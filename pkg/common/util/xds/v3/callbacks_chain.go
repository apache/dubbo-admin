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

package v3

import (
	"context"

	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_sd "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	envoy_xds "github.com/envoyproxy/go-control-plane/pkg/server/v3"
)

type CallbacksChain []envoy_xds.Callbacks

var _ envoy_xds.Callbacks = CallbacksChain{}

// OnStreamOpen is called once an xDS stream is open with a stream ID and the type URL (or "" for ADS).
// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
func (chain CallbacksChain) OnStreamOpen(ctx context.Context, streamID int64, typ string) error {
	for _, cb := range chain {
		if err := cb.OnStreamOpen(ctx, streamID, typ); err != nil {
			return err
		}
	}
	return nil
}

// OnStreamClosed is called immediately prior to closing an xDS stream with a stream ID.
func (chain CallbacksChain) OnStreamClosed(streamID int64, node *envoy_core.Node) {
	for i := len(chain) - 1; i >= 0; i-- {
		cb := chain[i]
		cb.OnStreamClosed(streamID, node)
	}
}

// OnStreamRequest is called once a request is received on a stream.
// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
func (chain CallbacksChain) OnStreamRequest(streamID int64, req *envoy_sd.DiscoveryRequest) error {
	for _, cb := range chain {
		if err := cb.OnStreamRequest(streamID, req); err != nil {
			return err
		}
	}
	return nil
}

// OnStreamResponse is called immediately prior to sending a response on a stream.
func (chain CallbacksChain) OnStreamResponse(ctx context.Context, streamID int64, req *envoy_sd.DiscoveryRequest, resp *envoy_sd.DiscoveryResponse) {
	for i := len(chain) - 1; i >= 0; i-- {
		cb := chain[i]
		cb.OnStreamResponse(ctx, streamID, req, resp)
	}
}

// OnFetchRequest is called for each Fetch request. Returning an error will end processing of the
// request and respond with an error.
func (chain CallbacksChain) OnFetchRequest(ctx context.Context, req *envoy_sd.DiscoveryRequest) error {
	for _, cb := range chain {
		if err := cb.OnFetchRequest(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// OnFetchRequest is called for each Fetch request. Returning an error will end processing of the
// request and respond with an error.
// OnFetchResponse is called immediately prior to sending a response.
func (chain CallbacksChain) OnFetchResponse(req *envoy_sd.DiscoveryRequest, resp *envoy_sd.DiscoveryResponse) {
	for i := len(chain) - 1; i >= 0; i-- {
		cb := chain[i]
		cb.OnFetchResponse(req, resp)
	}
}

func (chain CallbacksChain) OnDeltaStreamOpen(ctx context.Context, streamID int64, typeURL string) error {
	for _, cb := range chain {
		if err := cb.OnDeltaStreamOpen(ctx, streamID, typeURL); err != nil {
			return err
		}
	}

	return nil
}

func (chain CallbacksChain) OnDeltaStreamClosed(streamID int64, node *envoy_core.Node) {
	for i := len(chain) - 1; i >= 0; i-- {
		cb := chain[i]
		cb.OnDeltaStreamClosed(streamID, node)
	}
}

func (chain CallbacksChain) OnStreamDeltaRequest(streamID int64, request *envoy_sd.DeltaDiscoveryRequest) error {
	for _, cb := range chain {
		if err := cb.OnStreamDeltaRequest(streamID, request); err != nil {
			return err
		}
	}

	return nil
}

func (chain CallbacksChain) OnStreamDeltaResponse(streamID int64, request *envoy_sd.DeltaDiscoveryRequest, response *envoy_sd.DeltaDiscoveryResponse) {
	for i := len(chain) - 1; i >= 0; i-- {
		cb := chain[i]
		cb.OnStreamDeltaResponse(streamID, request, response)
	}
}
