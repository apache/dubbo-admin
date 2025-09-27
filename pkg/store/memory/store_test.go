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

package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// mockResource is a mock implementation of model.Resource for testing
type mockResource struct {
	kind      model.ResourceKind
	key       string
	mesh      string
	meta      metav1.ObjectMeta
	spec      model.ResourceSpec
	objectRef runtime.Object
}

func (mr *mockResource) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}

func (mr *mockResource) DeepCopyObject() runtime.Object {
	return mr.objectRef
}

func (mr *mockResource) ResourceKind() model.ResourceKind {
	return mr.kind
}

func (mr *mockResource) ResourceKey() string {
	return mr.key
}

func (mr *mockResource) MeshName() string {
	return mr.mesh
}

func (mr *mockResource) ResourceMeta() metav1.ObjectMeta {
	return mr.meta
}

func (mr *mockResource) ResourceSpec() model.ResourceSpec {
	return mr.spec
}

func TestNewMemoryResourceStore(t *testing.T) {
	store := NewMemoryResourceStore()
	assert.NotNil(t, store)
}

func TestResourceStore_Init(t *testing.T) {
	store := &resourceStore{}
	err := store.Init(nil)
	assert.NoError(t, err)
	assert.NotNil(t, store.storeProxy)
}

func TestResourceStore_AddAndGet(t *testing.T) {
	store := NewMemoryResourceStore()
	err := store.Init(nil)
	assert.NoError(t, err)

	// Create a mock resource
	mockRes := &mockResource{
		kind: "TestResource",
		key:  "test-key",
		mesh: "default",
		meta: metav1.ObjectMeta{
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
	assert.Equal(t, mockRes, item)

	// Get by key
	item, exists, err = store.GetByKey("test-key")
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, mockRes, item)
}

func TestResourceStore_Update(t *testing.T) {
	store := NewMemoryResourceStore()
	err := store.Init(nil)
	assert.NoError(t, err)

	// Create a mock resource
	mockRes := &mockResource{
		kind: "TestResource",
		key:  "test-key",
		mesh: "default",
		meta: metav1.ObjectMeta{
			Name:      "test-resource",
			Namespace: "default",
		},
	}

	// Add the resource
	err = store.Add(mockRes)
	assert.NoError(t, err)

	// Update the resource
	updatedRes := &mockResource{
		kind: "TestResource",
		key:  "test-key",
		mesh: "default",
		meta: metav1.ObjectMeta{
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
	assert.Equal(t, updatedRes, item)
	assert.Equal(t, "updated-resource", item.(*mockResource).meta.Name)
}

func TestResourceStore_Delete(t *testing.T) {
	store := NewMemoryResourceStore()
	err := store.Init(nil)
	assert.NoError(t, err)

	// Create a mock resource
	mockRes := &mockResource{
		kind: "TestResource",
		key:  "test-key",
		mesh: "default",
		meta: metav1.ObjectMeta{
			Name:      "test-resource",
			Namespace: "default",
		},
	}

	// Add the resource
	err = store.Add(mockRes)
	assert.NoError(t, err)

	// Delete the resource
	err = store.Delete(mockRes)
	assert.NoError(t, err)

	// Try to get the deleted resource
	_, exists, err := store.Get(mockRes)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestResourceStore_List(t *testing.T) {
	store := NewMemoryResourceStore()
	err := store.Init(nil)
	assert.NoError(t, err)

	// Create mock resources
	mockRes1 := &mockResource{
		kind: "TestResource",
		key:  "test-key-1",
		mesh: "default",
		meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}

	mockRes2 := &mockResource{
		kind: "TestResource",
		key:  "test-key-2",
		mesh: "default",
		meta: metav1.ObjectMeta{Name: "test-resource-2"},
	}

	// Add resources
	err = store.Add(mockRes1)
	assert.NoError(t, err)
	err = store.Add(mockRes2)
	assert.NoError(t, err)

	// List resources
	list := store.List()
	assert.Len(t, list, 2)
	assert.Contains(t, list, mockRes1)
	assert.Contains(t, list, mockRes2)
}

func TestResourceStore_ListKeys(t *testing.T) {
	store := NewMemoryResourceStore()
	err := store.Init(nil)
	assert.NoError(t, err)

	// Create mock resources
	mockRes1 := &mockResource{
		kind: "TestResource",
		key:  "test-key-1",
		mesh: "default",
		meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}

	mockRes2 := &mockResource{
		kind: "TestResource",
		key:  "test-key-2",
		mesh: "default",
		meta: metav1.ObjectMeta{Name: "test-resource-2"},
	}

	// Add resources
	err = store.Add(mockRes1)
	assert.NoError(t, err)
	err = store.Add(mockRes2)
	assert.NoError(t, err)

	// List keys
	keys := store.ListKeys()
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "test-key-1")
	assert.Contains(t, keys, "test-key-2")
}

func TestResourceStore_Replace(t *testing.T) {
	store := NewMemoryResourceStore()
	err := store.Init(nil)
	assert.NoError(t, err)

	// Create initial mock resources
	mockRes1 := &mockResource{
		kind: "TestResource",
		key:  "test-key-1",
		mesh: "default",
		meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}

	mockRes2 := &mockResource{
		kind: "TestResource",
		key:  "test-key-2",
		mesh: "default",
		meta: metav1.ObjectMeta{Name: "test-resource-2"},
	}

	// Add resources
	err = store.Add(mockRes1)
	assert.NoError(t, err)

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

func TestResourceStore_GetByKeys(t *testing.T) {
	store := NewMemoryResourceStore()
	err := store.Init(nil)
	assert.NoError(t, err)

	// Create mock resources
	mockRes1 := &mockResource{
		kind: "TestResource",
		key:  "test-key-1",
		mesh: "default",
		meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}

	mockRes2 := &mockResource{
		kind: "TestResource",
		key:  "test-key-2",
		mesh: "default",
		meta: metav1.ObjectMeta{Name: "test-resource-2"},
	}

	// Add only first resource
	err = store.Add(mockRes1)
	assert.NoError(t, err)

	err = store.Add(mockRes2)
	assert.NoError(t, err)

	// Get by multiple keys
	keys := []string{"test-key-1", "test-key-2", "test-key-3"}
	resources, err := store.GetByKeys(keys)
	assert.NoError(t, err)
	assert.Len(t, resources, 3)
	assert.Equal(t, mockRes1, resources["test-key-1"])
	assert.Equal(t, mockRes2, resources["test-key-2"])
	// test-key-3 doesn't exist
	assert.Nil(t, resources["test-key-3"])
}

func TestResourceStore_ListByIndexes(t *testing.T) {
	store := NewMemoryResourceStore()
	err := store.Init(nil)
	assert.NoError(t, err)

	// Create mock resources
	mockRes1 := &mockResource{
		kind: "TestResource",
		key:  "mesh1/test-key-1",
		mesh: "mesh1",
		meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}

	mockRes2 := &mockResource{
		kind: "TestResource",
		key:  "mesh1/test-key-2",
		mesh: "mesh1",
		meta: metav1.ObjectMeta{Name: "test-resource-2"},
	}

	mockRes3 := &mockResource{
		kind: "TestResource",
		key:  "mesh2/test-key-3",
		mesh: "mesh2",
		meta: metav1.ObjectMeta{Name: "test-resource-3"},
	}

	// Add indexers
	indexers := map[string]cache.IndexFunc{
		"by-mesh": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.MeshName()}, nil
		},
	}
	err = store.AddIndexers(indexers)
	assert.NoError(t, err)

	// Add resources
	err = store.Add(mockRes1)
	assert.NoError(t, err)
	err = store.Add(mockRes2)
	assert.NoError(t, err)
	err = store.Add(mockRes3)
	assert.NoError(t, err)

	// List by indexes
	indexes := map[string]string{"by-mesh": "mesh1"}
	resources, err := store.ListByIndexes(indexes)
	assert.NoError(t, err)
	assert.Len(t, resources, 2)
	// Should be sorted by ResourceKey
	assert.Equal(t, mockRes1, resources[0])
	assert.Equal(t, mockRes2, resources[1])
}

func TestResourceStore_PageListByIndexes(t *testing.T) {
	store := NewMemoryResourceStore()
	err := store.Init(nil)
	assert.NoError(t, err)

	// Create mock resources
	mockRes1 := &mockResource{
		kind: "TestResource",
		key:  "mesh1/test-key-1",
		mesh: "mesh1",
		meta: metav1.ObjectMeta{Name: "test-resource-1"},
	}

	mockRes2 := &mockResource{
		kind: "TestResource",
		key:  "mesh1/test-key-2",
		mesh: "mesh1",
		meta: metav1.ObjectMeta{Name: "test-resource-2"},
	}

	mockRes3 := &mockResource{
		kind: "TestResource",
		key:  "mesh1/test-key-3",
		mesh: "mesh1",
		meta: metav1.ObjectMeta{Name: "test-resource-3"},
	}

	// Add indexers
	indexers := map[string]cache.IndexFunc{
		"by-mesh": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.MeshName()}, nil
		},
	}
	err = store.AddIndexers(indexers)
	assert.NoError(t, err)

	// Add resources
	err = store.Add(mockRes1)
	assert.NoError(t, err)
	err = store.Add(mockRes2)
	assert.NoError(t, err)
	err = store.Add(mockRes3)
	assert.NoError(t, err)

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
	assert.Equal(t, mockRes1, pageData.Data[0])
	assert.Equal(t, mockRes2, pageData.Data[1])

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
	assert.Equal(t, mockRes3, pageData.Data[0])
}
