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

package xds

import (
	"context"

	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// DiscoveryRequest defines interface over real Envoy's DiscoveryRequest.
type DiscoveryRequest interface {
	NodeId() string
	// Node returns either a v2 or v3 Node
	Node() interface{}
	Metadata() *structpb.Struct
	VersionInfo() string
	GetTypeUrl() string
	GetResponseNonce() string
	GetResourceNames() []string
	HasErrors() bool
	ErrorMsg() string
}

// DiscoveryResponse defines interface over real Envoy's DiscoveryResponse.
type DiscoveryResponse interface {
	GetTypeUrl() string
	VersionInfo() string
	GetResources() []*anypb.Any
	GetNonce() string
}

type DeltaDiscoveryRequest interface {
	NodeId() string
	// Node returns either a v2 or v3 Node
	Node() interface{}
	Metadata() *structpb.Struct
	GetTypeUrl() string
	GetResponseNonce() string
	GetResourceNamesSubscribe() []string
	GetInitialResourceVersions() map[string]string
	HasErrors() bool
	ErrorMsg() string
}

// DeltaDiscoveryResponse defines interface over real Envoy's DeltaDiscoveryResponse.
type DeltaDiscoveryResponse interface {
	GetTypeUrl() string
	GetResources() []*discoveryv3.Resource
	GetRemovedResources() []string
	GetNonce() string
}

// Callbacks defines Callbacks for xDS streaming requests. The difference over real go-control-plane Callbacks is that it takes an DiscoveryRequest / DiscoveryResponse interface.
// It helps us to implement Callbacks once for many different versions of Envoy API.
type Callbacks interface {
	// OnStreamOpen is called once an xDS stream is opened with a stream ID and the type URL (or "" for ADS).
	// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
	OnStreamOpen(context.Context, int64, string) error
	// OnStreamClosed is called immediately prior to closing an xDS stream with a stream ID.
	OnStreamClosed(int64)
	// OnStreamRequest is called once a request is received on a stream.
	// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
	OnStreamRequest(int64, DiscoveryRequest) error
	// OnStreamResponse is called immediately prior to sending a response on a stream.
	OnStreamResponse(int64, DiscoveryRequest, DiscoveryResponse)
}

type DeltaCallbacks interface {
	// OnDeltaStreamOpen is called once an xDS stream is opened with a stream ID and the type URL (or "" for ADS).
	// Returning an error will end processing and close the stream. OnDeltaStreamClosed will still be called.
	OnDeltaStreamOpen(context.Context, int64, string) error
	// OnDeltaStreamClosed is called immediately prior to closing an xDS stream with a stream ID.
	OnDeltaStreamClosed(int64)
	// OnStreamDeltaRequest is called once a request is received on a stream.
	// Returning an error will end processing and close the stream. OnDeltaStreamClosed will still be called.
	OnStreamDeltaRequest(int64, DeltaDiscoveryRequest) error
	// OnStreamDeltaResponse is called immediately prior to sending a response on a stream.
	OnStreamDeltaResponse(int64, DeltaDiscoveryRequest, DeltaDiscoveryResponse)
}

// RestCallbacks defines rest.Callbacks for xDS fetch requests. The difference over real go-control-plane
// Callbacks is that it takes an DiscoveryRequest / DiscoveryResponse interface.
// It helps us to implement Callbacks once for many different versions of Envoy API.
type RestCallbacks interface {
	// OnFetchRequest is called when a new rest request comes in.
	// Returning an error will end processing. OnFetchResponse will not be called.
	OnFetchRequest(ctx context.Context, request DiscoveryRequest) error
	// OnFetchResponse is called immediately prior to sending a rest response.
	OnFetchResponse(request DiscoveryRequest, response DiscoveryResponse)
}

// MultiCallbacks implements callbacks for both rest and streaming xDS requests.
type MultiCallbacks interface {
	Callbacks
	RestCallbacks
}
