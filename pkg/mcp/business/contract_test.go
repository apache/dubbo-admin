/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package business

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

func TestResolveOperationAndRestoreArguments(t *testing.T) {
	operation := testOperation()
	bindings := DefaultParameterBindings(operation.ParameterTypes)
	bindings[0].Alias = "tenantId"
	bindings[1].Alias = "request"

	contract, err := ResolveOperation("mesh-1", operation, bindings, []*meshresource.ServiceProviderMetadataResource{
		testProviderMetadata("provider-a", map[string]string{
			"id":    "long",
			"items": "java.util.List<java.lang.Long>",
		}),
	})
	require.NoError(t, err)
	require.NotEmpty(t, contract.InputFingerprint)
	require.NotEmpty(t, contract.OutputFingerprint)

	args, err := RestoreArguments(contract, map[string]any{
		"request": map[string]any{
			"id":    "9223372036854775807",
			"items": []any{"1", "2"},
		},
		"tenantId": "tenant-a",
	})
	require.NoError(t, err)
	require.Equal(t, "tenant-a", args[0])
	request := args[1].(map[string]any)
	require.Equal(t, "9223372036854775807", request["id"])
	require.Equal(t, []any{"1", "2"}, request["items"])
}

func TestResolveOperationRejectsUnknownProperty(t *testing.T) {
	operation := testOperation()
	contract, err := ResolveOperation("mesh-1", operation, nil, []*meshresource.ServiceProviderMetadataResource{
		testProviderMetadata("provider-a", map[string]string{"id": "long"}),
	})
	require.NoError(t, err)

	_, err = RestoreArguments(contract, map[string]any{
		"arg0": "tenant-a",
		"arg1": map[string]any{"unknown": true},
	})
	require.ErrorContains(t, err, "arguments do not match input schema")
}

func TestResolveOperationRejectsConflictingDefinitions(t *testing.T) {
	operation := testOperation()
	_, err := ResolveOperation("mesh-1", operation, nil, []*meshresource.ServiceProviderMetadataResource{
		testProviderMetadata("provider-a", map[string]string{"id": "long"}),
		testProviderMetadata("provider-b", map[string]string{"id": "java.lang.String"}),
	})
	require.ErrorContains(t, err, "provider definitions disagree")
}

func TestResolveOperationFingerprintsIgnoreProviderAndPropertyOrder(t *testing.T) {
	operation := testOperation()
	first := testProviderMetadata("provider-a", map[string]string{"id": "long", "name": "java.lang.String"})
	second := testProviderMetadata("provider-b", map[string]string{"name": "java.lang.String", "id": "long"})
	before, err := ResolveOperation("mesh-1", operation, nil, []*meshresource.ServiceProviderMetadataResource{first, second})
	require.NoError(t, err)
	after, err := ResolveOperation("mesh-1", operation, nil, []*meshresource.ServiceProviderMetadataResource{second, first})
	require.NoError(t, err)
	require.Equal(t, before.InputFingerprint, after.InputFingerprint)
	require.Equal(t, before.OutputFingerprint, after.OutputFingerprint)

	second.Spec.Types[2].Properties["status"] = "java.lang.String"
	_, err = ResolveOperation("mesh-1", operation, nil, []*meshresource.ServiceProviderMetadataResource{first, second})
	require.ErrorContains(t, err, "provider definitions disagree")
	changed, err := ResolveOperation("mesh-1", operation, nil, []*meshresource.ServiceProviderMetadataResource{second})
	require.NoError(t, err)
	require.Equal(t, before.InputFingerprint, changed.InputFingerprint)
	require.NotEqual(t, before.OutputFingerprint, changed.OutputFingerprint)
}

func TestResolveOperationRejectsAmbiguousMethodAndRawCollection(t *testing.T) {
	operation := testOperation()
	provider := testProviderMetadata("provider-a", map[string]string{"id": "long"})
	provider.Spec.Methods = append(provider.Spec.Methods, &meshproto.Method{
		Name:           operation.MethodName,
		ParameterTypes: operation.ParameterTypes,
		ReturnType:     "java.lang.String",
	})
	_, err := ResolveOperation("mesh-1", operation, nil, []*meshresource.ServiceProviderMetadataResource{provider})
	require.ErrorContains(t, err, "conflicting return types")

	provider.Spec.Methods = provider.Spec.Methods[:1]
	provider.Spec.Types[0].Properties["raw"] = "java.util.List"
	provider.Spec.Types = append(provider.Spec.Types, &meshproto.Type{
		Type:  "java.util.List",
		Items: []string{"java.lang.String"},
	})
	_, err = ResolveOperation("mesh-1", operation, nil, []*meshresource.ServiceProviderMetadataResource{provider})
	require.ErrorContains(t, err, "not supported")
}

func TestResolveOperationRejectsInvalidReturnContract(t *testing.T) {
	provider := testProviderMetadata("provider-a", map[string]string{"id": "long"})
	provider.Spec.Methods[0].ReturnType = "java.util.Map<java.lang.Integer,java.lang.String>"
	provider.Spec.Types = append(provider.Spec.Types, &meshproto.Type{
		Type:  provider.Spec.Methods[0].ReturnType,
		Items: []string{"java.lang.Integer", "java.lang.String"},
	})
	_, err := ResolveOperation("mesh-1", testOperation(), nil, []*meshresource.ServiceProviderMetadataResource{provider})
	require.ErrorContains(t, err, "non-string key")
}

func TestRestoreArgumentsRejectsOutOfRangeNestedLong(t *testing.T) {
	contract, err := ResolveOperation("mesh-1", testOperation(), nil, []*meshresource.ServiceProviderMetadataResource{
		testProviderMetadata("provider-a", map[string]string{"id": "long"}),
	})
	require.NoError(t, err)
	_, err = RestoreArguments(contract, map[string]any{
		"arg0": "tenant-a",
		"arg1": map[string]any{"id": "9223372036854775808"},
	})
	require.ErrorContains(t, err, "invalid long value")
}

func TestValidateParameterBindings(t *testing.T) {
	parameterTypes := []string{"java.lang.String", "int"}
	require.NoError(t, ValidateParameterBindings(DefaultParameterBindings(parameterTypes), parameterTypes))

	require.Error(t, ValidateParameterBindings([]*meshproto.MCPParameterBinding{
		{Position: 0, Alias: "same"},
		{Position: 1, Alias: "same"},
	}, parameterTypes))
	require.Error(t, ValidateParameterBindings([]*meshproto.MCPParameterBinding{
		{Position: 0, Alias: "arg0"},
		{Position: 0, Alias: "arg1"},
	}, parameterTypes))
}

func testOperation() *meshproto.MCPOperation {
	return &meshproto.MCPOperation{
		ServiceName:    "com.example.OrderService",
		Group:          "trade",
		Version:        "1.0.0",
		MethodName:     "createOrder",
		ParameterTypes: []string{"java.lang.String", "com.example.OrderRequest"},
		Protocol:       "dubbo",
	}
}

func testProviderMetadata(name string, requestProperties map[string]string) *meshresource.ServiceProviderMetadataResource {
	resource := meshresource.NewServiceProviderMetadataResourceWithAttributes(name, "mesh-1")
	resource.ObjectMeta = metav1.ObjectMeta{Name: name}
	resource.Mesh = "mesh-1"
	resource.Spec = &meshproto.ServiceProviderMetadata{
		ServiceName:     "com.example.OrderService",
		Group:           "trade",
		Version:         "1.0.0",
		ProviderAppName: name,
		Methods: []*meshproto.Method{{
			Name:           "createOrder",
			ParameterTypes: []string{"java.lang.String", "com.example.OrderRequest"},
			ReturnType:     "com.example.Order",
		}},
		Types: []*meshproto.Type{
			{
				Type:       "com.example.OrderRequest",
				Properties: requestProperties,
			},
			{
				Type:  "java.util.List<java.lang.Long>",
				Items: []string{"java.lang.Long"},
			},
			{
				Type:       "com.example.Order",
				Properties: map[string]string{"id": "long"},
			},
		},
	}
	return resource
}
