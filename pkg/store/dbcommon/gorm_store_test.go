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
	"github.com/apache/dubbo-admin/pkg/core/store/index"
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

func (mr *mockResource) ResourceMesh() string {
	return mr.Mesh
}

func (mr *mockResource) String() string {
	b, err := json.Marshal(mr)
	if err != nil {
		return ""
	}
	return string(b)
}

type mockResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []*mockResource `json:"items"`
}

func (m mockResourceList) DeepCopyObject() runtime.Object {
	out := &mockResourceList{
		TypeMeta: m.TypeMeta,
	}
	m.ListMeta.DeepCopyInto(&out.ListMeta)

	if len(m.Items) == 0 {
		return out
	}
	out.Items = make([]*mockResource, len(m.Items))
	for i := range m.Items {
		out.Items[i] = m.Items[i].DeepCopyObject().(*mockResource)
	}
	return out
}

func (m mockResourceList) SetItems(items []model.Resource) {
	m.Items = make([]*mockResource, len(items))
	for i := range items {
		m.Items[i] = items[i].(*mockResource)
	}
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
		}, func() model.ResourceList {
			return &mockResourceList{
				TypeMeta: metav1.TypeMeta{
					Kind: kind.ToString(),
				},
				Items: []*mockResource{},
			}
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
	assert.NotNil(t, store.indexers)
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
			return []string{resource.ResourceMesh()}, nil
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
			return []string{resource.ResourceMesh()}, nil
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
			return []string{resource.ResourceMesh()}, nil
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
	indexes := []index.IndexCondition{{IndexName: "by-mesh", Value: "mesh1", Operator: index.Equals}}
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
	resources, err := store.ListByIndexes([]index.IndexCondition{})
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
			return []string{resource.ResourceMesh()}, nil
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
	indexes := []index.IndexCondition{{IndexName: "by-mesh", Value: "mesh1", Operator: index.Equals}}
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
			return []string{resource.ResourceMesh()}, nil
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
	indexes := []index.IndexCondition{{IndexName: "by-mesh", Value: "default", Operator: index.Equals}}
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
			return []string{resource.ResourceMesh()}, nil
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
	indexes := []index.IndexCondition{
		{IndexName: "by-mesh", Value: "mesh1", Operator: index.Equals},
		{IndexName: "by-namespace", Value: "default", Operator: index.Equals},
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
			return []string{resource.ResourceMesh()}, nil
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
			return []string{resource.ResourceMesh()}, nil
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
			return []string{resource.ResourceMesh()}, nil
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
			return []string{resource.ResourceMesh()}, nil
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
	}, func() model.ResourceList {
		return &mockResourceList{
			TypeMeta: metav1.TypeMeta{
				Kind: kind.ToString(),
			},
			Items: []*mockResource{},
		}
	})

	store := NewGormStore(kind, "test-address", pool)
	err = store.Init(nil)
	require.NoError(t, err)

	// Add indexer
	indexers := map[string]cache.IndexFunc{
		"by-mesh": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.ResourceMesh()}, nil
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

func TestGormStore_IndexPersistence(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add indexer for IP addresses
	indexers := map[string]cache.IndexFunc{
		"by-ip": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			// Simulate an IP address from the resource key
			return []string{resource.ResourceKey()}, nil
		},
	}
	err = store.AddIndexers(indexers)
	require.NoError(t, err)

	// Create and add resources with IP-like keys
	mockRes1 := &mockResource{
		Kind: "TestResource",
		Key:  "192.168.1.1",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "resource-1"},
	}
	mockRes2 := &mockResource{
		Kind: "TestResource",
		Key:  "192.168.1.2",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "resource-2"},
	}

	err = store.Add(mockRes1)
	require.NoError(t, err)
	err = store.Add(mockRes2)
	require.NoError(t, err)

	// Verify indices are persisted to resource_indices table
	db := store.pool.GetDB()
	var count int64
	err = db.Model(&ResourceIndexModel{}).
		Where("resource_kind = ? AND index_name = ?", "TestResource", "by-ip").
		Count(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count, "Both index entries should be persisted to resource_indices table")

	// Verify operator field is set to "Equals"
	var entries []ResourceIndexModel
	err = db.Where("resource_kind = ? AND index_name = ?", "TestResource", "by-ip").
		Find(&entries).Error
	assert.NoError(t, err)
	for _, entry := range entries {
		assert.Equal(t, "Equals", entry.Operator, "Operator field should be set to Equals")
	}
}

func TestGormStore_ListByIndexes_HasPrefix(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add indexer for IP addresses
	indexers := map[string]cache.IndexFunc{
		"by-ip": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.ResourceKey()}, nil
		},
	}
	err = store.AddIndexers(indexers)
	require.NoError(t, err)

	// Create resources with IP-like keys
	mockRes1 := &mockResource{
		Kind: "TestResource",
		Key:  "192.168.1.1",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "resource-1"},
	}
	mockRes2 := &mockResource{
		Kind: "TestResource",
		Key:  "192.168.1.2",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "resource-2"},
	}
	mockRes3 := &mockResource{
		Kind: "TestResource",
		Key:  "10.0.0.1",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "resource-3"},
	}

	err = store.Add(mockRes1)
	require.NoError(t, err)
	err = store.Add(mockRes2)
	require.NoError(t, err)
	err = store.Add(mockRes3)
	require.NoError(t, err)

	// Query with HasPrefix operator
	indexes := []index.IndexCondition{
		{IndexName: "by-ip", Value: "192.168", Operator: index.HasPrefix},
	}
	resources, err := store.ListByIndexes(indexes)
	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Verify the correct resources were returned
	keys := make([]string, len(resources))
	for i, res := range resources {
		keys[i] = res.ResourceKey()
	}
	assert.Contains(t, keys, "192.168.1.1")
	assert.Contains(t, keys, "192.168.1.2")
	assert.NotContains(t, keys, "10.0.0.1")
}

func TestGormStore_PageListByIndexes_HasPrefix(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add indexer for IP addresses
	indexers := map[string]cache.IndexFunc{
		"by-ip": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.ResourceKey()}, nil
		},
	}
	err = store.AddIndexers(indexers)
	require.NoError(t, err)

	// Create resources with IP-like keys
	for i := 1; i <= 5; i++ {
		mockRes := &mockResource{
			Kind: "TestResource",
			Key:  fmt.Sprintf("192.168.1.%d", i),
			Mesh: "default",
			Meta: metav1.ObjectMeta{Name: fmt.Sprintf("resource-%d", i)},
		}
		err = store.Add(mockRes)
		require.NoError(t, err)
	}

	// Query with HasPrefix operator and pagination
	indexes := []index.IndexCondition{
		{IndexName: "by-ip", Value: "192.168", Operator: index.HasPrefix},
	}
	pageReq := model.PageReq{
		PageOffset: 0,
		PageSize:   2,
	}
	pageData, err := store.PageListByIndexes(indexes, pageReq)
	assert.NoError(t, err)
	assert.Equal(t, 5, pageData.Total)
	assert.Len(t, pageData.Data, 2)

	// Verify second page
	pageReq.PageOffset = 2
	pageData, err = store.PageListByIndexes(indexes, pageReq)
	assert.NoError(t, err)
	assert.Equal(t, 5, pageData.Total)
	assert.Len(t, pageData.Data, 2)

	// Verify last page
	pageReq.PageOffset = 4
	pageData, err = store.PageListByIndexes(indexes, pageReq)
	assert.NoError(t, err)
	assert.Equal(t, 5, pageData.Total)
	assert.Len(t, pageData.Data, 1)
}

func TestGormStore_DeleteIndex_RemovesFromDB(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add indexer
	indexers := map[string]cache.IndexFunc{
		"by-name": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.ResourceMeta().Name}, nil
		},
	}
	err = store.AddIndexers(indexers)
	require.NoError(t, err)

	// Create and add resource
	mockRes := &mockResource{
		Kind: "TestResource",
		Key:  "test-key",
		Mesh: "default",
		Meta: metav1.ObjectMeta{Name: "test-resource"},
	}
	err = store.Add(mockRes)
	require.NoError(t, err)

	// Verify index entry exists in DB
	db := store.pool.GetDB()
	var count int64
	err = db.Model(&ResourceIndexModel{}).
		Where("resource_kind = ? AND resource_key = ?", "TestResource", "test-key").
		Count(&count).Error
	assert.NoError(t, err)
	assert.Greater(t, count, int64(0), "Index entry should exist after Add")

	// Delete the resource
	err = store.Delete(mockRes)
	require.NoError(t, err)

	// Verify index entry is removed from DB
	count = 0
	err = db.Model(&ResourceIndexModel{}).
		Where("resource_kind = ? AND resource_key = ?", "TestResource", "test-key").
		Count(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count, "Index entry should be removed after Delete")
}

func TestGormStore_UpdateIndex_UpdatesInDB(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.Init(nil)
	require.NoError(t, err)

	// Add indexer
	indexers := map[string]cache.IndexFunc{
		"by-mesh": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.ResourceMesh()}, nil
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

	// Verify initial index entry in DB
	db := store.pool.GetDB()
	var count int64
	err = db.Model(&ResourceIndexModel{}).
		Where("resource_kind = ? AND resource_key = ? AND index_value = ?", "TestResource", "test-key", "mesh1").
		Count(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count, "Initial index entry should exist")

	// Update resource with different mesh
	updatedRes := &mockResource{
		Kind: "TestResource",
		Key:  "test-key",
		Mesh: "mesh2",
		Meta: metav1.ObjectMeta{Name: "test-resource"},
	}
	err = store.Update(updatedRes)
	require.NoError(t, err)

	// Verify old index entry is removed
	count = 0
	err = db.Model(&ResourceIndexModel{}).
		Where("resource_kind = ? AND resource_key = ? AND index_value = ?", "TestResource", "test-key", "mesh1").
		Count(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count, "Old index entry should be removed")

	// Verify new index entry exists
	count = 0
	err = db.Model(&ResourceIndexModel{}).
		Where("resource_kind = ? AND resource_key = ? AND index_value = ?", "TestResource", "test-key", "mesh2").
		Count(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count, "New index entry should exist")
}
