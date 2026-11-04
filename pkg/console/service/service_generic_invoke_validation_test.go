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
	"errors"
	"reflect"
	"testing"
	"time"

	dubboconstant "dubbo.apache.org/dubbo-go/v3/common/constant"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
)

func TestInvokeServiceGenericInvalidArgsNoRPC(t *testing.T) {
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
	ctx := &testContext{
		rm: manager.NewResourceManager(
			&testRouter{stores: stores},
			nil,
		),
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

	rpcName := meshresource.BuildInstanceResName(
		"calculator",
		"127.0.0.1",
		20880,
	)

	rpcInstance := meshresource.NewRPCInstanceResourceWithAttributes(
		rpcName,
		"test-mesh",
	)

	rpcInstance.Spec.Name = rpcName
	rpcInstance.Spec.AppName = "calculator"
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

	originalRPC := invokeGenericServiceRPC

	rpcCallCount := 0

	var hasDeadline bool
	var remaining time.Duration
	var receivedTraceID any
	var capturedInvocation genericInvocation

	invokeGenericServiceRPC = func(
		callCtx context.Context,
		invocation genericInvocation,
	) (any, error) {
		rpcCallCount++
		capturedInvocation = invocation
		deadline, ok := callCtx.Deadline()
		hasDeadline = ok

		if ok {
			remaining = time.Until(deadline)
		}
		attachments, ok := callCtx.Value(
			dubboconstant.AttachmentKey,
		).(map[string]any)

		if ok {
			receivedTraceID = attachments["traceId"]
		}
		return int32(30), nil
	}
	t.Cleanup(func() {
		invokeGenericServiceRPC = originalRPC
	})

	req := model.ServiceGenericInvokeReq{
		BaseServiceReq: model.BaseServiceReq{
			Mesh:        "test-mesh",
			ServiceName: "example.Calculator",
		},
		InstanceName: "provider-1",
		MethodName:   "add",
		Signature:    "int,int->int",
		Args: []json.RawMessage{
			json.RawMessage(`"hello"`),
			json.RawMessage(`20`),
		},
		TimeoutMs: 3000,
	}

	_, err := InvokeServiceGeneric(ctx, req)

	if err == nil {
		t.Fatal("expected invalid argument type error")
	}

	var fieldErr *GenericInvokeArgumentError

	if !errors.As(err, &fieldErr) {
		t.Fatalf(
			"expected structured argument error, got %T: %v",
			err,
			err,
		)
	}

	if fieldErr.Kind != ArgumentErrorType {
		t.Fatalf(
			"expected type error, got %s",
			fieldErr.Kind,
		)
	}

	if fieldErr.Field != "args[0]" {
		t.Fatalf(
			"expected field args[0], got %s",
			fieldErr.Field,
		)
	}

	if rpcCallCount != 0 {
		t.Fatalf(
			"invalid args triggered %d RPC calls",
			rpcCallCount,
		)
	}

	req.Args = []json.RawMessage{
		json.RawMessage(`10`),
		json.RawMessage(`20`),
	}

	req.TimeoutMs = 0
	req.Attachments = map[string]string{
		"traceId": "abc",
	}
	if err := req.Validate(); err != nil {
		t.Fatal(err)
	}
	if req.TimeoutMs != model.DefaultServiceGenericInvokeTimeoutMs {
		t.Fatalf(
			"expected default timeout %d, got %d",
			model.DefaultServiceGenericInvokeTimeoutMs,
			req.TimeoutMs,
		)
	}

	_, err = InvokeServiceGeneric(ctx, req)

	if err != nil {
		t.Fatalf("valid request failed: %v", err)
	}
	if rpcCallCount != 1 {
		t.Fatalf(
			"expected 1 RPC call, got %d",
			rpcCallCount,
		)
	}
	if !hasDeadline {
		t.Fatal("expected RPC context deadline")
	}

	expectedTimeout := time.Duration(
		model.DefaultServiceGenericInvokeTimeoutMs,
	) * time.Millisecond
	if remaining <= 0 || remaining > expectedTimeout {
		t.Fatalf(
			"unexpected remaining timeout: %v",
			remaining,
		)
	}
	if receivedTraceID != "abc" {
		t.Fatalf(
			"expected traceId abc, got %v",
			receivedTraceID,
		)
	}
	if capturedInvocation.MethodName != "add" {
		t.Fatalf(
			"expected method add, got %s",
			capturedInvocation.MethodName,
		)
	}

	expectedTypes := []string{"int", "int"}

	if !reflect.DeepEqual(
		capturedInvocation.ParameterTypes,
		expectedTypes,
	) {
		t.Fatalf(
			"expected parameter types %v, got %v",
			expectedTypes,
			capturedInvocation.ParameterTypes,
		)
	}
	if len(capturedInvocation.Args) != 2 {
		t.Fatalf(
			"expected 2 decoded args, got %d",
			len(capturedInvocation.Args),
		)
	}
	if !reflect.DeepEqual(
		capturedInvocation.Args[0],
		int32(10),
	) {
		t.Fatalf(
			"expected first arg int32(10), got %T: %v",
			capturedInvocation.Args[0],
			capturedInvocation.Args[0],
		)
	}
	if !reflect.DeepEqual(
		capturedInvocation.Args[1],
		int32(20),
	) {
		t.Fatalf(
			"expected second arg int32(20), got %T: %v",
			capturedInvocation.Args[1],
			capturedInvocation.Args[1],
		)
	}
	rpcCallCount = 0
	req.Args = []json.RawMessage{
		json.RawMessage(`10`),
	}

	_, err = InvokeServiceGeneric(ctx, req)

	if err == nil {
		t.Fatal("expected argument count mismatch error")
	}

	var countErr *GenericInvokeArgumentError

	if !errors.As(err, &countErr) {
		t.Fatalf(
			"expected structured count error, got %T: %v",
			err,
			err,
		)
	}
	if countErr.Kind != ArgumentErrorCount {
		t.Errorf(
			"expected count error, got %s",
			countErr.Kind,
		)
	}
	if countErr.Field != "args" {
		t.Errorf(
			"expected field args, got %s",
			countErr.Field,
		)
	}

	if countErr.Expected != "2" {
		t.Errorf(
			"expected 2 arguments, got %s",
			countErr.Expected,
		)
	}

	if countErr.Actual != "1" {
		t.Errorf(
			"expected 1 argument, got %s",
			countErr.Actual,
		)
	}

	if rpcCallCount != 0 {
		t.Fatalf(
			"count mismatch triggered %d RPC calls",
			rpcCallCount,
		)
	}

	rpcCallCount = 0

	req.Args = []json.RawMessage{
		json.RawMessage(`10`),
		json.RawMessage(`20`),
	}

	req.Signature = "int->int"

	_, err = InvokeServiceGeneric(ctx, req)

	if err == nil {
		t.Fatal("expected unknown signature error")
	}
	if rpcCallCount != 0 {
		t.Fatalf(
			"unknown signature triggered %d RPC calls",
			rpcCallCount,
		)
	}
	var bizErr bizerror.Error

	if !errors.As(err, &bizErr) {
		t.Fatalf(
			"expected bizerror, got %T: %v",
			err,
			err,
		)
	}
	if bizErr.Code() != bizerror.NotFoundError {
		t.Fatalf(
			"expected NotFoundError, got %s",
			bizErr.Code(),
		)
	}

	var signatureErr *GenericInvokeArgumentError

	if !errors.As(err, &signatureErr) {
		t.Fatalf(
			"expected structured signature error, got %T: %v",
			err,
			err,
		)
	}
	if signatureErr.Field != "signature" {
		t.Errorf(
			"expected field signature, got %s",
			signatureErr.Field,
		)
	}
	if signatureErr.Actual != "int->int" {
		t.Errorf(
			"expected actual int->int, got %s",
			signatureErr.Actual,
		)
	}
	if signatureErr.Expected != "int,int->int" {
		t.Errorf(
			"expected int,int->int, got %s",
			signatureErr.Expected,
		)
	}
}
