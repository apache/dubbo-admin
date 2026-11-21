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

package service

import (
	"context"
	"encoding/json"
	"math"
	"testing"

	hessian "github.com/apache/dubbo-go-hessian2"
	javabig "github.com/dubbogo/gost/math/big"
	"github.com/stretchr/testify/require"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
)

func TestGenericInvokeScalars(t *testing.T) {
	for _, tc := range []struct {
		types []string
		raw   string
		want  any
	}{
		{[]string{"byte", "java.lang.Byte", "int8"}, "-128", int8(-128)},
		{[]string{"short", "java.lang.Short", "int16"}, "-32768", int16(-32768)},
		{[]string{"int", "java.lang.Integer", "int32", "rune"}, "2147483647", int32(math.MaxInt32)},
		{[]string{"long", "java.lang.Long", "int64"}, `"9223372036854775807"`, int64(math.MaxInt64)},
		{[]string{"long"}, "9.223372036854775807e18", int64(math.MaxInt64)},
		{[]string{"uint8"}, "255", uint8(255)},
		{[]string{"uint16"}, "65535", uint16(65535)},
		{[]string{"uint32"}, "4294967295", uint32(math.MaxUint32)},
		{[]string{"uint64"}, "42", uint64(42)},
		{[]string{"uint"}, "42", uint(42)},
		{[]string{"float", "java.lang.Float", "float32"}, "1.5", float32(1.5)},
		{[]string{"double", "java.lang.Double", "float64"}, "1.5", float64(1.5)},
		{[]string{"boolean", "java.lang.Boolean", "bool"}, "false", false},
		{[]string{"char", "java.lang.Character"}, `"文"`, uint16('文')},
		{[]string{"string", "java.lang.String"}, `"3"`, "3"},
		{[]string{"long[]", "[]int64", "[1]int64"}, `["9223372036854775807"]`, []int64{math.MaxInt64}},
		{[]string{"java.lang.Long[]"}, `[null, 2]`, []any{nil, int64(2)}},
		{[]string{"java.util.Map"}, `{"count":9.223372036854775807e18,"fraction":0.5}`, map[string]any{"count": int64(math.MaxInt64), "fraction": 0.5}},
		{[]string{"map[int32][]int64"}, `{"1":[2]}`, map[any]any{int32(1): []int64{2}}},
	} {
		for _, typ := range tc.types {
			t.Run(typ, func(t *testing.T) {
				args, err := decodeGenericInvokeArgs(&model.ServiceMethodDetailResp{ParameterTypes: []string{typ}}, []json.RawMessage{json.RawMessage(tc.raw)}, false)
				require.NoError(t, err)
				require.Equal(t, tc.want, args[0])
			})
		}
	}
	detail := &model.ServiceMethodDetailResp{ParameterTypes: []string{"int", "byte", "java.math.BigInteger"}}
	args, err := decodeGenericInvokeArgs(detail, []json.RawMessage{json.RawMessage("42"), json.RawMessage("255"), json.RawMessage(`"123456789012345678901234567890"`)}, true)
	require.NoError(t, err)
	require.Equal(t, int(42), args[0])
	require.Equal(t, byte(255), args[1])
	encoder := hessian.NewEncoder()
	require.NoError(t, encoder.Encode(args[2]))
	decoded, err := hessian.NewDecoder(encoder.Buffer()).Decode()
	require.NoError(t, err)
	require.Equal(t, "123456789012345678901234567890", decoded.(*javabig.Integer).String())
}

func TestGenericInvokeRejectsUnsafeValues(t *testing.T) {
	for _, tc := range []struct{ typ, raw, message string }{
		{"long", "9223372036854775808", "out of range"},
		{"int64", "9223372036854775806.9", "expected an integer"},
		{"uint64", `"18446744073709551615"`, "wire range"},
		{"float32", "1e39", "out of range"},
		{"float32", "16777217", "loses precision"},
		{"double", "9223372036854775807", "loses precision"},
		{"bool", `"true"`, "got string"},
		{"int", "null", "got null"},
		{"java.lang.String", "42", "got number"},
		{"[]int16", "[32768]", "arg0[0]: expected int16"},
		{"[2]int32", "[]", "got 0 elements"},
		{"Unknown", "42", "metadata unavailable"},
		{"java.util.Map", `{"x":[9223372036854775808]}`, `arg0["x"][0]`},
		{"map[int32]string", `{"1":"a","01":"b"}`, "duplicate converted map key"},
	} {
		t.Run(tc.typ, func(t *testing.T) {
			_, err := decodeGenericInvokeArgs(&model.ServiceMethodDetailResp{ParameterTypes: []string{tc.typ}}, []json.RawMessage{json.RawMessage(tc.raw)}, false)
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestGenericInvokeProviderOverloads(t *testing.T) {
	user := &meshproto.Method{Name: "echo", ParameterTypes: []string{"User"}, Parameters: []*meshproto.Parameter{{Name: "user", Type: "User"}}, ReturnType: "void"}
	number := &meshproto.Method{Name: "echo", ParameterTypes: []string{"long"}, ReturnType: "void"}
	stores := map[coremodel.ResourceKind]store.ResourceStore{}
	for _, kind := range []coremodel.ResourceKind{meshresource.ServiceProviderMetadataKind, meshresource.InstanceKind, meshresource.RPCInstanceKind} {
		s := memoryst.NewMemoryResourceStore(kind)
		require.NoError(t, s.Init(nil))
		stores[kind] = s
	}
	ctx := &testContext{rm: manager.NewResourceManager(&testRouter{stores: stores}, nil)}
	instance := meshresource.NewInstanceResourceWithAttributes("instance", "mesh")
	instance.Spec.AppName, instance.Spec.Ip, instance.Spec.RpcPort = "app", "127.0.0.1", 20880
	require.NoError(t, stores[instance.ResourceKind()].Add(instance))
	rpc := meshresource.NewRPCInstanceResourceWithAttributes(meshresource.BuildInstanceResName("app", "127.0.0.1", 20880), "mesh")
	rpc.Spec.Ip = "127.0.0.1"
	rpc.Spec.Endpoints = []*meshproto.Endpoint{{Protocol: "dubbo", Port: 20880}}
	require.NoError(t, stores[rpc.ResourceKind()].Add(rpc))
	meta := meshresource.NewServiceProviderMetadataResourceWithAttributes("app", "mesh")
	meta.Spec.ServiceName, meta.Spec.ProviderAppName = "Service", "app"
	meta.Spec.Methods = []*meshproto.Method{user, number}
	meta.Spec.Types = []*meshproto.Type{
		{Type: "User", Properties: map[string]string{"profile": "Profile", "roles": "java.util.Map<java.lang.String,Role[]>"}},
		{Type: "Profile", Properties: map[string]string{"id": "long"}},
		{Type: "java.util.Map<java.lang.String,Role[]>", Items: []string{"java.lang.String", "Role[]"}},
		{Type: "Role", Enums: []string{"ADMIN", "USER"}},
	}
	require.NoError(t, stores[meta.ResourceKind()].Add(meta))
	other := meshresource.NewServiceProviderMetadataResourceWithAttributes("other", "mesh")
	other.Spec.ServiceName, other.Spec.ProviderAppName = "Service", "another-app"
	other.Spec.Methods = []*meshproto.Method{{Name: "echo", ParameterTypes: []string{"bool"}, ReturnType: "void"}}
	require.NoError(t, stores[other.ResourceKind()].Add(other))

	var calls []genericInvocation
	original := invokeGenericServiceRPC
	t.Cleanup(func() { invokeGenericServiceRPC = original })
	invokeGenericServiceRPC = func(_ context.Context, inv genericInvocation) (any, error) {
		calls = append(calls, inv)
		return nil, nil
	}
	req := model.ServiceGenericInvokeReq{BaseServiceReq: model.BaseServiceReq{Mesh: "mesh", ServiceName: "Service"}, InstanceName: "instance", MethodName: "echo", TimeoutMs: 3000}
	invoke := func(signature, raw string) error {
		req.Signature = signature
		require.NoError(t, json.Unmarshal([]byte(raw), &req.Args))
		_, err := InvokeServiceGeneric(ctx, req)
		return err
	}
	require.NoError(t, invoke("long->void", `["9223372036854775807"]`))
	require.Equal(t, []string{"long"}, calls[0].ParameterTypes)
	require.Equal(t, int64(math.MaxInt64), calls[0].Args[0])
	require.NoError(t, invoke("User->void", `[{"profile":{"id":9223372036854775807},"roles":{"a":["ADMIN"]}}]`))
	require.Equal(t, map[string]any{"profile": map[string]any{"id": int64(math.MaxInt64)}, "roles": map[string]any{"a": []any{"ADMIN"}}}, calls[1].Args[0])
	for _, tc := range []struct{ sig, raw, message string }{
		{"long->void", `["9223372036854775808"]`, "out of range"},
		{"User->void", `[{"profile":{},"roles":{}}]`, "user.profile.id: expected long, got missing"},
		{"User->void", `[{"profile":{"id":1},"roles":{"a":["ROOT"]}}]`, `user.roles["a"][0]: expected Role, got string (allowed values: ADMIN, USER)`},
		{"User->void", "[42]", "expected User, got number"},
		{"long->void", "[{}]", "expected long, got object"},
		{"long->void", "[]", "expected 1 parameters"},
		{"bool->void", "[true]", "not found"},
	} {
		require.ErrorContains(t, invoke(tc.sig, tc.raw), tc.message)
	}
	require.Len(t, calls, 2)
}
