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
}

func (c *testContext) ResourceManager() manager.ResourceManager { return c.rm }
func (c *testContext) CounterManager() counter.CounterManager   { return nil }
func (c *testContext) Config() appcfg.AdminConfig               { return *c.cfg }
func (c *testContext) AppContext() context.Context              { return context.Background() }
func (c *testContext) LockManager() lock.Lock                   { return nil }
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
	stores map[coremodel.ResourceKind]store.ResourceStore
}

func (g *noopGovernor) CreateRule(res coremodel.Resource) error {
	s, ok := g.stores[res.ResourceKind()]
	if !ok {
		return bizerror.New(bizerror.InvalidArgument, "store not found")
	}
	return s.Add(res)
}

func (g *noopGovernor) UpdateRule(res coremodel.Resource) error {
	s, ok := g.stores[res.ResourceKind()]
	if !ok {
		return bizerror.New(bizerror.InvalidArgument, "store not found")
	}
	return s.Update(res)
}

func (g *noopGovernor) DeleteRule(res coremodel.Resource) error {
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

func (s *failingResourceStore) Add(obj interface{}) error {
	if s.failNextAdd {
		s.failNextAdd = false
		return s.err
	}
	return s.ResourceStore.Add(obj)
}

func setupRollbackTestEnv(t *testing.T, wrapVersionStore ...func(store.ResourceStore) store.ResourceStore) *testContext {
	conditionStore := memoryst.NewMemoryResourceStore(meshresource.ConditionRouteKind)
	versionStore := memoryst.NewMemoryResourceStore(meshresource.RuleVersionKind)
	for _, s := range []store.ManagedResourceStore{conditionStore, versionStore} {
		require.NoError(t, s.Init(nil))
	}

	var versioningVersionStore store.ResourceStore = versionStore
	if len(wrapVersionStore) > 0 && wrapVersionStore[0] != nil {
		versioningVersionStore = wrapVersionStore[0](versionStore)
	}
	stores := map[coremodel.ResourceKind]store.ResourceStore{
		meshresource.ConditionRouteKind: conditionStore,
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
	}
}

func conditionRule(name, payload string) *meshresource.ConditionRouteResource {
	res := meshresource.NewConditionRouteResourceWithAttributes(name, "")
	res.Spec = &meshproto.ConditionRoute{Enabled: true, Key: name, Conditions: []string{payload}}
	return res
}

func kindName(name string) RuleKindName {
	return RuleKindName{Kind: meshresource.ConditionRouteKind, Name: name}
}

func TestUpdateAppendsBaselineBeforeFirstHistory(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	require.NoError(t, ctx.stores[meshresource.ConditionRouteKind].Add(conditionRule("demo-rule", "v1")))

	require.NoError(t, UpdateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v2"), RuleMutationOptions{Author: "admin"}))

	versions, err := ListRuleVersions(ctx, kindName("demo-rule"))
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

	versions, err := ListRuleVersions(ctx, kindName("demo-rule"))
	require.NoError(t, err)
	require.Len(t, versions.Items, 3)
	assert.Equal(t, versioning.OperationDelete, versions.Items[0].Operation)
	assert.Equal(t, versioning.OperationUpdate, versions.Items[1].Operation)
	assert.Equal(t, versioning.OperationCreate, versions.Items[2].Operation)
	assert.Equal(t, versioning.SourceAdmin, versions.Items[0].Source)
	assert.Contains(t, versions.Items[0].SpecJSON, "v2")
}

func TestMainWriteSucceedsWhenHistoryAppendFails(t *testing.T) {
	appendErr := errors.New("history append failed")
	failingVersionStore := &failingResourceStore{err: appendErr}
	ctx := setupRollbackTestEnv(t, func(base store.ResourceStore) store.ResourceStore {
		failingVersionStore.ResourceStore = base
		return failingVersionStore
	})
	failingVersionStore.failNextAdd = true

	err := CreateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v1"), RuleMutationOptions{Author: "admin"})
	require.NoError(t, err)
	current, exists, err := ctx.rm.GetByKey(meshresource.ConditionRouteKind, "/demo-rule")
	require.NoError(t, err)
	require.True(t, exists)
	assert.Contains(t, current.String(), "v1")

	versions, err := ListRuleVersions(ctx, kindName("demo-rule"))
	require.NoError(t, err)
	assert.Empty(t, versions.Items)
}

func TestDeleteMissingRuleDoesNotAppendHistory(t *testing.T) {
	ctx := setupRollbackTestEnv(t)

	require.NoError(t, DeleteConditionRuleWithOptions(ctx, "missing-rule", "", RuleMutationOptions{Author: "admin"}))

	versions, err := ListRuleVersions(ctx, kindName("missing-rule"))
	require.NoError(t, err)
	assert.Empty(t, versions.Items)
}

func TestRollbackUpsertsAndAppendsRollbackHistory(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	require.NoError(t, CreateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v1"), RuleMutationOptions{Author: "admin"}))
	versions, err := ListRuleVersions(ctx, kindName("demo-rule"))
	require.NoError(t, err)
	targetID := versions.Items[0].ID
	require.NoError(t, DeleteConditionRuleWithOptions(ctx, "demo-rule", "", RuleMutationOptions{Author: "admin"}))

	result, err := RollbackRuleVersion(ctx, kindName("demo-rule"), targetID, "restore", "admin")
	require.NoError(t, err)
	require.True(t, result.HistoryRecorded)

	current, exists, err := ctx.rm.GetByKey(meshresource.ConditionRouteKind, "/demo-rule")
	require.NoError(t, err)
	require.True(t, exists)
	assert.Contains(t, current.String(), "v1")
	versions, err = ListRuleVersions(ctx, kindName("demo-rule"))
	require.NoError(t, err)
	assert.Equal(t, versioning.SourceRollback, versions.Items[0].Source)
	assert.Equal(t, versioning.OperationCreate, versions.Items[0].Operation)
	require.NotNil(t, versions.Items[0].RolledBackFromID)
	assert.Equal(t, targetID, *versions.Items[0].RolledBackFromID)
}

func TestRollbackUpsertSuccessWithHistoryAppendFailure(t *testing.T) {
	appendErr := errors.New("history append failed")
	failingVersionStore := &failingResourceStore{err: appendErr}
	ctx := setupRollbackTestEnv(t, func(base store.ResourceStore) store.ResourceStore {
		failingVersionStore.ResourceStore = base
		return failingVersionStore
	})
	require.NoError(t, CreateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v1"), RuleMutationOptions{Author: "admin"}))
	require.NoError(t, UpdateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v2"), RuleMutationOptions{Author: "admin"}))
	versions, err := ListRuleVersions(ctx, kindName("demo-rule"))
	require.NoError(t, err)
	targetID := versions.Items[1].ID
	failingVersionStore.failNextAdd = true

	result, err := RollbackRuleVersion(ctx, kindName("demo-rule"), targetID, "restore", "admin")
	require.NoError(t, err)
	require.False(t, result.HistoryRecorded)
	assert.Zero(t, result.VersionID)
	assert.Zero(t, result.VersionNo)
	current, exists, err := ctx.rm.GetByKey(meshresource.ConditionRouteKind, "/demo-rule")
	require.NoError(t, err)
	require.True(t, exists)
	assert.Contains(t, current.String(), "v1")
}

func TestRollbackRejectsDeleteMarker(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	require.NoError(t, CreateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v1"), RuleMutationOptions{Author: "admin"}))
	require.NoError(t, DeleteConditionRuleWithOptions(ctx, "demo-rule", "", RuleMutationOptions{Author: "admin"}))
	versions, err := ListRuleVersions(ctx, kindName("demo-rule"))
	require.NoError(t, err)
	require.Equal(t, versioning.OperationDelete, versions.Items[0].Operation)

	_, err = RollbackRuleVersion(ctx, kindName("demo-rule"), versions.Items[0].ID, "restore delete marker", "admin")
	require.ErrorIs(t, err, versioning.ErrRollbackToDelete)
}

func TestRollbackNoOpRejectedAgainstActualCurrent(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	require.NoError(t, CreateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v1"), RuleMutationOptions{Author: "admin"}))
	versions, err := ListRuleVersions(ctx, kindName("demo-rule"))
	require.NoError(t, err)

	_, err = RollbackRuleVersion(ctx, kindName("demo-rule"), versions.Items[0].ID, "same content", "admin")
	require.ErrorIs(t, err, versioning.ErrRollbackToCurrent)
}

func TestDiffAgainstCurrentReadsActualResource(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	require.NoError(t, CreateConditionRuleWithOptions(ctx, conditionRule("demo-rule", "v1"), RuleMutationOptions{Author: "admin"}))
	versions, err := ListRuleVersions(ctx, kindName("demo-rule"))
	require.NoError(t, err)
	require.NoError(t, ctx.stores[meshresource.ConditionRouteKind].Update(conditionRule("demo-rule", "v2-outside-history")))

	diff, err := DiffRuleVersion(ctx, kindName("demo-rule"), versions.Items[0].ID, "current")
	require.NoError(t, err)
	assert.Contains(t, diff.Left.SpecJSON, "v1")
	assert.Contains(t, diff.Right.SpecJSON, "v2-outside-history")
}
