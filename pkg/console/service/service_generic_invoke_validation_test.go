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

	dubbocommon "dubbo.apache.org/dubbo-go/v3/common"
	dubboconstant "dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	protocolbase "dubbo.apache.org/dubbo-go/v3/protocol/base"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
)

func TestResolveGenericInvokeTimeout(t *testing.T) {
	invocation := genericInvocation{Timeout: 5 * time.Second}

	t.Run("uses remaining deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		deadline, _ := ctx.Deadline()
		before := ceilTimeoutToMillisecond(time.Until(deadline))
		got, err := resolveGenericInvokeTimeout(ctx, invocation)
		if err != nil || got <= 0 || got < time.Until(deadline) || got > before || got%time.Millisecond != 0 {
			t.Fatalf("expected remaining millisecond-aligned context timeout, got %v, %v", got, err)
		}
	})

	t.Run("rejects expired deadline", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
		_, err := resolveGenericInvokeTimeout(ctx, invocation)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected deadline exceeded, got %v", err)
		}
	})

	t.Run("rejects canceled context before deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cancel()
		_, err := resolveGenericInvokeTimeout(ctx, invocation)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context canceled, got %v", err)
		}
	})

	t.Run("uses shorter invocation timeout than context deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		got, err := resolveGenericInvokeTimeout(ctx, genericInvocation{Timeout: 2 * time.Second})
		if err != nil || got != 2*time.Second {
			t.Fatalf("expected shorter invocation timeout 2s, got %v, %v", got, err)
		}
	})

	t.Run("rounds sub-millisecond timeout up to 1ms", func(t *testing.T) {
		got, err := resolveGenericInvokeTimeout(context.Background(), genericInvocation{Timeout: 500 * time.Microsecond})
		if err != nil || got != time.Millisecond {
			t.Fatalf("expected 1ms, got %v, %v", got, err)
		}
	})

	t.Run("handles nil context", func(t *testing.T) {
		got, err := resolveGenericInvokeTimeout(nil, invocation)
		if err != nil || got != invocation.Timeout {
			t.Fatalf("expected invocation timeout %v, got %v, %v", invocation.Timeout, got, err)
		}
	})

	t.Run("uses request timeout without deadline", func(t *testing.T) {
		got, err := resolveGenericInvokeTimeout(context.Background(), invocation)
		if err != nil || got != invocation.Timeout {
			t.Fatalf("expected request timeout %v, got %v, %v", invocation.Timeout, got, err)
		}
	})
}

type genericInvokeReferenceRecorder struct {
	protocolbase.BaseProtocol
	url *dubbocommon.URL
}

func (p *genericInvokeReferenceRecorder) Refer(url *dubbocommon.URL) protocolbase.Invoker {
	p.url = url
	return protocolbase.NewBaseInvoker(url)
}

func TestInvokeGenericServiceRPCReferenceTimeout(t *testing.T) {
	originalDubbo := extension.GetProtocol(dubboconstant.DubboProtocol)
	originalTriple := extension.GetProtocol(dubboconstant.TriProtocol)
	t.Cleanup(func() {
		extension.SetProtocol(dubboconstant.DubboProtocol, func() protocolbase.Protocol { return originalDubbo })
		extension.SetProtocol(dubboconstant.TriProtocol, func() protocolbase.Protocol { return originalTriple })
	})

	for _, timeout := range []time.Duration{100 * time.Millisecond, 5 * time.Second} {
		t.Run("dubbo/"+timeout.String(), func(t *testing.T) {
			recorder := &genericInvokeReferenceRecorder{}
			extension.SetProtocol(dubboconstant.DubboProtocol, func() protocolbase.Protocol { return recorder })

			_, err := invokeGenericServiceRPC(context.Background(), genericInvocation{
				URL:           "dubbo://127.0.0.1:20880",
				Protocol:      dubboconstant.DubboProtocol,
				Serialization: dubboconstant.Hessian2Serialization,
				ServiceName:   "example.Calculator",
				MethodName:    "delay",
				Timeout:       timeout,
			})
			if err != nil {
				t.Fatalf("invoke generic service: %v", err)
			}
			if recorder.url == nil {
				t.Fatal("expected a Dubbo reference URL")
			}
			if got := recorder.url.GetParam(dubboconstant.TimeoutKey, ""); got != timeout.String() {
				t.Fatalf("expected reference timeout %v, got %q", timeout, got)
			}
		})
	}

	t.Run("dubbo/context-deadline", func(t *testing.T) {
		recorder := &genericInvokeReferenceRecorder{}
		extension.SetProtocol(dubboconstant.DubboProtocol, func() protocolbase.Protocol { return recorder })

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		_, err := invokeGenericServiceRPC(ctx, genericInvocation{
			URL:           "dubbo://127.0.0.1:20880",
			Protocol:      dubboconstant.DubboProtocol,
			Serialization: dubboconstant.Hessian2Serialization,
			ServiceName:   "example.Calculator",
			MethodName:    "delay",
			Timeout:       5 * time.Second,
		})
		if err != nil {
			t.Fatalf("invoke generic service: %v", err)
		}
		if recorder.url == nil {
			t.Fatal("expected a Dubbo reference URL")
		}
		rawTimeout := recorder.url.GetParam(dubboconstant.TimeoutKey, "")
		parsed, err := time.ParseDuration(rawTimeout)
		if err != nil || parsed <= 0 || parsed > 200*time.Millisecond || parsed%time.Millisecond != 0 {
			t.Fatalf("expected millisecond-aligned timeout in (0, 200ms], got %q (%v)", rawTimeout, err)
		}
	})

	t.Run("dubbo/expired-context-skips-refer", func(t *testing.T) {
		recorder := &genericInvokeReferenceRecorder{}
		extension.SetProtocol(dubboconstant.DubboProtocol, func() protocolbase.Protocol { return recorder })

		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
		_, err := invokeGenericServiceRPC(ctx, genericInvocation{
			URL:           "dubbo://127.0.0.1:20880",
			Protocol:      dubboconstant.DubboProtocol,
			Serialization: dubboconstant.Hessian2Serialization,
			ServiceName:   "example.Calculator",
			MethodName:    "delay",
			Timeout:       5 * time.Second,
		})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected deadline exceeded, got %v", err)
		}
		if recorder.url != nil {
			t.Fatalf("expected Refer not to be called for expired context, got %v", recorder.url)
		}
	})

	t.Run("triple/150ms", func(t *testing.T) {
		recorder := &genericInvokeReferenceRecorder{}
		extension.SetProtocol(dubboconstant.TriProtocol, func() protocolbase.Protocol { return recorder })

		_, err := invokeGenericServiceRPC(context.Background(), genericInvocation{
			URL:           "tri://127.0.0.1:50051",
			Protocol:      dubboconstant.TriProtocol,
			Serialization: dubboconstant.Hessian2Serialization,
			ServiceName:   "example.Calculator",
			MethodName:    "delay",
			Timeout:       150 * time.Millisecond,
		})
		if err != nil {
			t.Fatalf("invoke generic service: %v", err)
		}
		if recorder.url == nil {
			t.Fatal("expected a Triple reference URL")
		}
		if got := recorder.url.GetParam(dubboconstant.TimeoutKey, ""); got != (150 * time.Millisecond).String() {
			t.Fatalf("expected reference timeout 150ms, got %q", got)
		}
	})
}

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
	if capturedInvocation.Timeout != expectedTimeout {
		t.Fatalf("expected invocation timeout %v, got %v", expectedTimeout, capturedInvocation.Timeout)
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
