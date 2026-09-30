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
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/armon/go-radix"
	set "github.com/duke-git/lancet/v2/datastructure/set"
	"github.com/duke-git/lancet/v2/slice"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

type resourceStore struct {
	rk          coremodel.ResourceKind
	storeProxy  cache.Indexer
	prefixTrees map[string]*radix.Tree
	mutationMu  sync.Mutex
	treesMu     sync.RWMutex
}

var _ store.ManagedResourceStore = &resourceStore{}
var _ store.ConditionalResourceStore = &resourceStore{}

func copyVersionedResource(obj interface{}) interface{} {
	if resource, ok := obj.(store.VersionedResource); ok {
		return resource.DeepCopyObject()
	}
	return obj
}

func NewMemoryResourceStore(rk coremodel.ResourceKind) store.ManagedResourceStore {
	return &resourceStore{rk: rk}
}

func (rs *resourceStore) Init(_ runtime.BuilderContext) error {
	indexers := index.IndexersRegistry().Indexers(rs.rk)
	rs.storeProxy = cache.NewIndexer(
		func(obj interface{}) (string, error) {
			r, ok := obj.(coremodel.Resource)
			if !ok {
				return "", bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
			}
			return r.ResourceKey(), nil
		},
		indexers,
	)
	// Initialize RadixTree for each index for prefix matching support
	rs.prefixTrees = make(map[string]*radix.Tree)
	for indexName := range indexers {
		rs.prefixTrees[indexName] = radix.New()
	}
	return nil
}

func (rs *resourceStore) Start(_ runtime.Runtime, _ <-chan struct{}) error {
	return nil
}

func (rs *resourceStore) Add(obj interface{}) error {
	rs.mutationMu.Lock()
	defer rs.mutationMu.Unlock()

	if resource, ok := obj.(store.VersionedResource); ok {
		candidate := resource.DeepCopyObject().(store.VersionedResource)
		if err := store.PrepareInitialVersion(candidate); err != nil {
			return err
		}
		if err := rs.add(candidate); err != nil {
			return err
		}
		resource.SetResourceVersion(candidate.ResourceMeta().ResourceVersion)
		return nil
	}
	return rs.add(obj)
}

func (rs *resourceStore) add(obj interface{}) error {
	if resource, ok := obj.(store.VersionedResource); ok {
		if _, exists, err := rs.storeProxy.GetByKey(resource.ResourceKey()); err != nil {
			return err
		} else if exists {
			return store.ErrorResourceAlreadyExists(resource.ResourceKind().ToString(), resource.ResourceMeta().Name, resource.ResourceMesh())
		}
	}
	if err := rs.storeProxy.Add(obj); err != nil {
		return err
	}
	r, ok := obj.(coremodel.Resource)
	if ok {
		rs.addToTrees(r)
	}
	return nil
}

func (rs *resourceStore) Update(obj interface{}) error {
	if _, ok := obj.(store.VersionedResource); ok {
		return &store.PreconditionError{Reason: "versioned resources require CompareAndSwap"}
	}
	rs.mutationMu.Lock()
	defer rs.mutationMu.Unlock()
	return rs.update(obj)
}

func (rs *resourceStore) update(obj interface{}) error {
	if resource, ok := obj.(store.VersionedResource); ok {
		obj = resource.DeepCopyObject()
	}
	r, ok := obj.(coremodel.Resource)
	var oldRes coremodel.Resource
	if ok {
		// Fetch old resource before mutating the store
		oldObj, exists, err := rs.storeProxy.Get(r)
		if exists && err == nil {
			oldRes, _ = oldObj.(coremodel.Resource)
		}
	}
	if err := rs.storeProxy.Update(obj); err != nil {
		return err
	}
	// Only mutate trees after a successful store update
	if ok {
		if oldRes != nil {
			rs.removeFromTrees(oldRes)
		}
		rs.addToTrees(r)
	}
	return nil
}

func (rs *resourceStore) Delete(obj interface{}) error {
	if _, ok := obj.(store.VersionedResource); ok {
		return &store.PreconditionError{Reason: "versioned resources require CompareAndDelete"}
	}
	rs.mutationMu.Lock()
	defer rs.mutationMu.Unlock()
	return rs.delete(obj)
}

func (rs *resourceStore) delete(obj interface{}) error {
	if err := rs.storeProxy.Delete(obj); err != nil {
		return err
	}
	if r, ok := obj.(coremodel.Resource); ok {
		rs.removeFromTrees(r)
	}
	return nil
}

func (rs *resourceStore) List() []interface{} {
	items := rs.storeProxy.List()
	for i, item := range items {
		items[i] = copyVersionedResource(item)
	}
	return items
}

func (rs *resourceStore) ListKeys() []string {
	return rs.storeProxy.ListKeys()
}

func (rs *resourceStore) Get(obj interface{}) (item interface{}, exists bool, err error) {
	item, exists, err = rs.storeProxy.Get(obj)
	return copyVersionedResource(item), exists, err
}

func (rs *resourceStore) GetByKey(key string) (item interface{}, exists bool, err error) {
	item, exists, err = rs.storeProxy.GetByKey(key)
	return copyVersionedResource(item), exists, err
}

func (rs *resourceStore) Replace(i []interface{}, s string) error {
	if store.IsVersionedResourceKind(rs.rk) {
		return &store.PreconditionError{Reason: "versioned resources require conditional mutations"}
	}
	rs.mutationMu.Lock()
	defer rs.mutationMu.Unlock()

	// Clear all trees before replace
	rs.treesMu.Lock()
	for indexName := range rs.prefixTrees {
		rs.prefixTrees[indexName] = radix.New()
	}
	rs.treesMu.Unlock()

	if err := rs.storeProxy.Replace(i, s); err != nil {
		return err
	}

	// Add all new resources to trees
	for _, obj := range i {
		r, ok := obj.(coremodel.Resource)
		if ok {
			rs.addToTrees(r)
		}
	}
	return nil
}

func (rs *resourceStore) CompareAndSwap(obj coremodel.Resource, expectedVersion string) error {
	rs.mutationMu.Lock()
	defer rs.mutationMu.Unlock()

	currentObj, exists, err := rs.storeProxy.GetByKey(obj.ResourceKey())
	if err != nil {
		return err
	}
	if !exists {
		return store.ErrorResourceNotFound(obj.ResourceKind().ToString(), obj.ResourceMeta().Name, obj.ResourceMesh())
	}
	current, ok := currentObj.(coremodel.Resource)
	if !ok {
		return bizerror.NewAssertionError("Resource", reflect.TypeOf(currentObj).Name())
	}
	if current.ResourceMeta().ResourceVersion != expectedVersion {
		return store.ErrorResourceConflict(obj.ResourceKind().ToString(), obj.ResourceMeta().Name, obj.ResourceMesh())
	}
	nextVersion, err := store.NextResourceVersion(obj, expectedVersion)
	if err != nil {
		return err
	}
	candidate, ok := obj.DeepCopyObject().(coremodel.Resource)
	if !ok {
		return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj.DeepCopyObject()).Name())
	}
	candidateVersioned, ok := candidate.(store.VersionedResource)
	if !ok {
		return &store.PreconditionError{Reason: "resource does not support conditional mutations"}
	}
	candidateVersioned.SetResourceVersion(nextVersion)
	if err := rs.update(candidate); err != nil {
		return err
	}
	obj.(store.VersionedResource).SetResourceVersion(nextVersion)
	return nil
}

func (rs *resourceStore) CompareAndDelete(obj coremodel.Resource, expectedVersion string) error {
	rs.mutationMu.Lock()
	defer rs.mutationMu.Unlock()

	currentObj, exists, err := rs.storeProxy.GetByKey(obj.ResourceKey())
	if err != nil {
		return err
	}
	if !exists {
		return store.ErrorResourceNotFound(obj.ResourceKind().ToString(), obj.ResourceMeta().Name, obj.ResourceMesh())
	}
	current, ok := currentObj.(coremodel.Resource)
	if !ok {
		return bizerror.NewAssertionError("Resource", reflect.TypeOf(currentObj).Name())
	}
	if current.ResourceMeta().ResourceVersion != expectedVersion {
		return store.ErrorResourceConflict(obj.ResourceKind().ToString(), obj.ResourceMeta().Name, obj.ResourceMesh())
	}
	if _, err := store.NextResourceVersion(obj, expectedVersion); err != nil {
		return err
	}
	return rs.delete(current)
}

func (rs *resourceStore) Resync() error {
	return rs.storeProxy.Resync()
}

func (rs *resourceStore) Index(indexName string, obj interface{}) ([]interface{}, error) {
	items, err := rs.storeProxy.Index(indexName, obj)
	for i, item := range items {
		items[i] = copyVersionedResource(item)
	}
	return items, err
}

func (rs *resourceStore) IndexKeys(indexName, indexedValue string) ([]string, error) {
	return rs.storeProxy.IndexKeys(indexName, indexedValue)
}

func (rs *resourceStore) ListIndexFuncValues(indexName string) []string {
	return rs.storeProxy.ListIndexFuncValues(indexName)
}

func (rs *resourceStore) ByIndex(indexName, indexedValue string) ([]interface{}, error) {
	items, err := rs.storeProxy.ByIndex(indexName, indexedValue)
	for i, item := range items {
		items[i] = copyVersionedResource(item)
	}
	return items, err
}

func (rs *resourceStore) GetIndexers() cache.Indexers {
	return rs.storeProxy.GetIndexers()
}

func (rs *resourceStore) AddIndexers(newIndexers cache.Indexers) error {
	rs.treesMu.Lock()
	defer rs.treesMu.Unlock()

	if err := rs.storeProxy.AddIndexers(newIndexers); err != nil {
		return err
	}

	// Add RadixTrees for new indexers
	for indexName := range newIndexers {
		if _, exists := rs.prefixTrees[indexName]; !exists {
			rs.prefixTrees[indexName] = radix.New()
		}
	}
	return nil
}

func (rs *resourceStore) GetByKeys(keys []string) ([]coremodel.Resource, error) {
	resources := make([]coremodel.Resource, 0)
	for _, key := range keys {
		r, exists, err := rs.storeProxy.GetByKey(key)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		res, ok := r.(coremodel.Resource)
		if !ok {
			return nil, bizerror.NewAssertionError("Resource", reflect.TypeOf(r).Name())
		}
		resources = append(resources, copyVersionedResource(res).(coremodel.Resource))
	}
	return resources, nil
}

func (rs *resourceStore) ListByIndexes(indexes []index.IndexCondition) ([]coremodel.Resource, error) {
	keys, err := rs.getKeysByIndexes(indexes)
	if err != nil {
		return nil, err
	}
	resources, err := rs.GetByKeys(keys)
	if err != nil {
		return nil, err
	}
	slice.SortBy(resources, func(r1 coremodel.Resource, r2 coremodel.Resource) bool {
		return r1.ResourceKey() < r2.ResourceKey()
	})
	return resources, nil
}

func (rs *resourceStore) PageListByIndexes(indexes []index.IndexCondition, pq coremodel.PageReq) (*coremodel.PageData[coremodel.Resource], error) {
	keys, err := rs.getKeysByIndexes(indexes)
	if err != nil {
		return nil, err
	}
	sort.Strings(keys)
	total := len(keys)
	resources := make([]coremodel.Resource, 0, pq.PageSize)
	for i := pq.PageOffset; len(resources) < pq.PageSize && i < total; i++ {
		r, exists, err := rs.storeProxy.GetByKey(keys[i])
		if err != nil {
			return nil, err
		}
		if !exists {
			total -= 1
			continue
		}
		res, ok := r.(coremodel.Resource)
		if !ok {
			return nil, bizerror.NewAssertionError("Resource", reflect.TypeOf(r).Name())
		}
		resources = append(resources, copyVersionedResource(res).(coremodel.Resource))
	}
	pageData := coremodel.NewPageData(total, pq.PageOffset, pq.PageSize, resources)
	return pageData, nil
}

func (rs *resourceStore) getKeysByIndexes(indexes []index.IndexCondition) ([]string, error) {
	if len(indexes) == 0 {
		return []string{}, nil
	}
	keySet := set.New[string]()
	first := true
	for _, condition := range indexes {
		var keys []string
		var err error

		switch condition.Operator {
		case index.Equals:
			keys, err = rs.storeProxy.IndexKeys(condition.IndexName, condition.Value)
		case index.HasPrefix:
			keys, err = rs.getKeysByPrefix(condition.IndexName, condition.Value)
		default:
			return nil, bizerror.New(bizerror.InvalidArgument, "operator not yet supported: "+string(condition.Operator))
		}

		if err != nil {
			return nil, err
		}

		if first {
			keySet = set.FromSlice(keys)
			first = false
		} else {
			nextSet := set.FromSlice(keys)
			keySet = keySet.Intersection(nextSet)
		}
	}
	return keySet.ToSlice(), nil
}

// addToTrees adds a resource to all relevant RadixTrees for prefix matching
func (rs *resourceStore) addToTrees(resource coremodel.Resource) {
	rs.treesMu.Lock()
	defer rs.treesMu.Unlock()

	// Get indexers from storeProxy, not from global registry
	// This ensures we include both init-time and dynamically-added indexers
	indexers := rs.storeProxy.GetIndexers()
	for indexName, indexFunc := range indexers {
		values, err := indexFunc(resource)
		if err != nil {
			continue
		}
		tree, ok := rs.prefixTrees[indexName]
		if !ok || tree == nil {
			continue
		}
		for _, v := range values {
			// Key format: "indexValue/resourceKey"
			key := v + "/" + resource.ResourceKey()
			tree.Insert(key, struct{}{})
		}
	}
}

// removeFromTrees removes a resource from all relevant RadixTrees
func (rs *resourceStore) removeFromTrees(resource coremodel.Resource) {
	rs.treesMu.Lock()
	defer rs.treesMu.Unlock()

	// Get indexers from storeProxy, not from global registry
	// This ensures we include both init-time and dynamically-added indexers
	indexers := rs.storeProxy.GetIndexers()
	for indexName, indexFunc := range indexers {
		values, err := indexFunc(resource)
		if err != nil {
			continue
		}
		tree, ok := rs.prefixTrees[indexName]
		if !ok || tree == nil {
			continue
		}
		for _, v := range values {
			// Key format: "indexValue/resourceKey"
			key := v + "/" + resource.ResourceKey()
			tree.Delete(key)
		}
	}
}

// getKeysByPrefix retrieves resource keys by prefix match using RadixTree
func (rs *resourceStore) getKeysByPrefix(indexName, prefix string) ([]string, error) {
	rs.treesMu.RLock()
	defer rs.treesMu.RUnlock()

	tree, ok := rs.prefixTrees[indexName]
	if !ok {
		return nil, fmt.Errorf("index %s does not exist", indexName)
	}

	var keys []string
	tree.WalkPrefix(prefix, func(k string, v interface{}) bool {
		// Key format: "indexValue/resourceKey"
		// Key format: "indexValue/resourceKey"
		// Use Index (first "/") because resourceKey itself contains "/" (mesh/name)
		idx := strings.Index(k, "/")
		if idx >= 0 && idx < len(k)-1 {
			keys = append(keys, k[idx+1:])
		}
		return false // Continue walking
	})

	return keys, nil
}
