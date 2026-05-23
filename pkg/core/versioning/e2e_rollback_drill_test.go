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

package versioning

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/cache"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/events"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

func TestE2ERollbackDrill(t *testing.T) {
	store := NewMemoryStore()
	hints := NewAdminHintRegistry()
	maxVersions := int64(5)
	svc := NewServiceWithRollbackWait(true, maxVersions, 0, 30*time.Second, time.Second, store, hints)
	sub := NewSubscriber(meshresource.ConditionRouteKind, store, hints, maxVersions, 0)
	bus := newTestEventBus(t)
	defer bus.WaitForDone()
	require.NoError(t, bus.Subscribe(sub))
	require.NoError(t, bus.Start(nil, nil))

	original := newE2EConditionRoute(1)
	require.NoError(t, RecordBootstrap(store, maxVersions, original))
	items := requireVersions(t, store, original.ResourceKey(), 1)
	require.Equal(t, SourceBootstrap, items[0].Source)
	require.Equal(t, OperationCreate, items[0].Operation)
	require.Equal(t, int64(1), items[0].VersionNo)
	require.Equal(t, "system:bootstrap", items[0].Author)
	bootstrapID := items[0].ID

	adminEdit := newE2EConditionRoute(2)
	_, err := svc.RecordMutation(adminEdit, OperationUpdate, SourceAdmin, "alice", "raise priority", nil)
	require.NoError(t, err)
	bus.Send(events.NewResourceChangedEvent(cache.Updated, original, adminEdit))
	items = requireVersions(t, store, original.ResourceKey(), 2)
	require.Equal(t, SourceAdmin, items[0].Source)
	require.Equal(t, "alice", items[0].Author)
	require.Equal(t, int64(2), items[0].VersionNo)

	upstreamPush := newE2EConditionRoute(3)
	bus.Send(events.NewResourceChangedEventWithContext(cache.Updated, adminEdit, upstreamPush, map[string]string{
		events.SourceRegistryContextKey: "zookeeper",
	}))
	items = requireVersions(t, store, original.ResourceKey(), 3)
	require.Equal(t, SourceUpstream, items[0].Source)
	require.Equal(t, "system:zookeeper", items[0].Author)
	require.Equal(t, int64(3), items[0].VersionNo)

	fromID := bootstrapID
	rollback, err := svc.RecordMutation(original, OperationUpdate, SourceRollback, "bob", "restore bootstrap baseline", &fromID)
	require.NoError(t, err)
	bus.Send(events.NewResourceChangedEvent(cache.Updated, upstreamPush, original))
	require.Equal(t, SourceRollback, rollback.Source)
	require.Equal(t, OperationUpdate, rollback.Operation)
	require.NotNil(t, rollback.RolledBackFromID)
	require.Equal(t, bootstrapID, *rollback.RolledBackFromID)
	require.Equal(t, "bob", rollback.Author)
	require.Equal(t, int64(4), rollback.VersionNo)

	items = requireVersions(t, store, original.ResourceKey(), 4)
	requireAuditChainReadable(t, items, []Source{SourceRollback, SourceUpstream, SourceAdmin, SourceBootstrap})

	previous := newE2EConditionRoute(1)
	for priority := int32(4); priority <= 9; priority++ {
		next := newE2EConditionRoute(priority)
		_, err := svc.RecordMutation(next, OperationUpdate, SourceAdmin, "alice", "bulk edit", nil)
		require.NoError(t, err)
		bus.Send(events.NewResourceChangedEvent(cache.Updated, previous, next))
		previous = next
		require.Eventually(t, func() bool {
			latest, err := store.LatestVersion(meshresource.ConditionRouteKind, original.ResourceKey())
			return err == nil && latest != nil && latest.VersionNo == int64(priority+1)
		}, time.Second, 10*time.Millisecond)
	}

	items = requireVersions(t, store, original.ResourceKey(), int(maxVersions))
	require.Equal(t, []int64{10, 9, 8, 7, 6}, versionNumbers(items))
	for _, item := range items {
		require.NotEmpty(t, item.Author)
		require.False(t, item.CreatedAt.IsZero())
	}
}

func newE2EConditionRoute(priority int32) *meshresource.ConditionRouteResource {
	res := meshresource.NewConditionRouteResourceWithAttributes("demo.condition-router", "mesh")
	res.Spec = &meshproto.ConditionRoute{
		Key:        "demo",
		Enabled:    true,
		Priority:   priority,
		Conditions: []string{"host = 127.0.0.1"},
	}
	return res
}

func requireVersions(t *testing.T, store Store, resourceKey string, count int) []Version {
	t.Helper()
	var items []Version
	require.Eventually(t, func() bool {
		var err error
		items, err = store.ListVersions(meshresource.ConditionRouteKind, resourceKey)
		return err == nil && len(items) == count
	}, time.Second, 10*time.Millisecond)
	return items
}

func requireAuditChainReadable(t *testing.T, items []Version, sources []Source) {
	t.Helper()
	require.Len(t, items, len(sources))
	for i, item := range items {
		require.Equal(t, sources[i], item.Source)
		require.NotEmpty(t, item.Author)
		require.False(t, item.CreatedAt.IsZero())
		if i > 0 {
			require.False(t, items[i-1].CreatedAt.Before(item.CreatedAt))
		}
	}
}

func versionNumbers(items []Version) []int64 {
	numbers := make([]int64, 0, len(items))
	for _, item := range items {
		numbers = append(numbers, item.VersionNo)
	}
	return numbers
}
