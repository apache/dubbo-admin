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
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
)

func TestServiceGenericInvokeHTTPRealJavaProvider(t *testing.T) {
	if os.Getenv("RUN_DUBBO_E2E") != "1" {
		t.Skip("requires Java Dubbo Provider on 127.0.0.1:20880")
	}
	stores := make(map[coremodel.ResourceKind]store.ResourceStore)

	for _, kind := range []coremodel.ResourceKind{
		meshresource.InstanceKind,
		meshresource.ServiceProviderMetadataKind,
		meshresource.RPCInstanceKind,
	} {
		s := memoryst.NewMemoryResourceStore(kind)
		if err := s.Init(nil); err != nil {
			t.Fatal(err)
		}
		stores[kind] = s
	}

	instance := meshresource.NewInstanceResourceWithAttributes(
		"provider-1", "test-mesh",
	)
	instance.Spec.AppName = "generic-e2e-provider"
	instance.Spec.Ip = "127.0.0.1"
	instance.Spec.RpcPort = 20880

	if err := stores[meshresource.InstanceKind].Add(instance); err != nil {
		t.Fatal(err)
	}

	metadata := meshresource.NewServiceProviderMetadataResourceWithAttributes(
		"echo-metadata", "test-mesh",
	)
	metadata.Spec.ServiceName = "example.EchoService"
	metadata.Spec.Methods = []*meshproto.Method{
		{
			Name:           "echo",
			ParameterTypes: []string{"java.lang.String"},
			ReturnType:     "java.lang.String",
		},
	}

	if err := stores[meshresource.ServiceProviderMetadataKind].Add(metadata); err != nil {
		t.Fatal(err)
	}

	rpcName := meshresource.BuildInstanceResName(
		"generic-e2e-provider",
		"127.0.0.1",
		20880,
	)

	rpcInstance := meshresource.NewRPCInstanceResourceWithAttributes(
		rpcName, "test-mesh",
	)
	rpcInstance.Spec.Name = rpcName
	rpcInstance.Spec.AppName = "generic-e2e-provider"
	rpcInstance.Spec.Ip = "127.0.0.1"
	rpcInstance.Spec.Port = 20880
	rpcInstance.Spec.Endpoints = []*meshproto.Endpoint{
		{
			Protocol: "dubbo",
			Port:     20880,
		},
	}

	if err := stores[meshresource.RPCInstanceKind].Add(rpcInstance); err != nil {
		t.Fatal(err)
	}

	ctx := &genericHTTPTestContext{
		rm: manager.NewResourceManager(
			&genericHTTPTestRouter{stores: stores},
			nil,
		),
	}

	router := gin.New()
	router.POST("/test/invoke", ServiceGenericInvoke(ctx))

	body := `{
		"mesh": "test-mesh",
		"serviceName": "example.EchoService",
		"instanceName": "provider-1",
		"methodName": "echo",
		"signature": "java.lang.String->java.lang.String",
		"args": ["hello"],
		"timeoutMs": 0,
		"attachments": {
			"traceId": "abc"
		}
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/test/invoke",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected HTTP 200, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			RawResult string `json:"rawResult"`
			ElapsedMs int64  `json:"elapsedMs"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "Success" {
		t.Fatalf(
			"expected Success, got %s: %s",
			response.Code,
			recorder.Body.String(),
		)
	}
	if response.Data.RawResult != "hello" {
		t.Fatalf(
			"expected hello, got %q",
			response.Data.RawResult,
		)
	}

	t.Logf("HTTP response: %s", recorder.Body.String())
}

func TestServiceGenericInvokeHTTPRealJavaProviderTriple(t *testing.T) {
	if os.Getenv("RUN_DUBBO_E2E") != "1" {
		t.Skip("requires Triple Java Provider on 127.0.0.1:50051")
	}

	const (
		meshName     = "test-mesh"
		instanceName = "provider-1"
		appName      = "generic-e2e-provider"
		serviceName  = "example.EchoService"
		providerIP   = "127.0.0.1"
		providerPort = 50051
	)

	stores := make(map[coremodel.ResourceKind]store.ResourceStore)

	for _, kind := range []coremodel.ResourceKind{
		meshresource.InstanceKind,
		meshresource.ServiceProviderMetadataKind,
		meshresource.RPCInstanceKind,
	} {
		s := memoryst.NewMemoryResourceStore(kind)
		if err := s.Init(nil); err != nil {
			t.Fatal(err)
		}
		stores[kind] = s
	}
	instance := meshresource.NewInstanceResourceWithAttributes(
		instanceName,
		meshName,
	)

	instance.Spec.AppName = appName
	instance.Spec.Ip = providerIP
	instance.Spec.RpcPort = providerPort

	if err := stores[meshresource.InstanceKind].Add(instance); err != nil {
		t.Fatal(err)
	}
	metadata := meshresource.NewServiceProviderMetadataResourceWithAttributes(
		"echo-metadata",
		meshName,
	)

	metadata.Spec.ServiceName = serviceName
	metadata.Spec.Methods = []*meshproto.Method{
		{
			Name:           "echo",
			ParameterTypes: []string{"java.lang.String"},
			ReturnType:     "java.lang.String",
		},
	}

	if err := stores[meshresource.ServiceProviderMetadataKind].Add(metadata); err != nil {
		t.Fatal(err)
	}
	rpcName := meshresource.BuildInstanceResName(
		appName,
		providerIP,
		providerPort,
	)

	rpcInstance := meshresource.NewRPCInstanceResourceWithAttributes(
		rpcName,
		meshName,
	)

	rpcInstance.Spec.Name = rpcName
	rpcInstance.Spec.AppName = appName
	rpcInstance.Spec.Ip = providerIP
	rpcInstance.Spec.Port = providerPort

	rpcInstance.Spec.Endpoints = []*meshproto.Endpoint{
		{
			Protocol: "tri",
			Port:     providerPort,
		},
	}

	if err := stores[meshresource.RPCInstanceKind].Add(rpcInstance); err != nil {
		t.Fatal(err)
	}
	ctx := &genericHTTPTestContext{
		rm: manager.NewResourceManager(
			&genericHTTPTestRouter{stores: stores},
			nil,
		),
	}
	router := gin.New()
	router.POST("/test/invoke", ServiceGenericInvoke(ctx))
	body := `{
		"mesh": "test-mesh",
		"serviceName": "example.EchoService",
		"instanceName": "provider-1",
		"methodName": "echo",
		"signature": "java.lang.String->java.lang.String",
		"args": ["hello"],
		"timeoutMs": 0,
		"attachments": {
			"traceId": "abc"
		}
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/test/invoke",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected HTTP 200, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			RawResult string `json:"rawResult"`
			ElapsedMs int64  `json:"elapsedMs"`
		} `json:"data"`
	}

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatalf("invalid response JSON: %v", err)
	}

	if response.Code != "Success" {
		t.Fatalf(
			"expected Success, got %q: %s",
			response.Code,
			recorder.Body.String(),
		)
	}

	if response.Data.RawResult != "hello" {
		t.Fatalf(
			"expected hello, got %q",
			response.Data.RawResult,
		)
	}

	t.Logf("Triple HTTP response: %s", recorder.Body.String())
	t.Logf("Triple RPC elapsed: %d ms", response.Data.ElapsedMs)
}
