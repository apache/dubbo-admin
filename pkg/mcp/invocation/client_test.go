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

package invocation

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/registry"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	discovery "github.com/apache/dubbo-admin/pkg/config/discovery"
)

func TestRegistryReferenceUsesSafeCluster(t *testing.T) {
	extension.SetRegistry("zookeeper", registry.NewMockRegistry)
	for _, mode := range []string{constant.RegistryTypeInterface, constant.RegistryTypeService} {
		for _, wire := range []string{constant.DubboProtocol, constant.TriProtocol} {
			t.Run(mode+"/"+wire, func(t *testing.T) {
				config := &discovery.Config{
					ID:   "mcp-test-" + mode + "-" + wire,
					Type: discovery.Zookeeper,
					Address: discovery.AddressConfig{
						Registry: "zookeeper://127.0.0.1:2181",
					},
				}
				consumer := NewClient()
				operation := &meshproto.MCPOperation{
					ServiceName: "org.apache.dubbo.mcp.TestService" + mode + wire,
					MethodName:  "update",
					Protocol:    wire,
				}
				_, err := consumer.Call(context.Background(), "test-mesh", config,
					mode, operation, nil, time.Second)
				var callErr *CallError
				if !errors.As(err, &callErr) || callErr.Outcome != NotExecuted {
					t.Fatalf("no-provider call returned %v, want not-executed", err)
				}
				if len(consumer.refs) != 1 {
					t.Fatalf("created %d references, want one", len(consumer.refs))
				}
				for _, ref := range consumer.refs {
					if _, ok := ref.options.GetInvoker().(*safeInvoker); !ok {
						t.Fatalf("effective invoker is %T", ref.options.GetInvoker())
					}
				}
			})
		}
	}
}

func TestRegistryOptionsPreserveAddressConfiguration(t *testing.T) {
	registryURL, err := url.Parse("nacos://admin:secret@127.0.0.1:8848?" +
		"namespace=public&group=APP&timeout=7s&ttl=2m&simplified=true&" +
		"preferred=true&zone=zone-a&weight=11&accesskey=ak&secretkey=sk&" +
		"custom=value")
	if err != nil {
		t.Fatal(err)
	}
	opts, err := registryOptions(registryURL, constant.RegistryTypeService)
	if err != nil {
		t.Fatal(err)
	}
	config := registry.NewOptions(opts...).Registry
	if config.Address != "nacos://127.0.0.1:8848" ||
		config.Username != "admin" || config.Password != "secret" ||
		config.Namespace != "public" || config.Group != "APP" ||
		config.Timeout != "7s" || config.TTL != "2m0s" ||
		!config.Simplified || !config.Preferred ||
		config.Zone != "zone-a" || config.Weight != 11 ||
		config.RegistryType != constant.RegistryTypeService ||
		config.Params[constant.RegistryAccessKey] != "ak" ||
		config.Params[constant.RegistrySecretKey] != "sk" ||
		config.Params["custom"] != "value" {
		t.Fatalf("registry configuration lost: %+v", config)
	}
}

func TestInvalidInvocationNeverCreatesReference(t *testing.T) {
	client := NewClient()
	_, err := client.Call(context.Background(), "test-mesh",
		&discovery.Config{ID: "test", Type: discovery.Zookeeper},
		constant.RegistryTypeInterface,
		&meshproto.MCPOperation{ServiceName: "Test", MethodName: "update", Protocol: constant.TriProtocol},
		nil, 0)
	var callErr *CallError
	if !errors.As(err, &callErr) || callErr.Outcome != NotExecuted || len(client.refs) != 0 {
		t.Fatalf("invalid invocation returned %v and %d references", err, len(client.refs))
	}
}
