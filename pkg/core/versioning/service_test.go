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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type unorderedHistoryStore struct {
	versions []Version
}

func (s *unorderedHistoryStore) InsertVersion(context.Context, InsertRequest, int64) (*Version, error) {
	return nil, nil
}

func (s *unorderedHistoryStore) ListLatestVersions(coremodel.ResourceKind) ([]Version, error) {
	return nil, nil
}

func (s *unorderedHistoryStore) ListVersions(coremodel.ResourceKind, string) ([]Version, error) {
	return append([]Version(nil), s.versions...), nil
}

func (s *unorderedHistoryStore) HistorySnapshot(coremodel.ResourceKind, string) (*HistorySnapshot, error) {
	return &HistorySnapshot{Versions: append([]Version(nil), s.versions...)}, nil
}

func (s *unorderedHistoryStore) GetVersion(_ coremodel.ResourceKind, _ string, id int64) (*Version, error) {
	for i := range s.versions {
		if s.versions[i].ID == id {
			v := s.versions[i]
			return &v, nil
		}
	}
	return nil, ErrVersionNotFound
}

func (s *unorderedHistoryStore) LatestVersion(coremodel.ResourceKind, string) (*Version, error) {
	return nil, nil
}

func TestDiffHistoryVersionsPreviousSortsUnorderedStoreResults(t *testing.T) {
	svc := NewService(10, &unorderedHistoryStore{versions: []Version{
		{ID: 11, VersionNo: 1, SpecJSON: `{"version":"v1"}`},
		{ID: 33, VersionNo: 3, SpecJSON: `{"version":"v3"}`},
		{ID: 22, VersionNo: 2, SpecJSON: `{"version":"v2"}`},
	}})

	diff, err := svc.DiffHistoryVersions(meshresource.ConditionRouteKind, "", "demo-rule", 33, "previous")
	require.NoError(t, err)
	assert.Equal(t, int64(33), diff.Left.ID)
	assert.Equal(t, int64(22), diff.Right.ID)
	assert.Contains(t, diff.Right.SpecJSON, "v2")
}

func TestDiffHistoryVersionsUsesExplicitVersionID(t *testing.T) {
	svc := NewService(10, &unorderedHistoryStore{versions: []Version{
		{ID: 11, VersionNo: 1, SpecJSON: `{"version":"v1"}`},
		{ID: 22, VersionNo: 2, SpecJSON: `{"version":"v2"}`},
	}})

	diff, err := svc.DiffHistoryVersions(meshresource.ConditionRouteKind, "", "demo-rule", 22, "11")
	require.NoError(t, err)
	assert.Equal(t, int64(22), diff.Left.ID)
	assert.Equal(t, int64(11), diff.Right.ID)
}
