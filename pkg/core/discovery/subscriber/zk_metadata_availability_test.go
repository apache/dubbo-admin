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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/events"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/store/memory"
)

type availabilityStoreRouter struct {
	store store.ResourceStore
}

func (r availabilityStoreRouter) ResourceRoute(coremodel.Resource) (store.ResourceStore, error) {
	return r.store, nil
}

func (r availabilityStoreRouter) ResourceKindRoute(kind coremodel.ResourceKind) (store.ResourceStore, error) {
	if kind != meshresource.RPCInstanceMetadataKind {
		return nil, fmt.Errorf("unexpected kind %s", kind)
	}
	return r.store, nil
}

type availabilityEmitter struct{}

func (availabilityEmitter) Send(events.Event) {}

func TestZKApplicationRevisionNormalizesAndRemovesMetadata(t *testing.T) {
	st := memory.NewMemoryResourceStore(meshresource.RPCInstanceMetadataKind)
	require.NoError(t, st.Init(nil))
	subscriber := NewZKMetadataEventSubscriber(availabilityEmitter{}, availabilityStoreRouter{store: st})
	resource := meshresource.NewZKMetadataResourceWithAttributes("/dubbo/metadata/orders/rev-2", "mesh-1")
	resource.Spec = &meshproto.ZKMetadata{
		NodePath: "/dubbo/metadata/orders/rev-2",
		NodeData: `{"services":{"/com.example.OrderService::tri":{"port":50051}}}`,
	}
	require.NoError(t, subscriber.processUpsert(resource))
	raw, exists, err := st.GetByKey(coremodel.BuildResourceKey("mesh-1", "orders@rev-2"))
	require.NoError(t, err)
	require.True(t, exists)
	normalized := raw.(*meshresource.RPCInstanceMetadataResource)
	require.Equal(t, "tri", normalized.Spec.Services["/com.example.OrderService::tri"].Protocol)

	require.NoError(t, subscriber.processApplicationMetadataDelete(resource))
	_, exists, err = st.GetByKey(normalized.ResourceKey())
	require.NoError(t, err)
	require.False(t, exists)
}
