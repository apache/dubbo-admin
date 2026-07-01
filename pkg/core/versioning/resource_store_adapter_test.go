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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
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
	assert.Equal(t, v2.ID, snapshot.Head.ID)
	assert.Equal(t, int64(2), snapshot.Head.VersionNo)
	assert.True(t, snapshot.Versions[0].IsCurrent)
	assert.False(t, snapshot.Versions[1].IsCurrent)
	assert.NotEqual(t, v1.ID, v2.ID)
}

func TestResourceStoreAdapter_DeleteVersionMarksSnapshotDeleted(t *testing.T) {
	versionStore := newVersionStore(t)
	adapter := NewResourceStoreAdapter(versionStore)
	res := conditionRouteForVersionTest("demo-rule", "v1")
	_, err := adapter.InsertVersion(context.Background(), insertRequestForTest(t, res, OperationCreate), 10)
	require.NoError(t, err)
	deleteReq := insertRequestForTest(t, res, OperationDelete)
	deleteReq.SpecJSON = `{"conditions":["v1"]}`
	deleteReq.ContentHash = HashSpecJSON(deleteReq.SpecJSON)
	_, err = adapter.InsertVersion(context.Background(), deleteReq, 10)
	require.NoError(t, err)

	snapshot, err := adapter.HistorySnapshot(meshresource.ConditionRouteKind, res.ResourceKey())
	require.NoError(t, err)
	require.True(t, snapshot.Deleted)
	require.NotNil(t, snapshot.Head)
	assert.Equal(t, OperationDelete, snapshot.Head.Operation)
	assert.Contains(t, snapshot.Head.SpecJSON, "v1")
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
	if op == OperationDelete {
		hash, specJSON, err := NormalizeResource(res)
		require.NoError(t, err)
		req.ContentHash = hash
		req.SpecJSON = specJSON
	}
	return req
}
