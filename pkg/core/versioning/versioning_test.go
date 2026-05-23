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
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/cache"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	appconfig "github.com/apache/dubbo-admin/pkg/config/app"
	eventbusconfig "github.com/apache/dubbo-admin/pkg/config/eventbus"
	"github.com/apache/dubbo-admin/pkg/config/mode"
	versioningcfg "github.com/apache/dubbo-admin/pkg/config/versioning"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	coreruntime "github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

func TestNormalizeSpecHashStable(t *testing.T) {
	hash1, spec1, err := NormalizeSpec(&meshproto.ConditionRoute{
		Enabled:    true,
		Conditions: []string{"host = 127.0.0.1"},
		Key:        "demo",
	})
	require.NoError(t, err)
	hash2, spec2, err := NormalizeSpec(&meshproto.ConditionRoute{
		Key:        "demo",
		Conditions: []string{"host = 127.0.0.1"},
		Enabled:    true,
	})
	require.NoError(t, err)
	require.Equal(t, spec1, spec2)
	require.Equal(t, hash1, hash2)
	require.NotEmpty(t, hash1)
}

func TestAdminHintTTLHitAndMiss(t *testing.T) {
	now := time.Now()
	reg := NewAdminHintRegistry()
	reg.now = func() time.Time { return now }
	reg.Put(meshresource.ConditionRouteKind, "m/r", "hash", AdminHint{
		Source:    SourceAdmin,
		Author:    "alice",
		ExpiresAt: now.Add(time.Second),
	})
	hint, ok := reg.Take(meshresource.ConditionRouteKind, "m/r", "hash")
	require.True(t, ok)
	require.Equal(t, "alice", hint.Author)
	_, ok = reg.Take(meshresource.ConditionRouteKind, "m/r", "hash")
	require.False(t, ok)

	reg.Put(meshresource.ConditionRouteKind, "m/r", "expired", AdminHint{ExpiresAt: now.Add(-time.Second)})
	_, ok = reg.Take(meshresource.ConditionRouteKind, "m/r", "expired")
	require.False(t, ok)
}

func TestAdminHintPrunesExpiredOnPutAndTake(t *testing.T) {
	now := time.Now()
	reg := NewAdminHintRegistry()
	reg.now = func() time.Time { return now }

	reg.Put(meshresource.ConditionRouteKind, "m/stale", "stale", AdminHint{ExpiresAt: now.Add(-time.Second)})
	require.Len(t, reg.hints, 1)

	reg.Put(meshresource.ConditionRouteKind, "m/fresh", "fresh", AdminHint{
		ExpiresAt: now.Add(time.Second),
	})
	require.Len(t, reg.hints, 1)

	reg.Put(meshresource.ConditionRouteKind, "m/stale", "stale", AdminHint{ExpiresAt: now.Add(-time.Second)})
	hint, ok := reg.Take(meshresource.ConditionRouteKind, "m/fresh", "fresh")
	require.True(t, ok)
	require.Empty(t, hint.Author)
	require.Empty(t, reg.hints)
}

func TestMemoryStoreRetentionCurrentPointerAndDelete(t *testing.T) {
	store := NewMemoryStore()
	key := "mesh/demo.condition-router"
	for i := 0; i < 4; i++ {
		hash, specJSON, err := NormalizeSpec(&meshproto.ConditionRoute{Key: "demo", Priority: int32(i + 1)})
		require.NoError(t, err)
		_, err = store.InsertVersion(InsertRequest{
			RuleKind:    meshresource.ConditionRouteKind,
			Mesh:        "mesh",
			ResourceKey: key,
			RuleName:    "demo.condition-router",
			SpecJSON:    specJSON,
			ContentHash: hash,
			Source:      SourceAdmin,
			Operation:   OperationUpdate,
			Author:      "alice",
			CreatedAt:   time.Now().Add(time.Duration(i) * time.Second),
		}, 2)
		require.NoError(t, err)
	}
	items, err := store.ListVersions(meshresource.ConditionRouteKind, key)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, int64(4), items[0].VersionNo)
	require.Equal(t, int64(3), items[1].VersionNo)
	require.True(t, items[0].IsCurrent)
	meta, err := store.CurrentMeta(meshresource.ConditionRouteKind, key)
	require.NoError(t, err)
	require.Equal(t, int64(4), meta.LastVersionNo)

	_, err = store.InsertVersion(InsertRequest{
		RuleKind:    meshresource.ConditionRouteKind,
		Mesh:        "mesh",
		ResourceKey: key,
		RuleName:    "demo.condition-router",
		SpecJSON:    DeleteSpecJSON,
		ContentHash: HashSpecJSON(DeleteSpecJSON),
		Source:      SourceAdmin,
		Operation:   OperationDelete,
		Author:      "alice",
		CreatedAt:   time.Now().Add(5 * time.Second),
	}, 2)
	require.NoError(t, err)
	meta, err = store.CurrentMeta(meshresource.ConditionRouteKind, key)
	require.NoError(t, err)
	require.Nil(t, meta.CurrentVersion)
	require.Equal(t, int64(5), meta.LastVersionNo)
}

func TestMemoryStoreDeleteIsNotDedupedAgainstEmptyCurrentSpec(t *testing.T) {
	store := NewMemoryStore()
	key := "mesh/demo.condition-router"
	created, err := store.InsertVersion(InsertRequest{
		RuleKind:    meshresource.ConditionRouteKind,
		Mesh:        "mesh",
		ResourceKey: key,
		RuleName:    "demo.condition-router",
		SpecJSON:    DeleteSpecJSON,
		ContentHash: HashSpecJSON(DeleteSpecJSON),
		Source:      SourceAdmin,
		Operation:   OperationCreate,
		Author:      "alice",
		CreatedAt:   time.Now(),
	}, 5)
	require.NoError(t, err)
	deleted, err := store.InsertVersion(InsertRequest{
		RuleKind:    meshresource.ConditionRouteKind,
		Mesh:        "mesh",
		ResourceKey: key,
		RuleName:    "demo.condition-router",
		SpecJSON:    DeleteSpecJSON,
		ContentHash: HashSpecJSON(DeleteSpecJSON),
		Source:      SourceAdmin,
		Operation:   OperationDelete,
		Author:      "alice",
		CreatedAt:   time.Now().Add(time.Second),
	}, 5)
	require.NoError(t, err)

	require.NotEqual(t, created.ID, deleted.ID)
	require.Equal(t, int64(2), deleted.VersionNo)
	items, err := store.ListVersions(meshresource.ConditionRouteKind, key)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, OperationDelete, items[0].Operation)
	meta, err := store.CurrentMeta(meshresource.ConditionRouteKind, key)
	require.NoError(t, err)
	require.Nil(t, meta.CurrentVersion)
}

func TestMemoryStoreIntentGetListOpenAndFailWithReason(t *testing.T) {
	store := NewMemoryStore()
	key := "mesh/demo.condition-router"
	intent, err := store.CreateIntent(InsertRequest{
		RuleKind:    meshresource.ConditionRouteKind,
		Mesh:        "mesh",
		ResourceKey: key,
		RuleName:    "demo.condition-router",
		SpecJSON:    `{"priority":1}`,
		ContentHash: "hash-1",
		Source:      SourceAdmin,
		Operation:   OperationUpdate,
		Author:      "alice",
		CreatedAt:   time.Now(),
	}, nil)
	require.NoError(t, err)

	got, err := store.GetIntent(intent.ID)
	require.NoError(t, err)
	require.Equal(t, IntentStatusPending, got.Status)
	open, err := store.ListOpenIntents()
	require.NoError(t, err)
	require.Len(t, open, 1)
	require.Equal(t, intent.ID, open[0].ID)

	require.NoError(t, store.MarkIntentFailedWithReason(intent.ID, "registry rejected mutation"))
	got, err = store.GetIntent(intent.ID)
	require.NoError(t, err)
	require.Equal(t, IntentStatusFailed, got.Status)
	require.Equal(t, "registry rejected mutation", got.LastError)
	open, err = store.ListOpenIntents()
	require.NoError(t, err)
	require.Empty(t, open)
	require.ErrorIs(t, store.MarkIntentFailedWithReason(intent.ID, "again"), ErrVersionIntentNotOpen)
	_, err = store.GetIntent(404)
	require.ErrorIs(t, err, ErrVersionIntentNotFound)
}

func TestSubscriberCoalescesBursts(t *testing.T) {
	store := NewMemoryStore()
	hints := NewAdminHintRegistry()
	sub := NewSubscriber(meshresource.ConditionRouteKind, store, hints, 5, time.Hour)
	key := "mesh/demo.condition-router"
	first := meshresource.NewConditionRouteResourceWithAttributes("demo.condition-router", "mesh")
	first.Spec = &meshproto.ConditionRoute{Key: "demo", Priority: 1}
	second := meshresource.NewConditionRouteResourceWithAttributes("demo.condition-router", "mesh")
	second.Spec = &meshproto.ConditionRoute{Key: "demo", Priority: 2}
	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Updated, nil, first)))
	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Updated, first, second)))
	sub.FlushAll()
	items, err := store.ListVersions(meshresource.ConditionRouteKind, key)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Contains(t, items[0].SpecJSON, `"priority":2`)
}

func TestSubscriberCapturesAdminAndUpstreamSources(t *testing.T) {
	store := NewMemoryStore()
	hints := NewAdminHintRegistry()
	sub := NewSubscriber(meshresource.ConditionRouteKind, store, hints, 5, 0)
	adminRes := meshresource.NewConditionRouteResourceWithAttributes("demo.condition-router", "mesh")
	adminRes.Spec = &meshproto.ConditionRoute{Key: "demo", Priority: 1}
	hash, _, err := NormalizeResource(adminRes)
	require.NoError(t, err)
	hints.Put(adminRes.ResourceKind(), adminRes.ResourceKey(), hash, AdminHint{
		Source:    SourceAdmin,
		Author:    "alice",
		Operation: OperationCreate,
		ExpiresAt: time.Now().Add(time.Second),
	})
	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, adminRes)))
	upstreamRes := meshresource.NewConditionRouteResourceWithAttributes("demo.condition-router", "mesh")
	upstreamRes.Spec = &meshproto.ConditionRoute{Key: "demo", Priority: 2}
	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Updated, adminRes, upstreamRes)))
	items, err := store.ListVersions(meshresource.ConditionRouteKind, adminRes.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, SourceUpstream, items[0].Source)
	require.Equal(t, SourceAdmin, items[1].Source)
	require.Equal(t, "alice", items[1].Author)
}

func TestSubscriberRespectsRegistrySourceContext(t *testing.T) {
	store := NewMemoryStore()
	hints := NewAdminHintRegistry()
	sub := NewSubscriber(meshresource.ConditionRouteKind, store, hints, 5, 0)

	upstreamRes := meshresource.NewConditionRouteResourceWithAttributes("demo.condition-router", "mesh")
	upstreamRes.Spec = &meshproto.ConditionRoute{Key: "demo", Priority: 2}
	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEventWithContext(
		cache.Updated,
		nil,
		upstreamRes,
		map[string]string{events.SourceRegistryContextKey: "zookeeper"},
	)))

	items, err := store.ListVersions(meshresource.ConditionRouteKind, upstreamRes.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, SourceUpstream, items[0].Source)
	require.Equal(t, "system:zookeeper", items[0].Author)
}

func TestSubscriberSkipsNoopUpstreamEchoAfterBootstrap(t *testing.T) {
	store := NewMemoryStore()
	hints := NewAdminHintRegistry()
	sub := NewSubscriber(meshresource.ConditionRouteKind, store, hints, 5, 0)

	original := meshresource.NewConditionRouteResourceWithAttributes("demo.condition-router", "mesh")
	original.Spec = &meshproto.ConditionRoute{Key: "demo", Priority: 1}
	require.NoError(t, RecordBootstrap(store, 5, original))
	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEventWithContext(
		cache.Updated,
		nil,
		original,
		map[string]string{events.SourceRegistryContextKey: "zookeeper"},
	)))

	items, err := store.ListVersions(meshresource.ConditionRouteKind, original.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, SourceBootstrap, items[0].Source)
	require.True(t, items[0].IsCurrent)

	changed := meshresource.NewConditionRouteResourceWithAttributes("demo.condition-router", "mesh")
	changed.Spec = &meshproto.ConditionRoute{Key: "demo", Priority: 2}
	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEventWithContext(
		cache.Updated,
		original,
		changed,
		map[string]string{events.SourceRegistryContextKey: "zookeeper"},
	)))

	items, err = store.ListVersions(meshresource.ConditionRouteKind, original.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, SourceUpstream, items[0].Source)
	require.Equal(t, "system:zookeeper", items[0].Author)
	require.True(t, items[0].IsCurrent)
}

func TestSubscriberRecordsEmptyCreateAfterDelete(t *testing.T) {
	store := NewMemoryStore()
	hints := NewAdminHintRegistry()
	sub := NewSubscriber(meshresource.ConditionRouteKind, store, hints, 5, 0)
	res := meshresource.NewConditionRouteResourceWithAttributes("demo.condition-router", "mesh")
	res.Spec = &meshproto.ConditionRoute{}

	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, res)))
	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Deleted, res, nil)))
	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, res)))

	items, err := store.ListVersions(meshresource.ConditionRouteKind, res.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 3)
	require.Equal(t, OperationCreate, items[0].Operation)
	require.True(t, items[0].IsCurrent)
	require.Equal(t, OperationDelete, items[1].Operation)
	require.False(t, items[1].IsCurrent)
}

func TestSubscriberRecordsDeleteWithAdminHintSnapshot(t *testing.T) {
	store := NewMemoryStore()
	hints := NewAdminHintRegistry()
	svc := NewService(true, 5, 0, time.Second, store, hints)
	sub := NewSubscriber(meshresource.TagRouteKind, store, hints, 5, 0)

	res := meshresource.NewTagRouteResourceWithAttributes("demo.tag-router", "mesh")
	res.Spec = &meshproto.TagRoute{
		Key:      "demo",
		Priority: 7,
	}
	require.NoError(t, svc.PutAdminHint(res, OperationDelete, SourceAdmin, "alice", "", nil))
	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Deleted, res, nil)))

	items, err := store.ListVersions(meshresource.TagRouteKind, res.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, OperationDelete, items[0].Operation)
	require.Equal(t, SourceAdmin, items[0].Source)
	require.Equal(t, "alice", items[0].Author)
	require.Equal(t, DeleteSpecJSON, items[0].SpecJSON)
	require.Equal(t, HashSpecJSON(DeleteSpecJSON), items[0].ContentHash)
	require.Equal(t, res.ResourceKey(), items[0].ResourceKey)
	require.Equal(t, res.ResourceMeta().Name, items[0].RuleName)

	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Deleted, res, nil)))
	items, err = store.ListVersions(meshresource.TagRouteKind, res.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 1)
}

func TestServiceRecordMutationCreatesAdminVersionAndDedupsEcho(t *testing.T) {
	store := NewMemoryStore()
	hints := NewAdminHintRegistry()
	svc := NewService(true, 5, 0, time.Second, store, hints)
	sub := NewSubscriber(meshresource.ConditionRouteKind, store, hints, 5, 0)

	res := meshresource.NewConditionRouteResourceWithAttributes("demo.condition-router", "mesh")
	res.Spec = &meshproto.ConditionRoute{Key: "demo", Priority: 1}
	recorded, err := svc.RecordMutation(res, OperationUpdate, SourceAdmin, " alice ", "", nil)
	require.NoError(t, err)
	require.Equal(t, SourceAdmin, recorded.Source)
	require.Equal(t, "alice", recorded.Author)
	require.True(t, recorded.IsCurrent)

	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Updated, nil, res)))
	items, err := store.ListVersions(meshresource.ConditionRouteKind, res.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, recorded.ID, items[0].ID)
	require.Equal(t, SourceAdmin, items[0].Source)
}

func TestSubscriberCommitsMatchingAdminIntent(t *testing.T) {
	store := NewMemoryStore()
	hints := NewAdminHintRegistry()
	svc := NewService(true, 5, 0, time.Second, store, hints)
	sub := NewSubscriber(meshresource.ConditionRouteKind, store, hints, 5, 0)

	res := meshresource.NewConditionRouteResourceWithAttributes("demo.condition-router", "mesh")
	res.Spec = &meshproto.ConditionRoute{Key: "demo", Priority: 1}
	intent, err := svc.BeginMutationIntent(res, OperationUpdate, SourceAdmin, "alice", "admin edit", nil, nil)
	require.NoError(t, err)
	require.Equal(t, IntentStatusPending, intent.Status)

	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Updated, nil, res)))
	items, err := store.ListVersions(meshresource.ConditionRouteKind, res.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, SourceAdmin, items[0].Source)
	require.Equal(t, "alice", items[0].Author)
	require.Equal(t, "admin edit", items[0].Reason)
	open, err := store.OpenIntent(meshresource.ConditionRouteKind, res.ResourceKey())
	require.NoError(t, err)
	require.Nil(t, open)
}

func TestServiceRepairIntentByIDCommitsOnlyMatchingPendingIntent(t *testing.T) {
	store := NewMemoryStore()
	svc := NewService(true, 5, 0, time.Second, store, NewAdminHintRegistry())
	res := meshresource.NewConditionRouteResourceWithAttributes("demo.condition-router", "mesh")
	res.Spec = &meshproto.ConditionRoute{Key: "demo", Priority: 1}
	intent, err := svc.BeginMutationIntent(res, OperationUpdate, SourceAdmin, "alice", "admin edit", nil, nil)
	require.NoError(t, err)

	_, err = svc.RepairIntentByID(intent.ID, nil, false)
	require.ErrorIs(t, err, ErrVersionIntentPending)

	version, err := svc.RepairIntentByID(intent.ID, res, false)
	require.NoError(t, err)
	require.Equal(t, SourceAdmin, version.Source)
	require.Equal(t, "alice", version.Author)
	repaired, err := store.GetIntent(intent.ID)
	require.NoError(t, err)
	require.Equal(t, IntentStatusCommitted, repaired.Status)
	require.NotNil(t, repaired.VersionID)
}

func TestServiceRecordMutationDeleteClearsCurrent(t *testing.T) {
	store := NewMemoryStore()
	hints := NewAdminHintRegistry()
	svc := NewService(true, 5, 0, time.Second, store, hints)

	res := meshresource.NewConditionRouteResourceWithAttributes("demo.condition-router", "mesh")
	res.Spec = &meshproto.ConditionRoute{Key: "demo", Priority: 1}
	created, err := svc.RecordMutation(res, OperationCreate, SourceAdmin, "alice", "", nil)
	require.NoError(t, err)
	require.True(t, created.IsCurrent)

	deleted, err := svc.RecordMutation(res, OperationDelete, SourceAdmin, "alice", "", nil)
	require.NoError(t, err)
	require.Equal(t, OperationDelete, deleted.Operation)
	require.False(t, deleted.IsCurrent)

	meta, err := store.CurrentMeta(meshresource.ConditionRouteKind, res.ResourceKey())
	require.NoError(t, err)
	require.Nil(t, meta.CurrentVersion)
	items, err := store.ListVersions(meshresource.ConditionRouteKind, res.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, OperationDelete, items[0].Operation)
	require.False(t, items[0].IsCurrent)
}

func TestDisabledServiceRecordMutationNoops(t *testing.T) {
	store := NewMemoryStore()
	svc := NewService(false, 5, 0, time.Second, store, NewAdminHintRegistry())
	res := meshresource.NewConditionRouteResourceWithAttributes("demo.condition-router", "mesh")
	res.Spec = &meshproto.ConditionRoute{Key: "demo", Priority: 1}

	recorded, err := svc.RecordMutation(res, OperationUpdate, SourceAdmin, "alice", "", nil)
	require.NoError(t, err)
	require.Nil(t, recorded)

	items, err := store.ListVersions(meshresource.ConditionRouteKind, res.ResourceKey())
	require.NoError(t, err)
	require.Empty(t, items)
}

func TestDisabledServiceHistoryReturnsFeatureDisabled(t *testing.T) {
	svc := NewService(false, 5, 0, time.Second, NewMemoryStore(), NewAdminHintRegistry())
	_, err := svc.List(meshresource.ConditionRouteKind, "mesh", "demo.condition-router")
	require.ErrorIs(t, err, ErrFeatureDisabled)
}

func TestComponentFlushesPendingVersionsOnStop(t *testing.T) {
	store := NewMemoryStore()
	sub := NewSubscriber(meshresource.ConditionRouteKind, store, NewAdminHintRegistry(), 5, time.Hour)
	comp := &component{
		store:       store,
		subscribers: []*Subscriber{sub},
	}
	res := meshresource.NewConditionRouteResourceWithAttributes("demo.condition-router", "mesh")
	res.Spec = &meshproto.ConditionRoute{Key: "demo", Priority: 1}
	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, res)))

	stop := make(chan struct{})
	require.NoError(t, comp.Start(testRuntime{
		cfg: appconfig.AdminConfig{
			Versioning: &versioningcfg.Config{
				Enabled:            true,
				MaxVersionsPerRule: 5,
			},
		},
		components: map[coreruntime.ComponentType]coreruntime.Component{
			coreruntime.ResourceManager: testRMComponent{rm: fakeNoopResourceManager{}},
		},
	}, stop))
	close(stop)

	require.Eventually(t, func() bool {
		items, err := store.ListVersions(meshresource.ConditionRouteKind, res.ResourceKey())
		return err == nil && len(items) == 1
	}, time.Second, 10*time.Millisecond)
}

type fakeVersionResourceManager struct {
	subscriber *Subscriber
}

func (f fakeVersionResourceManager) GetByKey(model.ResourceKind, string) (model.Resource, bool, error) {
	return nil, false, nil
}

func (f fakeVersionResourceManager) GetByKeys(model.ResourceKind, []string) ([]model.Resource, error) {
	return nil, nil
}

func (f fakeVersionResourceManager) List(model.ResourceKind) ([]model.Resource, error) {
	return nil, nil
}

func (f fakeVersionResourceManager) ListByIndexes(model.ResourceKind, []index.IndexCondition) ([]model.Resource, error) {
	return nil, nil
}

func (f fakeVersionResourceManager) PageListByIndexes(model.ResourceKind, []index.IndexCondition, model.PageReq) (*model.PageData[model.Resource], error) {
	return nil, nil
}

func (f fakeVersionResourceManager) Add(r model.Resource) error {
	return f.subscriber.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, r))
}

func (f fakeVersionResourceManager) Update(r model.Resource) error {
	return f.subscriber.ProcessEvent(events.NewResourceChangedEvent(cache.Updated, nil, r))
}

func (f fakeVersionResourceManager) Upsert(r model.Resource) error {
	return f.Update(r)
}

func (f fakeVersionResourceManager) DeleteByKey(model.ResourceKind, string, string) error {
	return nil
}

type fakeNoopResourceManager struct{}

func (f fakeNoopResourceManager) GetByKey(model.ResourceKind, string) (model.Resource, bool, error) {
	return nil, false, nil
}

func (f fakeNoopResourceManager) GetByKeys(model.ResourceKind, []string) ([]model.Resource, error) {
	return nil, nil
}

func (f fakeNoopResourceManager) List(model.ResourceKind) ([]model.Resource, error) {
	return nil, nil
}

func (f fakeNoopResourceManager) ListByIndexes(model.ResourceKind, []index.IndexCondition) ([]model.Resource, error) {
	return nil, nil
}

func (f fakeNoopResourceManager) PageListByIndexes(model.ResourceKind, []index.IndexCondition, model.PageReq) (*model.PageData[model.Resource], error) {
	return nil, nil
}

func (f fakeNoopResourceManager) Add(model.Resource) error {
	return nil
}

func (f fakeNoopResourceManager) Update(model.Resource) error {
	return nil
}

func (f fakeNoopResourceManager) Upsert(model.Resource) error {
	return nil
}

func (f fakeNoopResourceManager) DeleteByKey(model.ResourceKind, string, string) error {
	return nil
}

type eventBusVersionResourceManager struct {
	emitter events.Emitter
}

func (f eventBusVersionResourceManager) GetByKey(model.ResourceKind, string) (model.Resource, bool, error) {
	return nil, false, nil
}

func (f eventBusVersionResourceManager) GetByKeys(model.ResourceKind, []string) ([]model.Resource, error) {
	return nil, nil
}

func (f eventBusVersionResourceManager) List(model.ResourceKind) ([]model.Resource, error) {
	return nil, nil
}

func (f eventBusVersionResourceManager) ListByIndexes(model.ResourceKind, []index.IndexCondition) ([]model.Resource, error) {
	return nil, nil
}

func (f eventBusVersionResourceManager) PageListByIndexes(model.ResourceKind, []index.IndexCondition, model.PageReq) (*model.PageData[model.Resource], error) {
	return nil, nil
}

func (f eventBusVersionResourceManager) Add(r model.Resource) error {
	f.emitter.Send(events.NewResourceChangedEvent(cache.Added, nil, r))
	return nil
}

func (f eventBusVersionResourceManager) Update(r model.Resource) error {
	f.emitter.Send(events.NewResourceChangedEvent(cache.Updated, nil, r))
	return nil
}

func (f eventBusVersionResourceManager) Upsert(r model.Resource) error {
	return f.Update(r)
}

func (f eventBusVersionResourceManager) DeleteByKey(model.ResourceKind, string, string) error {
	return nil
}

type testEventBus interface {
	events.EventBusComponent
	coreruntime.GracefulComponent
}

func newTestEventBus(t *testing.T) testEventBus {
	t.Helper()
	prototype, err := coreruntime.ComponentRegistry().EventBus()
	require.NoError(t, err)
	bus := reflect.New(reflect.TypeOf(prototype).Elem()).Interface().(testEventBus)
	bufferSize := uint(1)
	require.NoError(t, bus.Init(testBuilderContext{
		cfg: appconfig.AdminConfig{
			EventBus: &eventbusconfig.Config{BufferSize: bufferSize},
		},
	}))
	return bus
}

type testBuilderContext struct {
	cfg appconfig.AdminConfig
}

func (c testBuilderContext) Config() appconfig.AdminConfig {
	return c.cfg
}

func (c testBuilderContext) GetActivatedComponent(coreruntime.ComponentType) (coreruntime.Component, error) {
	return nil, nil
}

func (c testBuilderContext) ActivateComponent(coreruntime.Component) error {
	return nil
}

type testRuntime struct {
	cfg        appconfig.AdminConfig
	components map[coreruntime.ComponentType]coreruntime.Component
}

func (r testRuntime) GetInstanceId() string {
	return "test-instance"
}

func (r testRuntime) GetClusterId() string {
	return "test-cluster"
}

func (r testRuntime) GetStartTime() time.Time {
	return time.Now()
}

func (r testRuntime) GetMode() mode.Mode {
	return mode.Test
}

func (r testRuntime) Config() appconfig.AdminConfig {
	return r.cfg
}

func (r testRuntime) GetComponent(typ coreruntime.ComponentType) (coreruntime.Component, error) {
	return r.components[typ], nil
}

func (r testRuntime) AppContext() context.Context {
	return context.Background()
}

func (r testRuntime) Add(...coreruntime.Component) {}

func (r testRuntime) Start(<-chan struct{}) error {
	return nil
}

type testRMComponent struct {
	rm manager.ResourceManager
}

func (c testRMComponent) Type() coreruntime.ComponentType {
	return coreruntime.ResourceManager
}

func (c testRMComponent) Order() int {
	return 0
}

func (c testRMComponent) RequiredDependencies() []coreruntime.ComponentType {
	return nil
}

func (c testRMComponent) Init(coreruntime.BuilderContext) error {
	return nil
}

func (c testRMComponent) Start(coreruntime.Runtime, <-chan struct{}) error {
	return nil
}

func (c testRMComponent) ResourceManager() manager.ResourceManager {
	return c.rm
}
