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

package subscriber

import (
	"testing"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/events"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/store/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/cache"
)

type recordingEmitter struct {
	events []events.Event
}

func (r *recordingEmitter) Send(event events.Event) {
	r.events = append(r.events, event)
}

func TestServiceProviderMetadataEventSubscriberSyncsServiceResource(t *testing.T) {
	appStore := memory.NewMemoryResourceStore(meshresource.ApplicationKind)
	serviceStore := memory.NewMemoryResourceStore(meshresource.ServiceKind)
	providerStore := memory.NewMemoryResourceStore(meshresource.ServiceProviderMetadataKind)
	require.NoError(t, appStore.Init(nil))
	require.NoError(t, serviceStore.Init(nil))
	require.NoError(t, providerStore.Init(nil))

	providerRes := newProviderMetadataResource("demo", "org.apache.DemoService", "v1", "g1", "shop-detail", "sayHello", "sayHello", "sayGoodbye")
	require.NoError(t, providerStore.Add(providerRes))

	emitter := &recordingEmitter{}
	sub := NewServiceProviderMetadataEventSubscriber(appStore, serviceStore, providerStore, emitter)
	err := sub.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, providerRes))
	require.NoError(t, err)

	appRaw, exists, err := appStore.GetByKey(coremodel.BuildResourceKey("demo", "shop-detail"))
	require.NoError(t, err)
	require.True(t, exists)
	appRes := appRaw.(*meshresource.ApplicationResource)
	assert.Equal(t, "shop-detail", appRes.Spec.Name)

	serviceKey := coremodel.BuildResourceKey("demo", meshresource.BuildServiceIdentityKey("org.apache.DemoService", "v1", "g1"))
	serviceRaw, exists, err := serviceStore.GetByKey(serviceKey)
	require.NoError(t, err)
	require.True(t, exists)
	serviceRes := serviceRaw.(*meshresource.ServiceResource)
	assert.Equal(t, "org.apache.DemoService", serviceRes.Spec.Name)
	assert.Equal(t, "v1", serviceRes.Spec.Version)
	assert.Equal(t, "g1", serviceRes.Spec.Group)
	assert.Equal(t, []string{"sayGoodbye", "sayHello"}, serviceRes.Spec.Methods)
}

func TestServiceProviderMetadataEventSubscriberDeletesServiceWhenLastProviderRemoved(t *testing.T) {
	appStore := memory.NewMemoryResourceStore(meshresource.ApplicationKind)
	serviceStore := memory.NewMemoryResourceStore(meshresource.ServiceKind)
	providerStore := memory.NewMemoryResourceStore(meshresource.ServiceProviderMetadataKind)
	require.NoError(t, appStore.Init(nil))
	require.NoError(t, serviceStore.Init(nil))
	require.NoError(t, providerStore.Init(nil))

	providerRes := newProviderMetadataResource("demo", "org.apache.DemoService", "v1", "g1", "shop-detail", "sayHello")
	require.NoError(t, providerStore.Add(providerRes))

	serviceRes := meshresource.NewServiceResourceWithAttributes(
		meshresource.BuildServiceIdentityKey("org.apache.DemoService", "v1", "g1"),
		"demo",
	)
	serviceRes.Spec = &meshproto.Service{
		Name:    "org.apache.DemoService",
		Group:   "g1",
		Version: "v1",
		Methods: []string{"sayHello"},
	}
	require.NoError(t, serviceStore.Add(serviceRes))
	require.NoError(t, providerStore.Delete(providerRes))

	emitter := &recordingEmitter{}
	sub := NewServiceProviderMetadataEventSubscriber(appStore, serviceStore, providerStore, emitter)
	err := sub.ProcessEvent(events.NewResourceChangedEvent(cache.Deleted, providerRes, nil))
	require.NoError(t, err)

	serviceKey := coremodel.BuildResourceKey("demo", meshresource.BuildServiceIdentityKey("org.apache.DemoService", "v1", "g1"))
	_, exists, err := serviceStore.GetByKey(serviceKey)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestServiceProviderMetadataEventSubscriberUpdatesServiceWhenIdentityChanges(t *testing.T) {
	appStore := memory.NewMemoryResourceStore(meshresource.ApplicationKind)
	serviceStore := memory.NewMemoryResourceStore(meshresource.ServiceKind)
	providerStore := memory.NewMemoryResourceStore(meshresource.ServiceProviderMetadataKind)
	require.NoError(t, appStore.Init(nil))
	require.NoError(t, serviceStore.Init(nil))
	require.NoError(t, providerStore.Init(nil))

	oldProviderRes := newProviderMetadataResource("demo", "org.apache.DemoService", "v1", "g1", "shop-detail", "sayHello")
	newProviderRes := newProviderMetadataResource("demo", "org.apache.DemoService", "v2", "g2", "shop-detail", "sayHelloV2")
	require.NoError(t, providerStore.Add(newProviderRes))

	oldServiceRes := meshresource.NewServiceResourceWithAttributes(
		meshresource.BuildServiceIdentityKey("org.apache.DemoService", "v1", "g1"),
		"demo",
	)
	oldServiceRes.Spec = &meshproto.Service{
		Name:    "org.apache.DemoService",
		Group:   "g1",
		Version: "v1",
		Methods: []string{"sayHello"},
	}
	require.NoError(t, serviceStore.Add(oldServiceRes))

	emitter := &recordingEmitter{}
	sub := NewServiceProviderMetadataEventSubscriber(appStore, serviceStore, providerStore, emitter)
	err := sub.ProcessEvent(events.NewResourceChangedEvent(cache.Updated, oldProviderRes, newProviderRes))
	require.NoError(t, err)

	oldServiceKey := coremodel.BuildResourceKey("demo", meshresource.BuildServiceIdentityKey("org.apache.DemoService", "v1", "g1"))
	_, exists, err := serviceStore.GetByKey(oldServiceKey)
	require.NoError(t, err)
	assert.False(t, exists)

	newServiceKey := coremodel.BuildResourceKey("demo", meshresource.BuildServiceIdentityKey("org.apache.DemoService", "v2", "g2"))
	serviceRaw, exists, err := serviceStore.GetByKey(newServiceKey)
	require.NoError(t, err)
	require.True(t, exists)
	serviceRes := serviceRaw.(*meshresource.ServiceResource)
	assert.Equal(t, "v2", serviceRes.Spec.Version)
	assert.Equal(t, "g2", serviceRes.Spec.Group)
	assert.Equal(t, []string{"sayHelloV2"}, serviceRes.Spec.Methods)
}

func newProviderMetadataResource(mesh, serviceName, version, group, providerAppName string, methods ...string) *meshresource.ServiceProviderMetadataResource {
	resKey := meshresource.BuildServiceKey(serviceName, version, group, providerAppName)
	res := meshresource.NewServiceProviderMetadataResourceWithAttributes(resKey, mesh)
	methodSpecs := make([]*meshproto.Method, 0, len(methods))
	for _, method := range methods {
		methodSpecs = append(methodSpecs, &meshproto.Method{Name: method})
	}
	res.Spec = &meshproto.ServiceProviderMetadata{
		ServiceName:     serviceName,
		ProviderAppName: providerAppName,
		Version:         version,
		Group:           group,
		Methods:         methodSpecs,
	}
	return res
}
