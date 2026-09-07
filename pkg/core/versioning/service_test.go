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
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

func TestDiffHistoryVersionsPreviousUsesPriorRecordedVersion(t *testing.T) {
	adapter := NewResourceStoreAdapter(newVersionStore(t))
	svc := NewService(10, adapter)
	v1, err := adapter.InsertVersion(t.Context(), insertRequestForTest(t, conditionRouteForVersionTest("demo-rule", "v1"), OperationCreate), 10)
	require.NoError(t, err)
	v2, err := adapter.InsertVersion(t.Context(), insertRequestForTest(t, conditionRouteForVersionTest("demo-rule", "v2"), OperationUpdate), 10)
	require.NoError(t, err)
	v3, err := adapter.InsertVersion(t.Context(), insertRequestForTest(t, conditionRouteForVersionTest("demo-rule", "v3"), OperationUpdate), 10)
	require.NoError(t, err)
	require.NotEqual(t, v1.VersionNo, v2.VersionNo)

	diff, err := svc.DiffHistoryVersions(meshresource.ConditionRouteKind, "", "demo-rule", v3.VersionNo, "previous")
	require.NoError(t, err)
	assert.Equal(t, v3.VersionNo, diff.Left.VersionNo)
	assert.Equal(t, v2.VersionNo, diff.Right.VersionNo)
	assert.Contains(t, diff.Right.SpecJSON, "v2")
}

func TestDiffHistoryVersionsUsesExplicitVersionNo(t *testing.T) {
	adapter := NewResourceStoreAdapter(newVersionStore(t))
	svc := NewService(10, adapter)
	v1, err := adapter.InsertVersion(t.Context(), insertRequestForTest(t, conditionRouteForVersionTest("demo-rule", "v1"), OperationCreate), 10)
	require.NoError(t, err)
	v2, err := adapter.InsertVersion(t.Context(), insertRequestForTest(t, conditionRouteForVersionTest("demo-rule", "v2"), OperationUpdate), 10)
	require.NoError(t, err)

	diff, err := svc.DiffHistoryVersions(meshresource.ConditionRouteKind, "", "demo-rule", v2.VersionNo, strconv.FormatInt(v1.VersionNo, 10))
	require.NoError(t, err)
	assert.Equal(t, v2.VersionNo, diff.Left.VersionNo)
	assert.Equal(t, v1.VersionNo, diff.Right.VersionNo)
}

func TestDiffHistoryVersionsRejectsNonPositiveAgainstVersionNo(t *testing.T) {
	adapter := NewResourceStoreAdapter(newVersionStore(t))
	svc := NewService(10, adapter)
	version, err := adapter.InsertVersion(t.Context(), insertRequestForTest(t, conditionRouteForVersionTest("demo-rule", "v1"), OperationCreate), 10)
	require.NoError(t, err)

	_, err = svc.DiffHistoryVersions(meshresource.ConditionRouteKind, "", "demo-rule", version.VersionNo, "0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version number")
}
