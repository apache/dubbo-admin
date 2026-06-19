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

// testContext implements consolectx.Context for rollback tests.
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

// testRouter routes resource kinds to their in-memory stores.
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

// noopGovernor writes to the store and emits a synchronous event — simulates real governor.
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

// noopGovernorRouter routes all meshes to the same noop governor.
type noopGovernorRouter struct {
	gov *noopGovernor
}

func (r *noopGovernorRouter) ResourceRoute(res coremodel.Resource) (governor.RuleGovernor, error) {
	return r.gov, nil
}

func (r *noopGovernorRouter) ResourceMeshRoute(mesh string) (governor.RuleGovernor, error) {
	return r.gov, nil
}

// simpleBus is a minimal synchronous EventBus for tests.
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

func (b *simpleBus) Unsubscribe(sub events.Subscriber) error { return nil }

func (b *simpleBus) Send(event events.Event) {
	obj := event.NewObj()
	if obj == nil {
		obj = event.OldObj()
	}
	if obj == nil {
		return
	}
	kind := obj.ResourceKind()
	if b.muted[kind] {
		return
	}
	for _, sub := range b.subscribers[kind] {
		if !sub.AsyncEnabled() {
			_ = sub.ProcessEvent(event)
		}
	}
}

// setupRollbackTestEnv builds an in-memory ResourceManager with versioning
// subscribers for all three governor-managed rule kinds.
func setupRollbackTestEnv(t *testing.T) *testContext {
	return setupRollbackTestEnvWithMax(t, 5)
}

func setupRollbackTestEnvWithMax(t *testing.T, maxVersions int64) *testContext {
	conditionStore := memoryst.NewMemoryResourceStore(meshresource.ConditionRouteKind)
	tagStore := memoryst.NewMemoryResourceStore(meshresource.TagRouteKind)
	dynamicStore := memoryst.NewMemoryResourceStore(meshresource.DynamicConfigKind)
	versionStore := memoryst.NewMemoryResourceStore(meshresource.RuleVersionKind)
	intentStore := memoryst.NewMemoryResourceStore(meshresource.RuleIntentKind)
	metaStore := memoryst.NewMemoryResourceStore(meshresource.RuleMetaKind)

	for _, s := range []store.ManagedResourceStore{conditionStore, tagStore, dynamicStore, versionStore, intentStore, metaStore} {
		require.NoError(t, s.Init(nil))
	}

	stores := map[coremodel.ResourceKind]store.ResourceStore{
		meshresource.ConditionRouteKind: conditionStore,
		meshresource.TagRouteKind:       tagStore,
		meshresource.DynamicConfigKind:  dynamicStore,
		meshresource.RuleVersionKind:    versionStore,
		meshresource.RuleIntentKind:     intentStore,
		meshresource.RuleMetaKind:       metaStore,
	}

	storeRouter := &testRouter{stores: stores}
	bus := newSimpleBus()

	// Wire governor that writes to store and emits events
	gov := &noopGovernor{stores: stores, emitter: bus}
	govRouter := &noopGovernorRouter{gov: gov}

	rm := manager.NewResourceManager(storeRouter, govRouter)

	// Create versioning service + subscriber for each rule kind, sharing the
	// same adapter (RuleVersion/RuleIntent/RuleMeta stores).
	adapter := versioning.NewResourceStoreAdapter(versionStore, intentStore, metaStore)
	versioningSvc := versioning.NewService(true, maxVersions, adapter)
	lockMgr := locallock.NewLocalLock()
	for _, kind := range []coremodel.ResourceKind{
		meshresource.ConditionRouteKind,
		meshresource.TagRouteKind,
		meshresource.DynamicConfigKind,
	} {
		require.NoError(t, bus.Subscribe(versioning.NewSubscriber(kind, adapter, maxVersions, lockMgr)))
	}

	cfg := &appcfg.AdminConfig{
		RuleVersioning: &versioningcfg.Config{Enabled: true, MaxVersionsPerRule: maxVersions},
	}

	return &testContext{
		rm:            rm,
		versioningSvc: versioningSvc,
		adapter:       adapter,
		cfg:           cfg,
		bus:           bus,
		lockMgr:       lockMgr,
	}
}

func beginMutationForTest(ctx *testContext, res coremodel.Resource, op versioning.Operation, source versioning.Source, author string) (*versioning.Intent, error) {
	var intent *versioning.Intent
	kindName := RuleKindName{Kind: res.ResourceKind(), Mesh: res.ResourceMesh(), Name: res.ResourceMeta().Name}
	err := withRuleLock(ctx, kindName, func(leaseCtx context.Context) error {
		var inner error
		intent, inner = ctx.versioningSvc.BeginMutation(leaseCtx, res, op, source, author, "", nil)
		return inner
	})
	return intent, err
}

func TestAdminMutationSuccessCommitsLedgerBeforeReturn(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	f := conditionFactory()
	kindName := RuleKindName{Kind: f.kind, Name: "admin-rule"}

	create := f.build("admin-rule", "v1").(*meshresource.ConditionRouteResource)
	require.NoError(t, CreateConditionRuleWithOptions(ctx, create, RuleMutationOptions{Author: "admin"}))
	versions, err := ListRuleVersions(ctx, kindName)
	require.NoError(t, err)
	require.Len(t, versions.Items, 1)
	require.Equal(t, versioning.SourceAdmin, versions.Items[0].Source)
	require.Equal(t, versioning.OperationCreate, versions.Items[0].Operation)
	require.NotZero(t, versions.Items[0].IntentID)
	createVersionID := versions.Items[0].ID

	update := f.build("admin-rule", "v2").(*meshresource.ConditionRouteResource)
	require.NoError(t, UpdateConditionRuleWithOptions(ctx, update, RuleMutationOptions{
		ExpectedVersionID: &createVersionID,
		Author:            "admin",
	}))
	versions, err = ListRuleVersions(ctx, kindName)
	require.NoError(t, err)
	require.Len(t, versions.Items, 2)
	require.Equal(t, versioning.OperationUpdate, versions.Items[0].Operation)
	updateVersionID := versions.Items[0].ID

	require.NoError(t, DeleteConditionRuleWithOptions(ctx, "admin-rule", "", RuleMutationOptions{
		ExpectedVersionID: &updateVersionID,
		Author:            "admin",
	}))
	versions, err = ListRuleVersions(ctx, kindName)
	require.NoError(t, err)
	require.Len(t, versions.Items, 3)
	require.Equal(t, versioning.OperationDelete, versions.Items[0].Operation)
	require.Equal(t, versioning.DeleteSpecJSON, versions.Items[0].SpecJSON)

	require.True(t, versions.Deleted)
	require.Nil(t, versions.CurrentVersionID)
}

// ruleFactory builds a rule resource of a given kind with a discriminating
// payload, so different "versions" produce different content hashes.
type ruleFactory struct {
	kind  coremodel.ResourceKind
	build func(name, payload string) coremodel.Resource
}

func conditionFactory() ruleFactory {
	return ruleFactory{
		kind: meshresource.ConditionRouteKind,
		build: func(name, payload string) coremodel.Resource {
			res := meshresource.NewConditionRouteResourceWithAttributes(name, "")
			res.Spec = &meshproto.ConditionRoute{Enabled: true, Key: name, Conditions: []string{payload}}
			return res
		},
	}
}

func tagFactory() ruleFactory {
	return ruleFactory{
		kind: meshresource.TagRouteKind,
		build: func(name, payload string) coremodel.Resource {
			res := meshresource.NewTagRouteResourceWithAttributes(name, "")
			res.Spec = &meshproto.TagRoute{Enabled: true, Key: name, ConfigVersion: payload}
			return res
		},
	}
}

func dynamicFactory() ruleFactory {
	return ruleFactory{
		kind: meshresource.DynamicConfigKind,
		build: func(name, payload string) coremodel.Resource {
			res := meshresource.NewDynamicConfigResourceWithAttributes(name, "")
			res.Spec = &meshproto.DynamicConfig{Key: name, Enabled: true, ConfigVersion: payload}
			return res
		},
	}
}

func TestRollbackRuleVersion_Success(t *testing.T) {
	ctx := setupRollbackTestEnv(t)

	// Create initial rule (v1)
	rule1 := meshresource.NewConditionRouteResourceWithAttributes("demo-rule", "")
	rule1.Spec = &meshproto.ConditionRoute{
		Enabled:    true,
		Key:        "demo-rule",
		Conditions: []string{"host=1.2.3.4 => host=5.6.7.8"},
	}
	require.NoError(t, ctx.rm.Add(context.Background(), rule1))

	// Update rule (v2)
	rule2 := meshresource.NewConditionRouteResourceWithAttributes("demo-rule", "")
	rule2.Spec = &meshproto.ConditionRoute{
		Enabled:    true,
		Key:        "demo-rule",
		Conditions: []string{"host=9.9.9.9 => host=10.10.10.10"},
	}
	require.NoError(t, ctx.rm.Update(context.Background(), rule2))

	// Verify v1 and v2 exist
	versions, err := ListRuleVersions(ctx, RuleKindName{Kind: meshresource.ConditionRouteKind, Mesh: "", Name: "demo-rule"})
	require.NoError(t, err)
	require.Len(t, versions.Items, 2)
	assert.Equal(t, int64(2), versions.Items[0].VersionNo)
	assert.Equal(t, int64(1), versions.Items[1].VersionNo)
	assert.True(t, versions.Items[0].IsCurrent)

	v1ID := versions.Items[1].ID
	v2ID := versions.Items[0].ID

	// Rollback to v1
	result, err := RollbackRuleVersion(ctx, RuleKindName{Kind: meshresource.ConditionRouteKind, Mesh: "", Name: "demo-rule"}, v1ID, "test rollback", &v2ID, "admin")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, v1ID, result.RolledBackFromID)

	// Verify v3 was created with source=ROLLBACK and rolledBackFromId=v1
	versions, err = ListRuleVersions(ctx, RuleKindName{Kind: meshresource.ConditionRouteKind, Mesh: "", Name: "demo-rule"})
	require.NoError(t, err)
	require.Len(t, versions.Items, 3)
	v3 := versions.Items[0]
	assert.Equal(t, int64(3), v3.VersionNo)
	assert.True(t, v3.IsCurrent)
	assert.Equal(t, versioning.SourceRollback, v3.Source)
	assert.NotNil(t, v3.RolledBackFromID)
	assert.Equal(t, v1ID, *v3.RolledBackFromID)
	assert.Equal(t, "test rollback", v3.Reason)
	assert.Equal(t, v3.ID, result.VersionID)
	assert.Equal(t, v3.VersionNo, result.VersionNo)
	assert.Equal(t, string(versioning.SourceRollback), result.Source)
	assert.True(t, result.Committed)

	// Verify current rule spec matches v1
	current, exists, err := ctx.rm.GetByKey(meshresource.ConditionRouteKind, "/demo-rule")
	require.NoError(t, err)
	require.True(t, exists)
	currentRule, ok := current.(*meshresource.ConditionRouteResource)
	require.True(t, ok)
	assert.Equal(t, "host=1.2.3.4 => host=5.6.7.8", currentRule.Spec.Conditions[0])
}

func TestRollbackRuleVersion_RejectDeleteMarker(t *testing.T) {
	ctx := setupRollbackTestEnv(t)

	rule := meshresource.NewConditionRouteResourceWithAttributes("demo-rule", "")
	rule.Spec = &meshproto.ConditionRoute{Enabled: true, Key: "demo-rule"}
	require.NoError(t, ctx.rm.Add(context.Background(), rule))
	require.NoError(t, ctx.rm.DeleteByKey(context.Background(), meshresource.ConditionRouteKind, "", "/demo-rule"))

	versions, err := ListRuleVersions(ctx, RuleKindName{Kind: meshresource.ConditionRouteKind, Mesh: "", Name: "demo-rule"})
	require.NoError(t, err)
	deleteVersion := versions.Items[0]
	assert.Equal(t, versioning.OperationDelete, deleteVersion.Operation)

	_, err = RollbackRuleVersion(ctx, RuleKindName{Kind: meshresource.ConditionRouteKind, Mesh: "", Name: "demo-rule"}, deleteVersion.ID, "rollback", nil, "admin")
	require.Error(t, err)
	assert.True(t, errors.Is(err, versioning.ErrRollbackToDelete))
}

func TestRollbackRuleVersion_RejectCurrent(t *testing.T) {
	ctx := setupRollbackTestEnv(t)

	rule := meshresource.NewConditionRouteResourceWithAttributes("demo-rule", "")
	rule.Spec = &meshproto.ConditionRoute{Enabled: true, Key: "demo-rule"}
	require.NoError(t, ctx.rm.Add(context.Background(), rule))

	versions, err := ListRuleVersions(ctx, RuleKindName{Kind: meshresource.ConditionRouteKind, Mesh: "", Name: "demo-rule"})
	require.NoError(t, err)
	currentID := versions.Items[0].ID

	_, err = RollbackRuleVersion(ctx, RuleKindName{Kind: meshresource.ConditionRouteKind, Mesh: "", Name: "demo-rule"}, currentID, "rollback", nil, "admin")
	require.Error(t, err)
	assert.True(t, errors.Is(err, versioning.ErrRollbackToCurrent))
}

func TestRollbackRuleVersion_RestoresDeletedCurrentRule(t *testing.T) {
	ctx := setupRollbackTestEnv(t)

	f := conditionFactory()
	kindName := RuleKindName{Kind: f.kind, Name: "demo-rule"}
	require.NoError(t, ctx.rm.Add(context.Background(), f.build("demo-rule", "v1")))
	require.NoError(t, ctx.rm.Update(context.Background(), f.build("demo-rule", "v2")))

	versions, err := ListRuleVersions(ctx, kindName)
	require.NoError(t, err)
	v1 := versions.Items[1]
	v2 := versions.Items[0]

	require.NoError(t, ctx.rm.DeleteByKey(context.Background(), f.kind, "", "/demo-rule"))
	versions, err = ListRuleVersions(ctx, kindName)
	require.NoError(t, err)
	require.Equal(t, versioning.OperationDelete, versions.Items[0].Operation)

	result, err := RollbackRuleVersion(ctx, kindName, v1.ID, "restore deleted rule", nil, "admin")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, v1.ID, result.RolledBackFromID)

	versions, err = ListRuleVersions(ctx, kindName)
	require.NoError(t, err)
	require.Len(t, versions.Items, 4)
	restored := versions.Items[0]
	assert.Equal(t, int64(4), restored.VersionNo)
	assert.True(t, restored.IsCurrent)
	assert.Equal(t, versioning.SourceRollback, restored.Source)
	assert.Equal(t, versioning.OperationCreate, restored.Operation)
	require.NotNil(t, restored.RolledBackFromID)
	assert.Equal(t, v1.ID, *restored.RolledBackFromID)
	assert.Equal(t, restored.ID, result.VersionID)
	assert.Equal(t, restored.VersionNo, result.VersionNo)
	assert.True(t, result.Committed)

	current, exists, err := ctx.rm.GetByKey(f.kind, "/demo-rule")
	require.NoError(t, err)
	require.True(t, exists)
	curHash, _, err := versioning.NormalizeResource(current)
	require.NoError(t, err)
	assert.Equal(t, v1.ContentHash, curHash)
	assert.NotEqual(t, v2.ContentHash, curHash)
}

func TestRollbackRuleVersion_DeletedStateCASRace(t *testing.T) {
	ctx := setupRollbackTestEnv(t)

	f := conditionFactory()
	kindName := RuleKindName{Kind: f.kind, Name: "demo-rule"}
	require.NoError(t, ctx.rm.Add(context.Background(), f.build("demo-rule", "v1")))
	require.NoError(t, ctx.rm.Update(context.Background(), f.build("demo-rule", "v2")))

	versions, err := ListRuleVersions(ctx, kindName)
	require.NoError(t, err)
	v1 := versions.Items[1]
	require.NoError(t, ctx.rm.DeleteByKey(context.Background(), f.kind, "", "/demo-rule"))

	// T1 read the deleted state and therefore sends expectedVersionId=0.
	expectedDeleted := int64(0)
	require.NoError(t, ctx.rm.Add(context.Background(), f.build("demo-rule", "v3")))

	_, err = RollbackRuleVersion(ctx, kindName, v1.ID, "restore stale deleted view", &expectedDeleted, "admin")
	var conflict *versioning.ConflictError
	require.ErrorAs(t, err, &conflict)
	require.NotNil(t, conflict.CurrentVersionID)

	current, exists, err := ctx.rm.GetByKey(f.kind, "/demo-rule")
	require.NoError(t, err)
	require.True(t, exists)
	curHash, _, err := versioning.NormalizeResource(current)
	require.NoError(t, err)
	latest, err := ListRuleVersions(ctx, kindName)
	require.NoError(t, err)
	assert.Equal(t, latest.Items[0].ContentHash, curHash)
	assert.NotEqual(t, v1.ContentHash, curHash)
}

func TestRollbackRuleVersion_RepairsWhenCommitNotObserved(t *testing.T) {
	ctx := setupRollbackTestEnv(t)

	f := conditionFactory()
	kindName := RuleKindName{Kind: f.kind, Name: "demo-rule"}
	require.NoError(t, ctx.rm.Add(context.Background(), f.build("demo-rule", "v1")))
	require.NoError(t, ctx.rm.Update(context.Background(), f.build("demo-rule", "v2")))

	versions, err := ListRuleVersions(ctx, kindName)
	require.NoError(t, err)
	require.Len(t, versions.Items, 2)
	v1 := versions.Items[1]
	v2 := versions.Items[0]

	ctx.bus.muted[f.kind] = true
	result, err := RollbackRuleVersion(ctx, kindName, v1.ID, "repair rollback", &v2.ID, "admin")
	ctx.bus.muted[f.kind] = false

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Committed)
	assert.Equal(t, v1.ID, result.RolledBackFromID)

	versions, err = ListRuleVersions(ctx, kindName)
	require.NoError(t, err)
	require.Len(t, versions.Items, 3)
	repaired := versions.Items[0]
	assert.Equal(t, int64(3), repaired.VersionNo)
	assert.True(t, repaired.IsCurrent)
	assert.Equal(t, versioning.SourceRollback, repaired.Source)
	require.NotNil(t, repaired.RolledBackFromID)
	assert.Equal(t, v1.ID, *repaired.RolledBackFromID)
	assert.Equal(t, repaired.ID, result.VersionID)
	assert.Equal(t, repaired.VersionNo, result.VersionNo)

	current, exists, err := ctx.rm.GetByKey(f.kind, "/demo-rule")
	require.NoError(t, err)
	require.True(t, exists)
	curHash, _, err := versioning.NormalizeResource(current)
	require.NoError(t, err)
	assert.Equal(t, v1.ContentHash, curHash)
}

func TestRollbackRuleVersion_VersionConflict(t *testing.T) {
	ctx := setupRollbackTestEnv(t)

	rule1 := meshresource.NewConditionRouteResourceWithAttributes("demo-rule", "")
	rule1.Spec = &meshproto.ConditionRoute{Enabled: true, Key: "demo-rule", Conditions: []string{"v1"}}
	require.NoError(t, ctx.rm.Add(context.Background(), rule1))

	rule2 := meshresource.NewConditionRouteResourceWithAttributes("demo-rule", "")
	rule2.Spec = &meshproto.ConditionRoute{Enabled: true, Key: "demo-rule", Conditions: []string{"v2"}}
	require.NoError(t, ctx.rm.Update(context.Background(), rule2))

	versions, err := ListRuleVersions(ctx, RuleKindName{Kind: meshresource.ConditionRouteKind, Mesh: "", Name: "demo-rule"})
	require.NoError(t, err)
	v1ID := versions.Items[1].ID
	v2ID := versions.Items[0].ID

	// Try rollback with stale expectedVersionID
	staleExpected := int64(99999)
	_, err = RollbackRuleVersion(ctx, RuleKindName{Kind: meshresource.ConditionRouteKind, Mesh: "", Name: "demo-rule"}, v1ID, "rollback", &staleExpected, "admin")
	require.Error(t, err)
	var conflictErr *versioning.ConflictError
	assert.True(t, errors.As(err, &conflictErr))
	assert.Equal(t, v2ID, *conflictErr.CurrentVersionID)
}

func TestRollbackRuleVersion_EmptyReasonRejected(t *testing.T) {
	ctx := setupRollbackTestEnv(t)

	f := conditionFactory()
	require.NoError(t, ctx.rm.Add(context.Background(), f.build("demo-rule", "v1")))
	require.NoError(t, ctx.rm.Update(context.Background(), f.build("demo-rule", "v2")))

	versions, err := ListRuleVersions(ctx, RuleKindName{Kind: f.kind, Name: "demo-rule"})
	require.NoError(t, err)
	v1ID := versions.Items[1].ID

	_, err = RollbackRuleVersion(ctx, RuleKindName{Kind: f.kind, Name: "demo-rule"}, v1ID, "   ", nil, "admin")
	require.Error(t, err)
	var bizErr bizerror.Error
	require.True(t, errors.As(err, &bizErr))
	assert.Equal(t, bizerror.InvalidArgument, bizErr.Code())
}

func TestRollbackRuleVersion_RepeatedSameContentRejected(t *testing.T) {
	ctx := setupRollbackTestEnv(t)

	f := conditionFactory()
	require.NoError(t, ctx.rm.Add(context.Background(), f.build("demo-rule", "A")))    // v1: A
	require.NoError(t, ctx.rm.Update(context.Background(), f.build("demo-rule", "B"))) // v2: B

	versions, err := ListRuleVersions(ctx, RuleKindName{Kind: f.kind, Name: "demo-rule"})
	require.NoError(t, err)
	require.Len(t, versions.Items, 2)
	v1 := versions.Items[1]
	v2 := versions.Items[0]

	first, err := RollbackRuleVersion(ctx, RuleKindName{Kind: f.kind, Name: "demo-rule"}, v1.ID, "first rollback", &v2.ID, "admin")
	require.NoError(t, err)

	versions, err = ListRuleVersions(ctx, RuleKindName{Kind: f.kind, Name: "demo-rule"})
	require.NoError(t, err)
	require.Len(t, versions.Items, 3)
	v3 := versions.Items[0]
	require.Equal(t, first.VersionID, v3.ID)
	require.NotZero(t, v3.IntentID)
	require.Equal(t, v1.ContentHash, v3.ContentHash)

	_, err = RollbackRuleVersion(ctx, RuleKindName{Kind: f.kind, Name: "demo-rule"}, v1.ID, "second rollback", &v3.ID, "admin")
	require.ErrorIs(t, err, versioning.ErrRollbackToCurrent)

	versions, err = ListRuleVersions(ctx, RuleKindName{Kind: f.kind, Name: "demo-rule"})
	require.NoError(t, err)
	require.Len(t, versions.Items, 3)
	assert.Equal(t, first.VersionID, versions.Items[0].ID)
}

// TestRollbackRuleVersion_PendingIntentBlocks verifies that a stale PENDING
// intent that does not match the current resource blocks rollback with
// VERSION_LEDGER_PENDING.
func TestRollbackRuleVersion_PendingIntentBlocks(t *testing.T) {
	ctx := setupRollbackTestEnv(t)

	f := conditionFactory()
	require.NoError(t, ctx.rm.Add(context.Background(), f.build("demo-rule", "v1")))
	require.NoError(t, ctx.rm.Update(context.Background(), f.build("demo-rule", "v2")))

	versions, err := ListRuleVersions(ctx, RuleKindName{Kind: f.kind, Name: "demo-rule"})
	require.NoError(t, err)
	v1ID := versions.Items[1].ID

	// Inject a stale PENDING intent whose desired spec ("phantom") does not
	// match the current resource ("v2"), so repair cannot auto-clear it.
	phantom := f.build("demo-rule", "phantom-divergent")
	_, err = beginMutationForTest(ctx, phantom, versioning.OperationUpdate, versioning.SourceAdmin, "other-admin")
	require.NoError(t, err)

	_, err = RollbackRuleVersion(ctx, RuleKindName{Kind: f.kind, Name: "demo-rule"}, v1ID, "rollback", nil, "admin")
	require.Error(t, err)
	assert.True(t, errors.Is(err, versioning.ErrVersionIntentPending))
}

func TestAbandonRuleVersionIntent_CleansAppliedIntent(t *testing.T) {
	ctx := setupRollbackTestEnv(t)

	f := conditionFactory()
	require.NoError(t, ctx.rm.Add(context.Background(), f.build("demo-rule", "current")))

	stale := f.build("demo-rule", "stale-applied")
	intent, err := beginMutationForTest(ctx, stale, versioning.OperationUpdate, versioning.SourceAdmin, "admin")
	require.NoError(t, err)
	require.NoError(t, ctx.adapter.MarkIntentApplied(context.Background(), intent.ID))

	require.NoError(t, AbandonRuleVersionIntent(ctx, intent.ID, "operator decided not to repair"))

	_, err = ctx.versioningSvc.GetIntent(intent.ID)
	require.ErrorIs(t, err, versioning.ErrVersionIntentNotFound)
}

// TestRollbackRuleVersion_DuplicateEventSingleVersion verifies that a redundant
// upstream event with the rollback's content hash does not create a second
// version: the rollback intent is committed exactly once.
func TestRollbackRuleVersion_DuplicateEventSingleVersion(t *testing.T) {
	ctx := setupRollbackTestEnv(t)

	f := conditionFactory()
	require.NoError(t, ctx.rm.Add(context.Background(), f.build("demo-rule", "v1")))
	require.NoError(t, ctx.rm.Update(context.Background(), f.build("demo-rule", "v2")))

	versions, err := ListRuleVersions(ctx, RuleKindName{Kind: f.kind, Name: "demo-rule"})
	require.NoError(t, err)
	v1ID := versions.Items[1].ID
	v2ID := versions.Items[0].ID

	_, err = RollbackRuleVersion(ctx, RuleKindName{Kind: f.kind, Name: "demo-rule"}, v1ID, "rollback", &v2ID, "admin")
	require.NoError(t, err)

	// Re-apply the same spec (simulates a duplicate upstream re-registration of
	// the now-current rule). Dedup must skip it: no new version.
	require.NoError(t, ctx.rm.Update(context.Background(), f.build("demo-rule", "v1")))

	versions, err = ListRuleVersions(ctx, RuleKindName{Kind: f.kind, Name: "demo-rule"})
	require.NoError(t, err)
	assert.Len(t, versions.Items, 3, "duplicate content must not create a 4th version")
	assert.Equal(t, versioning.SourceRollback, versions.Items[0].Source)
}

// TestRollbackRuleVersion_AfterRetentionTrim verifies version numbers stay
// monotonic across rollback even after old versions are trimmed.
func TestRollbackRuleVersion_AfterRetentionTrim(t *testing.T) {
	ctx := setupRollbackTestEnvWithMax(t, 3) // keep only 3 versions

	f := conditionFactory()
	require.NoError(t, ctx.rm.Add(context.Background(), f.build("demo-rule", "p1")))    // v1 (trimmed)
	require.NoError(t, ctx.rm.Update(context.Background(), f.build("demo-rule", "p2"))) // v2 (trimmed)
	require.NoError(t, ctx.rm.Update(context.Background(), f.build("demo-rule", "p3"))) // v3
	require.NoError(t, ctx.rm.Update(context.Background(), f.build("demo-rule", "p4"))) // v4

	versions, err := ListRuleVersions(ctx, RuleKindName{Kind: f.kind, Name: "demo-rule"})
	require.NoError(t, err)
	require.Len(t, versions.Items, 3) // trimmed to 3
	// Newest first: v4, v3, v2
	assert.Equal(t, int64(4), versions.Items[0].VersionNo)
	targetID := versions.Items[1].ID // v3
	targetNo := versions.Items[1].VersionNo
	curID := versions.Items[0].ID

	result, err := RollbackRuleVersion(ctx, RuleKindName{Kind: f.kind, Name: "demo-rule"}, targetID, "rollback to v3", &curID, "admin")
	require.NoError(t, err)
	assert.Equal(t, targetID, result.RolledBackFromID)

	versions, err = ListRuleVersions(ctx, RuleKindName{Kind: f.kind, Name: "demo-rule"})
	require.NoError(t, err)
	// New version must be v5 (monotonic), even though v3 was the target.
	assert.Equal(t, int64(5), versions.Items[0].VersionNo)
	assert.Greater(t, versions.Items[0].VersionNo, targetNo)
	assert.Equal(t, versioning.SourceRollback, versions.Items[0].Source)
}

// TestRollbackRuleVersion_AllKinds is the end-to-end rollback drill across all
// three governor-managed rule kinds. It exercises the real ResourceManager →
// governor → event-bus → versioning subscriber path (no mocked rollback store):
//
//	v1 BOOTSTRAP-like create -> v2 edit -> v3 edit -> rollback(v1) -> v4
//
// asserting current spec == v1, latest == v4, v4.source == ROLLBACK,
// v4.rolledBackFromId == v1.id, and history ordering / versionNo are correct.
func TestRollbackRuleVersion_AllKinds(t *testing.T) {
	factories := map[string]ruleFactory{
		"condition": conditionFactory(),
		"tag":       tagFactory(),
		"dynamic":   dynamicFactory(),
	}

	for name, f := range factories {
		t.Run(name, func(t *testing.T) {
			ctx := setupRollbackTestEnv(t)
			kindName := RuleKindName{Kind: f.kind, Name: "drill-rule"}

			require.NoError(t, ctx.rm.Add(context.Background(), f.build("drill-rule", "spec-1")))    // v1
			require.NoError(t, ctx.rm.Update(context.Background(), f.build("drill-rule", "spec-2"))) // v2
			require.NoError(t, ctx.rm.Update(context.Background(), f.build("drill-rule", "spec-3"))) // v3

			versions, err := ListRuleVersions(ctx, kindName)
			require.NoError(t, err)
			require.Len(t, versions.Items, 3)
			v1 := versions.Items[2]
			v3 := versions.Items[0]
			require.Equal(t, int64(1), v1.VersionNo)
			require.Equal(t, int64(3), v3.VersionNo)
			require.True(t, v3.IsCurrent)

			// Rollback to v1
			result, err := RollbackRuleVersion(ctx, kindName, v1.ID, "drill rollback", &v3.ID, "admin")
			require.NoError(t, err)
			assert.Equal(t, v1.ID, result.RolledBackFromID)

			// Assert v4 created via subscriber path
			versions, err = ListRuleVersions(ctx, kindName)
			require.NoError(t, err)
			require.Len(t, versions.Items, 4)
			v4 := versions.Items[0]
			assert.Equal(t, int64(4), v4.VersionNo, "versionNo monotonic")
			assert.True(t, v4.IsCurrent)
			assert.Equal(t, versioning.SourceRollback, v4.Source)
			require.NotNil(t, v4.RolledBackFromID)
			assert.Equal(t, v1.ID, *v4.RolledBackFromID)
			assert.Equal(t, v4.ID, result.VersionID)
			assert.Equal(t, v4.VersionNo, result.VersionNo)
			assert.Equal(t, string(versioning.SourceRollback), result.Source)
			assert.True(t, result.Committed)

			// History ordering: v4 > v3 > v2 > v1
			assert.Equal(t, []int64{4, 3, 2, 1}, []int64{
				versions.Items[0].VersionNo,
				versions.Items[1].VersionNo,
				versions.Items[2].VersionNo,
				versions.Items[3].VersionNo,
			})

			// v1/v2/v3 unchanged (append-only): same content hashes as before.
			assert.Equal(t, v1.ContentHash, versions.Items[3].ContentHash)
			assert.Equal(t, v3.ContentHash, versions.Items[1].ContentHash)
			// v4 re-publishes v1's content.
			assert.Equal(t, v1.ContentHash, v4.ContentHash)

			// Current rule spec == v1 spec (verify hash equivalence via re-normalize).
			current, exists, err := ctx.rm.GetByKey(f.kind, "/drill-rule")
			require.NoError(t, err)
			require.True(t, exists)
			curHash, _, err := versioning.NormalizeResource(current)
			require.NoError(t, err)
			assert.Equal(t, v1.ContentHash, curHash)
		})
	}
}
