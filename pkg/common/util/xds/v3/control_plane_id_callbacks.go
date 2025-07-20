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
	envoy_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	envoy_xds "github.com/envoyproxy/go-control-plane/pkg/server/v3"
)

// controlPlaneIdCallbacks adds Control Plane ID to the DiscoveryResponse
type controlPlaneIdCallbacks struct {
	NoopCallbacks
	id string
}

var _ envoy_xds.Callbacks = &controlPlaneIdCallbacks{}

func NewControlPlaneIdCallbacks(id string) envoy_xds.Callbacks {
	return &controlPlaneIdCallbacks{
		id: id,
	}
}

func (c *controlPlaneIdCallbacks) OnStreamResponse(ctx context.Context, streamID int64, request *envoy_discovery.DiscoveryRequest, response *envoy_discovery.DiscoveryResponse) {
	if c.id != "" {
		response.ControlPlane = &envoy_core.ControlPlane{
			Identifier: c.id,
		}
	}
}

func (c *controlPlaneIdCallbacks) OnStreamDeltaResponse(streamID int64, request *envoy_discovery.DeltaDiscoveryRequest, response *envoy_discovery.DeltaDiscoveryResponse) {
	if c.id != "" {
		response.ControlPlane = &envoy_core.ControlPlane{
			Identifier: c.id,
		}
	}
}
