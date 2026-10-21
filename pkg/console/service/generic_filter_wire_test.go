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
	"net/url"
	"reflect"
	"testing"

	hessian "github.com/apache/dubbo-go-hessian2"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	_ "dubbo.apache.org/dubbo-go/v3/filter/generic"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type genericWireCaptureInvoker struct {
	*base.BaseInvoker

	captured base.Invocation
	calls    int
}

func (c *genericWireCaptureInvoker) Invoke(
	_ context.Context,
	inv base.Invocation,
) result.Result {
	c.calls++
	c.captured = inv

	return &result.RPCResult{}
}

func TestGenericFilterTripleNonIDLWireContract(t *testing.T) {
	gf, ok := extension.GetFilter(constant.GenericFilterKey)
	if !ok {
		t.Fatal("Generic Filter is not registered")
	}
	u := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(
			constant.GenericKey,
			constant.GenericSerializationDefault,
		),
	)

	next := &genericWireCaptureInvoker{
		BaseInvoker: base.NewBaseInvoker(u),
	}

	types := []string{"java.lang.String"}
	args := []hessian.Object{"hello"}

	var reply any

	original := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName(constant.Generic),
		invocation.WithArguments([]any{
			"echo",
			types,
			args,
		}),
		invocation.WithReply(&reply),
		invocation.WithAttachments(map[string]any{
			"traceId": "abc",
		}),
	)
	res := gf.Invoke(
		context.Background(),
		next,
		original,
	)

	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if err := res.Error(); err != nil {
		t.Fatalf("unexpected filter error: %v", err)
	}
	if next.calls != 1 {
		t.Fatalf(
			"expected 1 downstream invocation, got %d",
			next.calls,
		)
	}

	if next.captured == nil {
		t.Fatal("expected captured invocation")
	}

	got := next.captured
	if got.MethodName() != constant.Generic {
		t.Fatalf(
			"expected method %q, got %q",
			constant.Generic,
			got.MethodName(),
		)
	}
	wantArgs := []any{"echo", types, args}

	if !reflect.DeepEqual(got.Arguments(), wantArgs) {
		t.Fatalf(
			"unexpected Arguments:\n got: %#v\nwant: %#v",
			got.Arguments(),
			wantArgs,
		)
	}
	raw := got.ParameterRawValues()

	if len(raw) != 4 {
		t.Fatalf(
			"expected 4 raw values, got %d: %#v",
			len(raw),
			raw,
		)
	}
	if !reflect.DeepEqual(raw[:3], wantArgs) {
		t.Fatalf(
			"unexpected request raw values:\n got: %#v\nwant: %#v",
			raw[:3],
			wantArgs,
		)
	}
	if raw[3] != &reply {
		t.Fatalf(
			"reply pointer changed: got %#v",
			raw[3],
		)
	}

	if got.Reply() != &reply {
		t.Fatal("Invocation.Reply pointer changed")
	}
	callType, exists := got.GetAttribute(constant.CallTypeKey)

	if !exists || callType != constant.CallUnary {
		t.Fatalf(
			"expected CallUnary, got %#v, exists=%v",
			callType,
			exists,
		)
	}
	if traceID := got.GetAttachmentWithDefaultValue(
		"traceId", "",
	); traceID != "abc" {
		t.Fatalf(
			"expected traceId=abc, got %q",
			traceID,
		)
	}

	t.Log("Generic Filter wire contract verified")
	t.Logf("method: %s", got.MethodName())
	t.Logf("Arguments length: %d", len(got.Arguments()))
	t.Logf("ParameterRawValues length: %d", len(raw))
	t.Logf("CallType: %v", callType)
}

func TestGenericFilterPreservesExplicitCallType(t *testing.T) {
	gf, ok := extension.GetFilter(constant.GenericFilterKey)
	if !ok {
		t.Fatal("Generic Filter is not registered")
	}

	u := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(
			constant.GenericKey,
			constant.GenericSerializationDefault,
		),
	)

	next := &genericWireCaptureInvoker{
		BaseInvoker: base.NewBaseInvoker(u),
	}

	types := []string{"java.lang.String"}
	args := []hessian.Object{"hello"}

	var reply any

	original := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName(constant.Generic),
		invocation.WithArguments([]any{
			"echo",
			types,
			args,
		}),
		invocation.WithReply(&reply),
		invocation.WithAttachments(map[string]any{}),
	)
	original.SetAttribute(
		constant.CallTypeKey,
		constant.CallServerStream,
	)

	res := gf.Invoke(context.Background(), next, original)

	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if err := res.Error(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if next.calls != 1 || next.captured == nil {
		t.Fatalf(
			"expected one captured invocation, got %d",
			next.calls,
		)
	}

	callType, exists := next.captured.GetAttribute(
		constant.CallTypeKey,
	)

	if !exists || callType != constant.CallServerStream {
		t.Fatalf(
			"expected preserved CallServerStream, got %#v",
			callType,
		)
	}

	t.Logf("explicit CallType preserved: %v", callType)
}
