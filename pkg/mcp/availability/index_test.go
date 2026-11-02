/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package availability

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

func TestLiveServiceIndexStatesAndGracePeriod(t *testing.T) {
	service := meshresource.NewServiceResourceWithAttributes("orders", "mesh-1")
	service.Spec = &meshproto.Service{Name: "com.example.OrderService"}
	provider := meshresource.NewServiceProviderMetadataResourceWithAttributes("provider", "mesh-1")
	provider.Spec = &meshproto.ServiceProviderMetadata{
		ServiceName: "com.example.OrderService", ProviderAppName: "orders-app",
	}
	instance := meshresource.NewRPCInstanceResourceWithAttributes("instance", "mesh-1")
	instance.Spec = &meshproto.RPCInstance{
		AppName: "orders-app", Protocol: "tri",
		Endpoints: []*meshproto.Endpoint{{Protocol: "tri"}},
	}
	now := time.Unix(100, 0)
	index := NewLiveServiceIndex(time.Minute)
	results := index.Refresh(Snapshot{
		Services: []*meshresource.ServiceResource{service}, Providers: []*meshresource.ServiceProviderMetadataResource{provider},
		Instances: []*meshresource.RPCInstanceResource{instance}, Sync: SyncState{InitialSyncComplete: true, Healthy: true}, Now: now,
	})
	require.Len(t, results, 2)
	tri, ok := index.Get(Identity{Mesh: "mesh-1", ServiceName: "com.example.OrderService"}, "tri")
	require.True(t, ok)
	require.Equal(t, Active, tri.State)
	require.Equal(t, []string{"tri"}, tri.Evidence.Protocols)
	dubbo, ok := index.Get(Identity{Mesh: "mesh-1", ServiceName: "com.example.OrderService"}, "dubbo")
	require.True(t, ok)
	require.Equal(t, Inactive, dubbo.State)

	instance.Spec.UnregisterTime = "down"
	results = index.Refresh(Snapshot{
		Services: []*meshresource.ServiceResource{service}, Providers: []*meshresource.ServiceProviderMetadataResource{provider},
		Instances: []*meshresource.RPCInstanceResource{instance}, Sync: SyncState{InitialSyncComplete: true, Healthy: true}, Now: now.Add(30 * time.Second),
	})
	dubbo, _ = index.Get(Identity{Mesh: "mesh-1", ServiceName: "com.example.OrderService"}, "dubbo")
	require.Equal(t, Inactive, dubbo.State)
	require.True(t, index.Visible(dubbo, now.Add(30*time.Second)))
	results = index.Refresh(Snapshot{
		Services: []*meshresource.ServiceResource{service}, Providers: []*meshresource.ServiceProviderMetadataResource{provider},
		Instances: []*meshresource.RPCInstanceResource{instance}, Sync: SyncState{InitialSyncComplete: true, Healthy: true}, Now: now.Add(91 * time.Second),
	})
	dubbo, _ = index.Get(Identity{Mesh: "mesh-1", ServiceName: "com.example.OrderService"}, "dubbo")
	require.False(t, index.Visible(dubbo, now.Add(91*time.Second)))
}

func TestLiveServiceIndexUnknownAndProtocolEvidence(t *testing.T) {
	service := meshresource.NewServiceResourceWithAttributes("orders", "mesh-1")
	service.Spec = &meshproto.Service{Name: "com.example.OrderService"}
	provider := meshresource.NewServiceProviderMetadataResourceWithAttributes("provider", "mesh-1")
	provider.Spec = &meshproto.ServiceProviderMetadata{
		ServiceName: "com.example.OrderService", ProviderAppName: "orders-app",
	}
	index := NewLiveServiceIndex(time.Minute)
	results := index.Refresh(Snapshot{
		Services: []*meshresource.ServiceResource{service}, Providers: []*meshresource.ServiceProviderMetadataResource{provider},
		Sync: SyncState{InitialSyncComplete: false}, Now: time.Unix(100, 0),
	})
	require.Len(t, results, 2)
	for _, result := range results {
		require.Equal(t, Unknown, result.State)
		require.False(t, result.Evidence.Complete)
	}
}
