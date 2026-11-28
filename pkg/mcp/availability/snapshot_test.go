/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package availability

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

type snapshotManager struct {
	manager.ResourceManager
	t    *testing.T
	list func(model.ResourceKind) ([]model.Resource, error)
}

func (m snapshotManager) ListByIndexes(kind model.ResourceKind, conditions []index.IndexCondition) ([]model.Resource, error) {
	require.Equal(m.t, []index.IndexCondition{{IndexName: index.ByMeshIndex, Value: "mesh-1", Operator: index.Equals}}, conditions)
	return m.list(kind)
}

type syncSourceFunc func(string) SyncState

func (f syncSourceFunc) DiscoverySyncState(mesh string) SyncState { return f(mesh) }

func TestReadSnapshotRequiresStableSynchronization(t *testing.T) {
	for _, changed := range []bool{false, true} {
		state := SyncState{InitialSyncComplete: true, Healthy: true, Generation: 2}
		source := syncSourceFunc(func(mesh string) SyncState { require.Equal(t, "mesh-1", mesh); return state })
		instance := meshresource.NewRPCInstanceResourceWithAttributes("orders-1", "mesh-1")
		metadata := meshresource.NewRPCInstanceMetadataResourceWithAttributes("orders@r1", "mesh-1")
		resources := snapshotManager{t: t, list: func(kind model.ResourceKind) ([]model.Resource, error) {
			if kind == meshresource.RPCInstanceKind {
				return []model.Resource{instance}, nil
			}
			require.Equal(t, meshresource.RPCInstanceMetadataKind, kind)
			if changed {
				state.Generation += 2
			}
			return []model.Resource{metadata}, nil
		}}
		snapshot, err := ReadSnapshot(resources, source, "mesh-1")
		require.NoError(t, err)
		require.Equal(t, []*meshresource.RPCInstanceResource{instance}, snapshot.Instances)
		require.Equal(t, []*meshresource.RPCInstanceMetadataResource{metadata}, snapshot.Metadata)
		if changed {
			require.Equal(t, SyncState{}, snapshot.Sync)
		} else {
			require.Equal(t, state, snapshot.Sync)
		}
		snapshot, err = ReadSnapshot(resources, nil, "mesh-1")
		require.NoError(t, err)
		require.False(t, snapshot.Sync.InitialSyncComplete)
	}
	resources := snapshotManager{t: t, list: func(model.ResourceKind) ([]model.Resource, error) { return nil, errors.New("store offline") }}
	_, err := ReadSnapshot(resources, nil, "mesh-1")
	require.ErrorContains(t, err, "store offline")
}
