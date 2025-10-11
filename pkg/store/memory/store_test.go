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
	assert.Len(t, resources, 2)
	assert.Equal(t, mockRes1, resources[0])
	assert.Equal(t, mockRes2, resources[1])
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

func TestResourceStore_MultipleIndexes(t *testing.T) {
	store := NewMemoryResourceStore()
	err := store.Init(nil)
	assert.NoError(t, err)

	// Create mock resources
	mockRes1 := &mockResource{
		kind: "TestApplication",
		key:  "mesh1/app1",
		mesh: "mesh1",
		meta: metav1.ObjectMeta{
			Name:      "app1",
			Namespace: "default",
			Labels: map[string]string{
				"version": "v1",
				"env":     "prod",
			},
		},
	}

	mockRes2 := &mockResource{
		kind: "TestApplication",
		key:  "mesh1/app2",
		mesh: "mesh1",
		meta: metav1.ObjectMeta{
			Name:      "app2",
			Namespace: "default",
			Labels: map[string]string{
				"version": "v2",
				"env":     "prod",
			},
		},
	}

	mockRes3 := &mockResource{
		kind: "TestApplication",
		key:  "mesh2/app3",
		mesh: "mesh2",
		meta: metav1.ObjectMeta{
			Name:      "app3",
			Namespace: "default",
			Labels: map[string]string{
				"version": "v1",
				"env":     "dev",
			},
		},
	}

	// Add indexers for multiple fields
	indexers := map[string]cache.IndexFunc{
		"by-mesh": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			return []string{resource.MeshName()}, nil
		},
		"by-version": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			version := resource.ResourceMeta().Labels["version"]
			return []string{version}, nil
		},
		"by-env": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			env := resource.ResourceMeta().Labels["env"]
			return []string{env}, nil
		},
	}
	err = store.AddIndexers(indexers)
	assert.NoError(t, err)

	// Add resources
	resources := []model.Resource{mockRes1, mockRes2, mockRes3}
	for _, res := range resources {
		err = store.Add(res)
		assert.NoError(t, err)
	}

	// Test multiple indexes - get all prod env resources in mesh1
	indexes := map[string]string{
		"by-mesh":    "mesh1",
		"by-version": "v1",
	}
	result, err := store.ListByIndexes(indexes)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	// Should contain app1
	keys := make([]string, len(result))
	for i, res := range result {
		keys[i] = res.ResourceKey()
	}
	assert.Contains(t, keys, "mesh1/app1")
}

func TestResourceStore_IndexKeys(t *testing.T) {
	store := NewMemoryResourceStore()
	err := store.Init(nil)
	assert.NoError(t, err)

	// Create mock resources
	mockRes1 := &mockResource{
		kind: "TestService",
		key:  "mesh1/service1",
		mesh: "mesh1",
		meta: metav1.ObjectMeta{
			Name: "service1",
			Labels: map[string]string{
				"group": "frontend",
			},
		},
	}

	mockRes2 := &mockResource{
		kind: "TestService",
		key:  "mesh1/service2",
		mesh: "mesh1",
		meta: metav1.ObjectMeta{
			Name: "service2",
			Labels: map[string]string{
				"group": "backend",
			},
		},
	}

	mockRes3 := &mockResource{
		kind: "TestService",
		key:  "mesh1/service3",
		mesh: "mesh1",
		meta: metav1.ObjectMeta{
			Name: "service3",
			Labels: map[string]string{
				"group": "frontend",
			},
		},
	}

	// Add indexer
	indexers := map[string]cache.IndexFunc{
		"by-group": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			group := resource.ResourceMeta().Labels["group"]
			return []string{group}, nil
		},
	}
	err = store.AddIndexers(indexers)
	assert.NoError(t, err)

	// Add resources
	resources := []model.Resource{mockRes1, mockRes2, mockRes3}
	for _, res := range resources {
		err = store.Add(res)
		assert.NoError(t, err)
	}

	// Test IndexKeys method
	keys, err := store.IndexKeys("by-group", "frontend")
	assert.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "mesh1/service1")
	assert.Contains(t, keys, "mesh1/service3")

	keys, err = store.IndexKeys("by-group", "backend")
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Contains(t, keys, "mesh1/service2")
}

func TestResourceStore_ByIndex(t *testing.T) {
	store := NewMemoryResourceStore()
	err := store.Init(nil)
	assert.NoError(t, err)

	// Create mock resources
	mockRes1 := &mockResource{
		kind: "TestInstance",
		key:  "mesh1/instance1",
		mesh: "mesh1",
		meta: metav1.ObjectMeta{
			Name: "instance1",
			Labels: map[string]string{
				"type": "web",
			},
		},
	}

	mockRes2 := &mockResource{
		kind: "TestInstance",
		key:  "mesh1/instance2",
		mesh: "mesh1",
		meta: metav1.ObjectMeta{
			Name: "instance2",
			Labels: map[string]string{
				"type": "database",
			},
		},
	}

	// Add indexer
	indexers := map[string]cache.IndexFunc{
		"by-type": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			instanceType := resource.ResourceMeta().Labels["type"]
			return []string{instanceType}, nil
		},
	}
	err = store.AddIndexers(indexers)
	assert.NoError(t, err)

	// Add resources
	err = store.Add(mockRes1)
	assert.NoError(t, err)
	err = store.Add(mockRes2)
	assert.NoError(t, err)

	// Test ByIndex method
	items, err := store.ByIndex("by-type", "web")
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, mockRes1, items[0])

	items, err = store.ByIndex("by-type", "database")
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, mockRes2, items[0])
}

func TestResourceStore_GetIndexers(t *testing.T) {
	store := NewMemoryResourceStore()
	err := store.Init(nil)
	assert.NoError(t, err)

	// Initially no indexers
	indexers := store.GetIndexers()
	assert.Empty(t, indexers)

	// Add indexers
	newIndexers := map[string]cache.IndexFunc{
		"by-name": func(obj interface{}) ([]string, error) {
			return []string{obj.(model.Resource).ResourceMeta().Name}, nil
		},
		"by-kind": func(obj interface{}) ([]string, error) {
			return []string{string(obj.(model.Resource).ResourceKind())}, nil
		},
	}
	err = store.AddIndexers(newIndexers)
	assert.NoError(t, err)

	// Check if indexers were added
	indexers = store.GetIndexers()
	assert.Len(t, indexers, 2)
	assert.Contains(t, indexers, "by-name")
	assert.Contains(t, indexers, "by-kind")
}

func TestResourceStore_ListIndexFuncValues(t *testing.T) {
	store := NewMemoryResourceStore()
	err := store.Init(nil)
	assert.NoError(t, err)

	// Create mock resources
	mockRes1 := &mockResource{
		kind: "TestResource",
		key:  "mesh1/resource1",
		mesh: "mesh1",
		meta: metav1.ObjectMeta{
			Name: "resource1",
			Labels: map[string]string{
				"status": "active",
			},
		},
	}

	mockRes2 := &mockResource{
		kind: "TestResource",
		key:  "mesh1/resource2",
		mesh: "mesh1",
		meta: metav1.ObjectMeta{
			Name: "resource2",
			Labels: map[string]string{
				"status": "inactive",
			},
		},
	}

	mockRes3 := &mockResource{
		kind: "TestResource",
		key:  "mesh1/resource3",
		mesh: "mesh1",
		meta: metav1.ObjectMeta{
			Name: "resource3",
			Labels: map[string]string{
				"status": "active",
			},
		},
	}

	// Add indexer
	indexers := map[string]cache.IndexFunc{
		"by-status": func(obj interface{}) ([]string, error) {
			resource := obj.(model.Resource)
			status := resource.ResourceMeta().Labels["status"]
			return []string{status}, nil
		},
	}
	err = store.AddIndexers(indexers)
	assert.NoError(t, err)

	// Add resources
	resources := []model.Resource{mockRes1, mockRes2, mockRes3}
	for _, res := range resources {
		err = store.Add(res)
		assert.NoError(t, err)
	}

	// Test ListIndexFuncValues method
	values := store.ListIndexFuncValues("by-status")
	assert.Len(t, values, 2)
	assert.Contains(t, values, "active")
	assert.Contains(t, values, "inactive")
}
