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

package dbcommon

import (
	"fmt"
	"sync"

	set "github.com/duke-git/lancet/v2/datastructure/set"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// ValueIndex represents the mapping from indexed values to resource keys for a single index.
// Structure: map[indexedValue]set[resourceKey]
// Example: map["default"]{"resource1", "resource2", "resource3"}
type ValueIndex struct {
	values map[string]set.Set[string]
}

// NewValueIndex creates a new ValueIndex
func NewValueIndex() *ValueIndex {
	return &ValueIndex{
		values: make(map[string]set.Set[string]),
	}
}

// Add adds a resource key to the specified indexed value
func (vi *ValueIndex) Add(indexedValue, resourceKey string) {
	if vi.values[indexedValue] == nil {
		vi.values[indexedValue] = set.New[string]()
	}
	vi.values[indexedValue].Add(resourceKey)
}

// Remove removes a resource key from the specified indexed value
// Returns true if the value entry becomes empty after removal
func (vi *ValueIndex) Remove(indexedValue, resourceKey string) bool {
	if vi.values[indexedValue] == nil {
		return false
	}

	vi.values[indexedValue].Delete(resourceKey)

	// Check if the set is now empty
	if vi.values[indexedValue].Size() == 0 {
		delete(vi.values, indexedValue)
		return true
	}

	return false
}

// GetKeys returns all resource keys for the specified indexed value
func (vi *ValueIndex) GetKeys(indexedValue string) []string {
	if vi.values[indexedValue] == nil {
		return []string{}
	}
	return vi.values[indexedValue].ToSlice()
}

// GetAllValues returns all indexed values in this ValueIndex
func (vi *ValueIndex) GetAllValues() []string {
	if len(vi.values) == 0 {
		return []string{}
	}

	result := make([]string, 0, len(vi.values))
	for value := range vi.values {
		result = append(result, value)
	}
	return result
}

// IsEmpty returns true if the ValueIndex has no entries
func (vi *ValueIndex) IsEmpty() bool {
	return len(vi.values) == 0
}

// Index is a thread-safe in-memory index structure that manages multiple named indices.
// Each index maps values to sets of resource keys.
//
// Structure: map[indexName]*ValueIndex
// Example: map["mesh"]*ValueIndex where ValueIndex contains {"default": {"res1", "res2"}}
type Index struct {
	mu       sync.RWMutex
	indices  map[string]*ValueIndex // map[indexName]*ValueIndex
	indexers cache.Indexers         // Index functions for creating indices
}

// NewIndex creates a new empty Index instance
func NewIndex() *Index {
	return &Index{
		indices:  make(map[string]*ValueIndex),
		indexers: cache.Indexers{},
	}
}

// AddIndexers adds new indexer functions to the Index
// Returns an error if an indexer with the same name already exists
func (idx *Index) AddIndexers(newIndexers cache.Indexers) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	for name, indexFunc := range newIndexers {
		if _, exists := idx.indexers[name]; exists {
			return fmt.Errorf("indexer %s already exists", name)
		}
		idx.indexers[name] = indexFunc
	}

	return nil
}

// GetIndexers returns a copy of all registered indexers
func (idx *Index) GetIndexers() cache.Indexers {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	result := make(cache.Indexers, len(idx.indexers))
	for k, v := range idx.indexers {
		result[k] = v
	}
	return result
}

// UpdateResource atomically updates all indices for a resource
// If oldResource is nil, it's treated as an add operation
// If oldResource is not nil, it's treated as an update operation (remove old, add new)
// This is a high-level atomic operation that handles all indexers internally
func (idx *Index) UpdateResource(newResource model.Resource, oldResource model.Resource) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Remove old resource from indices if this is an update
	if oldResource != nil {
		idx.removeResourceUnsafe(oldResource)
	}

	// Add new resource to indices
	idx.addResourceUnsafe(newResource)
}

// RemoveResource atomically removes a resource from all indices
func (idx *Index) RemoveResource(resource model.Resource) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.removeResourceUnsafe(resource)
}

// GetKeys returns all resource keys for a given index name and value
// Returns an empty slice if the index name or value doesn't exist
func (idx *Index) GetKeys(indexName, indexValue string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	valueIndex := idx.indices[indexName]
	if valueIndex == nil {
		return []string{}
	}

	return valueIndex.GetKeys(indexValue)
}

// ListIndexFuncValues returns all indexed values for a given index name
// This directly retrieves values from the in-memory index without recalculating
func (idx *Index) ListIndexFuncValues(indexName string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	valueIndex := idx.indices[indexName]
	if valueIndex == nil {
		return []string{}
	}

	return valueIndex.GetAllValues()
}

// Clear removes all entries from the index
func (idx *Index) Clear() {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.indices = make(map[string]*ValueIndex)
}

// IndexExists checks if an indexer with the given name exists
func (idx *Index) IndexExists(indexName string) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	_, exists := idx.indexers[indexName]
	return exists
}

// addResourceUnsafe adds a resource to all indices (must be called with lock held)
func (idx *Index) addResourceUnsafe(resource model.Resource) {
	for indexName, indexFunc := range idx.indexers {
		values, err := indexFunc(resource)
		if err != nil {
			continue
		}

		// Ensure the ValueIndex exists for this index name
		if idx.indices[indexName] == nil {
			idx.indices[indexName] = NewValueIndex()
		}

		// Add resource key to each indexed value
		for _, value := range values {
			idx.indices[indexName].Add(value, resource.ResourceKey())
		}
	}
}

// removeResourceUnsafe removes a resource from all indices (must be called with lock held)
func (idx *Index) removeResourceUnsafe(resource model.Resource) {
	for indexName, indexFunc := range idx.indexers {
		values, err := indexFunc(resource)
		if err != nil {
			continue
		}

		valueIndex := idx.indices[indexName]
		if valueIndex == nil {
			continue
		}

		// Remove resource key from each indexed value
		for _, value := range values {
			valueIndex.Remove(value, resource.ResourceKey())
		}

		// Clean up empty ValueIndex
		if valueIndex.IsEmpty() {
			delete(idx.indices, indexName)
		}
	}
}
