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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	appcfg "github.com/apache/dubbo-admin/pkg/config/app"
	versioningcfg "github.com/apache/dubbo-admin/pkg/config/versioning"
	"github.com/apache/dubbo-admin/pkg/console/counter"
	"github.com/apache/dubbo-admin/pkg/core/governor"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
)

type testContext struct {
	rm            manager.ResourceManager
	versioningSvc *versioning.Service
	adapter       *versioning.ResourceStoreAdapter
	cfg           *appcfg.AdminConfig
	stores        map[coremodel.ResourceKind]store.ResourceStore
	gov           *noopGovernor
	lockMgr       lock.Lock
}

func (c *testContext) ResourceManager() manager.ResourceManager { return c.rm }
func (c *testContext) CounterManager() counter.CounterManager   { return nil }
func (c *testContext) Config() appcfg.AdminConfig               { return *c.cfg }
func (c *testContext) AppContext() context.Context              { return context.Background() }
func (c *testContext) LockManager() lock.Lock                   { return c.lockMgr }
func (c *testContext) RuleVersioning() *versioning.Service      { return c.versioningSvc }

type recordingLock struct {
	mu     sync.Mutex
	held   bool
	events []string
}

func (l *recordingLock) Lock(context.Context, string, time.Duration) error { return nil }

func (l *recordingLock) TryLock(context.Context, string, time.Duration) (bool, error) {
	return true, nil
}

func (l *recordingLock) Unlock(context.Context, string) error { return nil }

func (l *recordingLock) Renew(context.Context, string, time.Duration) error { return nil }

func (l *recordingLock) IsLocked(context.Context, string) (bool, error) { return false, nil }

func (l *recordingLock) CleanupExpiredLocks(context.Context) error { return nil }

func (l *recordingLock) WithLock(_ context.Context, key string, _ time.Duration, fn func() error) error {
	l.mu.Lock()
	l.events = append(l.events, "lock:"+key)
	l.held = true
	l.mu.Unlock()

	err := fn()

	l.mu.Lock()
	l.held = false
	l.events = append(l.events, "unlock:"+key)
	l.mu.Unlock()
	return err
}

func (l *recordingLock) record(event string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.held {
		l.events = append(l.events, event+":locked")
		return
	}
	l.events = append(l.events, event+":unlocked")
}

func (l *recordingLock) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = nil
}

func (l *recordingLock) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.events...)
}

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
	stores         map[coremodel.ResourceKind]store.ResourceStore
	failNextCreate error
	failNextUpdate error
	failNextDelete error
	trace          *recordingLock
}

func (g *noopGovernor) CreateRule(res coremodel.Resource) error {
	if g.trace != nil {
		g.trace.record("registry:create")
	}
	if g.failNextCreate != nil {
		err := g.failNextCreate
		g.failNextCreate = nil
		return err
	}
	s, ok := g.stores[res.ResourceKind()]
	if !ok {
		return bizerror.New(bizerror.InvalidArgument, "store not found")
	}
	return s.Add(res)
}

func (g *noopGovernor) UpdateRule(res coremodel.Resource) error {
	if g.trace != nil {
		g.trace.record("registry:update")
	}
	if g.failNextUpdate != nil {
		err := g.failNextUpdate
		g.failNextUpdate = nil
		return err
	}
	s, ok := g.stores[res.ResourceKind()]
	if !ok {
		return bizerror.New(bizerror.InvalidArgument, "store not found")
	}
	return s.Update(res)
}

func (g *noopGovernor) DeleteRule(res coremodel.Resource) error {
	if g.trace != nil {
		g.trace.record("registry:delete")
	}
	if g.failNextDelete != nil {
		err := g.failNextDelete
		g.failNextDelete = nil
		return err
	}
	s, ok := g.stores[res.ResourceKind()]
	if !ok {
		return bizerror.New(bizerror.InvalidArgument, "store not found")
	}
	return s.Delete(res)
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

type failingResourceStore struct {
	store.ResourceStore
	failNextAdd bool
	err         error
}

type recordingVersionStore struct {
	store.ResourceStore
	trace *recordingLock
}

func (s *recordingVersionStore) Add(obj interface{}) error {
	if s.trace != nil {
		s.trace.record("history:add")
	}
	return s.ResourceStore.Add(obj)
}

func (s *failingResourceStore) Add(obj interface{}) error {
	if s.failNextAdd {
		s.failNextAdd = false
		return s.err
	}
	return s.ResourceStore.Add(obj)
}

func setupRollbackTestEnv(t *testing.T, wrapVersionStore ...func(store.ResourceStore) store.ResourceStore) *testContext {
	affinityStore := memoryst.NewMemoryResourceStore(meshresource.AffinityRouteKind)
	conditionStore := memoryst.NewMemoryResourceStore(meshresource.ConditionRouteKind)
	dynamicConfigStore := memoryst.NewMemoryResourceStore(meshresource.DynamicConfigKind)
	scriptStore := memoryst.NewMemoryResourceStore(meshresource.ScriptRouteKind)
	tagStore := memoryst.NewMemoryResourceStore(meshresource.TagRouteKind)
	versionStore := memoryst.NewMemoryResourceStore(meshresource.RuleVersionKind)
	for _, s := range []store.ManagedResourceStore{affinityStore, conditionStore, dynamicConfigStore, scriptStore, tagStore, versionStore} {
		require.NoError(t, s.Init(nil))
	}

	var versioningVersionStore store.ResourceStore = versionStore
	if len(wrapVersionStore) > 0 && wrapVersionStore[0] != nil {
		versioningVersionStore = wrapVersionStore[0](versionStore)
	}
	stores := map[coremodel.ResourceKind]store.ResourceStore{
		meshresource.AffinityRouteKind:  affinityStore,
		meshresource.ConditionRouteKind: conditionStore,
		meshresource.DynamicConfigKind:  dynamicConfigStore,
		meshresource.ScriptRouteKind:    scriptStore,
		meshresource.TagRouteKind:       tagStore,
		meshresource.RuleVersionKind:    versioningVersionStore,
	}

	gov := &noopGovernor{stores: stores}
	rm := manager.NewResourceManager(&testRouter{stores: stores}, &noopGovernorRouter{gov: gov})
	adapter := versioning.NewResourceStoreAdapter(versioningVersionStore)
	return &testContext{
		rm:            rm,
		versioningSvc: versioning.NewService(5, adapter),
		adapter:       adapter,
		cfg:           &appcfg.AdminConfig{RuleVersioning: &versioningcfg.Config{MaxVersionsPerRule: 5}},
		stores:        stores,
		gov:           gov,
	}
}

func conditionRule(name, payload string) *meshresource.ConditionRouteResource {
	res := meshresource.NewConditionRouteResourceWithAttributes(name, "")
	res.Spec = &meshproto.ConditionRoute{Enabled: true, Key: name, Conditions: []string{payload}}
	return res
}

func dynamicConfigRule(name, payload string) *meshresource.DynamicConfigResource {
	res := meshresource.NewDynamicConfigResourceWithAttributes(name, "")
	res.Spec = &meshproto.DynamicConfig{Enabled: true, Key: name, ConfigVersion: payload}
	return res
}

func tagRule(name, payload string) *meshresource.TagRouteResource {
	res := meshresource.NewTagRouteResourceWithAttributes(name, "")
	res.Spec = &meshproto.TagRoute{Enabled: true, Key: name, ConfigVersion: payload}
	return res
}

func ruleRef(name string) RuleRef {
	return RuleRef{Kind: meshresource.ConditionRouteKind, Name: name}
}

func TestRuleMutationsUsePerRuleDistributedLock(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		create  func(*testContext) error
		update  func(*testContext) error
		remove  func(*testContext) error
		lockKey string
	}{
		{
			name: "condition",
			key:  "condition-rule",
			create: func(ctx *testContext) error {
				return CreateConditionRuleWithOptions(ctx, conditionRule("condition-rule", "v1"), RuleMutationOptions{Author: "admin"})
			},
			update: func(ctx *testContext) error {
				return UpdateConditionRuleWithOptions(ctx, conditionRule("condition-rule", "v2"), RuleMutationOptions{Author: "admin"})
			},
			remove: func(ctx *testContext) error {
				return DeleteConditionRuleWithOptions(ctx, "condition-rule", "", RuleMutationOptions{Author: "admin"})
			},
			lockKey: lock.BuildConditionRuleLockKey("", "condition-rule"),
		},
		{
			name: "dynamic config",
			key:  "dynamic-rule",
			create: func(ctx *testContext) error {
				return CreateConfiguratorWithOptions(ctx, dynamicConfigRule("dynamic-rule", "v1"), RuleMutationOptions{Author: "admin"})
			},
			update: func(ctx *testContext) error {
				return UpdateConfiguratorWithOptions(ctx, dynamicConfigRule("dynamic-rule", "v2"), RuleMutationOptions{Author: "admin"})
			},
			remove: func(ctx *testContext) error {
				return DeleteConfiguratorWithOptions(ctx, "dynamic-rule", "", RuleMutationOptions{Author: "admin"})
			},
			lockKey: lock.BuildConfiguratorRuleLockKey("", "dynamic-rule"),
		},
		{
			name: "tag",
			key:  "tag-rule",
			create: func(ctx *testContext) error {
				return CreateTagRuleWithOptions(ctx, tagRule("tag-rule", "v1"), RuleMutationOptions{Author: "admin"})
			},
			update: func(ctx *testContext) error {
				return UpdateTagRuleWithOptions(ctx, tagRule("tag-rule", "v2"), RuleMutationOptions{Author: "admin"})
			},
			remove: func(ctx *testContext) error {
				return DeleteTagRuleWithOptions(ctx, "tag-rule", "", RuleMutationOptions{Author: "admin"})
			},
			lockKey: lock.BuildTagRouteLockKey("", "tag-rule"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trace := &recordingLock{}
			ctx := setupRollbackTestEnv(t, func(base store.ResourceStore) store.ResourceStore {
				return &recordingVersionStore{ResourceStore: base, trace: trace}
			})
			ctx.lockMgr = trace
			ctx.gov.trace = trace

			require.NoError(t, tt.create(ctx))
			assert.Contains(t, trace.snapshot(), "lock:"+tt.lockKey)
			assert.Contains(t, trace.snapshot(), "history:add:locked")
			assert.Contains(t, trace.snapshot(), "registry:create:locked")

			trace.reset()
			require.NoError(t, tt.update(ctx))
			assert.Contains(t, trace.snapshot(), "lock:"+tt.lockKey)
			assert.Contains(t, trace.snapshot(), "history:add:locked")
			assert.Contains(t, trace.snapshot(), "registry:update:locked")

			trace.reset()
			require.NoError(t, tt.remove(ctx))
			assert.Contains(t, trace.snapshot(), "lock:"+tt.lockKey)
			assert.Contains(t, trace.snapshot(), "history:add:locked")
			assert.Contains(t, trace.snapshot(), "registry:delete:locked")
		})
	}
}

func TestRollbackRunsHistoryAppendAndRegistryMutationInsideRuleLock(t *testing.T) {
	trace := &recordingLock{}
	ctx := setupRollbackTestEnv(t, func(base store.ResourceStore) store.ResourceStore {
		return &recordingVersionStore{ResourceStore: base, trace: trace}
	})
	ctx.lockMgr = trace
	ctx.gov.trace = trace

	require.NoError(t, CreateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v1"), RuleMutationOptions{Author: "admin"}))
	versions, err := ListRuleVersions(ctx, ruleRef("demo-rule"))
	require.NoError(t, err)
	targetVersionNo := versions.Items[0].VersionNo
	require.NoError(t, UpdateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v2"), RuleMutationOptions{Author: "admin"}))

	trace.reset()
	result, err := RollbackRuleVersion(ctx, ruleRef("demo-rule"), targetVersionNo, "restore", "admin")
	require.NoError(t, err)
	require.NotNil(t, result)

	events := trace.snapshot()
	assert.Contains(t, events, "lock:"+lock.BuildConditionRuleLockKey("", "demo-rule"))
	assert.Contains(t, events, "history:add:locked")
	assert.Contains(t, events, "registry:update:locked")
}

func TestUpdateAppendsBaselineBeforeFirstHistory(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	require.NoError(t, ctx.stores[meshresource.ConditionRouteKind].Add(conditionRule("demo-rule", "v1")))

	require.NoError(t, UpdateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v2"), RuleMutationOptions{Author: "admin"}))

	versions, err := ListRuleVersions(ctx, ruleRef("demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions.Items, 2)
	assert.Equal(t, versioning.OperationUpdate, versions.Items[0].Operation)
	assert.Equal(t, versioning.SourceAdmin, versions.Items[0].Source)
	assert.Equal(t, versioning.OperationCreate, versions.Items[1].Operation)
	assert.Equal(t, versioning.SourceBootstrap, versions.Items[1].Source)
	assert.Contains(t, versions.Items[1].SpecJSON, "v1")
}

func TestCreateUpdateDeleteAppendHistory(t *testing.T) {
	ctx := setupRollbackTestEnv(t)

	require.NoError(t, CreateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v1"), RuleMutationOptions{Author: "admin"}))
	require.NoError(t, UpdateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v2"), RuleMutationOptions{Author: "admin"}))
	require.NoError(t, DeleteConditionRuleWithOptions(ctx, "demo-rule", "", RuleMutationOptions{Author: "admin"}))

	versions, err := ListRuleVersions(ctx, ruleRef("demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions.Items, 3)
	assert.Equal(t, versioning.OperationDelete, versions.Items[0].Operation)
	assert.Equal(t, versioning.OperationUpdate, versions.Items[1].Operation)
	assert.Equal(t, versioning.OperationCreate, versions.Items[2].Operation)
	assert.Equal(t, versioning.SourceAdmin, versions.Items[0].Source)
	assert.Equal(t, versioning.DeleteSpecJSON, versions.Items[0].SpecJSON)
}

func TestMutationFailsClosedWhenHistoryAppendFails(t *testing.T) {
	appendErr := errors.New("history append failed")
	failingVersionStore := &failingResourceStore{err: appendErr}
	ctx := setupRollbackTestEnv(t, func(base store.ResourceStore) store.ResourceStore {
		failingVersionStore.ResourceStore = base
		return failingVersionStore
	})
	failingVersionStore.failNextAdd = true

	err := CreateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v1"), RuleMutationOptions{Author: "admin"})
	require.Error(t, err)
	_, exists, err := ctx.rm.GetByKey(meshresource.ConditionRouteKind, "/demo-rule")
	require.NoError(t, err)
	require.False(t, exists)

	versions, err := ListRuleVersions(ctx, ruleRef("demo-rule"))
	require.NoError(t, err)
	assert.Empty(t, versions.Items)
}

func TestMutationFailsClosedWhenVersioningUnavailable(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	ctx.versioningSvc = nil

	err := CreateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v1"), RuleMutationOptions{Author: "admin"})
	require.ErrorIs(t, err, versioning.ErrVersionStoreError)
	_, exists, err := ctx.rm.GetByKey(meshresource.ConditionRouteKind, "/demo-rule")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestZeroRetentionConfigStillRecordsVersions(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	ctx.versioningSvc = versioning.NewService(0, ctx.adapter)
	ctx.cfg.RuleVersioning.MaxVersionsPerRule = 0

	require.NoError(t, CreateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v1"), RuleMutationOptions{Author: "admin"}))

	versions, err := ListRuleVersions(ctx, ruleRef("demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions.Items, 1)
	assert.Equal(t, versioning.OperationCreate, versions.Items[0].Operation)
}

func TestRegistryWriteFailureReturnsErrorAfterLedgerAppend(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	ctx.gov.failNextCreate = errors.New("registry write failed")

	err := CreateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v1"), RuleMutationOptions{Author: "admin"})
	require.Error(t, err)
	_, exists, getErr := ctx.rm.GetByKey(meshresource.ConditionRouteKind, "/demo-rule")
	require.NoError(t, getErr)
	assert.False(t, exists)
	versions, listErr := ListRuleVersions(ctx, ruleRef("demo-rule"))
	require.NoError(t, listErr)
	require.Len(t, versions.Items, 1)
	assert.Equal(t, versioning.OperationCreate, versions.Items[0].Operation)
}

func TestDeleteMissingRuleDoesNotAppendHistory(t *testing.T) {
	ctx := setupRollbackTestEnv(t)

	require.NoError(t, DeleteConditionRuleWithOptions(ctx, "missing-rule", "", RuleMutationOptions{Author: "admin"}))

	versions, err := ListRuleVersions(ctx, ruleRef("missing-rule"))
	require.NoError(t, err)
	assert.Empty(t, versions.Items)
}

func TestRollbackUpsertsAndAppendsRollbackHistory(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	require.NoError(t, CreateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v1"), RuleMutationOptions{Author: "admin"}))
	versions, err := ListRuleVersions(ctx, ruleRef("demo-rule"))
	require.NoError(t, err)
	targetVersionNo := versions.Items[0].VersionNo
	require.NoError(t, DeleteConditionRuleWithOptions(ctx, "demo-rule", "", RuleMutationOptions{Author: "admin"}))

	result, err := RollbackRuleVersion(ctx, ruleRef("demo-rule"), targetVersionNo, "restore", "admin")
	require.NoError(t, err)
	assert.Equal(t, targetVersionNo, result.RolledBackFromVersionNo)
	assert.NotZero(t, result.VersionNo)

	current, exists, err := ctx.rm.GetByKey(meshresource.ConditionRouteKind, "/demo-rule")
	require.NoError(t, err)
	require.True(t, exists)
	assert.Contains(t, current.String(), "v1")
	versions, err = ListRuleVersions(ctx, ruleRef("demo-rule"))
	require.NoError(t, err)
	assert.Equal(t, versioning.SourceRollback, versions.Items[0].Source)
	assert.Equal(t, versioning.OperationCreate, versions.Items[0].Operation)
	require.NotNil(t, versions.Items[0].RolledBackFromVersionNo)
	assert.Equal(t, targetVersionNo, *versions.Items[0].RolledBackFromVersionNo)
}

func TestRollbackFailsClosedWhenHistoryAppendFails(t *testing.T) {
	appendErr := errors.New("history append failed")
	failingVersionStore := &failingResourceStore{err: appendErr}
	ctx := setupRollbackTestEnv(t, func(base store.ResourceStore) store.ResourceStore {
		failingVersionStore.ResourceStore = base
		return failingVersionStore
	})
	require.NoError(t, CreateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v1"), RuleMutationOptions{Author: "admin"}))
	require.NoError(t, UpdateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v2"), RuleMutationOptions{Author: "admin"}))
	versions, err := ListRuleVersions(ctx, ruleRef("demo-rule"))
	require.NoError(t, err)
	targetVersionNo := versions.Items[1].VersionNo
	failingVersionStore.failNextAdd = true

	_, err = RollbackRuleVersion(ctx, ruleRef("demo-rule"), targetVersionNo, "restore", "admin")
	require.Error(t, err)
	current, exists, err := ctx.rm.GetByKey(meshresource.ConditionRouteKind, "/demo-rule")
	require.NoError(t, err)
	require.True(t, exists)
	assert.Contains(t, current.String(), "v2")
}

func TestRollbackRejectsDeleteMarker(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	require.NoError(t, CreateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v1"), RuleMutationOptions{Author: "admin"}))
	require.NoError(t, DeleteConditionRuleWithOptions(ctx, "demo-rule", "", RuleMutationOptions{Author: "admin"}))
	versions, err := ListRuleVersions(ctx, ruleRef("demo-rule"))
	require.NoError(t, err)
	require.Equal(t, versioning.OperationDelete, versions.Items[0].Operation)

	_, err = RollbackRuleVersion(ctx, ruleRef("demo-rule"), versions.Items[0].VersionNo, "restore delete marker", "admin")
	require.ErrorIs(t, err, versioning.ErrRollbackToDelete)
}

func TestRollbackNoOpRejectedAgainstActualCurrent(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	require.NoError(t, CreateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v1"), RuleMutationOptions{Author: "admin"}))
	versions, err := ListRuleVersions(ctx, ruleRef("demo-rule"))
	require.NoError(t, err)

	_, err = RollbackRuleVersion(ctx, ruleRef("demo-rule"), versions.Items[0].VersionNo, "same content", "admin")
	require.ErrorIs(t, err, versioning.ErrRollbackToCurrent)
}

func TestDiffAgainstCurrentReadsLiveResourceManagerState(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	require.NoError(t, CreateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v1"), RuleMutationOptions{Author: "admin"}))
	versions, err := ListRuleVersions(ctx, ruleRef("demo-rule"))
	require.NoError(t, err)
	require.NoError(t, ctx.stores[meshresource.ConditionRouteKind].Update(conditionRule("demo-rule", "v2-outside-history")))

	diff, err := DiffRuleVersion(ctx, ruleRef("demo-rule"), versions.Items[0].VersionNo, "current")
	require.NoError(t, err)
	assert.Contains(t, diff.Left.SpecJSON, "v1")
	assert.Contains(t, diff.Right.SpecJSON, "v2-outside-history")
}
