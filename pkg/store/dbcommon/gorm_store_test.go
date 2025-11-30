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
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"

	storecfg "github.com/apache/dubbo-admin/pkg/config/store"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// mockResource is a mock implementation of model.Resource for testing
type mockResource struct {
	Kind      model.ResourceKind `json:"kind"`
	Key       string             `json:"key"`
	Mesh      string             `json:"mesh"`
	Meta      metav1.ObjectMeta  `json:"meta"`
	Spec      model.ResourceSpec `json:"spec"`
	ObjectRef runtime.Object     `json:"-"`
}

func (mr *mockResource) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}

func (mr *mockResource) DeepCopyObject() runtime.Object {
	return mr.ObjectRef
}

func (mr *mockResource) ResourceKind() model.ResourceKind {
	return mr.Kind
}

func (mr *mockResource) ResourceKey() string {
	return mr.Key
}

func (mr *mockResource) MeshName() string {
	return mr.Mesh
}

func (mr *mockResource) ResourceMeta() metav1.ObjectMeta {
	return mr.Meta
}

func (mr *mockResource) ResourceSpec() model.ResourceSpec {
	return mr.Spec
}

func (mr *mockResource) String() string {
	b, err := json.Marshal(mr)
	if err != nil {
		return ""
	}
	return string(b)
}

// setupTestStore creates a new GormStore with an in-memory SQLite database for testing
func setupTestStore(t *testing.T) (*GormStore, func()) {
	// Create temporary SQLite database file for better isolation and reliability
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("test-db-%s-*.db", t.Name()))
	require.NoError(t, err)
	dbPath := tmpFile.Name()
	tmpFile.Close()

	dialector := sqlite.Open(dbPath)
	pool, err := NewConnectionPool(dialector, storecfg.MySQL, t.Name(), DefaultConnectionPoolConfig())
	require.NoError(t, err)

	// Register the mock resource type (only once)
	kind := model.ResourceKind("TestResource")
	if _, err := model.ResourceSchemaRegistry().NewResourceFunc(kind); err != nil {
		model.RegisterResourceSchema(kind, func() model.Resource {
			return &mockResource{Kind: kind}
		})
	}

	store := NewGormStore(kind, t.Name(), pool)

	// Cleanup function
	cleanup := func() {
		pool.Close()
		os.Remove(dbPath)
	}

	return store, cleanup
}

func TestNewGormStore(t *testing.T) {
	dialector := sqlite.Open("file::memory:?cache=shared")
	pool, err := NewConnectionPool(dialector, storecfg.MySQL, "test-address", DefaultConnectionPoolConfig())
	require.NoError(t, err)
	defer func(pool *ConnectionPool) {
		err := pool.Close()
		if err != nil {
			return
		}
	}(pool)

	kind := model.ResourceKind("TestResource")
	store := NewGormStore(kind, "test-address", pool)

	assert.NotNil(t, store)
	assert.Equal(t, kind, store.kind)
	assert.Equal(t, "test-address", store.address)
	assert.NotNil(t, store.pool)
	assert.NotNil(t, store.indices)
	assert.NotNil(t, store.stopCh)
}

func TestGormStore_Init(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	assert.NoError(t, err)

	// Verify table was created by attempting to add a resource
	mockRes := &mockResource{
		Kind: "TestResource",
		Key:  "init-test-key",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "init-test"},
	}
	err = store.Add(mockRes)
	assert.NoError(t, err, "Should be able to add resource after Init")
}

func TestGormStore_AddAndGet(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Create a mock resource
	mockRes := &mockResource{
		Kind: "TestResource",
		Key:  "test-key",
		Mesh: "default",
		Meta: metav1.ObjectMeta{
			Name:      "test-resource",
			Namespace: "default",
		},
	}

	// Add the resource
	err = store.Add(mockRes)
	assert.NoError(t, err)

	// Get the resource
	item, exists, err := store.Get(mockRes)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.NotNil(t, item)

	// Verify the resource content
	retrieved := item.(*mockResource)
	assert.Equal(t, mockRes.ResourceKey(), retrieved.ResourceKey())
	assert.Equal(t, mockRes.MeshName(), retrieved.MeshName())
	assert.Equal(t, mockRes.ResourceMeta().Name, retrieved.ResourceMeta().Name)

	// Get by key
	item, exists, err = store.GetByKey("test-key")
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, mockRes.ResourceKey(), item.(model.Resource).ResourceKey())
}

func TestGormStore_AddDuplicate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	mockRes := &mockResource{
		Kind: "TestResource",
		Key:  "test-key",
		Mesh: "default",
		Meta: metav1.ObjectMeta{
			Name:      "test-resource",
			Namespace: "default",
		},
	}

	// Add the resource
	err = store.Add(mockRes)
	assert.NoError(t, err)

	// Try to add the same resource again
	err = store.Add(mockRes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestGormStore_AddWrongKind(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Create a resource with different kind
	mockRes := &mockResource{
		Kind: "DifferentKind",
		Key:  "test-key",
		Mesh: "default",
		Meta: metav1.ObjectMeta{
			Name: "test-resource",
		},
	}

	err = store.Add(mockRes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resource kind mismatch")
}

func TestGormStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Create a mock resource
	mockRes := &mockResource{
		Kind: "TestResource",
		Key:  "test-key",
		Mesh: "default",
		Meta: metav1.ObjectMeta{
			Name:      "test-resource",
			Namespace: "default",
		},
	}

	// Add the resource
	err = store.Add(mockRes)
	require.NoError(t, err)

	// Update the resource
	updatedRes := &mockResource{
		Kind: "TestResource",
		Key:  "test-key",
		Mesh: "default",
		Meta: metav1.ObjectMeta{
			Name:      "updated-resource",
			Namespace: "default",
		},
	}

	err = store.Update(updatedRes)
	assert.NoError(t, err)

	// Get the updated resource
	item, exists, err := store.Get(updatedRes)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, "updated-resource", item.(*mockResource).ResourceMeta().Name)
}

func TestGormStore_UpdateNonExistent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	mockRes := &mockResource{
		Kind: "TestResource",
		Key:  "non-existent-key",
		Mesh: "default",
		Meta: metav1.ObjectMeta{
			Name: "test-resource",
		},
	}

	err = store.Update(mockRes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGormStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Create a mock resource
	mockRes := &mockResource{
		Kind: "TestResource",
		Key:  "test-key",
		Mesh: "default",
		Meta: metav1.ObjectMeta{
			Name:      "test-resource",
			Namespace: "default",
		},
	}

	// Add the resource
	err = store.Add(mockRes)
	require.NoError(t, err)

	// Delete the resource
	err = store.Delete(mockRes)
	assert.NoError(t, err)

	// Try to get the deleted resource
	_, exists, err := store.Get(mockRes)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestGormStore_DeleteNonExistent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	mockRes := &mockResource{
		Kind: "TestResource",
		Key:  "non-existent-key",
		Mesh: "default",
		Meta: metav1.ObjectMeta{
			Name: "test-resource",
		},
	}

	err = store.Delete(mockRes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGormStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Create mock resources
	mockRes1 := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-1",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}

	mockRes2 := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-2",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "test-resource-2"},
	}

	// Add resources
	err = store.Add(mockRes1)
	require.NoError(t, err)
	err = store.Add(mockRes2)
	require.NoError(t, err)

	// List resources
	list := store.List()
	assert.Len(t, list, 2)

	// Verify resource keys are present
	keys := make([]string, len(list))
	for i, item := range list {
		keys[i] = item.(model.Resource).ResourceKey()
	}
	assert.Contains(t, keys, "test-key-1")
	assert.Contains(t, keys, "test-key-2")
}

func TestGormStore_ListKeys(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Create mock resources
	mockRes1 := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-1",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}

	mockRes2 := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-2",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "test-resource-2"},
	}

	// Add resources
	err = store.Add(mockRes1)
	require.NoError(t, err)
	err = store.Add(mockRes2)
	require.NoError(t, err)

	// List keys
	keys := store.ListKeys()
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "test-key-1")
	assert.Contains(t, keys, "test-key-2")
}

func TestGormStore_Replace(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Create initial mock resources
	mockRes1 := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-1",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}

	mockRes2 := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-2",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "test-resource-2"},
	}

	// Add initial resource
	err = store.Add(mockRes1)
	require.NoError(t, err)

	// Replace with new set of resources
	newResources := []interface{}{mockRes2}
	err = store.Replace(newResources, "version-1")
	assert.NoError(t, err)

	// Check that only the new resource exists
	keys := store.ListKeys()
	assert.Len(t, keys, 1)
	assert.Contains(t, keys, "test-key-2")
	assert.NotContains(t, keys, "test-key-1")
}

func TestGormStore_ReplaceEmpty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add a resource
	mockRes := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-1",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}
	err = store.Add(mockRes)
	require.NoError(t, err)

	// Replace with empty list
	err = store.Replace([]interface{}{}, "version-1")
	assert.NoError(t, err)

	// Check that all resources are removed
	keys := store.ListKeys()
	assert.Len(t, keys, 0)
}

func TestGormStore_GetByKeys(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Create mock resources
	mockRes1 := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-1",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}

	mockRes2 := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-2",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "test-resource-2"},
	}

	// Add resources
	err = store.Add(mockRes1)
	require.NoError(t, err)
	err = store.Add(mockRes2)
	require.NoError(t, err)

	// Get by multiple keys
	keys := []string{"test-key-1", "test-key-2", "test-key-3"}
	resources, err := store.GetByKeys(keys)
	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Verify resource keys
	resKeys := make([]string, len(resources))
	for i, res := range resources {
		resKeys[i] = res.ResourceKey()
	}
	assert.Contains(t, resKeys, "test-key-1")
	assert.Contains(t, resKeys, "test-key-2")
}

func TestGormStore_GetByKeysEmpty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	resources, err := store.GetByKeys([]string{})
	assert.NoError(t, err)
	assert.Len(t, resources, 0)
}

func TestGormStore_AddIndexers(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add indexers
	indexers := map[string]cache.IndexFunc{
		"by-mesh": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.MeshName()}, nil
		},
	}
	err = store.AddIndexers(indexers)
	assert.NoError(t, err)

	// Verify indexers were added
	storedIndexers := store.GetIndexers()
	assert.Len(t, storedIndexers, 1)
	assert.Contains(t, storedIndexers, "by-mesh")
}

func TestGormStore_IndexKeys(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add indexers
	indexers := map[string]cache.IndexFunc{
		"by-mesh": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.MeshName()}, nil
		},
	}
	err = store.AddIndexers(indexers)
	require.NoError(t, err)

	// Create mock resources
	mockRes1 := &mockResource{
		Kind: "TestResource",
		Key:  "mesh1/test-key-1",
		Mesh: "mesh1",
		Meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}

	mockRes2 := &mockResource{
		Kind: "TestResource",
		Key:  "mesh1/test-key-2",
		Mesh: "mesh1",
		Meta: metav1.ObjectMeta{Name: "test-resource-2"},
	}

	mockRes3 := &mockResource{
		Kind: "TestResource",
		Key:  "mesh2/test-key-3",
		Mesh: "mesh2",
		Meta: metav1.ObjectMeta{Name: "test-resource-3"},
	}

	// Add resources
	err = store.Add(mockRes1)
	require.NoError(t, err)
	err = store.Add(mockRes2)
	require.NoError(t, err)
	err = store.Add(mockRes3)
	require.NoError(t, err)

	// Test IndexKeys method
	keys, err := store.IndexKeys("by-mesh", "mesh1")
	assert.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "mesh1/test-key-1")
	assert.Contains(t, keys, "mesh1/test-key-2")

	keys, err = store.IndexKeys("by-mesh", "mesh2")
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Contains(t, keys, "mesh2/test-key-3")
}

func TestGormStore_ByIndex(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add indexers
	indexers := map[string]cache.IndexFunc{
		"by-name": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.ResourceMeta().Name}, nil
		},
	}
	err = store.AddIndexers(indexers)
	require.NoError(t, err)

	// Create mock resources
	mockRes1 := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-1",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "resource-a"},
	}

	mockRes2 := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-2",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "resource-b"},
	}

	// Add resources
	err = store.Add(mockRes1)
	require.NoError(t, err)
	err = store.Add(mockRes2)
	require.NoError(t, err)

	// Test ByIndex method
	items, err := store.ByIndex("by-name", "resource-a")
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "test-key-1", items[0].(model.Resource).ResourceKey())

	items, err = store.ByIndex("by-name", "resource-b")
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "test-key-2", items[0].(model.Resource).ResourceKey())
}

func TestGormStore_ListByIndexes(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add indexers
	indexers := map[string]cache.IndexFunc{
		"by-mesh": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.MeshName()}, nil
		},
	}
	err = store.AddIndexers(indexers)
	require.NoError(t, err)

	// Create mock resources
	mockRes1 := &mockResource{
		Kind: "TestResource",
		Key:  "mesh1/test-key-1",
		Mesh: "mesh1",
		Meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}

	mockRes2 := &mockResource{
		Kind: "TestResource",
		Key:  "mesh1/test-key-2",
		Mesh: "mesh1",
		Meta: metav1.ObjectMeta{Name: "test-resource-2"},
	}

	mockRes3 := &mockResource{
		Kind: "TestResource",
		Key:  "mesh2/test-key-3",
		Mesh: "mesh2",
		Meta: metav1.ObjectMeta{Name: "test-resource-3"},
	}

	// Add resources
	err = store.Add(mockRes1)
	require.NoError(t, err)
	err = store.Add(mockRes2)
	require.NoError(t, err)
	err = store.Add(mockRes3)
	require.NoError(t, err)

	// List by indexes
	indexes := map[string]string{"by-mesh": "mesh1"}
	resources, err := store.ListByIndexes(indexes)
	assert.NoError(t, err)
	assert.Len(t, resources, 2)
	// Should be sorted by ResourceKey
	assert.Equal(t, "mesh1/test-key-1", resources[0].ResourceKey())
	assert.Equal(t, "mesh1/test-key-2", resources[1].ResourceKey())
}

func TestGormStore_ListByIndexesEmpty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add some resources
	mockRes := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-1",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}
	err = store.Add(mockRes)
	require.NoError(t, err)

	// List with empty indexes should return all resources
	resources, err := store.ListByIndexes(map[string]string{})
	assert.NoError(t, err)
	assert.Len(t, resources, 1)
}

func TestGormStore_PageListByIndexes(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add indexers
	indexers := map[string]cache.IndexFunc{
		"by-mesh": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.MeshName()}, nil
		},
	}
	err = store.AddIndexers(indexers)
	require.NoError(t, err)

	// Create mock resources
	mockRes1 := &mockResource{
		Kind: "TestResource",
		Key:  "mesh1/test-key-1",
		Mesh: "mesh1",
		Meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}

	mockRes2 := &mockResource{
		Kind: "TestResource",
		Key:  "mesh1/test-key-2",
		Mesh: "mesh1",
		Meta: metav1.ObjectMeta{Name: "test-resource-2"},
	}

	mockRes3 := &mockResource{
		Kind: "TestResource",
		Key:  "mesh1/test-key-3",
		Mesh: "mesh1",
		Meta: metav1.ObjectMeta{Name: "test-resource-3"},
	}

	// Add resources
	err = store.Add(mockRes1)
	require.NoError(t, err)
	err = store.Add(mockRes2)
	require.NoError(t, err)
	err = store.Add(mockRes3)
	require.NoError(t, err)

	// Page list by indexes
	indexes := map[string]string{"by-mesh": "mesh1"}
	pageReq := model.PageReq{
		PageOffset: 0,
		PageSize:   2,
	}
	pageData, err := store.PageListByIndexes(indexes, pageReq)
	assert.NoError(t, err)
	// Total 3 resources
	assert.Equal(t, 3, pageData.Total)
	// Page offset 0
	assert.Equal(t, 0, pageData.PageOffset)
	// Page size 2
	assert.Equal(t, 2, pageData.PageSize)
	// 2 items in this page
	assert.Len(t, pageData.Data, 2)
	// Sorted by key
	assert.Equal(t, "mesh1/test-key-1", pageData.Data[0].ResourceKey())
	assert.Equal(t, "mesh1/test-key-2", pageData.Data[1].ResourceKey())

	// Second page
	pageReq.PageOffset = 2
	pageData, err = store.PageListByIndexes(indexes, pageReq)
	assert.NoError(t, err)
	// Total still 3
	assert.Equal(t, 3, pageData.Total)
	// Page offset 2
	assert.Equal(t, 2, pageData.PageOffset)
	// Page size 2
	assert.Equal(t, 2, pageData.PageSize)
	// Only 1 item left
	assert.Len(t, pageData.Data, 1)
	// Last item
	assert.Equal(t, "mesh1/test-key-3", pageData.Data[0].ResourceKey())
}

func TestGormStore_PageListByIndexesOffsetBeyondTotal(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add indexers
	indexers := map[string]cache.IndexFunc{
		"by-mesh": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.MeshName()}, nil
		},
	}
	err = store.AddIndexers(indexers)
	require.NoError(t, err)

	// Add one resource
	mockRes := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-1",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}
	err = store.Add(mockRes)
	require.NoError(t, err)

	// Request page beyond total
	indexes := map[string]string{"by-mesh": "default"}
	pageReq := model.PageReq{
		PageOffset: 10,
		PageSize:   2,
	}
	pageData, err := store.PageListByIndexes(indexes, pageReq)
	assert.NoError(t, err)
	assert.Equal(t, 1, pageData.Total)
	assert.Len(t, pageData.Data, 0)
}

func TestGormStore_MultipleIndexes(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add indexers for multiple fields
	indexers := map[string]cache.IndexFunc{
		"by-mesh": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.MeshName()}, nil
		},
		"by-namespace": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.ResourceMeta().Namespace}, nil
		},
	}
	err = store.AddIndexers(indexers)
	require.NoError(t, err)

	// Create mock resources
	mockRes1 := &mockResource{
		Kind: "TestResource",
		Key:  "mesh1/default/app1",
		Mesh: "mesh1",
		Meta: metav1.ObjectMeta{
			Name:      "app1",
			Namespace: "default",
		},
	}

	mockRes2 := &mockResource{
		Kind: "TestResource",
		Key:  "mesh1/default/app2",
		Mesh: "mesh1",
		Meta: metav1.ObjectMeta{
			Name:      "app2",
			Namespace: "default",
		},
	}

	mockRes3 := &mockResource{
		Kind: "TestResource",
		Key:  "mesh1/custom/app3",
		Mesh: "mesh1",
		Meta: metav1.ObjectMeta{
			Name:      "app3",
			Namespace: "custom",
		},
	}

	mockRes4 := &mockResource{
		Kind: "TestResource",
		Key:  "mesh2/default/app4",
		Mesh: "mesh2",
		Meta: metav1.ObjectMeta{
			Name:      "app4",
			Namespace: "default",
		},
	}

	// Add resources
	resources := []model.Resource{mockRes1, mockRes2, mockRes3, mockRes4}
	for _, res := range resources {
		err = store.Add(res)
		require.NoError(t, err)
	}

	// Test multiple indexes - get all resources in mesh1 and default namespace
	indexes := map[string]string{
		"by-mesh":      "mesh1",
		"by-namespace": "default",
	}
	result, err := store.ListByIndexes(indexes)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Should contain app1 and app2
	keys := make([]string, len(result))
	for i, res := range result {
		keys[i] = res.ResourceKey()
	}
	assert.Contains(t, keys, "mesh1/default/app1")
	assert.Contains(t, keys, "mesh1/default/app2")
}

func TestGormStore_ListIndexFuncValues(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add indexer
	indexers := map[string]cache.IndexFunc{
		"by-mesh": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.MeshName()}, nil
		},
	}
	err = store.AddIndexers(indexers)
	require.NoError(t, err)

	// Create mock resources with different meshes
	mockRes1 := &mockResource{
		Kind: "TestResource",
		Key:  "mesh1/resource1",
		Mesh: "mesh1",
		Meta: metav1.ObjectMeta{Name: "resource1"},
	}

	mockRes2 := &mockResource{
		Kind: "TestResource",
		Key:  "mesh2/resource2",
		Mesh: "mesh2",
		Meta: metav1.ObjectMeta{Name: "resource2"},
	}

	mockRes3 := &mockResource{
		Kind: "TestResource",
		Key:  "mesh1/resource3",
		Mesh: "mesh1",
		Meta: metav1.ObjectMeta{Name: "resource3"},
	}

	// Add resources
	resources := []model.Resource{mockRes1, mockRes2, mockRes3}
	for _, res := range resources {
		err = store.Add(res)
		require.NoError(t, err)
	}

	// Test ListIndexFuncValues method
	values := store.ListIndexFuncValues("by-mesh")
	assert.Len(t, values, 2)
	assert.Contains(t, values, "mesh1")
	assert.Contains(t, values, "mesh2")
}

func TestGormStore_IndexNonExistent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	mockRes := &mockResource{
		Kind: "TestResource",
		Key:  "test-key",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "test-resource"},
	}

	// Try to use non-existent index
	_, err = store.IndexKeys("non-existent-index", "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")

	_, err = store.ByIndex("non-existent-index", "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")

	_, err = store.Index("non-existent-index", mockRes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestGormStore_UpdateIndices(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add indexer
	indexers := map[string]cache.IndexFunc{
		"by-mesh": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.MeshName()}, nil
		},
	}
	err = store.AddIndexers(indexers)
	require.NoError(t, err)

	// Create and add initial resource
	mockRes := &mockResource{
		Kind: "TestResource",
		Key:  "test-key",
		Mesh: "mesh1",
		Meta: metav1.ObjectMeta{Name: "test-resource"},
	}
	err = store.Add(mockRes)
	require.NoError(t, err)

	// Verify initial index
	keys, err := store.IndexKeys("by-mesh", "mesh1")
	assert.NoError(t, err)
	assert.Contains(t, keys, "test-key")

	// Update resource with different mesh
	updatedRes := &mockResource{
		Kind: "TestResource",
		Key:  "test-key",
		Mesh: "mesh2",
		Meta: metav1.ObjectMeta{Name: "test-resource"},
	}
	err = store.Update(updatedRes)
	require.NoError(t, err)

	// Verify index was updated
	keys, err = store.IndexKeys("by-mesh", "mesh1")
	assert.NoError(t, err)
	assert.NotContains(t, keys, "test-key")

	keys, err = store.IndexKeys("by-mesh", "mesh2")
	assert.NoError(t, err)
	assert.Contains(t, keys, "test-key")
}

func TestGormStore_DeleteFromIndices(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add indexer
	indexers := map[string]cache.IndexFunc{
		"by-mesh": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.MeshName()}, nil
		},
	}
	err = store.AddIndexers(indexers)
	require.NoError(t, err)

	// Create and add resource
	mockRes := &mockResource{
		Kind: "TestResource",
		Key:  "test-key",
		Mesh: "mesh1",
		Meta: metav1.ObjectMeta{Name: "test-resource"},
	}
	err = store.Add(mockRes)
	require.NoError(t, err)

	// Verify initial index
	keys, err := store.IndexKeys("by-mesh", "mesh1")
	assert.NoError(t, err)
	assert.Contains(t, keys, "test-key")

	// Delete resource
	err = store.Delete(mockRes)
	require.NoError(t, err)

	// Verify index was updated
	keys, err = store.IndexKeys("by-mesh", "mesh1")
	assert.NoError(t, err)
	assert.NotContains(t, keys, "test-key")
}

func TestGormStore_ReplaceIndices(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add indexer
	indexers := map[string]cache.IndexFunc{
		"by-mesh": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.MeshName()}, nil
		},
	}
	err = store.AddIndexers(indexers)
	require.NoError(t, err)

	// Add initial resources
	mockRes1 := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-1",
		Mesh: "mesh1",
		Meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}
	err = store.Add(mockRes1)
	require.NoError(t, err)

	// Replace with new resources
	mockRes2 := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-2",
		Mesh: "mesh2",
		Meta: metav1.ObjectMeta{Name: "test-resource-2"},
	}
	err = store.Replace([]interface{}{mockRes2}, "version-1")
	require.NoError(t, err)

	// Verify indices were cleared and rebuilt
	keys, err := store.IndexKeys("by-mesh", "mesh1")
	assert.NoError(t, err)
	assert.Len(t, keys, 0)

	keys, err = store.IndexKeys("by-mesh", "mesh2")
	assert.NoError(t, err)
	assert.Contains(t, keys, "test-key-2")
}

func TestGormStore_InitRebuildIndices(t *testing.T) {
	// This test verifies that indices are rebuilt from existing data during Init()
	// Simulates the scenario where a GormStore starts with existing data in the database

	// Create store and add data
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add indexer before adding data
	indexers := map[string]cache.IndexFunc{
		"by-mesh": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.MeshName()}, nil
		},
	}
	err = store.AddIndexers(indexers)
	require.NoError(t, err)

	// Add some resources to the database
	mockRes1 := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-1",
		Mesh: "mesh1",
		Meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}
	mockRes2 := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-2",
		Mesh: "mesh2",
		Meta: metav1.ObjectMeta{Name: "test-resource-2"},
	}
	err = store.Add(mockRes1)
	require.NoError(t, err)
	err = store.Add(mockRes2)
	require.NoError(t, err)

	// Verify indices are populated
	keys, err := store.IndexKeys("by-mesh", "mesh1")
	assert.NoError(t, err)
	assert.Contains(t, keys, "test-key-1")

	// Now simulate a restart by creating a new store instance with the same pool
	// This simulates the scenario where existing data exists in the database
	pool := store.pool
	pool.IncrementRef() // Increment ref count since we're creating another store using it

	newStore := NewGormStore("TestResource", pool.Address(), pool)

	// Add indexers BEFORE Init to ensure they're available during index rebuild
	err = newStore.AddIndexers(indexers)
	require.NoError(t, err)

	// Init should rebuild indices from existing database data
	err = newStore.Init(nil)
	require.NoError(t, err)

	// Verify indices were rebuilt with existing data
	keys, err = newStore.IndexKeys("by-mesh", "mesh1")
	assert.NoError(t, err)
	assert.Contains(t, keys, "test-key-1", "Index should contain existing data after Init()")

	keys, err = newStore.IndexKeys("by-mesh", "mesh2")
	assert.NoError(t, err)
	assert.Contains(t, keys, "test-key-2", "Index should contain existing data after Init()")

	// Verify all keys are present
	allKeys := newStore.ListKeys()
	assert.Len(t, allKeys, 2)
}

func TestGormStore_Resync(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Resync should return nil (no-op for database stores)
	err = store.Resync()
	assert.NoError(t, err)
}

func TestGormStore_TransactionRollback(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add initial resource
	mockRes1 := &mockResource{
		Kind: "TestResource",
		Key:  "test-key-1",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}
	err = store.Add(mockRes1)
	require.NoError(t, err)

	// Try to replace with invalid resource (wrong type)
	// This should cause transaction rollback
	invalidList := []interface{}{"not-a-resource"}
	err = store.Replace(invalidList, "version-1")
	assert.Error(t, err)

	// Verify original resource still exists (transaction was rolled back)
	keys := store.ListKeys()
	assert.Len(t, keys, 1)
	assert.Contains(t, keys, "test-key-1")
}

func TestGormStore_ConcurrentOperations(t *testing.T) {
	// Use WAL mode for better concurrent write support in SQLite
	dialector := sqlite.Open("file::memory:?cache=shared&_journal_mode=WAL")
	pool, err := NewConnectionPool(dialector, storecfg.MySQL, "test-address", DefaultConnectionPoolConfig())
	require.NoError(t, err)
	defer pool.Close()

	// Register the mock resource type
	kind := model.ResourceKind("TestResource")
	model.RegisterResourceSchema(kind, func() model.Resource {
		return &mockResource{Kind: kind}
	})

	store := NewGormStore(kind, "test-address", pool)
	err = store.Init(nil)
	require.NoError(t, err)

	// Add indexer
	indexers := map[string]cache.IndexFunc{
		"by-mesh": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.MeshName()}, nil
		},
	}
	err = store.AddIndexers(indexers)
	require.NoError(t, err)

	// Perform concurrent Add operations with less concurrency for SQLite
	const numGoroutines = 5
	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			mockRes := &mockResource{
				Kind: "TestResource",
				Key:  string(rune('a' + index)),
				Mesh: "default",
				Meta: metav1.ObjectMeta{Name: string(rune('a' + index))},
			}
			done <- store.Add(mockRes)
		}(i)
	}

	// Wait for all goroutines to complete and check for errors
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err == nil {
			successCount++
		}
	}

	// Due to SQLite concurrency limitations, at least some operations should succeed
	assert.Greater(t, successCount, 0, "At least some concurrent operations should succeed")

	// Verify resources were added
	keys := store.ListKeys()
	assert.Greater(t, len(keys), 0, "Should have added some resources")
	assert.LessOrEqual(t, len(keys), numGoroutines, "Should not have more resources than goroutines")
}

func TestGormStore_GetNonExistent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	mockRes := &mockResource{
		Kind: "TestResource",
		Key:  "non-existent-key",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "non-existent"},
	}

	item, exists, err := store.Get(mockRes)
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, item)
}

func TestGormStore_GetByKeyNonExistent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	item, exists, err := store.GetByKey("non-existent-key")
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, item)
}

func TestGormStore_InvalidResourceType(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Try to add invalid type
	err = store.Add("not-a-resource")
	assert.Error(t, err)

	// Try to update invalid type
	err = store.Update("not-a-resource")
	assert.Error(t, err)

	// Try to delete invalid type
	err = store.Delete("not-a-resource")
	assert.Error(t, err)

	// Try to get invalid type
	_, _, err = store.Get("not-a-resource")
	assert.Error(t, err)
}
