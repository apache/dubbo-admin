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
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

type SyncStateSource interface {
	DiscoverySyncState(string) SyncState
}

// ReadSnapshot reads unified Resources between two synchronization checks. A
// concurrent discovery update cannot turn a mixed snapshot into confirmed evidence.
func ReadSnapshot(resources manager.ResourceManager, source SyncStateSource, mesh string) (Snapshot, error) {
	snapshot := Snapshot{}
	if source != nil {
		snapshot.Sync = source.DiscoverySyncState(mesh)
	}
	conditions := []index.IndexCondition{{IndexName: index.ByMeshIndex, Value: mesh, Operator: index.Equals}}
	var err error
	snapshot.Instances, err = manager.ListByIndexes[*meshresource.RPCInstanceResource](resources, meshresource.RPCInstanceKind, conditions)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.Metadata, err = manager.ListByIndexes[*meshresource.RPCInstanceMetadataResource](resources, meshresource.RPCInstanceMetadataKind, conditions)
	if err != nil {
		return Snapshot{}, err
	}
	if source != nil && source.DiscoverySyncState(mesh) != snapshot.Sync {
		snapshot.Sync = SyncState{}
	}
	return snapshot, nil
}
