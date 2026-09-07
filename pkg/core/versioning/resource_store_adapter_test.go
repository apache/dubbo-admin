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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
)

func TestResourceStoreAdapter_AppendsAndListsByParentRule(t *testing.T) {
	versionStore := newVersionStore(t)
	adapter := NewResourceStoreAdapter(versionStore)
	res := conditionRouteForVersionTest("demo-rule", "v1")

	v1, err := adapter.InsertVersion(context.Background(), insertRequestForTest(t, res, OperationCreate), 10)
	require.NoError(t, err)
	updated := conditionRouteForVersionTest("demo-rule", "v2")
	v2, err := adapter.InsertVersion(context.Background(), insertRequestForTest(t, updated, OperationUpdate), 10)
	require.NoError(t, err)

	snapshot, err := adapter.HistorySnapshot(meshresource.ConditionRouteKind, res.ResourceKey())
	require.NoError(t, err)
	require.Len(t, snapshot.Versions, 2)
	assert.Equal(t, v2.VersionNo, snapshot.Head.VersionNo)
	assert.Equal(t, int64(2), snapshot.Head.VersionNo)
	assert.True(t, snapshot.Versions[0].IsLatestRecorded)
	assert.False(t, snapshot.Versions[1].IsLatestRecorded)
	assert.Equal(t, int64(1), v1.VersionNo)
	assert.Equal(t, int64(2), v2.VersionNo)
}

func TestResourceStoreAdapter_UsesDeterministicVersionResourceKey(t *testing.T) {
	versionStore := newVersionStore(t)
	adapter := NewResourceStoreAdapter(versionStore)
	res := conditionRouteForVersionTest("demo-rule", "v1")

	version, err := adapter.InsertVersion(t.Context(), insertRequestForTest(t, res, OperationCreate), 10)
	require.NoError(t, err)

	item, exists, err := versionStore.GetByKey(buildVersionResourceKey(res.ResourceKind(), res.ResourceKey(), version.VersionNo))
	require.NoError(t, err)
	require.True(t, exists)
	ruleVersion := item.(*meshresource.RuleVersionResource)
	assert.Empty(t, ruleVersion.Annotations)
}

func TestResourceStoreAdapter_GetVersionValidatesDeterministicKeyContents(t *testing.T) {
	versionStore := newVersionStore(t)
	adapter := NewResourceStoreAdapter(versionStore)
	res := conditionRouteForVersionTest("demo-rule", "v1")
	req := insertRequestForTest(t, res, OperationCreate)
	ruleVersion := newRuleVersionResource(req, 1, time.Now(), time.Now())
	ruleVersion.Spec.ParentRuleName = "different-parent"
	require.NoError(t, versionStore.Add(ruleVersion))

	_, err := adapter.GetVersion(res.ResourceKind(), res.ResourceKey(), 1)
	require.ErrorIs(t, err, ErrVersionStoreError)
}

func TestResourceStoreAdapter_RetriesVersionNumberAfterDeterministicKeyConflict(t *testing.T) {
	versionStore := newVersionStore(t)
	conflictingStore := &addConflictOnceStore{ResourceStore: versionStore}
	adapter := NewResourceStoreAdapter(conflictingStore)
	res := conditionRouteForVersionTest("demo-rule", "v1")

	version, err := adapter.InsertVersion(t.Context(), insertRequestForTest(t, res, OperationCreate), 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), version.VersionNo)

	versions, err := adapter.ListVersions(res.ResourceKind(), res.ResourceKey())
	require.NoError(t, err)
	require.Len(t, versions, 2)
	assert.Equal(t, int64(2), versions[0].VersionNo)
	assert.Equal(t, int64(1), versions[1].VersionNo)
}

func TestResourceStoreAdapter_RetentionDoesNotReuseVersionNumbers(t *testing.T) {
	adapter := NewResourceStoreAdapter(newVersionStore(t))
	res := conditionRouteForVersionTest("demo-rule", "v1")

	for i := 0; i < 4; i++ {
		res = conditionRouteForVersionTest("demo-rule", string(rune('a'+i)))
		version, err := adapter.InsertVersion(t.Context(), insertRequestForTest(t, res, OperationUpdate), 2)
		require.NoError(t, err)
		assert.Equal(t, int64(i+1), version.VersionNo)
	}

	versions, err := adapter.ListVersions(res.ResourceKind(), res.ResourceKey())
	require.NoError(t, err)
	require.Len(t, versions, 2)
	assert.Equal(t, []int64{4, 3}, []int64{versions[0].VersionNo, versions[1].VersionNo})
}

func TestResourceStoreAdapter_ZeroRetentionLimitStillRecordsHistory(t *testing.T) {
	adapter := NewResourceStoreAdapter(newVersionStore(t))
	res := conditionRouteForVersionTest("demo-rule", "v1")

	for i := 0; i < 3; i++ {
		_, err := adapter.InsertVersion(t.Context(), insertRequestForTest(t, res, OperationUpdate), 0)
		require.NoError(t, err)
	}

	versions, err := adapter.ListVersions(res.ResourceKind(), res.ResourceKey())
	require.NoError(t, err)
	assert.Len(t, versions, 3)
}

func TestResourceStoreAdapter_DeleteVersionStoresAbsenceMarker(t *testing.T) {
	versionStore := newVersionStore(t)
	adapter := NewResourceStoreAdapter(versionStore)
	res := conditionRouteForVersionTest("demo-rule", "v1")
	_, err := adapter.InsertVersion(context.Background(), insertRequestForTest(t, res, OperationCreate), 10)
	require.NoError(t, err)
	deleteReq := insertRequestForTest(t, res, OperationDelete)
	_, err = adapter.InsertVersion(context.Background(), deleteReq, 10)
	require.NoError(t, err)

	snapshot, err := adapter.HistorySnapshot(meshresource.ConditionRouteKind, res.ResourceKey())
	require.NoError(t, err)
	require.True(t, snapshot.Deleted)
	require.NotNil(t, snapshot.Head)
	assert.Equal(t, OperationDelete, snapshot.Head.Operation)
	assert.Equal(t, DeleteSpecJSON, snapshot.Head.SpecJSON)
}

func TestRecordBootstrapStateCreatesOnlyInitialBaseline(t *testing.T) {
	versionStore := newVersionStore(t)
	adapter := NewResourceStoreAdapter(versionStore)
	res := conditionRouteForVersionTest("demo-rule", "v1")

	require.NoError(t, RecordBootstrapState(context.Background(), adapter, 10, res))

	snapshot, err := adapter.HistorySnapshot(meshresource.ConditionRouteKind, res.ResourceKey())
	require.NoError(t, err)
	require.Len(t, snapshot.Versions, 1)
	assert.Equal(t, OperationCreate, snapshot.Versions[0].Operation)
	assert.Equal(t, SourceBootstrap, snapshot.Versions[0].Source)
	assert.Contains(t, snapshot.Versions[0].SpecJSON, "v1")
}

func TestRecordBootstrapStateDoesNotReconcileWhenHistoryExists(t *testing.T) {
	versionStore := newVersionStore(t)
	adapter := NewResourceStoreAdapter(versionStore)
	res := conditionRouteForVersionTest("demo-rule", "v1")
	require.NoError(t, RecordBootstrapState(context.Background(), adapter, 10, res))

	require.NoError(t, RecordBootstrapState(context.Background(), adapter, 10, conditionRouteForVersionTest("demo-rule", "v2-outside-history")))

	snapshot, err := adapter.HistorySnapshot(meshresource.ConditionRouteKind, res.ResourceKey())
	require.NoError(t, err)
	require.Len(t, snapshot.Versions, 1)
	assert.Equal(t, OperationCreate, snapshot.Versions[0].Operation)
	assert.Contains(t, snapshot.Versions[0].SpecJSON, "v1")
}

func newVersionStore(t *testing.T) store.ManagedResourceStore {
	t.Helper()
	s := memoryst.NewMemoryResourceStore(meshresource.RuleVersionKind)
	require.NoError(t, s.Init(nil))
	return s
}

func conditionRouteForVersionTest(name, payload string) *meshresource.ConditionRouteResource {
	res := meshresource.NewConditionRouteResourceWithAttributes(name, "")
	res.Spec = &meshproto.ConditionRoute{Enabled: true, Key: name, Conditions: []string{payload}}
	return res
}

func insertRequestForTest(t *testing.T, res *meshresource.ConditionRouteResource, op Operation) InsertRequest {
	t.Helper()
	req, err := BuildInsertRequest(res, op, SourceAdmin, "admin", "", nil, time.Now())
	require.NoError(t, err)
	return req
}

type addConflictOnceStore struct {
	store.ResourceStore
	once sync.Once
}

func (s *addConflictOnceStore) Add(obj interface{}) error {
	var (
		first    bool
		addErr   error
		conflict error
	)
	s.once.Do(func() {
		first = true
		res := obj.(coremodel.Resource)
		addErr = s.ResourceStore.Add(obj)
		if addErr == nil {
			conflict = store.ErrorResourceAlreadyExists(res.ResourceKind().ToString(), res.ResourceMeta().Name, res.ResourceMesh())
		}
	})
	if first {
		if addErr != nil {
			return addErr
		}
		return conflict
	}
	return s.ResourceStore.Add(obj)
}
