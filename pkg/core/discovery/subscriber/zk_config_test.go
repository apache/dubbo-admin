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
	"k8s.io/client-go/tools/cache"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/events"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	corestore "github.com/apache/dubbo-admin/pkg/core/store"
	memorystore "github.com/apache/dubbo-admin/pkg/store/memory"
)

func TestZKConfigDeleteUsesLocalOldRule(t *testing.T) {
	ruleStore := memorystore.NewMemoryResourceStore(meshresource.TagRouteKind)
	require.NoError(t, ruleStore.Init(nil))
	oldRule := meshresource.NewTagRouteResourceWithAttributes("demo.tag-router", "mesh")
	oldRule.Spec = &meshproto.TagRoute{Key: "demo", Priority: 7}
	require.NoError(t, ruleStore.Add(oldRule))

	emitter := &capturingEmitter{}
	sub := NewZKConfigEventSubscriber(emitter, singleStoreRouter{store: ruleStore})
	zkConfig := newZKRuleConfig("demo.tag-router", "mesh", "")

	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Deleted, zkConfig, nil)))

	_, exists, err := ruleStore.GetByKey(oldRule.ResourceKey())
	require.NoError(t, err)
	require.False(t, exists)
	require.Len(t, emitter.events, 1)
	require.Equal(t, cache.Deleted, emitter.events[0].Type())
	require.Equal(t, oldRule.ResourceKey(), emitter.events[0].OldObj().ResourceKey())
	require.Nil(t, emitter.events[0].NewObj())
}

func TestZKConfigDeleteMissingLocalRuleIsNoop(t *testing.T) {
	ruleStore := memorystore.NewMemoryResourceStore(meshresource.TagRouteKind)
	require.NoError(t, ruleStore.Init(nil))

	emitter := &capturingEmitter{}
	sub := NewZKConfigEventSubscriber(emitter, singleStoreRouter{store: ruleStore})
	zkConfig := newZKRuleConfig("demo.tag-router", "mesh", "")

	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Deleted, zkConfig, nil)))
	require.Empty(t, emitter.events)
}

type singleStoreRouter struct {
	store corestore.ResourceStore
}

func (r singleStoreRouter) ResourceRoute(model.Resource) (corestore.ResourceStore, error) {
	return r.store, nil
}

func (r singleStoreRouter) ResourceKindRoute(model.ResourceKind) (corestore.ResourceStore, error) {
	if r.store == nil {
		return nil, fmt.Errorf("no store configured")
	}
	return r.store, nil
}

type capturingEmitter struct {
	events []events.Event
}

func (e *capturingEmitter) Send(event events.Event) {
	e.events = append(e.events, event)
}

func newZKRuleConfig(name, mesh, nodeData string) *meshresource.ZKConfigResource {
	res := meshresource.NewZKConfigResourceWithAttributes(name, mesh)
	res.Spec.NodeName = name
	res.Spec.NodeData = nodeData
	return res
}
