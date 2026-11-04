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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
)

type genericHTTPTestContext struct {
	consolectx.Context
	rm manager.ResourceManager
}

func (c *genericHTTPTestContext) AppContext() context.Context {
	return context.Background()
}

func (c *genericHTTPTestContext) ResourceManager() manager.ResourceManager {
	return c.rm
}

type genericHTTPTestRouter struct {
	stores map[coremodel.ResourceKind]store.ResourceStore
}

func (r *genericHTTPTestRouter) ResourceRoute(
	res coremodel.Resource,
) (store.ResourceStore, error) {
	return r.ResourceKindRoute(res.ResourceKind())
}

func (r *genericHTTPTestRouter) ResourceKindRoute(
	kind coremodel.ResourceKind,
) (store.ResourceStore, error) {
	s, ok := r.stores[kind]
	if !ok {
		return nil, fmt.Errorf("store not found for %s", kind)
	}
	return s, nil
}

func TestServiceGenericInvokeHTTPValidation(t *testing.T) {
	stores := make(map[coremodel.ResourceKind]store.ResourceStore)

	for _, kind := range []coremodel.ResourceKind{
		meshresource.InstanceKind,
		meshresource.ServiceProviderMetadataKind,
	} {
		s := memoryst.NewMemoryResourceStore(kind)

		if err := s.Init(nil); err != nil {
			t.Fatal(err)
		}

		stores[kind] = s
	}

	instance := meshresource.NewInstanceResourceWithAttributes(
		"provider-1",
		"test-mesh",
	)

	instance.Spec.AppName = "calculator"
	instance.Spec.Ip = "127.0.0.1"
	instance.Spec.RpcPort = 20880

	if err := stores[meshresource.InstanceKind].Add(instance); err != nil {
		t.Fatal(err)
	}

	metadata := meshresource.NewServiceProviderMetadataResourceWithAttributes(
		"calculator-metadata",
		"test-mesh",
	)

	metadata.Spec.ServiceName = "example.Calculator"
	metadata.Spec.Methods = []*meshproto.Method{
		{
			Name:           "add",
			ParameterTypes: []string{"int", "int"},
			ReturnType:     "int",
		},
	}

	if err := stores[meshresource.ServiceProviderMetadataKind].Add(metadata); err != nil {
		t.Fatal(err)
	}

	ctx := &genericHTTPTestContext{
		rm: manager.NewResourceManager(
			&genericHTTPTestRouter{stores: stores},
			nil,
		),
	}

	router := gin.New()

	router.POST(
		"/test/invoke",
		ServiceGenericInvoke(ctx),
	)

	tests := []struct {
		name           string
		body           string
		wantHTTPStatus int
		wantCode       string
		wantMessage    string
		wantField      string
		wantExpected   string
		wantActual     string
	}{
		{
			name: "count_mismatch",
			body: `{
				"mesh": "test-mesh",
				"serviceName": "example.Calculator",
				"instanceName": "provider-1",
				"methodName": "add",
				"signature": "int,int->int",
				"args": [10],
				"timeoutMs": 3000
			}`,

			wantHTTPStatus: http.StatusOK,
			wantCode:       "InvalidArgument",
			wantMessage:    "invalid argument count",
			wantField:      "args",
			wantExpected:   "2",
			wantActual:     "1",
		},
		{
			name: "unknown_signature",
			body: `{
				"mesh": "test-mesh",
				"serviceName": "example.Calculator",
				"instanceName": "provider-1",
				"methodName": "add",
				"signature": "int->int",
				"args": [10, 20],
				"timeoutMs": 3000
			}`,

			wantHTTPStatus: http.StatusOK,
			wantCode:       "NotFoundError",
			wantMessage:    "method add not found for service example.Calculator",
			wantField:      "signature",
			wantExpected:   "int,int->int",
			wantActual:     "int->int",
		},
		{
			name: "empty_args",

			body: `{
				"mesh": "test-mesh",
				"serviceName": "example.Calculator",
				"instanceName": "provider-1",
				"methodName": "add",
				"signature": "int,int->int",
				"args": [],
				"timeoutMs": 3000
			}`,

			wantHTTPStatus: http.StatusOK,
			wantCode:       "InvalidArgument",
			wantMessage:    "invalid argument count",
			wantField:      "args",
			wantExpected:   "2",
			wantActual:     "0",
		},
		{
			name: "missing_signature",
			body: `{
				"mesh": "test-mesh",
				"serviceName": "example.Calculator",
				"instanceName": "provider-1",
				"methodName": "add",
				"args": [10, 20],
				"timeoutMs": 3000
			}`,

			wantHTTPStatus: http.StatusOK,
			wantCode:       "InvalidArgument",
			wantMessage:    "signature is empty",
			wantField:      "signature",
			wantExpected:   "non-empty",
			wantActual:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost,
				"/test/invoke",
				bytes.NewBufferString(tt.body),
			)

			req.Header.Set(
				"Content-Type",
				"application/json",
			)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			if recorder.Code != tt.wantHTTPStatus {
				t.Fatalf(
					"expected HTTP status %d, got %d, body: %s",
					tt.wantHTTPStatus,
					recorder.Code,
					recorder.Body.String(),
				)
			}

			var response struct {
				Code     string `json:"code"`
				Message  string `json:"message"`
				Field    string `json:"field"`
				Expected string `json:"expected"`
				Actual   string `json:"actual"`
			}

			if err := json.Unmarshal(
				recorder.Body.Bytes(),
				&response,
			); err != nil {
				t.Fatalf(
					"invalid response JSON: %v, body: %s",
					err,
					recorder.Body.String(),
				)
			}

			if response.Code != tt.wantCode {
				t.Errorf(
					"expected code %q, got %q",
					tt.wantCode,
					response.Code,
				)
			}

			if response.Message != tt.wantMessage {
				t.Errorf(
					"expected message %q, got %q",
					tt.wantMessage,
					response.Message,
				)
			}
			if response.Field != tt.wantField {
				t.Errorf(
					"expected field %q, got %q",
					tt.wantField,
					response.Field,
				)
			}
			if response.Expected != tt.wantExpected {
				t.Errorf(
					"expected value %q, got %q",
					tt.wantExpected,
					response.Expected,
				)
			}
			if response.Actual != tt.wantActual {
				t.Errorf(
					"expected actual %q, got %q",
					tt.wantActual,
					response.Actual,
				)
			}
		})
	}
}
