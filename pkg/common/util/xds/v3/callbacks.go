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
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/apache/dubbo-admin/pkg/common/util/xds"
)

// stream callbacks
type adapterCallbacks struct {
	NoopCallbacks
	callbacks xds.Callbacks
}

// AdaptCallbacks translate dubbo callbacks to real go-control-plane Callbacks
func AdaptCallbacks(callbacks xds.Callbacks) envoy_xds.Callbacks {
	return &adapterCallbacks{
		callbacks: callbacks,
	}
}

var _ envoy_xds.Callbacks = &adapterCallbacks{}

func (a *adapterCallbacks) OnStreamOpen(ctx context.Context, streamID int64, typeURL string) error {
	return a.callbacks.OnStreamOpen(ctx, streamID, typeURL)
}

func (a *adapterCallbacks) OnStreamClosed(streamID int64, _ *envoy_core.Node) {
	a.callbacks.OnStreamClosed(streamID)
}

func (a *adapterCallbacks) OnStreamRequest(streamID int64, request *envoy_sd.DiscoveryRequest) error {
	return a.callbacks.OnStreamRequest(streamID, &discoveryRequest{request})
}

func (a *adapterCallbacks) OnStreamResponse(ctx context.Context, streamID int64, request *envoy_sd.DiscoveryRequest, response *envoy_sd.DiscoveryResponse) {
	a.callbacks.OnStreamResponse(streamID, &discoveryRequest{request}, &discoveryResponse{response})
}

// delta callbacks

type adapterDeltaCallbacks struct {
	NoopCallbacks
	callbacks xds.DeltaCallbacks
}

// AdaptDeltaCallbacks translate dubbo callbacks to real go-control-plane Callbacks
func AdaptDeltaCallbacks(callbacks xds.DeltaCallbacks) envoy_xds.Callbacks {
	return &adapterDeltaCallbacks{
		callbacks: callbacks,
	}
}

var _ envoy_xds.Callbacks = &adapterDeltaCallbacks{}

func (a *adapterDeltaCallbacks) OnDeltaStreamOpen(ctx context.Context, streamID int64, typeURL string) error {
	return a.callbacks.OnDeltaStreamOpen(ctx, streamID, typeURL)
}

func (a *adapterDeltaCallbacks) OnDeltaStreamClosed(streamID int64, _ *envoy_core.Node) {
	a.callbacks.OnDeltaStreamClosed(streamID)
}

func (a *adapterDeltaCallbacks) OnStreamDeltaRequest(streamID int64, request *envoy_sd.DeltaDiscoveryRequest) error {
	return a.callbacks.OnStreamDeltaRequest(streamID, &deltaDiscoveryRequest{request})
}

func (a *adapterDeltaCallbacks) OnStreamDeltaResponse(streamID int64, request *envoy_sd.DeltaDiscoveryRequest, response *envoy_sd.DeltaDiscoveryResponse) {
	a.callbacks.OnStreamDeltaResponse(streamID, &deltaDiscoveryRequest{request}, &deltaDiscoveryResponse{response})
}

// rest callbacks

type adapterRestCallbacks struct {
	NoopCallbacks
	callbacks xds.RestCallbacks
}

// AdaptRestCallbacks translate dubbo callbacks to real go-control-plane Callbacks
func AdaptRestCallbacks(callbacks xds.RestCallbacks) envoy_xds.Callbacks {
	return &adapterRestCallbacks{
		callbacks: callbacks,
	}
}

func (a *adapterRestCallbacks) OnFetchRequest(ctx context.Context, request *envoy_sd.DiscoveryRequest) error {
	return a.callbacks.OnFetchRequest(ctx, &discoveryRequest{request})
}

func (a *adapterRestCallbacks) OnFetchResponse(request *envoy_sd.DiscoveryRequest, response *envoy_sd.DiscoveryResponse) {
	a.callbacks.OnFetchResponse(&discoveryRequest{request}, &discoveryResponse{response})
}

// Both rest and stream

type adapterMultiCallbacks struct {
	NoopCallbacks
	callbacks xds.MultiCallbacks
}

// AdaptMultiCallbacks translate dubbo callbacks to real go-control-plane Callbacks
func AdaptMultiCallbacks(callbacks xds.MultiCallbacks) envoy_xds.Callbacks {
	return &adapterMultiCallbacks{
		callbacks: callbacks,
	}
}

func (a *adapterMultiCallbacks) OnFetchRequest(ctx context.Context, request *envoy_sd.DiscoveryRequest) error {
	return a.callbacks.OnFetchRequest(ctx, &discoveryRequest{request})
}

func (a *adapterMultiCallbacks) OnFetchResponse(request *envoy_sd.DiscoveryRequest, response *envoy_sd.DiscoveryResponse) {
	a.callbacks.OnFetchResponse(&discoveryRequest{request}, &discoveryResponse{response})
}

func (a *adapterMultiCallbacks) OnStreamOpen(ctx context.Context, streamID int64, typeURL string) error {
	return a.callbacks.OnStreamOpen(ctx, streamID, typeURL)
}

func (a *adapterMultiCallbacks) OnStreamClosed(streamID int64, _ *envoy_core.Node) {
	a.callbacks.OnStreamClosed(streamID)
}

func (a *adapterMultiCallbacks) OnStreamRequest(streamID int64, request *envoy_sd.DiscoveryRequest) error {
	return a.callbacks.OnStreamRequest(streamID, &discoveryRequest{request})
}

func (a *adapterMultiCallbacks) OnStreamResponse(ctx context.Context, streamID int64, request *envoy_sd.DiscoveryRequest, response *envoy_sd.DiscoveryResponse) {
	a.callbacks.OnStreamResponse(streamID, &discoveryRequest{request}, &discoveryResponse{response})
}

// DiscoveryRequest facade

type discoveryRequest struct {
	*envoy_sd.DiscoveryRequest
}

func (d *discoveryRequest) Metadata() *structpb.Struct {
	return d.GetNode().GetMetadata()
}

func (d *discoveryRequest) VersionInfo() string {
	return d.GetVersionInfo()
}

func (d *discoveryRequest) NodeId() string {
	return d.GetNode().GetId()
}

func (d *discoveryRequest) Node() interface{} {
	return d.GetNode()
}

func (d *discoveryRequest) HasErrors() bool {
	return d.ErrorDetail != nil
}

func (d *discoveryRequest) ErrorMsg() string {
	return d.GetErrorDetail().GetMessage()
}

func (d *discoveryRequest) ResourceNames() []string {
	return d.GetResourceNames()
}

var _ xds.DiscoveryRequest = &discoveryRequest{}

type discoveryResponse struct {
	*envoy_sd.DiscoveryResponse
}

func (d *discoveryResponse) VersionInfo() string {
	return d.GetVersionInfo()
}

type deltaDiscoveryRequest struct {
	*envoy_sd.DeltaDiscoveryRequest
}

func (d *deltaDiscoveryRequest) Metadata() *structpb.Struct {
	return d.GetNode().GetMetadata()
}

func (d *deltaDiscoveryRequest) NodeId() string {
	return d.GetNode().GetId()
}

func (d *deltaDiscoveryRequest) Node() interface{} {
	return d.GetNode()
}

func (d *deltaDiscoveryRequest) HasErrors() bool {
	return d.ErrorDetail != nil
}

func (d *deltaDiscoveryRequest) ErrorMsg() string {
	return d.GetErrorDetail().GetMessage()
}

func (d *deltaDiscoveryRequest) ResourceNames() []string {
	return d.GetResourceNamesSubscribe()
}

func (d *deltaDiscoveryRequest) GetInitialResourceVersions() map[string]string {
	return d.InitialResourceVersions
}

var _ xds.DeltaDiscoveryRequest = &deltaDiscoveryRequest{}

type deltaDiscoveryResponse struct {
	*envoy_sd.DeltaDiscoveryResponse
}

var _ xds.DeltaDiscoveryResponse = &deltaDiscoveryResponse{}

func (d *deltaDiscoveryResponse) GetTypeUrl() string {
	return d.TypeUrl
}
