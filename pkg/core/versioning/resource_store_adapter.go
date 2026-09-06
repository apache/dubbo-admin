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
	"fmt"
	"hash/fnv"
	"sync"

	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

const parentLockStripes = 256
const maxVersionAllocateAttempts = 16

// ResourceStoreAdapter routes RuleVersion resources through the existing
// resource store. The striped mutexes only keep a single adapter instance
// internally consistent while it derives monotonically increasing version
// numbers for audit entries.
type ResourceStoreAdapter struct {
	versionStore store.ResourceStore
	parentLocks  [parentLockStripes]sync.Mutex
}

func NewResourceStoreAdapter(versionStore store.ResourceStore) *ResourceStoreAdapter {
	return &ResourceStoreAdapter{
		versionStore: versionStore,
	}
}

func (a *ResourceStoreAdapter) ensureStores() error {
	if a == nil || a.versionStore == nil {
		return fmt.Errorf("%w: RuleVersion store is required", ErrVersionStoreError)
	}
	return nil
}

func (a *ResourceStoreAdapter) LatestVersion(kind coremodel.ResourceKind, resourceKey string) (*Version, error) {
	snapshot, err := a.HistorySnapshot(kind, resourceKey)
	if err != nil {
		return nil, err
	}
	if snapshot.Head == nil {
		return nil, ErrVersionNotFound
	}
	return snapshot.Head, nil
}

func (a *ResourceStoreAdapter) withParentLock(kind coremodel.ResourceKind, resourceKey string, fn func() error) error {
	key := fmt.Sprintf("%s/%s", kind, resourceKey)
	mu := &a.parentLocks[parentLockIndex(key)]
	mu.Lock()
	defer mu.Unlock()
	return fn()
}

func parentLockIndex(key string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return h.Sum32() % parentLockStripes
}
