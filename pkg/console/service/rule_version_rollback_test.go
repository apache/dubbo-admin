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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	appcfg "github.com/apache/dubbo-admin/pkg/config/app"
	versioningcfg "github.com/apache/dubbo-admin/pkg/config/versioning"
	"github.com/apache/dubbo-admin/pkg/console/counter"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/governor"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
	locallock "github.com/apache/dubbo-admin/pkg/lock/local"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
)

type testContext struct {
	rm            manager.ResourceManager
	versioningSvc *versioning.Service
	adapter       *versioning.ResourceStoreAdapter
	cfg           *appcfg.AdminConfig
	bus           *simpleBus
	lockMgr       lock.Lock
}

func (c *testContext) ResourceManager() manager.ResourceManager { return c.rm }
func (c *testContext) CounterManager() counter.CounterManager   { return nil }
func (c *testContext) Config() appcfg.AdminConfig               { return *c.cfg }
func (c *testContext) AppContext() context.Context              { return context.Background() }
func (c *testContext) LockManager() lock.Lock                   { return c.lockMgr }
func (c *testContext) RuleVersioning() *versioning.Service      { return c.versioningSvc }

type testRouter struct {
	stores map[coremodel.ResourceKind]store.ResourceStore
}

func (r *testRouter) ResourceRoute(res coremodel.Resource) (store.ResourceStore, error) {
	return r.ResourceKindRoute(res.ResourceKind())
}

func (r *testRouter) ResourceKindRoute(kind coremodel.ResourceKind) (store.ResourceStore, error) {
	s, ok := r.stores[kind]
	if !ok {
		return nil, bizerror.New(bizerror.InvalidArgument, "store not found for kind")
	}
	return s, nil
}

type noopGovernor struct {
	stores  map[coremodel.ResourceKind]store.ResourceStore
	emitter events.Emitter
}

func (g *noopGovernor) CreateRule(ctx context.Context, res coremodel.Resource) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s, ok := g.stores[res.ResourceKind()]
	if !ok {
		return bizerror.New(bizerror.InvalidArgument, "store not found")
	}
	if err := s.Add(res); err != nil {
		return err
	}
	g.emitter.Send(events.NewResourceChangedEvent("Added", nil, res))
	return nil
}

func (g *noopGovernor) UpdateRule(ctx context.Context, res coremodel.Resource) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s, ok := g.stores[res.ResourceKind()]
	if !ok {
		return bizerror.New(bizerror.InvalidArgument, "store not found")
	}
	old, exists, _ := s.GetByKey(res.ResourceKey())
	var oldRes coremodel.Resource
	if exists {
		oldRes, _ = old.(coremodel.Resource)
	}
	if err := s.Update(res); err != nil {
		return err
	}
	g.emitter.Send(events.NewResourceChangedEvent("Updated", oldRes, res))
	return nil
}

func (g *noopGovernor) DeleteRule(ctx context.Context, res coremodel.Resource) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s, ok := g.stores[res.ResourceKind()]
	if !ok {
		return bizerror.New(bizerror.InvalidArgument, "store not found")
	}
	if err := s.Delete(res); err != nil {
		return err
	}
	g.emitter.Send(events.NewResourceChangedEvent("Deleted", res, nil))
	return nil
}

type noopGovernorRouter struct {
	gov *noopGovernor
}

func (r *noopGovernorRouter) ResourceRoute(coremodel.Resource) (governor.RuleGovernor, error) {
	return r.gov, nil
}

func (r *noopGovernorRouter) ResourceMeshRoute(string) (governor.RuleGovernor, error) {
	return r.gov, nil
}

type simpleBus struct {
	subscribers map[coremodel.ResourceKind][]events.Subscriber
	muted       map[coremodel.ResourceKind]bool
}

func newSimpleBus() *simpleBus {
	return &simpleBus{
		subscribers: make(map[coremodel.ResourceKind][]events.Subscriber),
		muted:       make(map[coremodel.ResourceKind]bool),
	}
}

func (b *simpleBus) Subscribe(sub events.Subscriber) error {
	kind := sub.ResourceKind()
	b.subscribers[kind] = append(b.subscribers[kind], sub)
	return nil
}

func (b *simpleBus) Unsubscribe(events.Subscriber) error { return nil }

func (b *simpleBus) Send(event events.Event) {
	obj := event.NewObj()
	if obj == nil {
		obj = event.OldObj()
	}
	if obj == nil || b.muted[obj.ResourceKind()] {
		return
	}
	for _, sub := range b.subscribers[obj.ResourceKind()] {
		if !sub.AsyncEnabled() {
			_ = sub.ProcessEvent(event)
		}
	}
}

type failingResourceStore struct {
	store.ResourceStore
	failNextAdd    bool
	failNextUpdate bool
	failNextDelete bool
	err            error
}

func (s *failingResourceStore) Add(obj interface{}) error {
	if s.failNextAdd {
		s.failNextAdd = false
		return s.err
	}
	return s.ResourceStore.Add(obj)
}

func (s *failingResourceStore) Update(obj interface{}) error {
	if s.failNextUpdate {
		s.failNextUpdate = false
		return s.err
	}
	return s.ResourceStore.Update(obj)
}

func (s *failingResourceStore) UpdateIfUnchanged(expected coremodel.Resource, updated coremodel.Resource) (bool, error) {
	if s.failNextUpdate {
		s.failNextUpdate = false
		return false, s.err
	}
	cas, ok := s.ResourceStore.(store.ConditionalResourceStore)
	if !ok {
		return false, fmt.Errorf("wrapped store does not support conditional updates")
	}
	return cas.UpdateIfUnchanged(expected, updated)
}

func (s *failingResourceStore) Delete(obj interface{}) error {
	if s.failNextDelete {
		s.failNextDelete = false
		return s.err
	}
	return s.ResourceStore.Delete(obj)
}

func setupRollbackTestEnv(t *testing.T) *testContext {
	return setupRollbackTestEnvWithStoreWrappers(t, nil, nil)
}

func setupRollbackTestEnvWithStoreWrappers(t *testing.T, wrapVersionStore, wrapIntentStore func(store.ResourceStore) store.ResourceStore) *testContext {
	conditionStore := memoryst.NewMemoryResourceStore(meshresource.ConditionRouteKind)
	versionStore := memoryst.NewMemoryResourceStore(meshresource.RuleVersionKind)
	intentStore := memoryst.NewMemoryResourceStore(meshresource.RuleIntentKind)
	for _, s := range []store.ManagedResourceStore{conditionStore, versionStore, intentStore} {
		require.NoError(t, s.Init(nil))
	}

	var versioningVersionStore store.ResourceStore = versionStore
	if wrapVersionStore != nil {
		versioningVersionStore = wrapVersionStore(versionStore)
	}
	var versioningIntentStore store.ResourceStore = intentStore
	if wrapIntentStore != nil {
		versioningIntentStore = wrapIntentStore(intentStore)
	}
	stores := map[coremodel.ResourceKind]store.ResourceStore{
		meshresource.ConditionRouteKind: conditionStore,
		meshresource.RuleVersionKind:    versioningVersionStore,
		meshresource.RuleIntentKind:     versioningIntentStore,
	}

	bus := newSimpleBus()
	gov := &noopGovernor{stores: stores, emitter: bus}
	rm := manager.NewResourceManager(&testRouter{stores: stores}, &noopGovernorRouter{gov: gov})
	adapter := versioning.NewResourceStoreAdapter(versioningVersionStore, versioningIntentStore)
	lockMgr := locallock.NewLocalLock()
	require.NoError(t, bus.Subscribe(versioning.NewSubscriber(meshresource.ConditionRouteKind, adapter, 5, lockMgr, context.Background())))

	return &testContext{
		rm:            rm,
		versioningSvc: versioning.NewService(5, adapter),
		adapter:       adapter,
		cfg:           &appcfg.AdminConfig{RuleVersioning: &versioningcfg.Config{MaxVersionsPerRule: 5}},
		bus:           bus,
		lockMgr:       lockMgr,
	}
}

func mustVersionStoreForTest(t *testing.T) store.ResourceStore {
	s := memoryst.NewMemoryResourceStore(meshresource.RuleVersionKind)
	require.NoError(t, s.Init(nil))
	return s
}

func mustIntentStoreForTest(t *testing.T) store.ResourceStore {
	s := memoryst.NewMemoryResourceStore(meshresource.RuleIntentKind)
	require.NoError(t, s.Init(nil))
	return s
}

func conditionRule(name, payload string) *meshresource.ConditionRouteResource {
	res := meshresource.NewConditionRouteResourceWithAttributes(name, "")
	res.Spec = &meshproto.ConditionRoute{Enabled: true, Key: name, Conditions: []string{payload}}
	return res
}

func kindName(name string) RuleKindName {
	return RuleKindName{Kind: meshresource.ConditionRouteKind, Name: name}
}

func beginMutationForTest(ctx *testContext, res coremodel.Resource) (*versioning.Intent, error) {
	var intent *versioning.Intent
	err := withRuleLock(ctx, RuleKindName{Kind: res.ResourceKind(), Mesh: res.ResourceMesh(), Name: res.ResourceMeta().Name}, func(leaseCtx context.Context) error {
		var inner error
		intent, inner = ctx.versioningSvc.BeginMutation(leaseCtx, res, versioning.OperationUpdate, versioning.SourceAdmin, "admin", "", nil)
		return inner
	})
	return intent, err
}

func TestRuleMutationFailClosedWithoutVersioningService(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	ctx.versioningSvc = nil

	res := conditionRule("demo-rule", "v1")
	err := CreateConditionRuleWithOptions(ctx, res, RuleMutationOptions{Author: "admin"})
	require.ErrorIs(t, err, versioning.ErrVersionLedgerCorrupt)

	_, exists, getErr := ctx.rm.GetByKey(res.ResourceKind(), res.ResourceKey())
	require.NoError(t, getErr)
	assert.False(t, exists)
}

func TestRuleMutationFailClosedWithoutLockManager(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	ctx.lockMgr = nil

	res := conditionRule("demo-rule", "v1")
	err := CreateConditionRuleWithOptions(ctx, res, RuleMutationOptions{Author: "admin"})
	require.ErrorIs(t, err, lock.ErrLockUnavailable)

	_, exists, getErr := ctx.rm.GetByKey(res.ResourceKind(), res.ResourceKey())
	require.NoError(t, getErr)
	assert.False(t, exists)
}

func TestRuleMutationFailClosedWithoutIntentOrVersionStore(t *testing.T) {
	for name, adapter := range map[string]*versioning.ResourceStoreAdapter{
		"intent-store-nil":  versioning.NewResourceStoreAdapter(mustVersionStoreForTest(t), nil),
		"version-store-nil": versioning.NewResourceStoreAdapter(nil, mustIntentStoreForTest(t)),
	} {
		t.Run(name, func(t *testing.T) {
			ctx := setupRollbackTestEnv(t)
			ctx.adapter = adapter
			ctx.versioningSvc = versioning.NewService(5, adapter)

			res := conditionRule("demo-rule", "v1")
			err := CreateConditionRuleWithOptions(ctx, res, RuleMutationOptions{Author: "admin"})
			require.ErrorIs(t, err, versioning.ErrVersionLedgerCorrupt)

			_, exists, getErr := ctx.rm.GetByKey(res.ResourceKind(), res.ResourceKey())
			require.NoError(t, getErr)
			assert.False(t, exists)
		})
	}
}

func TestRollbackRuleVersion_Success(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	require.NoError(t, ctx.rm.Add(context.Background(), conditionRule("demo-rule", "v1")))
	require.NoError(t, ctx.rm.Update(context.Background(), conditionRule("demo-rule", "v2")))

	versions, err := ListRuleVersions(ctx, kindName("demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions.Items, 2)
	v1ID := versions.Items[1].ID
	v2ID := versions.Items[0].ID

	result, err := RollbackRuleVersion(ctx, kindName("demo-rule"), v1ID, "test rollback", &v2ID, "admin")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, v1ID, result.RolledBackFromID)
	assert.True(t, result.Committed)

	versions, err = ListRuleVersions(ctx, kindName("demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions.Items, 3)
	assert.Equal(t, versioning.SourceRollback, versions.Items[0].Source)
	require.NotNil(t, versions.Items[0].RolledBackFromID)
	assert.Equal(t, v1ID, *versions.Items[0].RolledBackFromID)
}

func TestRollbackRuleVersion_DeletedStateCASRace(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	require.NoError(t, ctx.rm.Add(context.Background(), conditionRule("demo-rule", "v1")))
	require.NoError(t, ctx.rm.Update(context.Background(), conditionRule("demo-rule", "v2")))

	versions, err := ListRuleVersions(ctx, kindName("demo-rule"))
	require.NoError(t, err)
	v1 := versions.Items[1]
	require.NoError(t, ctx.rm.DeleteByKey(context.Background(), meshresource.ConditionRouteKind, "", "/demo-rule"))

	expectedDeleted := int64(0)
	require.NoError(t, ctx.rm.Add(context.Background(), conditionRule("demo-rule", "v3")))

	_, err = RollbackRuleVersion(ctx, kindName("demo-rule"), v1.ID, "restore stale deleted view", &expectedDeleted, "admin")
	var conflict *versioning.ConflictError
	require.ErrorAs(t, err, &conflict)
	require.NotNil(t, conflict.CurrentVersionID)
}

func TestRollbackRuleVersion_RepairsWhenCommitNotObserved(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	require.NoError(t, ctx.rm.Add(context.Background(), conditionRule("demo-rule", "v1")))
	require.NoError(t, ctx.rm.Update(context.Background(), conditionRule("demo-rule", "v2")))

	versions, err := ListRuleVersions(ctx, kindName("demo-rule"))
	require.NoError(t, err)
	v1 := versions.Items[1]
	v2 := versions.Items[0]

	ctx.bus.muted[meshresource.ConditionRouteKind] = true
	result, err := RollbackRuleVersion(ctx, kindName("demo-rule"), v1.ID, "repair rollback", &v2.ID, "admin")
	ctx.bus.muted[meshresource.ConditionRouteKind] = false

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Committed)
	assert.Equal(t, v1.ID, result.RolledBackFromID)

	versions, err = ListRuleVersions(ctx, kindName("demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions.Items, 3)
	assert.Equal(t, versioning.SourceRollback, versions.Items[0].Source)
}

func TestRollbackRuleVersion_PendingIntentBlocks(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	require.NoError(t, ctx.rm.Add(context.Background(), conditionRule("demo-rule", "v1")))
	require.NoError(t, ctx.rm.Update(context.Background(), conditionRule("demo-rule", "v2")))

	versions, err := ListRuleVersions(ctx, kindName("demo-rule"))
	require.NoError(t, err)
	v1ID := versions.Items[1].ID

	_, err = beginMutationForTest(ctx, conditionRule("demo-rule", "phantom-divergent"))
	require.NoError(t, err)

	_, err = RollbackRuleVersion(ctx, kindName("demo-rule"), v1ID, "rollback", nil, "admin")
	require.ErrorIs(t, err, versioning.ErrVersionIntentPending)
}

func TestAbandonRuleVersionIntent_CrashBeforeReconcileKeepsIntentOpen(t *testing.T) {
	versionErr := errors.New("version add failed before reconcile")
	failingVersionStore := &failingResourceStore{err: versionErr}
	ctx := setupRollbackTestEnvWithStoreWrappers(t, func(base store.ResourceStore) store.ResourceStore {
		failingVersionStore.ResourceStore = base
		return failingVersionStore
	}, nil)

	require.NoError(t, ctx.rm.Add(context.Background(), conditionRule("demo-rule", "v1")))
	intent, err := beginMutationForTest(ctx, conditionRule("demo-rule", "admin-pending"))
	require.NoError(t, err)
	require.NoError(t, ctx.rm.Update(context.Background(), conditionRule("demo-rule", "external-change")))

	failingVersionStore.failNextAdd = true
	err = AbandonRuleVersionIntent(ctx, intent.ID, "operator chose external state")
	require.ErrorIs(t, err, versionErr)

	open, err := ctx.versioningSvc.GetIntent(intent.ID)
	require.NoError(t, err)
	assert.Equal(t, versioning.IntentStatusOutcomeUnknown, open.Status)
}

func TestAbandonRuleVersionIntent_RuleVersionAddBeforeMarkFailedCrashIsRepairable(t *testing.T) {
	intentErr := errors.New("mark failed crash")
	failingIntentStore := &failingResourceStore{err: intentErr}
	ctx := setupRollbackTestEnvWithStoreWrappers(t, nil, func(base store.ResourceStore) store.ResourceStore {
		failingIntentStore.ResourceStore = base
		return failingIntentStore
	})

	require.NoError(t, ctx.rm.Add(context.Background(), conditionRule("demo-rule", "v1")))
	intent, err := beginMutationForTest(ctx, conditionRule("demo-rule", "admin-pending"))
	require.NoError(t, err)
	external := conditionRule("demo-rule", "external-change")
	require.NoError(t, ctx.rm.Update(context.Background(), external))

	failingIntentStore.failNextUpdate = true
	err = AbandonRuleVersionIntent(ctx, intent.ID, "operator chose external state")
	require.ErrorIs(t, err, intentErr)

	open, err := ctx.versioningSvc.GetIntent(intent.ID)
	require.NoError(t, err)
	require.True(t, open.ReconcileRequired)

	require.NoError(t, AbandonRuleVersionIntent(ctx, intent.ID, "operator chose external state"))
	versions, err := ListRuleVersions(ctx, kindName("demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions.Items, 2)
	hash, _, err := versioning.NormalizeResource(external)
	require.NoError(t, err)
	assert.Equal(t, hash, versions.Items[0].ContentHash)
	_, err = ctx.versioningSvc.GetIntent(intent.ID)
	require.ErrorIs(t, err, versioning.ErrVersionIntentNotFound)
}

func TestAbandonRuleVersionIntent_MarkFailedBeforeCleanupCrashSweepsOnRetry(t *testing.T) {
	cleanupErr := errors.New("cleanup failed")
	failingIntentStore := &failingResourceStore{err: cleanupErr}
	ctx := setupRollbackTestEnvWithStoreWrappers(t, nil, func(base store.ResourceStore) store.ResourceStore {
		failingIntentStore.ResourceStore = base
		return failingIntentStore
	})

	require.NoError(t, ctx.rm.Add(context.Background(), conditionRule("demo-rule", "v1")))
	intent, err := beginMutationForTest(ctx, conditionRule("demo-rule", "admin-pending"))
	require.NoError(t, err)
	require.NoError(t, ctx.rm.Update(context.Background(), conditionRule("demo-rule", "external-change")))

	failingIntentStore.failNextDelete = true
	err = AbandonRuleVersionIntent(ctx, intent.ID, "operator chose external state")
	require.ErrorIs(t, err, cleanupErr)

	terminal, err := ctx.versioningSvc.GetIntent(intent.ID)
	require.NoError(t, err)
	require.Equal(t, versioning.IntentStatusFailed, terminal.Status)

	require.NoError(t, AbandonRuleVersionIntent(ctx, intent.ID, "operator chose external state"))
	_, err = ctx.versioningSvc.GetIntent(intent.ID)
	require.ErrorIs(t, err, versioning.ErrVersionIntentNotFound)
}
