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

package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/cache"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	appconfig "github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/console/counter"
	"github.com/apache/dubbo-admin/pkg/core/events"
	corelock "github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
)

func TestAdminMutationRecordsSynchronouslyAndConflictsOnStaleExpected(t *testing.T) {
	ctx, store, _ := newRuleVersionTestContext()
	initial := newTestConditionRule(1)
	ctx.rm.Put(initial)
	current, err := ctx.versioning.RecordMutation(initial, versioning.OperationCreate, versioning.SourceBootstrap, "system:bootstrap", "", nil)
	require.NoError(t, err)

	expected := current.ID
	firstUpdate := newTestConditionRule(2)
	err = UpdateConditionRuleWithOptions(ctx, firstUpdate, RuleMutationOptions{
		ExpectedVersionID: &expected,
		Author:            "alice",
	})
	require.NoError(t, err)

	secondUpdate := newTestConditionRule(3)
	err = UpdateConditionRuleWithOptions(ctx, secondUpdate, RuleMutationOptions{
		ExpectedVersionID: &expected,
		Author:            "bob",
	})
	var conflict *versioning.ConflictError
	require.ErrorAs(t, err, &conflict)

	items, err := store.ListVersions(meshresource.ConditionRouteKind, initial.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, versioning.SourceAdmin, items[0].Source)
	require.Equal(t, "alice", items[0].Author)
}

func TestRollbackAndUpdateShareRuleLockAndStaleExpectedConflicts(t *testing.T) {
	ctx, store, _ := newRuleVersionTestContext()
	original := newTestConditionRule(1)
	currentRes := newTestConditionRule(2)
	ctx.rm.Put(currentRes)
	target, err := ctx.versioning.RecordMutation(original, versioning.OperationCreate, versioning.SourceBootstrap, "system:bootstrap", "", nil)
	require.NoError(t, err)
	current, err := ctx.versioning.RecordMutation(currentRes, versioning.OperationUpdate, versioning.SourceAdmin, "alice", "", nil)
	require.NoError(t, err)

	expected := current.ID
	start := make(chan struct{})
	errs := make(chan error, 2)
	go func() {
		<-start
		_, err := RollbackRuleVersion(ctx, conditionKindName(), target.ID, "restore baseline", &expected, "bob")
		errs <- err
	}()
	go func() {
		<-start
		errs <- UpdateConditionRuleWithOptions(ctx, newTestConditionRule(3), RuleMutationOptions{
			ExpectedVersionID: &expected,
			Author:            "carol",
		})
	}()
	close(start)

	var successCount, conflictCount int
	for i := 0; i < 2; i++ {
		err := <-errs
		if err == nil {
			successCount++
			continue
		}
		var conflict *versioning.ConflictError
		if errors.As(err, &conflict) {
			conflictCount++
			continue
		}
		require.NoError(t, err)
	}
	require.Equal(t, 1, successCount)
	require.Equal(t, 1, conflictCount)

	items, err := store.ListVersions(meshresource.ConditionRouteKind, original.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 3)
	require.True(t, items[0].Source == versioning.SourceAdmin || items[0].Source == versioning.SourceRollback)
}

func TestRollbackCurrentVersionReturnsImmediately(t *testing.T) {
	ctx, store, _ := newRuleVersionTestContext()
	currentRes := newTestConditionRule(1)
	ctx.rm.Put(currentRes)
	current, err := ctx.versioning.RecordMutation(currentRes, versioning.OperationCreate, versioning.SourceAdmin, "alice", "", nil)
	require.NoError(t, err)

	_, err = RollbackRuleVersion(ctx, conditionKindName(), current.ID, "restore current", &current.ID, "bob")
	require.ErrorIs(t, err, versioning.ErrRollbackToCurrent)

	items, err := store.ListVersions(meshresource.ConditionRouteKind, currentRes.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 1)
}

func TestPendingIntentRepairPreventsStaleExpectedReuse(t *testing.T) {
	ctx, store, _ := newRuleVersionTestContext()
	currentRes := newTestConditionRule(1)
	ctx.rm.Put(currentRes)
	current, err := ctx.versioning.RecordMutation(currentRes, versioning.OperationCreate, versioning.SourceAdmin, "alice", "", nil)
	require.NoError(t, err)

	appliedRes := newTestConditionRule(2)
	_, err = ctx.versioning.BeginMutationIntent(appliedRes, versioning.OperationUpdate, versioning.SourceAdmin, "bob", "", &current.ID, nil)
	require.NoError(t, err)
	ctx.rm.Put(appliedRes)

	err = UpdateConditionRuleWithOptions(ctx, newTestConditionRule(3), RuleMutationOptions{
		ExpectedVersionID: &current.ID,
		Author:            "carol",
	})
	var conflict *versioning.ConflictError
	require.ErrorAs(t, err, &conflict)

	items, err := store.ListVersions(meshresource.ConditionRouteKind, currentRes.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "bob", items[0].Author)
	require.Equal(t, versioning.SourceAdmin, items[0].Source)
	intent, err := store.OpenIntent(meshresource.ConditionRouteKind, currentRes.ResourceKey())
	require.NoError(t, err)
	require.Nil(t, intent)
}

func TestPendingIntentWithoutAppliedResourceBlocksMutation(t *testing.T) {
	ctx, store, _ := newRuleVersionTestContext()
	currentRes := newTestConditionRule(1)
	ctx.rm.Put(currentRes)
	current, err := ctx.versioning.RecordMutation(currentRes, versioning.OperationCreate, versioning.SourceAdmin, "alice", "", nil)
	require.NoError(t, err)

	pendingRes := newTestConditionRule(2)
	_, err = ctx.versioning.BeginMutationIntent(pendingRes, versioning.OperationUpdate, versioning.SourceAdmin, "bob", "", &current.ID, nil)
	require.NoError(t, err)

	err = UpdateConditionRuleWithOptions(ctx, newTestConditionRule(3), RuleMutationOptions{
		ExpectedVersionID: &current.ID,
		Author:            "carol",
	})
	require.ErrorIs(t, err, versioning.ErrVersionIntentPending)

	items, err := store.ListVersions(meshresource.ConditionRouteKind, currentRes.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 1)
}

func TestRepairRuleVersionIntentCommitsMatchingPendingIntent(t *testing.T) {
	ctx, store, _ := newRuleVersionTestContext()
	currentRes := newTestConditionRule(1)
	ctx.rm.Put(currentRes)
	current, err := ctx.versioning.RecordMutation(currentRes, versioning.OperationCreate, versioning.SourceAdmin, "alice", "", nil)
	require.NoError(t, err)

	appliedRes := newTestConditionRule(2)
	intent, err := ctx.versioning.BeginMutationIntent(appliedRes, versioning.OperationUpdate, versioning.SourceAdmin, "bob", "admin edit", &current.ID, nil)
	require.NoError(t, err)
	ctx.rm.Put(appliedRes)

	repaired, err := RepairRuleVersionIntent(ctx, intent.ID)
	require.NoError(t, err)
	require.NotNil(t, repaired)
	require.Equal(t, versioning.SourceAdmin, repaired.Source)
	require.Equal(t, "bob", repaired.Author)
	repairedIntent, err := store.GetIntent(intent.ID)
	require.NoError(t, err)
	require.Equal(t, versioning.IntentStatusCommitted, repairedIntent.Status)
	require.NotNil(t, repairedIntent.VersionID)
	open, err := store.OpenIntent(meshresource.ConditionRouteKind, currentRes.ResourceKey())
	require.NoError(t, err)
	require.Nil(t, open)
}

func TestRepairRuleVersionIntentBlocksMismatchedPendingIntent(t *testing.T) {
	ctx, store, _ := newRuleVersionTestContext()
	currentRes := newTestConditionRule(1)
	ctx.rm.Put(currentRes)
	current, err := ctx.versioning.RecordMutation(currentRes, versioning.OperationCreate, versioning.SourceAdmin, "alice", "", nil)
	require.NoError(t, err)

	pendingRes := newTestConditionRule(2)
	intent, err := ctx.versioning.BeginMutationIntent(pendingRes, versioning.OperationUpdate, versioning.SourceAdmin, "bob", "admin edit", &current.ID, nil)
	require.NoError(t, err)

	_, err = RepairRuleVersionIntent(ctx, intent.ID)
	require.ErrorIs(t, err, versioning.ErrVersionIntentPending)
	repairedIntent, err := store.GetIntent(intent.ID)
	require.NoError(t, err)
	require.Equal(t, versioning.IntentStatusPending, repairedIntent.Status)
	items, err := store.ListVersions(meshresource.ConditionRouteKind, currentRes.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 1)
}

func TestAbandonRuleVersionIntentFailsMismatchedPendingAndUnblocksMutation(t *testing.T) {
	ctx, store, _ := newRuleVersionTestContext()
	currentRes := newTestConditionRule(1)
	ctx.rm.Put(currentRes)
	current, err := ctx.versioning.RecordMutation(currentRes, versioning.OperationCreate, versioning.SourceAdmin, "alice", "", nil)
	require.NoError(t, err)

	pendingRes := newTestConditionRule(2)
	intent, err := ctx.versioning.BeginMutationIntent(pendingRes, versioning.OperationUpdate, versioning.SourceAdmin, "bob", "admin edit", &current.ID, nil)
	require.NoError(t, err)

	err = AbandonRuleVersionIntent(ctx, intent.ID, "registry rejected mutation")
	require.NoError(t, err)
	abandoned, err := store.GetIntent(intent.ID)
	require.NoError(t, err)
	require.Equal(t, versioning.IntentStatusFailed, abandoned.Status)
	require.Equal(t, "registry rejected mutation", abandoned.LastError)
	open, err := store.OpenIntent(meshresource.ConditionRouteKind, currentRes.ResourceKey())
	require.NoError(t, err)
	require.Nil(t, open)

	err = UpdateConditionRuleWithOptions(ctx, newTestConditionRule(3), RuleMutationOptions{
		ExpectedVersionID: &current.ID,
		Author:            "carol",
	})
	require.NoError(t, err)
	items, err := store.ListVersions(meshresource.ConditionRouteKind, currentRes.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "carol", items[0].Author)
}

func TestAbandonRuleVersionIntentRejectsAppliedAndMatchingPending(t *testing.T) {
	ctx, store, _ := newRuleVersionTestContext()
	currentRes := newTestConditionRule(1)
	current, err := ctx.versioning.RecordMutation(currentRes, versioning.OperationCreate, versioning.SourceAdmin, "alice", "", nil)
	require.NoError(t, err)
	appliedRes := newTestConditionRule(2)
	applied, err := ctx.versioning.BeginMutationIntent(appliedRes, versioning.OperationUpdate, versioning.SourceAdmin, "bob", "admin edit", &current.ID, nil)
	require.NoError(t, err)
	require.NoError(t, ctx.versioning.MarkMutationIntentApplied(applied.ID))

	err = AbandonRuleVersionIntent(ctx, applied.ID, "operator abandon")
	requireInvalidArgument(t, err)
	unchanged, err := store.GetIntent(applied.ID)
	require.NoError(t, err)
	require.Equal(t, versioning.IntentStatusApplied, unchanged.Status)

	ctx, store, _ = newRuleVersionTestContext()
	matchingRes := newTestConditionRule(2)
	intent, err := ctx.versioning.BeginMutationIntent(matchingRes, versioning.OperationUpdate, versioning.SourceAdmin, "bob", "admin edit", nil, nil)
	require.NoError(t, err)
	ctx.rm.Put(matchingRes)

	err = AbandonRuleVersionIntent(ctx, intent.ID, "operator abandon")
	requireInvalidArgument(t, err)
	unchanged, err = store.GetIntent(intent.ID)
	require.NoError(t, err)
	require.Equal(t, versioning.IntentStatusPending, unchanged.Status)
}

func TestDeleteRecordsMarkerAndClearsCurrent(t *testing.T) {
	ctx, store, _ := newRuleVersionTestContext()
	currentRes := newTestConditionRule(1)
	ctx.rm.Put(currentRes)
	current, err := ctx.versioning.RecordMutation(currentRes, versioning.OperationCreate, versioning.SourceAdmin, "alice", "", nil)
	require.NoError(t, err)

	err = DeleteConditionRuleWithOptions(ctx, currentRes.Name, currentRes.Mesh, RuleMutationOptions{
		ExpectedVersionID: &current.ID,
		Author:            "alice",
	})
	require.NoError(t, err)

	meta, err := store.CurrentMeta(meshresource.ConditionRouteKind, currentRes.ResourceKey())
	require.NoError(t, err)
	require.Nil(t, meta.CurrentVersion)
	items, err := store.ListVersions(meshresource.ConditionRouteKind, currentRes.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, versioning.OperationDelete, items[0].Operation)
	require.False(t, items[0].IsCurrent)
}

func TestFailedAdminMutationDoesNotRecordOrPolluteNextUpstreamEvent(t *testing.T) {
	ctx, store, hints := newRuleVersionTestContext()
	currentRes := newTestConditionRule(1)
	ctx.rm.Put(currentRes)
	current, err := ctx.versioning.RecordMutation(currentRes, versioning.OperationCreate, versioning.SourceAdmin, "alice", "", nil)
	require.NoError(t, err)

	failedUpdate := newTestConditionRule(2)
	updateErr := errors.New("update failed")
	ctx.rm.FailUpdate(updateErr)
	err = UpdateConditionRuleWithOptions(ctx, failedUpdate, RuleMutationOptions{
		ExpectedVersionID: &current.ID,
		Author:            "alice",
	})
	require.ErrorIs(t, err, updateErr)
	items, err := store.ListVersions(meshresource.ConditionRouteKind, currentRes.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 1)

	sub := versioning.NewSubscriber(meshresource.ConditionRouteKind, store, hints, 5, 0)
	require.NoError(t, sub.ProcessEvent(events.NewResourceChangedEvent(cache.Updated, currentRes, failedUpdate)))
	items, err = store.ListVersions(meshresource.ConditionRouteKind, currentRes.ResourceKey())
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, versioning.SourceUpstream, items[0].Source)
}

func requireInvalidArgument(t *testing.T, err error) {
	t.Helper()
	var bizErr bizerror.Error
	require.ErrorAs(t, err, &bizErr)
	require.Equal(t, bizerror.InvalidArgument, bizErr.Code())
}

func newRuleVersionTestContext() (*ruleVersionTestContext, *versioning.MemoryStore, *versioning.AdminHintRegistry) {
	store := versioning.NewMemoryStore()
	hints := versioning.NewAdminHintRegistry()
	return &ruleVersionTestContext{
		rm:         newTestResourceManager(),
		lock:       &serialTestLock{},
		versioning: versioning.NewService(true, 5, 0, time.Second, store, hints),
	}, store, hints
}

func newTestConditionRule(priority int32) *meshresource.ConditionRouteResource {
	res := meshresource.NewConditionRouteResourceWithAttributes("demo.condition-router", "mesh")
	res.Spec = &meshproto.ConditionRoute{
		Key:        "demo",
		Enabled:    true,
		Priority:   priority,
		Conditions: []string{"host = 127.0.0.1"},
	}
	return res
}

func conditionKindName() RuleKindName {
	return RuleKindName{
		Kind: meshresource.ConditionRouteKind,
		Mesh: "mesh",
		Name: "demo.condition-router",
	}
}

type ruleVersionTestContext struct {
	rm         *testResourceManager
	lock       corelock.Lock
	versioning versioning.Service
}

func (c *ruleVersionTestContext) ResourceManager() manager.ResourceManager {
	return c.rm
}

func (c *ruleVersionTestContext) CounterManager() counter.CounterManager {
	return nil
}

func (c *ruleVersionTestContext) Config() appconfig.AdminConfig {
	return appconfig.AdminConfig{}
}

func (c *ruleVersionTestContext) AppContext() context.Context {
	return context.Background()
}

func (c *ruleVersionTestContext) LockManager() corelock.Lock {
	return c.lock
}

func (c *ruleVersionTestContext) RuleVersioning() versioning.Service {
	return c.versioning
}

type serialTestLock struct {
	mu sync.Mutex
}

func (l *serialTestLock) Lock(context.Context, string, time.Duration) error {
	l.mu.Lock()
	return nil
}

func (l *serialTestLock) TryLock(context.Context, string, time.Duration) (bool, error) {
	l.mu.Lock()
	return true, nil
}

func (l *serialTestLock) Unlock(context.Context, string) error {
	l.mu.Unlock()
	return nil
}

func (l *serialTestLock) Renew(context.Context, string, time.Duration) error {
	return nil
}

func (l *serialTestLock) IsLocked(context.Context, string) (bool, error) {
	return false, nil
}

func (l *serialTestLock) WithLock(_ context.Context, _ string, _ time.Duration, fn func() error) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return fn()
}

func (l *serialTestLock) CleanupExpiredLocks(context.Context) error {
	return nil
}

type testResourceManager struct {
	mu         sync.Mutex
	resources  map[coremodel.ResourceKind]map[string]coremodel.Resource
	updateFail error
}

func newTestResourceManager() *testResourceManager {
	return &testResourceManager{
		resources: make(map[coremodel.ResourceKind]map[string]coremodel.Resource),
	}
}

func (m *testResourceManager) Put(res coremodel.Resource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.putLocked(res)
}

func (m *testResourceManager) FailUpdate(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateFail = err
}

func (m *testResourceManager) GetByKey(kind coremodel.ResourceKind, key string) (coremodel.Resource, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	byKind := m.resources[kind]
	if byKind == nil {
		return nil, false, nil
	}
	res, ok := byKind[key]
	return res, ok, nil
}

func (m *testResourceManager) GetByKeys(kind coremodel.ResourceKind, keys []string) ([]coremodel.Resource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	items := make([]coremodel.Resource, 0, len(keys))
	for _, key := range keys {
		if res := m.resources[kind][key]; res != nil {
			items = append(items, res)
		}
	}
	return items, nil
}

func (m *testResourceManager) List(kind coremodel.ResourceKind) ([]coremodel.Resource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	items := make([]coremodel.Resource, 0, len(m.resources[kind]))
	for _, res := range m.resources[kind] {
		items = append(items, res)
	}
	return items, nil
}

func (m *testResourceManager) ListByIndexes(kind coremodel.ResourceKind, _ []index.IndexCondition) ([]coremodel.Resource, error) {
	return m.List(kind)
}

func (m *testResourceManager) PageListByIndexes(kind coremodel.ResourceKind, _ []index.IndexCondition, page coremodel.PageReq) (*coremodel.PageData[coremodel.Resource], error) {
	items, err := m.List(kind)
	if err != nil {
		return nil, err
	}
	return coremodel.NewPageData(len(items), page.PageOffset, page.PageSize, items), nil
}

func (m *testResourceManager) Add(res coremodel.Resource) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.putLocked(res)
	return nil
}

func (m *testResourceManager) Update(res coremodel.Resource) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateFail != nil {
		return m.updateFail
	}
	m.putLocked(res)
	return nil
}

func (m *testResourceManager) Upsert(res coremodel.Resource) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.putLocked(res)
	return nil
}

func (m *testResourceManager) DeleteByKey(kind coremodel.ResourceKind, _ string, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.resources[kind], key)
	return nil
}

func (m *testResourceManager) putLocked(res coremodel.Resource) {
	byKind := m.resources[res.ResourceKind()]
	if byKind == nil {
		byKind = make(map[string]coremodel.Resource)
		m.resources[res.ResourceKind()] = byKind
	}
	byKind[res.ResourceKey()] = res
}
