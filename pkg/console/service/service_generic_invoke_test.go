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
	"errors"
	"testing"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/console/model"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/stretchr/testify/require"
)

// Capture the invocation after the real SDK reference and filter chain.
type captureInvokeProtocol struct {
	base.BaseProtocol
	invoke func(*common.URL, base.Invocation) result.Result
}

func (p *captureInvokeProtocol) Refer(url *common.URL) base.Invoker {
	return &captureInvokeInvoker{BaseInvoker: base.NewBaseInvoker(url), invoke: p.invoke}
}

type captureInvokeInvoker struct {
	*base.BaseInvoker
	invoke func(*common.URL, base.Invocation) result.Result
}

func (i *captureInvokeInvoker) Invoke(_ context.Context, inv base.Invocation) result.Result {
	return i.invoke(i.GetURL(), inv)
}

func TestGenericInvokeSDKModes(t *testing.T) {
	for _, protocol := range []string{"dubbo", "tri"} {
		t.Run(protocol, func(t *testing.T) {
			original := extension.GetProtocol(protocol)
			t.Cleanup(func() { extension.SetProtocol(protocol, func() base.Protocol { return original }) })
			for _, mode := range []string{"true", "gson", "bean", "protobuf-json", "protobuf", "false"} {
				t.Run(mode, func(t *testing.T) {
					calls := 0
					capture := &captureInvokeProtocol{BaseProtocol: base.NewBaseProtocol()}
					capture.invoke = func(url *common.URL, inv base.Invocation) result.Result {
						calls++
						require.Equal(t, mode, url.GetParam(constant.GenericKey, ""))
						require.Equal(t, "hessian2", url.GetParam(constant.SerializationKey, ""))
						require.Equal(t, mode, inv.GetAttachmentWithDefaultValue(constant.GenericKey, ""))
						require.Equal(t, "abc", inv.GetAttachmentWithDefaultValue("traceId", ""))
						if mode == "false" {
							require.Equal(t, "echo", inv.MethodName())
							require.Equal(t, []any{"hello"}, inv.Arguments())
							require.Len(t, inv.ParameterRawValues(), 2) // raw argument and reply
						} else {
							require.Equal(t, "$invoke", inv.MethodName())
							require.Equal(t, "echo", inv.Arguments()[0])
							require.Equal(t, []string{"java.lang.String"}, inv.Arguments()[1])
							require.Equal(t, []hessian.Object{"hello"}, inv.Arguments()[2])
						}
						if reply, ok := inv.Reply().(*any); ok {
							*reply = "ok"
						}
						return &result.RPCResult{Rest: "ok"}
					}
					extension.SetProtocol(protocol, func() base.Protocol { return capture })
					attachments := map[string]any{"generic": "unknown", "traceId": "abc"}
					ctx := context.WithValue(context.Background(), constant.AttachmentKey, attachments)
					reply, invoked, err := invokeGenericServiceRPC(ctx, genericInvocation{
						Generic: mode, Protocol: protocol, Serialization: "hessian2",
						URL: protocol + "://127.0.0.1:20880", ServiceName: "org.example.Echo",
						MethodName: "echo", ParameterTypes: []string{"java.lang.String"}, Args: []hessian.Object{"hello"},
					})
					require.NoError(t, err)
					require.True(t, invoked)
					require.Equal(t, "ok", reply)
					require.Equal(t, 1, calls)
					require.Equal(t, "unknown", attachments["generic"], "must not mutate caller attachments")
				})
			}
		})
	}
}

func TestGenericInvokeReportsAttemptedTransport(t *testing.T) {
	original := invokeGenericServiceRPC
	t.Cleanup(func() { invokeGenericServiceRPC = original })
	instance := meshresource.NewRPCInstanceResourceWithAttributes("instance", "mesh")
	instance.Spec.Ip = "127.0.0.1"
	instance.Spec.Serialization = "hessian2"
	instance.Spec.PreferSerialization = "hessian2, fastjson2"
	instance.Spec.Endpoints = []*meshproto.Endpoint{{Protocol: "dubbo", Port: 20880}}
	targets, err := buildGenericInvokeTargets(instance)
	require.NoError(t, err)
	require.Len(t, targets, 2)
	for _, fail := range []bool{false, true} {
		calls := 0
		invokeGenericServiceRPC = func(_ context.Context, invocation genericInvocation) (any, bool, error) {
			calls++
			require.Equal(t, "gson", invocation.Generic)
			if calls == 1 {
				return nil, true, base.ErrClientClosed
			}
			require.Equal(t, "fastjson2", invocation.Serialization)
			if fail {
				return nil, true, errors.New("provider error")
			}
			return "ok", true, nil
		}
		resp, err := invokeGenericServiceWithTargets(context.Background(), model.ServiceGenericInvokeReq{}, targets, genericInvocation{Generic: "gson"})
		require.Equal(t, 2, calls)
		require.Equal(t, "gson", resp.Generic)
		require.Equal(t, "dubbo", resp.Protocol)
		require.Equal(t, "fastjson2", resp.Serialization)
		if fail {
			require.ErrorContains(t, err, "provider error")
		} else {
			require.NoError(t, err)
			require.Equal(t, "ok", resp.RawResult)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	invokeGenericServiceRPC = func(context.Context, genericInvocation) (any, bool, error) {
		t.Fatal("unexpected RPC")
		return nil, false, nil
	}
	resp, err := invokeGenericServiceWithTargets(ctx, model.ServiceGenericInvokeReq{}, targets, genericInvocation{Generic: "false"})
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, "false", resp.Generic)
	require.Empty(t, resp.Protocol)
	require.Empty(t, resp.Serialization)
	invokeGenericServiceRPC = func(context.Context, genericInvocation) (any, bool, error) {
		return nil, false, errors.New("client setup failed")
	}
	resp, err = invokeGenericServiceWithTargets(context.Background(), model.ServiceGenericInvokeReq{}, targets, genericInvocation{Generic: "true"})
	require.ErrorContains(t, err, "client setup failed")
	require.Empty(t, resp.Protocol)
	require.Empty(t, resp.Serialization)
	instance.Spec.Serialization, instance.Spec.PreferSerialization = "", ""
	require.Equal(t, []string{"hessian2"}, buildGenericInvokeSerializations(instance))
}
