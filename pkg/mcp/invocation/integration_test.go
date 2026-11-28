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
	"fmt"
	"net"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/server"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	discovery "github.com/apache/dubbo-admin/pkg/config/discovery"
)

type AttemptProvider struct{ attempts atomic.Int32 }

func (p *AttemptProvider) Call(name string) (string, error) {
	p.attempts.Add(1)
	if name == "fail" {
		return "", errors.New("provider rejected the request")
	}
	if name == "slow" {
		time.Sleep(100 * time.Millisecond)
	}
	return "ok " + name, nil
}

func TestRegistryBackedWireAttempts(t *testing.T) {
	for _, wire := range []string{constant.DubboProtocol, constant.TriProtocol} {
		t.Run(wire, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			port := listener.Addr().(*net.TCPAddr).Port
			listener.Close()

			serviceName := "org.apache.dubbo.mcp.AttemptProvider" + wire
			provider := &AttemptProvider{}
			var wireOption protocol.Option = protocol.WithDubbo()
			if wire == constant.TriProtocol {
				wireOption = protocol.WithTriple()
			}
			srv, err := server.NewServer(
				server.WithServerProtocol(wireOption, protocol.WithPort(port)),
				server.WithServerNotRegister(),
			)
			if err != nil {
				t.Fatal(err)
			}
			if err := srv.RegisterService(provider, server.WithInterface(serviceName),
				server.WithSerialization(constant.Hessian2Serialization), server.WithNotRegister()); err != nil {
				t.Fatal(err)
			}
			export := srv.GetServiceOptionsByInterfaceName(serviceName)
			if export == nil {
				t.Fatal("service registration missing")
			}
			if err := export.Export(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(export.Unexport)

			registries := make(chan *registry.MockRegistry, 1)
			extension.SetRegistry(constant.ZookeeperKey, func(url *common.URL) (registry.Registry, error) {
				reg, err := registry.NewMockRegistry(url)
				if err == nil {
					registries <- reg.(*registry.MockRegistry)
				}
				return reg, err
			})
			config := &discovery.Config{ID: "wire-" + wire, Type: discovery.Zookeeper, Address: discovery.AddressConfig{
				Registry: fmt.Sprintf("zookeeper://127.0.0.1:%d", port),
			}}
			op := &meshproto.MCPOperation{
				ServiceName: serviceName, MethodName: "Call", Protocol: wire,
				ParameterTypes: []string{"java.lang.String"},
			}
			consumer := NewClient()
			_, err = consumer.Call(context.Background(), "wire-mesh", config, constant.RegistryTypeInterface, op, []any{"hello"}, 2*time.Second)
			var callErr *CallError
			if !errors.As(err, &callErr) || callErr.Outcome != NotExecuted {
				t.Fatalf("call before discovery returned %v", err)
			}
			var reg *registry.MockRegistry
			select {
			case reg = <-registries:
			case <-time.After(2 * time.Second):
				t.Fatal("registry was not created")
			}
			providerURL := common.NewURLWithOptions(
				common.WithProtocol(wire), common.WithIp("127.0.0.1"),
				common.WithPort(strconv.Itoa(port)), common.WithPath(serviceName),
				common.WithMethods([]string{"Call"}),
				common.WithParamsValue(constant.ClusterKey, safeClusterName),
				common.WithParamsValue(constant.RetriesKey, "0"),
			)
			publish := func(providerURL *common.URL) {
				t.Helper()
				done := make(chan struct{})
				go func() {
					reg.MockEvents([]*registry.ServiceEvent{{Action: remoting.EventTypeAdd, Service: providerURL.Clone()}})
					close(done)
				}()
				select {
				case <-done:
				case <-time.After(3 * time.Second):
					t.Fatal("registry event not consumed")
				}
			}
			publish(providerURL)
			safe := consumer.refs[referenceKey{
				mesh: "wire-mesh", registryID: config.ID, registryURL: config.Address.Registry,
				mode: constant.RegistryTypeInterface, service: serviceName, protocol: wire,
			}].options.GetInvoker().(*safeInvoker)
			deadline := time.Now().Add(3 * time.Second)
			for len(safe.dir.List(invocation.NewRPCInvocation(
				constant.Generic, []any{"Call", []string{"java.lang.String"}, []any{"hello"}}, nil,
			))) == 0 {
				if time.Now().After(deadline) {
					t.Fatal("provider was not discovered")
				}
				time.Sleep(10 * time.Millisecond)
			}
			value, err := consumer.Call(context.Background(), "wire-mesh", config, constant.RegistryTypeInterface, op, []any{"hello"}, 2*time.Second)
			if err != nil || value != "ok hello" || provider.attempts.Load() != 1 {
				t.Fatalf("wire call value=%v error=%v attempts=%d", value, err, provider.attempts.Load())
			}

			// A governance update may replace the reference's effective URL
			// after its cluster has already been constructed.
			referenceURL := safe.dir.GetURL().SubURL
			for _, setting := range []struct{ name, key, value string }{
				{"cluster", constant.ClusterKey, constant.ClusterKeyBroadcast},
				{"retries", constant.RetriesKey, "2"},
				{"method retries", "methods.Call.retries", "1"},
			} {
				t.Run(setting.name, func(t *testing.T) {
					previous := referenceURL.GetParam(setting.key, "")
					referenceURL.SetParam(setting.key, setting.value)
					_, err := consumer.Call(context.Background(), "wire-mesh", config, constant.RegistryTypeInterface, op, []any{"hello"}, 2*time.Second)
					if !errors.As(err, &callErr) || callErr.Outcome != NotExecuted || provider.attempts.Load() != 1 {
						t.Fatalf("unsafe call error=%v attempts=%d", err, provider.attempts.Load())
					}
					referenceURL.SetParam(setting.key, previous)
				})
			}
			value, err = consumer.Call(context.Background(), "wire-mesh", config, constant.RegistryTypeInterface, op, []any{"hello"}, 2*time.Second)
			if err != nil || value != "ok hello" || provider.attempts.Load() != 2 {
				t.Fatalf("recovered wire call value=%v error=%v attempts=%d", value, err, provider.attempts.Load())
			}
			_, err = consumer.Call(context.Background(), "wire-mesh", config, constant.RegistryTypeInterface, op, []any{"fail"}, 2*time.Second)
			if !errors.As(err, &callErr) || callErr.Outcome != RemoteError || provider.attempts.Load() != 3 {
				t.Fatalf("provider error=%v attempts=%d", err, provider.attempts.Load())
			}
			_, err = consumer.Call(context.Background(), "wire-mesh", config, constant.RegistryTypeInterface, op, []any{"slow"}, 20*time.Millisecond)
			if !errors.As(err, &callErr) || callErr.Outcome != UnknownOutcome || provider.attempts.Load() != 4 {
				t.Fatalf("timed-out call error=%v attempts=%d", err, provider.attempts.Load())
			}
		})
	}
}
