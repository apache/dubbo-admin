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
	"google.golang.org/protobuf/types/known/structpb"

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
		AppName: "orders-app", Protocol: "tri", Revision: "r1",
		Endpoints: []*meshproto.Endpoint{{Protocol: "tri"}},
	}
	metadata := meshresource.NewRPCInstanceMetadataResourceWithAttributes("orders-app@r1", "mesh-1")
	metadata.Spec = &meshproto.RPCInstanceMetadata{App: "orders-app", Revision: "r1", Services: map[string]*meshproto.ServiceInfo{
		"com.example.OrderService:tri": {Name: "com.example.OrderService", Protocol: "tri"},
	}}
	now := time.Unix(100, 0)
	index := NewLiveServiceIndex(time.Minute)
	results := index.Refresh(Snapshot{
		Services: []*meshresource.ServiceResource{service}, Providers: []*meshresource.ServiceProviderMetadataResource{provider},
		Metadata: []*meshresource.RPCInstanceMetadataResource{metadata}, Instances: []*meshresource.RPCInstanceResource{instance}, Sync: SyncState{InitialSyncComplete: true, Healthy: true}, Now: now,
	})
	require.Len(t, results, 2)
	tri, ok := index.Get(Identity{Mesh: "mesh-1", ServiceName: "com.example.OrderService"}, "tri")
	require.True(t, ok)
	require.Equal(t, Active, tri.State)
	require.Equal(t, []string{"tri"}, tri.Evidence.Protocols)
	dubbo, ok := index.Get(Identity{Mesh: "mesh-1", ServiceName: "com.example.OrderService"}, "dubbo")
	require.True(t, ok)
	require.Equal(t, Inactive, dubbo.State)
	require.True(t, ConfirmedProtocolIncompatibility(dubbo))

	instance.Spec.UnregisterTime = "down"
	results = index.Refresh(Snapshot{
		Services: []*meshresource.ServiceResource{service}, Providers: []*meshresource.ServiceProviderMetadataResource{provider},
		Metadata: []*meshresource.RPCInstanceMetadataResource{metadata}, Instances: []*meshresource.RPCInstanceResource{instance}, Sync: SyncState{InitialSyncComplete: true, Healthy: true}, Now: now.Add(30 * time.Second),
	})
	tri, _ = index.Get(Identity{Mesh: "mesh-1", ServiceName: "com.example.OrderService"}, "tri")
	require.Equal(t, Inactive, tri.State)
	require.True(t, index.Visible(tri, now.Add(30*time.Second)))
	require.False(t, ConfirmedProtocolIncompatibility(tri))
	results = index.Refresh(Snapshot{
		Services: []*meshresource.ServiceResource{service}, Providers: []*meshresource.ServiceProviderMetadataResource{provider},
		Metadata: []*meshresource.RPCInstanceMetadataResource{metadata}, Instances: []*meshresource.RPCInstanceResource{instance}, Sync: SyncState{InitialSyncComplete: true, Healthy: true}, Now: now.Add(91 * time.Second),
	})
	tri, _ = index.Get(Identity{Mesh: "mesh-1", ServiceName: "com.example.OrderService"}, "tri")
	require.False(t, index.Visible(tri, now.Add(91*time.Second)))
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

func TestLiveServiceIndexUsesCurrentServiceExports(t *testing.T) {
	identity := Identity{Mesh: "mesh-1", ServiceName: "com.example.OrderService"}
	provider := meshresource.NewServiceProviderMetadataResourceWithAttributes("provider", identity.Mesh)
	provider.Spec = &meshproto.ServiceProviderMetadata{
		ServiceName: identity.ServiceName, ProviderAppName: "orders-app",
	}
	instance := meshresource.NewRPCInstanceResourceWithAttributes("instance", identity.Mesh)
	instance.Spec = &meshproto.RPCInstance{AppName: "orders-app", Revision: "r2", Protocol: "tri"}
	oldRevision := meshresource.NewRPCInstanceMetadataResourceWithAttributes("orders-app@r1", identity.Mesh)
	oldRevision.Spec = &meshproto.RPCInstanceMetadata{
		App: "orders-app", Revision: "r1",
		Services: map[string]*meshproto.ServiceInfo{
			"orders": {Name: identity.ServiceName, Protocol: "tri"},
		},
	}
	newRevision := meshresource.NewRPCInstanceMetadataResourceWithAttributes("orders-app@r2", identity.Mesh)
	newRevision.Spec = &meshproto.RPCInstanceMetadata{App: "orders-app", Revision: "r2", Services: map[string]*meshproto.ServiceInfo{}}
	snapshot := Snapshot{
		Providers: []*meshresource.ServiceProviderMetadataResource{provider},
		Instances: []*meshresource.RPCInstanceResource{instance},
		Metadata:  []*meshresource.RPCInstanceMetadataResource{oldRevision, newRevision},
		Sync:      SyncState{InitialSyncComplete: true, Healthy: true},
	}
	removed := Evaluate(snapshot, identity, "tri")
	require.Equal(t, Inactive, removed.State, "old revision must not keep a removed interface active")
	require.False(t, removed.Evidence.Available)

	snapshot.Metadata = []*meshresource.RPCInstanceMetadataResource{oldRevision}
	missing := Evaluate(snapshot, identity, "tri")
	require.Equal(t, Unknown, missing.State)
	require.False(t, missing.Evidence.Complete)
	require.False(t, ConfirmedProtocolIncompatibility(missing))

	snapshot.Sync.Healthy = false
	outage := Evaluate(snapshot, identity, "tri")
	require.Equal(t, Unknown, outage.State)
	require.True(t, NewLiveServiceIndex(0).Visible(outage, time.Now().Add(2*time.Hour)))
}

func TestLiveServiceIndexInterfaceRegistrationAndIncompleteProtocol(t *testing.T) {
	identity := Identity{Mesh: "mesh-1", ServiceName: "com.example.OrderService"}
	provider := meshresource.NewServiceProviderMetadataResourceWithAttributes("provider", identity.Mesh)
	provider.Spec = &meshproto.ServiceProviderMetadata{ServiceName: identity.ServiceName, ProviderAppName: "orders-app"}
	registration := meshresource.NewRPCInstanceMetadataResourceWithAttributes("interface:orders", identity.Mesh)
	available := true
	registration.Spec = &meshproto.RPCInstanceMetadata{
		App: "orders-app", InstanceName: "orders-app-1", Available: &available,
		Services: map[string]*meshproto.ServiceInfo{
			"orders": {Name: identity.ServiceName, Protocol: "tri"},
		},
	}
	snapshot := Snapshot{
		Providers: []*meshresource.ServiceProviderMetadataResource{provider},
		Metadata:  []*meshresource.RPCInstanceMetadataResource{registration},
		Sync:      SyncState{InitialSyncComplete: true, Healthy: true},
	}
	tri := Evaluate(snapshot, identity, "tri")
	require.Equal(t, Active, tri.State)
	dubbo := Evaluate(snapshot, identity, "dubbo")
	require.Equal(t, Inactive, dubbo.State)
	require.True(t, ConfirmedProtocolIncompatibility(dubbo))

	registration.Spec.Services["orders"].Protocol = ""
	unknown := Evaluate(snapshot, identity, "dubbo")
	require.Equal(t, Unknown, unknown.State)
	require.True(t, unknown.Evidence.Available)
	require.False(t, ConfirmedProtocolIncompatibility(unknown))

	available = false
	down := Evaluate(snapshot, identity, "tri")
	require.Equal(t, Inactive, down.State)
	require.False(t, down.Evidence.Available)
}

func TestLiveServiceIndexEquivalentNormalizedRegistrations(t *testing.T) {
	identity := Identity{Mesh: "mesh-1", ServiceName: "com.example.OrderService"}
	provider := meshresource.NewServiceProviderMetadataResourceWithAttributes("provider", identity.Mesh)
	provider.Spec = &meshproto.ServiceProviderMetadata{ServiceName: identity.ServiceName, ProviderAppName: "orders"}
	available := true
	direct := meshresource.NewRPCInstanceMetadataResourceWithAttributes("interface:orders", identity.Mesh)
	direct.Spec = &meshproto.RPCInstanceMetadata{
		App: "orders", InstanceName: "orders-1", Available: &available,
		Services: map[string]*meshproto.ServiceInfo{"service": {Name: identity.ServiceName, Protocol: "tri"}},
	}
	instance := meshresource.NewRPCInstanceResourceWithAttributes("orders-1", identity.Mesh)
	instance.Spec = &meshproto.RPCInstance{AppName: "orders", Revision: "rev-1"}
	revision := meshresource.NewRPCInstanceMetadataResourceWithAttributes("orders@rev-1", identity.Mesh)
	revision.Spec = &meshproto.RPCInstanceMetadata{
		App: "orders", Revision: "rev-1", Services: direct.Spec.Services,
	}
	base := Snapshot{
		Providers: []*meshresource.ServiceProviderMetadataResource{provider},
		Sync:      SyncState{InitialSyncComplete: true, Healthy: true},
	}
	nacos := base
	nacos.Metadata = []*meshresource.RPCInstanceMetadataResource{direct}
	zookeeper := base
	zookeeper.Instances = []*meshresource.RPCInstanceResource{instance}
	zookeeper.Metadata = []*meshresource.RPCInstanceMetadataResource{revision}
	for _, protocol := range []string{"tri", "dubbo"} {
		fromNacos := Evaluate(nacos, identity, protocol)
		fromZK := Evaluate(zookeeper, identity, protocol)
		require.Equal(t, fromNacos.State, fromZK.State)
		require.Equal(t, fromNacos.Evidence, fromZK.Evidence)
		require.Equal(t, ConfirmedProtocolIncompatibility(fromNacos), ConfirmedProtocolIncompatibility(fromZK))
	}
}

func TestLiveServiceIndexIgnoresRegistrationFromAnotherMesh(t *testing.T) {
	identity := Identity{Mesh: "mesh-1", ServiceName: "com.example.OrderService"}
	provider := meshresource.NewServiceProviderMetadataResourceWithAttributes("provider", identity.Mesh)
	provider.Spec = &meshproto.ServiceProviderMetadata{
		ServiceName: identity.ServiceName, ProviderAppName: "orders",
	}
	available := true
	unrelated := meshresource.NewRPCInstanceMetadataResourceWithAttributes("interface:unrelated", "another-mesh")
	unrelated.Spec = &meshproto.RPCInstanceMetadata{
		App: "unrelated", InstanceName: "unrelated-1", Available: &available,
		Services: map[string]*meshproto.ServiceInfo{
			"service": {Name: identity.ServiceName, Protocol: "grpc"},
		},
	}
	index := NewLiveServiceIndex(time.Minute)
	index.Refresh(Snapshot{
		Providers: []*meshresource.ServiceProviderMetadataResource{provider},
		Metadata:  []*meshresource.RPCInstanceMetadataResource{unrelated},
		Sync:      SyncState{InitialSyncComplete: true, Healthy: true},
	})
	result, found := index.Get(identity, "tri")
	require.True(t, found)
	require.False(t, result.Evidence.Available)
	require.Empty(t, result.Evidence.Protocols)
}

func TestFilterToolsRetainsSchemaAndUsesContinuousConfirmedDowntime(t *testing.T) {
	for _, grace := range []time.Duration{0, 15 * time.Second} {
		t.Run(grace.String(), func(t *testing.T) {
			window := grace
			if window == 0 {
				window = time.Minute
			}
			idx := NewLiveServiceIndex(grace)
			schema, err := structpb.NewStruct(map[string]interface{}{"type": "object"})
			require.NoError(t, err)
			tool := &meshproto.MCPToolBinding{
				ToolId: "orders", InputSchema: schema,
				Operation: &meshproto.MCPOperation{ServiceName: "orders", Protocol: "tri"},
			}
			tools := []*meshproto.MCPToolBinding{tool}
			start := time.Unix(100, 0)
			snapshot := Snapshot{Sync: SyncState{InitialSyncComplete: true, Healthy: true}, Now: start}
			visible := idx.FilterTools("mesh-1", tools, snapshot)
			require.Len(t, visible, 1, "removed discovery Resources still receive a grace period")
			require.Same(t, tool, visible[0])
			require.Same(t, schema, visible[0].InputSchema)

			snapshot.Now = start.Add(window + time.Second)
			require.Empty(t, idx.FilterTools("mesh-1", tools, snapshot))
			snapshot.Sync.Healthy = false
			visible = idx.FilterTools("mesh-1", tools, snapshot)
			require.Equal(t, tools, visible, "an outage must restore visibility")

			snapshot.Now = snapshot.Now.Add(time.Hour)
			require.Equal(t, tools, idx.FilterTools("mesh-1", tools, snapshot))
			snapshot.Sync.Healthy = true
			require.Equal(t, tools, idx.FilterTools("mesh-1", tools, snapshot), "recovery starts a fresh grace period")
			snapshot.Now = snapshot.Now.Add(window + time.Second)
			require.Empty(t, idx.FilterTools("mesh-1", tools, snapshot))

			snapshot.Sync.InitialSyncComplete = false
			require.Equal(t, tools, idx.FilterTools("mesh-1", tools, snapshot))
		})
	}
}
