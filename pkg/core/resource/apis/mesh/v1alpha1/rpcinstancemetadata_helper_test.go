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

package v1alpha1

import (
	"encoding/json"
	"testing"

	"dubbo.apache.org/dubbo-go/v3/common"
	metadatainfo "dubbo.apache.org/dubbo-go/v3/metadata/info"
	"github.com/stretchr/testify/require"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
)

func TestParseRPCInstanceMetadataNormalizesServiceKey(t *testing.T) {
	res, err := ParseRPCInstanceMetadata("mesh-1", "orders", "rev-2",
		`{"services":{"/com.example.OrderService:v1:tri":{"port":50051},"orders/com.example.OrderService:v1:dubbo":{"name":"com.example.OrderService","protocol":"dubbo"}}}`)
	require.NoError(t, err)
	require.Equal(t, "orders@rev-2", res.Name)
	require.Equal(t, "mesh-1", res.Mesh)
	require.Equal(t, "orders", res.Spec.App)
	require.Equal(t, "rev-2", res.Spec.Revision)
	export := res.Spec.Services["/com.example.OrderService:v1:tri"]
	require.Equal(t, "com.example.OrderService", export.Name)
	require.Equal(t, "v1", export.Version)
	require.Equal(t, "tri", export.Protocol)
	grouped := res.Spec.Services["orders/com.example.OrderService:v1:dubbo"]
	require.Equal(t, "orders", grouped.Group)
	require.Equal(t, "v1", grouped.Version)
}

func TestParseRPCInstanceMetadataSDKMetadata(t *testing.T) {
	const name = "com.example.OrderService"
	tests := []struct {
		name    string
		group   string
		version string
		key     string
	}{
		{"group and version", "orders", "v1", "orders/" + name + ":v1:tri"},
		{"empty group", "", "v1", name + ":v1:tri"},
		{"empty version", "orders", "", "orders/" + name + ":tri"},
		{"empty group and version", "", "", name + ":tri"},
		{"default version", "orders", "0.0.0", "orders/" + name + ":tri"},
		{"empty group and default version", "", "0.0.0", name + ":tri"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, omitted := range []string{"none", "name", "protocol", "all identity fields"} {
				t.Run(omitted, func(t *testing.T) {
					url := common.NewURLWithOptions(
						common.WithInterface(name),
						common.WithPath("/"+name),
						common.WithProtocol("tri"),
						common.WithPort("50051"),
						common.WithMethods([]string{"GetOrder"}),
						common.WithParamsValue("group", tt.group),
						common.WithParamsValue("version", tt.version),
						common.WithParamsValue("timeout", "3000"),
					)
					metadata := metadatainfo.NewAppMetadataInfo("orders")
					metadata.AddService(url)
					revision := metadata.CalAndGetRevision()
					key := common.MatchKey(common.ServiceKey(name, tt.group, tt.version), "tri")
					require.Equal(t, tt.key, key)
					service := metadata.Services[key]
					require.NotNil(t, service)
					expected := &meshproto.ServiceInfo{
						Name: name, Group: tt.group, Version: tt.version, Protocol: "tri",
						Port: int64(service.Port), Path: service.Path, Params: service.Params,
					}
					switch omitted {
					case "name":
						service.Name = ""
					case "protocol":
						service.Protocol = ""
					case "all identity fields":
						service.Name, service.Group, service.Version, service.Protocol = "", "", "", ""
						if tt.version == "0.0.0" {
							expected.Version = ""
						}
					}
					data, err := json.Marshal(metadata)
					require.NoError(t, err)

					res, err := ParseRPCInstanceMetadata("mesh-1", "orders", revision, string(data))
					require.NoError(t, err)
					require.Equal(t, "orders@"+revision, res.Name)
					require.Equal(t, "mesh-1", res.Mesh)
					require.Equal(t, "orders", res.Spec.App)
					require.Equal(t, revision, res.Spec.Revision)
					require.Len(t, res.Spec.Services, 1)
					require.Equal(t, expected, res.Spec.Services[key])
				})
			}
		})
	}
}

func TestParseRPCInstanceMetadataEmptyServices(t *testing.T) {
	for _, data := range []string{`{}`, `{"services":{}}`, `{"services":null}`} {
		t.Run(data, func(t *testing.T) {
			res, err := ParseRPCInstanceMetadata("mesh-1", "orders", "rev-2", data)
			require.NoError(t, err)
			require.Equal(t, "orders", res.Spec.App)
			require.Equal(t, "rev-2", res.Spec.Revision)
			require.Empty(t, res.Spec.Services)
		})
	}
}

func TestParseRPCInstanceMetadataRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"empty input", ""},
		{"invalid JSON", "{"},
		{"null metadata", "null"},
		{"array metadata", "[]"},
		{"string metadata", `"metadata"`},
		{"services array", `{"services":[]}`},
		{"null service", `{"services":{"com.example.OrderService:tri":null}}`},
		{"invalid service type", `{"services":{"com.example.OrderService:tri":"service"}}`},
		{"invalid port type", `{"services":{"com.example.OrderService:tri":{"port":"50051"}}}`},
		{"empty key", `{"services":{"":{}}}`},
		{"missing protocol", `{"services":{"com.example.OrderService":{}}}`},
		{"empty protocol", `{"services":{"com.example.OrderService:":{}}}`},
		{"empty name", `{"services":{":tri":{}}}`},
		{"empty grouped name", `{"services":{"orders/:v1:tri":{}}}`},
		{"extra identity separator", `{"services":{"orders/com.example.OrderService:v1:extra:tri":{}}}`},
		{"invalid key with complete identity", `{"services":{"invalid":{"name":"com.example.OrderService","group":"orders","version":"v1","protocol":"tri"}}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ParseRPCInstanceMetadata("mesh-1", "orders", "rev-2", tt.data)
			require.Error(t, err)
			require.Nil(t, res)
		})
	}
}

func TestParseRPCInstanceMetadataRejectsIdentityConflicts(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"application", `{"app":"different","services":{}}`},
		{"revision", `{"revision":"different","services":{}}`},
		{"service name", `{"services":{"orders/com.example.OrderService:v1:tri":{"name":"com.example.OtherService","group":"orders","version":"v1","protocol":"tri"}}}`},
		{"service group", `{"services":{"orders/com.example.OrderService:v1:tri":{"name":"com.example.OrderService","group":"other","version":"v1","protocol":"tri"}}}`},
		{"service version", `{"services":{"orders/com.example.OrderService:v1:tri":{"name":"com.example.OrderService","group":"orders","version":"v2","protocol":"tri"}}}`},
		{"service protocol", `{"services":{"orders/com.example.OrderService:v1:tri":{"name":"com.example.OrderService","group":"orders","version":"v1","protocol":"dubbo"}}}`},
		{"group absent from key", `{"services":{"com.example.OrderService:tri":{"group":"orders"}}}`},
		{"version absent from key", `{"services":{"com.example.OrderService:tri":{"version":"v1"}}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ParseRPCInstanceMetadata("mesh-1", "orders", "rev-2", tt.data)
			require.ErrorContains(t, err, "does not match")
			require.Nil(t, res)
		})
	}
}
