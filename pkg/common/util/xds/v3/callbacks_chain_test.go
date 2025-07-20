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

package v3_test

import (
	"context"
	"fmt"

	util_xds_v3 "github.com/apache/dubbo-admin/pkg/common/util/xds/v3"
	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_sd "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	envoy_xds "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CallbacksChain", func() {
	var first, second CallbacksFuncs

	type methodCall struct {
		obj    string
		method string
		args   []interface{}
	}
	var calls []methodCall

	BeforeEach(func() {
		calls = make([]methodCall, 0)
		first = CallbacksFuncs{
			OnStreamOpenFunc: func(ctx context.Context, streamID int64, typ string) error {
				calls = append(calls, methodCall{"1st", "OnStreamOpen()", []interface{}{ctx, streamID, typ}})
				return fmt.Errorf("1st: OnStreamOpen()")
			},
			OnStreamClosedFunc: func(streamID int64, n *envoy_core.Node) {
				calls = append(calls, methodCall{"1st", "OnStreamClosed()", []interface{}{streamID, n}})
			},
			OnStreamRequestFunc: func(streamID int64, req *envoy_sd.DiscoveryRequest) error {
				calls = append(calls, methodCall{"1st", "OnStreamRequest()", []interface{}{streamID, req}})
				return fmt.Errorf("1st: OnStreamRequest()")
			},
			OnStreamResponseFunc: func(ctx context.Context, streamID int64, req *envoy_sd.DiscoveryRequest, resp *envoy_sd.DiscoveryResponse) {
				calls = append(calls, methodCall{"1st", "OnStreamResponse()", []interface{}{ctx, streamID, req, resp}})
			},
		}
		second = CallbacksFuncs{
			OnStreamOpenFunc: func(ctx context.Context, streamID int64, typ string) error {
				calls = append(calls, methodCall{"2nd", "OnStreamOpen()", []interface{}{ctx, streamID, typ}})
				return fmt.Errorf("2nd: OnStreamOpen()")
			},
			OnStreamClosedFunc: func(streamID int64, n *envoy_core.Node) {
				calls = append(calls, methodCall{"2nd", "OnStreamClosed()", []interface{}{streamID, n}})
			},
			OnStreamRequestFunc: func(streamID int64, req *envoy_sd.DiscoveryRequest) error {
				calls = append(calls, methodCall{"2nd", "OnStreamRequest()", []interface{}{streamID, req}})
				return fmt.Errorf("2nd: OnStreamRequest()")
			},
			OnStreamResponseFunc: func(ctx context.Context, streamID int64, req *envoy_sd.DiscoveryRequest, resp *envoy_sd.DiscoveryResponse) {
				calls = append(calls, methodCall{"2nd", "OnStreamResponse()", []interface{}{ctx, streamID, req, resp}})
			},
		}
	})

	Describe("OnStreamOpen", func() {
		It("should be called sequentially and return after first error", func() {
			// given
			ctx := context.Background()
			streamID := int64(1)
			typ := "xDS"
			// setup
			chain := util_xds_v3.CallbacksChain{first, second}

			// when
			err := chain.OnStreamOpen(ctx, streamID, typ)

			// then
			Expect(calls).To(Equal([]methodCall{
				{"1st", "OnStreamOpen()", []interface{}{ctx, streamID, typ}},
			}))
			// and
			Expect(err).To(MatchError("1st: OnStreamOpen()"))
		})
	})
	Describe("OnStreamClose", func() {
		It("should be called in reverse order", func() {
			// given
			streamID := int64(1)
			n := &envoy_core.Node{Id: "my-node"}
			// setup
			chain := util_xds_v3.CallbacksChain{first, second}

			// when
			chain.OnStreamClosed(streamID, n)

			// then
			Expect(calls).To(Equal([]methodCall{
				{"2nd", "OnStreamClosed()", []interface{}{streamID, n}},
				{"1st", "OnStreamClosed()", []interface{}{streamID, n}},
			}))
		})
	})
	Describe("OnStreamRequest", func() {
		It("should be called sequentially and return after first error", func() {
			// given
			streamID := int64(1)
			req := &envoy_sd.DiscoveryRequest{}

			// setup
			chain := util_xds_v3.CallbacksChain{first, second}

			// when
			err := chain.OnStreamRequest(streamID, req)

			// then
			Expect(calls).To(Equal([]methodCall{
				{"1st", "OnStreamRequest()", []interface{}{streamID, req}},
			}))
			// and
			Expect(err).To(MatchError("1st: OnStreamRequest()"))
		})
	})
	Describe("OnStreamResponse", func() {
		It("should be called in reverse order", func() {
			// given
			chain := util_xds_v3.CallbacksChain{first, second}
			streamID := int64(1)
			req := &envoy_sd.DiscoveryRequest{}
			resp := &envoy_sd.DiscoveryResponse{}
			ctx := context.TODO()

			// when
			chain.OnStreamResponse(ctx, streamID, req, resp)

			// then
			Expect(calls).To(Equal([]methodCall{
				{"2nd", "OnStreamResponse()", []interface{}{ctx, streamID, req, resp}},
				{"1st", "OnStreamResponse()", []interface{}{ctx, streamID, req, resp}},
			}))
		})
	})
})

var _ envoy_xds.Callbacks = CallbacksFuncs{}

type CallbacksFuncs struct {
	OnStreamOpenFunc   func(context.Context, int64, string) error
	OnStreamClosedFunc func(int64, *envoy_core.Node)

	OnStreamRequestFunc  func(int64, *envoy_sd.DiscoveryRequest) error
	OnStreamResponseFunc func(context.Context, int64, *envoy_sd.DiscoveryRequest, *envoy_sd.DiscoveryResponse)

	OnFetchRequestFunc  func(context.Context, *envoy_sd.DiscoveryRequest) error
	OnFetchResponseFunc func(*envoy_sd.DiscoveryRequest, *envoy_sd.DiscoveryResponse)
}

func (f CallbacksFuncs) OnStreamOpen(ctx context.Context, streamID int64, typ string) error {
	if f.OnStreamOpenFunc != nil {
		return f.OnStreamOpenFunc(ctx, streamID, typ)
	}
	return nil
}

func (f CallbacksFuncs) OnStreamClosed(streamID int64, n *envoy_core.Node) {
	if f.OnStreamClosedFunc != nil {
		f.OnStreamClosedFunc(streamID, n)
	}
}

func (f CallbacksFuncs) OnStreamRequest(streamID int64, req *envoy_sd.DiscoveryRequest) error {
	if f.OnStreamRequestFunc != nil {
		return f.OnStreamRequestFunc(streamID, req)
	}
	return nil
}

func (f CallbacksFuncs) OnStreamResponse(ctx context.Context, streamID int64, req *envoy_sd.DiscoveryRequest, resp *envoy_sd.DiscoveryResponse) {
	if f.OnStreamResponseFunc != nil {
		f.OnStreamResponseFunc(ctx, streamID, req, resp)
	}
}

func (f CallbacksFuncs) OnFetchRequest(ctx context.Context, req *envoy_sd.DiscoveryRequest) error {
	if f.OnFetchRequestFunc != nil {
		return f.OnFetchRequestFunc(ctx, req)
	}
	return nil
}

func (f CallbacksFuncs) OnFetchResponse(req *envoy_sd.DiscoveryRequest, resp *envoy_sd.DiscoveryResponse) {
	if f.OnFetchResponseFunc != nil {
		f.OnFetchResponseFunc(req, resp)
	}
}

func (f CallbacksFuncs) OnDeltaStreamOpen(ctx context.Context, i int64, s string) error {
	return nil
}

func (f CallbacksFuncs) OnDeltaStreamClosed(i int64, n *envoy_core.Node) {
}

func (f CallbacksFuncs) OnStreamDeltaRequest(i int64, request *envoy_sd.DeltaDiscoveryRequest) error {
	return nil
}

func (f CallbacksFuncs) OnStreamDeltaResponse(i int64, request *envoy_sd.DeltaDiscoveryRequest, response *envoy_sd.DeltaDiscoveryResponse) {
}
