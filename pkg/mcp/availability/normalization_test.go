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

package availability_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/cache"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	"github.com/apache/dubbo-admin/pkg/core/discovery/subscriber"
	"github.com/apache/dubbo-admin/pkg/core/events"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/mcp/availability"
	"github.com/apache/dubbo-admin/pkg/store/memory"
)

type registrationRouter map[model.ResourceKind]store.ResourceStore

func (r registrationRouter) ResourceRoute(resource model.Resource) (store.ResourceStore, error) {
	return r.ResourceKindRoute(resource.ResourceKind())
}

func (r registrationRouter) ResourceKindRoute(kind model.ResourceKind) (store.ResourceStore, error) {
	result := r[kind]
	if result == nil {
		return nil, fmt.Errorf("unexpected resource kind %s", kind)
	}
	return result, nil
}

type registrationEmitter struct{}

func (registrationEmitter) Send(events.Event) {}

func newRegistrationRouter(t *testing.T) registrationRouter {
	t.Helper()
	result := make(registrationRouter)
	for _, kind := range []model.ResourceKind{
		meshresource.RPCInstanceKind,
		meshresource.RPCInstanceMetadataKind,
		meshresource.LifecycleEventKind,
	} {
		resourceStore := memory.NewMemoryResourceStore(kind)
		require.NoError(t, resourceStore.Init(nil))
		result[kind] = resourceStore
	}
	return result
}

func registrationSnapshot(t *testing.T, router registrationRouter) availability.Snapshot {
	t.Helper()
	snapshot := availability.Snapshot{Sync: availability.SyncState{InitialSyncComplete: true, Healthy: true}}
	for _, object := range router[meshresource.RPCInstanceKind].List() {
		snapshot.Instances = append(snapshot.Instances, object.(*meshresource.RPCInstanceResource))
	}
	for _, object := range router[meshresource.RPCInstanceMetadataKind].List() {
		snapshot.Metadata = append(snapshot.Metadata, object.(*meshresource.RPCInstanceMetadataResource))
	}
	return snapshot
}

func assertEquivalentEvidence(t *testing.T, nacos, zk availability.Snapshot, identity availability.Identity) {
	t.Helper()
	for _, protocol := range []string{"tri", "dubbo"} {
		fromNacos := availability.Evaluate(nacos, identity, protocol)
		fromZK := availability.Evaluate(zk, identity, protocol)
		require.Equal(t, fromNacos.State, fromZK.State, protocol)
		require.Equal(t, fromNacos.Evidence, fromZK.Evidence, protocol)
		require.Equal(t, availability.ConfirmedProtocolIncompatibility(fromNacos),
			availability.ConfirmedProtocolIncompatibility(fromZK), protocol)
	}
}

func TestNormalizedInterfaceRegistrationsHaveEquivalentEvidence(t *testing.T) {
	const (
		mesh = "mesh-1"
		name = "com.example.OrderService"
	)
	nacosRouter, zkRouter := newRegistrationRouter(t), newRegistrationRouter(t)
	nacosSubscriber := subscriber.NewNacosServiceEventSubscriber(registrationEmitter{}, nacosRouter)
	zkSubscriber := subscriber.NewZKMetadataEventSubscriber(registrationEmitter{}, zkRouter)
	nacos := meshresource.NewNacosServiceResourceWithAttributes("providers:"+name+":v1:orders", mesh)
	nacos.Spec.Instances = []*meshproto.NacosInstance{{
		Ip: "10.0.0.1", Port: 50051, Healthy: true, Enabled: true, Weight: 1,
		Metadata: map[string]string{
			constants.Application:  "orders",
			constants.URLParamsKey: `{"protocol":"tri"}`,
		},
	}}
	encoded := url.PathEscape("tri://10.0.0.1:50051/" + name +
		"?application=orders&group=orders&version=v1&enabled=true")
	path := "/dubbo/" + name + "/providers/" + encoded
	zk := meshresource.NewZKMetadataResourceWithAttributes(path, mesh)
	zk.Spec = &meshproto.ZKMetadata{NodePath: path}
	identity := availability.Identity{Mesh: mesh, ServiceName: name, Group: "orders", Version: "v1"}
	require.NoError(t, nacosSubscriber.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, nacos)))
	require.NoError(t, zkSubscriber.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, zk)))
	initial := registrationSnapshot(t, nacosRouter)
	require.Equal(t, availability.Active, availability.Evaluate(initial, identity, "tri").State)
	require.True(t, availability.ConfirmedProtocolIncompatibility(availability.Evaluate(initial, identity, "dubbo")))
	assertEquivalentEvidence(t, initial, registrationSnapshot(t, zkRouter), identity)

	nacos.Spec.Instances[0].Healthy = false
	downPath := "/dubbo/" + name + "/providers/" +
		url.PathEscape("tri://10.0.0.1:50051/"+name+"?application=orders&group=orders&version=v1&enabled=false")
	zkDown := meshresource.NewZKMetadataResourceWithAttributes(downPath, mesh)
	zkDown.Spec = &meshproto.ZKMetadata{NodePath: downPath}
	require.NoError(t, nacosSubscriber.ProcessEvent(events.NewResourceChangedEvent(cache.Updated, nil, nacos)))
	require.NoError(t, zkSubscriber.ProcessEvent(events.NewResourceChangedEvent(cache.Deleted, zk, nil)))
	require.NoError(t, zkSubscriber.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, zkDown)))
	down := registrationSnapshot(t, nacosRouter)
	require.Equal(t, availability.Inactive, availability.Evaluate(down, identity, "tri").State)
	require.False(t, availability.ConfirmedProtocolIncompatibility(availability.Evaluate(down, identity, "dubbo")))
	assertEquivalentEvidence(t, down, registrationSnapshot(t, zkRouter), identity)
}

func TestNormalizedApplicationRegistrationsHaveEquivalentEvidence(t *testing.T) {
	const (
		mesh = "mesh-1"
		name = "com.example.OrderService"
	)
	nacosRouter, zkRouter := newRegistrationRouter(t), newRegistrationRouter(t)
	nacosSubscriber := subscriber.NewNacosServiceEventSubscriber(registrationEmitter{}, nacosRouter)
	zkSubscriber := subscriber.NewZKMetadataEventSubscriber(registrationEmitter{}, zkRouter)
	metadata := `{"services":{"orders/` + name + `:v1:tri":{}}}`
	revision, err := meshresource.ParseRPCInstanceMetadata(mesh, "orders", "rev-1", metadata)
	require.NoError(t, err)
	nacos := meshresource.NewNacosServiceResourceWithAttributes("orders", mesh)
	nacos.Spec.Instances = []*meshproto.NacosInstance{{
		Ip: "10.0.0.1", Port: 50051, Healthy: true, Enabled: true, Weight: 1,
		Metadata: map[string]string{constants.MetadataRevisionKey: "rev-1"},
	}}
	nacos.Spec.Revisions = []*meshproto.RPCInstanceMetadata{revision.Spec}
	path := "/dubbo/metadata/orders/rev-1"
	zk := meshresource.NewZKMetadataResourceWithAttributes(path, mesh)
	zk.Spec = &meshproto.ZKMetadata{NodePath: path, NodeData: metadata}
	require.NoError(t, nacosSubscriber.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, nacos)))
	require.NoError(t, zkSubscriber.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, zk)))
	zkInstance := meshresource.ToRPCInstance(mesh, "orders", "10.0.0.1", 50051,
		map[string]string{constants.MetadataRevisionKey: "rev-1"})
	require.NoError(t, zkRouter[meshresource.RPCInstanceKind].Add(zkInstance))
	identity := availability.Identity{Mesh: mesh, ServiceName: name, Group: "orders", Version: "v1"}
	initial := registrationSnapshot(t, nacosRouter)
	require.Equal(t, availability.Active, availability.Evaluate(initial, identity, "tri").State)
	require.True(t, availability.ConfirmedProtocolIncompatibility(availability.Evaluate(initial, identity, "dubbo")))
	assertEquivalentEvidence(t, initial, registrationSnapshot(t, zkRouter), identity)

	nacos.Spec.Instances[0].Healthy = false
	require.NoError(t, nacosSubscriber.ProcessEvent(events.NewResourceChangedEvent(cache.Updated, nil, nacos)))
	require.NoError(t, zkRouter[meshresource.RPCInstanceKind].Delete(zkInstance))
	down := registrationSnapshot(t, nacosRouter)
	require.Equal(t, availability.Inactive, availability.Evaluate(down, identity, "tri").State)
	require.False(t, availability.ConfirmedProtocolIncompatibility(availability.Evaluate(down, identity, "dubbo")))
	assertEquivalentEvidence(t, down, registrationSnapshot(t, zkRouter), identity)
}
