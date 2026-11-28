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
	"encoding/json"
	"errors"
	"fmt"
	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func goProviderEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestGenericInvokeGoProvider(t *testing.T) {
	if os.Getenv("RUN_DUBBO_GO_E2E") != "1" {
		t.Skip("set RUN_DUBBO_GO_E2E=1 and start go-v330 Provider")
	}
	// This test intentionally runs sequentially: observation counts require an isolated Provider.
	for _, protocol := range []string{"dubbo", "tri"} {
		t.Run(protocol, func(t *testing.T) {
			address := goProviderEnv("GO_PROVIDER_DUBBO_ADDR", "127.0.0.1:20880")
			if protocol == "tri" {
				address = goProviderEnv("GO_PROVIDER_TRIPLE_ADDR", "127.0.0.1:50051")
			}
			host, p, err := net.SplitHostPort(address)
			if err != nil {
				t.Fatal(err)
			}
			port, err := strconv.ParseInt(p, 10, 32)
			if err != nil {
				t.Fatal(err)
			}
			conn, err := net.DialTimeout("tcp", address, 3*time.Second)
			if err != nil {
				t.Fatalf("enabled GoProvider unreachable at %s: %v", address, err)
			}
			conn.Close()
			stores := make(map[coremodel.ResourceKind]store.ResourceStore)
			for _, kind := range []coremodel.ResourceKind{meshresource.InstanceKind, meshresource.ServiceProviderMetadataKind, meshresource.RPCInstanceKind} {
				s := memoryst.NewMemoryResourceStore(kind)
				if err := s.Init(nil); err != nil {
					t.Fatal(err)
				}
				stores[kind] = s
			}
			instance := meshresource.NewInstanceResourceWithAttributes("go-provider", "test-mesh")
			instance.Spec.AppName = "generic-go-v330"
			instance.Spec.Ip = host
			instance.Spec.RpcPort = port
			if err := stores[meshresource.InstanceKind].Add(instance); err != nil {
				t.Fatal(err)
			}
			metadata := meshresource.NewServiceProviderMetadataResourceWithAttributes("go-metadata", "test-mesh")
			metadata.Spec.ServiceName = "example.GoEchoService"
			metadata.Spec.Methods = []*meshproto.Method{
				{Name: "echo", ParameterTypes: []string{"java.lang.String"}, ReturnType: "java.lang.String"},
				{Name: "echoInt", ParameterTypes: []string{"int"}, ReturnType: "int"},
				{Name: "echoNested", ParameterTypes: []string{"example.Envelope"}, ReturnType: "example.Envelope"},
				{Name: "noArgs", ReturnType: "java.lang.String"},
				{Name: "echoArray", ParameterTypes: []string{"int[]"}, ReturnType: "int[]"},
				{Name: "attachments", ReturnType: "java.lang.String"},
				{Name: "delay", ParameterTypes: []string{"int"}, ReturnType: "java.lang.String"},
			}
			if err := stores[meshresource.ServiceProviderMetadataKind].Add(metadata); err != nil {
				t.Fatal(err)
			}
			rpcName := meshresource.BuildInstanceResName("generic-go-v330", host, port)
			rpc := meshresource.NewRPCInstanceResourceWithAttributes(rpcName, "test-mesh")
			rpc.Spec.Name = rpcName
			rpc.Spec.AppName = "generic-go-v330"
			rpc.Spec.Ip = host
			rpc.Spec.Port = port
			rpc.Spec.Endpoints = []*meshproto.Endpoint{{Protocol: protocol, Port: port}}
			if err := stores[meshresource.RPCInstanceKind].Add(rpc); err != nil {
				t.Fatal(err)
			}
			ctx := &genericServiceE2EContext{rm: manager.NewResourceManager(&genericServiceE2ERouter{stores: stores}, nil)}
			invoke := func(method, signature string, args string, timeout int64) (string, any) {
				t.Helper()
				var values []json.RawMessage
				if err := json.Unmarshal([]byte(args), &values); err != nil {
					t.Fatal(err)
				}
				req := model.ServiceGenericInvokeReq{BaseServiceReq: model.BaseServiceReq{Mesh: "test-mesh", ServiceName: "example.GoEchoService"}, InstanceName: "go-provider", MethodName: method, Signature: signature, Args: values, TimeoutMs: timeout, Attachments: map[string]string{"traceId": "go-e2e"}}
				if err := req.Validate(); err != nil {
					t.Fatal(err)
				}
				resp, err := InvokeServiceGeneric(ctx, req)
				if err != nil {
					var fieldErr *GenericInvokeArgumentError
					if errors.As(err, &fieldErr) && fieldErr.Kind == ArgumentErrorSignature {
						return "NotFoundError", nil
					}
					return fmt.Sprint(err), nil
				}
				if resp == nil {
					t.Fatal("nil response")
				}
				return "Success", resp.RawResult
			}
			for _, tc := range []struct{ method, signature, args, want string }{
				{"echo", "java.lang.String->java.lang.String", `["hello"]`, `"hello"`},
				{"echoInt", "int->int", `[42]`, `42`},
				{"noArgs", "->java.lang.String", `[]`, `"no-args"`},
				{"echoArray", "int[]->int[]", `[[1,2]]`, `[1,2]`},
				{"attachments", "->java.lang.String", `[]`, `"go-e2e"`},
			} {
				t.Run(tc.method, func(t *testing.T) {
					code, value := invoke(tc.method, tc.signature, tc.args, 3000)
					if code != "Success" {
						t.Fatalf("RPC failed: %s", code)
					}
					got, err := json.Marshal(value)
					if err != nil {
						t.Fatal(err)
					}
					if string(got) != tc.want {
						t.Fatalf("want %s, got %s", tc.want, got)
					}
				})
			}
			t.Run("echoNested", func(t *testing.T) {
				code, value := invoke("echoNested", "example.Envelope->example.Envelope", `[{"item":{"value":42}}]`, 3000)
				if code != "Success" {
					t.Fatalf("nested RPC failed: %s", code)
				}
				envelope, ok := value.(map[string]any)
				if !ok {
					t.Fatalf("nested result: want object, got %T (%#v)", value, value)
				}
				item, ok := envelope["item"].(map[string]any)
				if !ok {
					t.Fatalf("nested item: want object, got %T (%#v)", envelope["item"], envelope["item"])
				}

				kind := reflect.ValueOf(item["value"]).Kind()
				switch kind {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					if reflect.ValueOf(item["value"]).Int() != 42 {
						t.Fatalf("nested value: want integer 42, got %#v", item["value"])
					}
				default:
					t.Fatalf("nested value: want an integer type, got %T (%#v)", item["value"], item["value"])
				}
			})
			count := func() int64 {
				t.Helper()
				client := &http.Client{Timeout: 3 * time.Second}
				resp, err := client.Get(goProviderEnv("GO_PROVIDER_OBSERVATION_URL", "http://127.0.0.1:18090") + "/count")
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					t.Fatalf("observation HTTP %d", resp.StatusCode)
				}
				var value struct {
					Count int64 `json:"count"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&value); err != nil {
					t.Fatal(err)
				}
				return value.Count
			}
			before := count()
			code, _ := invoke("echo", "int->java.lang.String", `[1]`, 3000)
			if code != "NotFoundError" {
				t.Fatalf("wrong signature: expected signature rejection, got %s", code)
			}
			if after := count(); after != before {
				t.Fatalf("signature rejection reached Provider: %d -> %d", before, after)
			}
			started := time.Now()
			before = count()
			code, _ = invoke("delay", "int->java.lang.String", `[1000]`, 100)
			if code == "Success" {
				t.Fatal("timeout unexpectedly succeeded")
			}
			if elapsed := time.Since(started); elapsed > 3*time.Second {
				t.Fatalf("timeout took %s", elapsed)
			}
			if after := count(); after <= before {
				t.Fatal("timeout did not reach Provider; transport failure cannot pass as timeout")
			}
			// TODO(#1631/#1647): assert large-integer overflow rejection, including nested integer fields, once fixed. Do not accept rounded values.
			// TODO(#1632): assert business exception classification once defined; do not accept a transport error as a business exception.
		})
	}
}
