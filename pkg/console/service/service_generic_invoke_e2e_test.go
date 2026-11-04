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
	"fmt"
	"os"
	"testing"
	"time"

	dubboconstant "dubbo.apache.org/dubbo-go/v3/common/constant"
	hessian "github.com/apache/dubbo-go-hessian2"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	appcfg "github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/console/counter"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
)

// These tests require the Java EchoService provider. Keep it running with:
//   dubbo://127.0.0.1:20880
//   tri://127.0.0.1:50051
// Enable tests explicitly using RUN_DUBBO_E2E=1.

const (
	genericE2EMeshName     = "test-mesh"
	genericE2EInstanceName = "provider-1"
	genericE2EAppName      = "generic-e2e-provider"
	genericE2EServiceName  = "example.EchoService"
	genericE2EProviderIP   = "127.0.0.1"
)

// Use an independently named test context to avoid coupling these E2E tests
// to helpers declared in other *_test.go files.
type genericServiceE2EContext struct {
	rm manager.ResourceManager
}

func (c *genericServiceE2EContext) ResourceManager() manager.ResourceManager {
	return c.rm
}
func (c *genericServiceE2EContext) AppContext() context.Context            { return context.Background() }
func (c *genericServiceE2EContext) CounterManager() counter.CounterManager { return nil }
func (c *genericServiceE2EContext) Config() appcfg.AdminConfig             { return appcfg.AdminConfig{} }
func (c *genericServiceE2EContext) LockManager() lock.Lock                 { return nil }
func (c *genericServiceE2EContext) RuleVersioning() *versioning.Service    { return nil }

type genericServiceE2ERouter struct {
	stores map[coremodel.ResourceKind]store.ResourceStore
}

func (r *genericServiceE2ERouter) ResourceRoute(res coremodel.Resource) (store.ResourceStore, error) {
	return r.ResourceKindRoute(res.ResourceKind())
}

func (r *genericServiceE2ERouter) ResourceKindRoute(kind coremodel.ResourceKind) (store.ResourceStore, error) {
	s, ok := r.stores[kind]
	if !ok {
		return nil, fmt.Errorf("E2E resource store not registered for %s", kind)
	}
	return s, nil
}

// The Service-layer test uses real Metadata/Instance/RPCInstance discovery,
// but runs against isolated in-memory stores rather than external registries.
func newGenericServiceJavaE2EContext(t *testing.T, protocol string, port int64, explicitSerialization bool) *genericServiceE2EContext {
	t.Helper()

	stores := make(map[coremodel.ResourceKind]store.ResourceStore)
	for _, kind := range []coremodel.ResourceKind{
		meshresource.InstanceKind,
		meshresource.ServiceProviderMetadataKind,
		meshresource.RPCInstanceKind,
	} {
		s := memoryst.NewMemoryResourceStore(kind)
		if err := s.Init(nil); err != nil {
			t.Fatalf("init %s store: %v", kind, err)
		}
		stores[kind] = s
	}

	instance := meshresource.NewInstanceResourceWithAttributes(
		genericE2EInstanceName, genericE2EMeshName,
	)
	instance.Spec.AppName = genericE2EAppName
	instance.Spec.Ip = genericE2EProviderIP
	instance.Spec.RpcPort = port
	if err := stores[meshresource.InstanceKind].Add(instance); err != nil {
		t.Fatalf("add instance: %v", err)
	}

	metadata := meshresource.NewServiceProviderMetadataResourceWithAttributes(
		"echo-metadata", genericE2EMeshName,
	)
	metadata.Spec.ServiceName = genericE2EServiceName
	metadata.Spec.Methods = []*meshproto.Method{{
		Name:           "echo",
		ParameterTypes: []string{"java.lang.String"},
		ReturnType:     "java.lang.String",
	}}
	if err := stores[meshresource.ServiceProviderMetadataKind].Add(metadata); err != nil {
		t.Fatalf("add provider metadata: %v", err)
	}

	rpcName := meshresource.BuildInstanceResName(
		genericE2EAppName, genericE2EProviderIP, port,
	)
	rpcInstance := meshresource.NewRPCInstanceResourceWithAttributes(
		rpcName, genericE2EMeshName,
	)
	rpcInstance.Spec.Name = rpcName
	rpcInstance.Spec.AppName = genericE2EAppName
	rpcInstance.Spec.Ip = genericE2EProviderIP
	rpcInstance.Spec.Port = port
	// Unset serialization covers the production Hessian2 fallback.
	if explicitSerialization {
		rpcInstance.Spec.Serialization = dubboconstant.Hessian2Serialization
	}
	rpcInstance.Spec.Endpoints = []*meshproto.Endpoint{{
		Protocol: protocol,
		Port:     port,
	}}
	if err := stores[meshresource.RPCInstanceKind].Add(rpcInstance); err != nil {
		t.Fatalf("add RPC instance: %v", err)
	}

	return &genericServiceE2EContext{
		rm: manager.NewResourceManager(
			&genericServiceE2ERouter{stores: stores}, nil,
		),
	}
}

func requireGenericJavaE2E(t *testing.T) {
	t.Helper()
	if os.Getenv("RUN_DUBBO_E2E") != "1" {
		t.Skip("set RUN_DUBBO_E2E=1 and start the Java EchoService provider")
	}
}

// Layer 1: Direct Go SDK -> Java Provider, without Admin Service or Metadata.
func runGenericSDKJavaE2E(t *testing.T, protocol string, port int64) {
	t.Helper()
	requireGenericJavaE2E(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ctx = context.WithValue(ctx, dubboconstant.AttachmentKey, map[string]any{
		"traceId": "abc",
	})

	got, err := invokeGenericServiceRPC(ctx, genericInvocation{
		URL:            fmt.Sprintf("%s://%s:%d", protocol, genericE2EProviderIP, port),
		Protocol:       protocol,
		Serialization:  dubboconstant.Hessian2Serialization,
		ServiceName:    genericE2EServiceName,
		MethodName:     "echo",
		ParameterTypes: []string{"java.lang.String"},
		Args:           []hessian.Object{"hello"},
	})
	if err != nil {
		t.Fatalf("%s SDK generic invoke failed: %v", protocol, err)
	}
	if value, ok := got.(string); !ok || value != "hello" {
		t.Fatalf("%s SDK result: want %q, got %#v (%T)", protocol, "hello", got, got)
	}

	t.Logf("%s SDK RPC result: %#v", protocol, got)
}

func TestGenericInvokeRealJavaProvider(t *testing.T) {
	runGenericSDKJavaE2E(t, dubboconstant.DubboProtocol, 20880)
}

func TestGenericInvokeRealJavaProviderTriple(t *testing.T) {
	runGenericSDKJavaE2E(t, dubboconstant.TriProtocol, 50051)
}

// Layer 2: Admin Service -> Metadata and target resolution -> real SDK -> Java Provider.
func runInvokeServiceGenericJavaE2E(t *testing.T, protocol string, port int64, explicitSerialization bool) {
	t.Helper()
	requireGenericJavaE2E(t)

	ctx := newGenericServiceJavaE2EContext(t, protocol, port, explicitSerialization)
	req := model.ServiceGenericInvokeReq{
		BaseServiceReq: model.BaseServiceReq{
			Mesh:        genericE2EMeshName,
			ServiceName: genericE2EServiceName,
		},
		InstanceName: genericE2EInstanceName,
		MethodName:   "echo",
		Signature:    "java.lang.String->java.lang.String",
		Args:         []json.RawMessage{json.RawMessage(`"hello"`)},
		TimeoutMs:    0,
		Attachments:  map[string]string{"traceId": "abc"},
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("validate generic invoke request: %v", err)
	}
	if req.TimeoutMs != model.DefaultServiceGenericInvokeTimeoutMs {
		t.Fatalf("default timeout: want %d ms, got %d ms",
			model.DefaultServiceGenericInvokeTimeoutMs, req.TimeoutMs)
	}

	resp, err := InvokeServiceGeneric(ctx, req)
	if err != nil {
		t.Fatalf("%s Service generic invoke failed: %v", protocol, err)
	}
	if resp == nil {
		t.Fatalf("%s Service generic invoke returned nil response", protocol)
	}
	if resp.RawResult != "hello" {
		t.Fatalf("%s Service result: want %q, got %#v", protocol, "hello", resp.RawResult)
	}
	if resp.ElapsedMs < 0 {
		t.Fatalf("%s Service elapsedMs is negative: %d", protocol, resp.ElapsedMs)
	}

	t.Logf("%s Service RPC result: %#v", protocol, resp.RawResult)
	t.Logf("%s Service RPC elapsed: %d ms", protocol, resp.ElapsedMs)
	// Confirm traceId=abc in the Java Provider's business logs separately.
}

// Verify the default Hessian2 fallback with omitted serialization metadata.
func TestInvokeServiceGenericRealJavaProvider(t *testing.T) {
	runInvokeServiceGenericJavaE2E(t, dubboconstant.DubboProtocol, 20880, false)
}

func TestInvokeServiceGenericRealJavaProviderTriple(t *testing.T) {
	runInvokeServiceGenericJavaE2E(t, dubboconstant.TriProtocol, 50051, true)
}
