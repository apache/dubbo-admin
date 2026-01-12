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
	"reflect"
	"sort"

	set "github.com/duke-git/lancet/v2/datastructure/set"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/util/slices"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
	"github.com/apache/dubbo-admin/pkg/store/indexer"
)

type resourceStore struct {
	rk         coremodel.ResourceKind
	storeProxy cache.Indexer
	indices    *indexer.IndexImpl
}

var _ store.ManagedResourceStore = &resourceStore{}

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

	// Create IndexWrapper with MemoryIndex for prefix query support
	memIndex := indexer.NewMemoryIndex()
	rs.indices = indexer.NewIndexImpl(memIndex)
	if err := rs.indices.AddIndexers(indexers); err != nil {
		return err
	}

	return nil
}

func (rs *resourceStore) Start(_ runtime.Runtime, _ <-chan struct{}) error {
	return nil
}

func (rs *resourceStore) Add(obj interface{}) error {
	if err := rs.storeProxy.Add(obj); err != nil {
		return err
	}

	// Update indices
	if resource, ok := obj.(coremodel.Resource); ok {
		rs.indices.UpdateResource(resource, nil)
	}

	return nil
}

func (rs *resourceStore) Update(obj interface{}) error {
	resource, ok := obj.(coremodel.Resource)
	if !ok {
		return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
	}

	// Get old resource for index update
	oldObj, exists, err := rs.storeProxy.GetByKey(resource.ResourceKey())
	if err != nil {
		return err
	}

	var oldResource coremodel.Resource
	if exists {
		oldResource, _ = oldObj.(coremodel.Resource)
	}

	if err := rs.storeProxy.Update(obj); err != nil {
		return err
	}

	// Update indices
	rs.indices.UpdateResource(resource, oldResource)

	return nil
}

func (rs *resourceStore) Delete(obj interface{}) error {
	if err := rs.storeProxy.Delete(obj); err != nil {
		return err
	}

	// Remove from indices
	if resource, ok := obj.(coremodel.Resource); ok {
		rs.indices.RemoveResource(resource)
	}

	return nil
}

func (rs *resourceStore) List() []interface{} {
	return rs.storeProxy.List()
}

func (rs *resourceStore) ListKeys() []string {
	return rs.storeProxy.ListKeys()
}

func (rs *resourceStore) Get(obj interface{}) (item interface{}, exists bool, err error) {
	return rs.storeProxy.Get(obj)
}

func (rs *resourceStore) GetByKey(key string) (item interface{}, exists bool, err error) {
	return rs.storeProxy.GetByKey(key)
}

func (rs *resourceStore) Replace(i []interface{}, s string) error {
	return rs.storeProxy.Replace(i, s)
}

func (rs *resourceStore) Resync() error {
	return rs.storeProxy.Resync()
}

func (rs *resourceStore) Index(indexName string, obj interface{}) ([]interface{}, error) {
	return rs.storeProxy.Index(indexName, obj)
}

func (rs *resourceStore) IndexKeys(indexName, indexedValue string) ([]string, error) {
	return rs.storeProxy.IndexKeys(indexName, indexedValue)
}

func (rs *resourceStore) ListIndexFuncValues(indexName string) []string {
	return rs.storeProxy.ListIndexFuncValues(indexName)
}

func (rs *resourceStore) ByIndex(indexName, indexedValue string) ([]interface{}, error) {
	return rs.storeProxy.ByIndex(indexName, indexedValue)
}

func (rs *resourceStore) GetIndexers() cache.Indexers {
	return rs.storeProxy.GetIndexers()
}

func (rs *resourceStore) AddIndexers(newIndexers cache.Indexers) error {
	return rs.storeProxy.AddIndexers(newIndexers)
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
		resources = append(resources, res)
	}
	return resources, nil
}

func (rs *resourceStore) ListByIndexes(indexes map[string]string) ([]coremodel.Resource, error) {
	keys, err := rs.getKeysByIndexes(indexes)
	if err != nil {
		return nil, err
	}
	resources, err := rs.GetByKeys(keys)
	if err != nil {
		return nil, err
	}
	resources = slices.SortBy(resources, func(r coremodel.Resource) string {
		return r.ResourceKey()
	})
	return resources, nil
}

func (rs *resourceStore) PageListByIndexes(indexes map[string]string, pq coremodel.PageReq) (*coremodel.PageData[coremodel.Resource], error) {
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
		resources = append(resources, res)
	}
	pageData := coremodel.NewPageData(total, pq.PageOffset, pq.PageSize, resources)
	return pageData, nil
}

func (rs *resourceStore) getKeysByIndexes(indexes map[string]string) ([]string, error) {
	if len(indexes) == 0 {
		return []string{}, nil
	}
	keySet := set.New[string]()
	first := true
	for indexName, indexValue := range indexes {
		keys, err := rs.storeProxy.IndexKeys(indexName, indexValue)
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

// getKeysByIndexesWithPrefix gets resource keys by combining exact and prefix indexes
func (rs *resourceStore) getKeysByIndexesWithPrefix(indexes map[string]string, prefixIndexes map[string]string) ([]string, error) {
	if len(indexes) == 0 && len(prefixIndexes) == 0 {
		return []string{}, nil
	}

	keySet := set.New[string]()
	first := true

	// Process exact match indexes using IndexWrapper
	for indexName, indexValue := range indexes {
		query := indexer.NewQuery(indexName, indexer.OperatorEquals, indexValue)
		keys, err := rs.indices.Query(query)
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

	// Process prefix match indexes using IndexWrapper
	for indexName, prefix := range prefixIndexes {
		query := indexer.NewQuery(indexName, indexer.OperatorPrefix, prefix)
		keys, err := rs.indices.Query(query)
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

// ListByIndexesWithPrefix lists resources by indexes with prefix matching support
func (rs *resourceStore) ListByIndexesWithPrefix(indexes map[string]string, prefixIndexes map[string]string) ([]coremodel.Resource, error) {
	keys, err := rs.getKeysByIndexesWithPrefix(indexes, prefixIndexes)
	if err != nil {
		return nil, err
	}

	resources, err := rs.GetByKeys(keys)
	if err != nil {
		return nil, err
	}

	resources = slices.SortBy(resources, func(r coremodel.Resource) string {
		return r.ResourceKey()
	})

	return resources, nil
}

// PageListByIndexesWithPrefix lists resources by indexes with prefix matching support, pageable
func (rs *resourceStore) PageListByIndexesWithPrefix(indexes map[string]string, prefixIndexes map[string]string, pq coremodel.PageReq) (*coremodel.PageData[coremodel.Resource], error) {
	keys, err := rs.getKeysByIndexesWithPrefix(indexes, prefixIndexes)
	if err != nil {
		return nil, err
	}

	sort.Strings(keys)
	total := len(keys)

	if pq.PageOffset < 0 || pq.PageOffset > total {
		return nil, store.ErrorInvalidOffset
	}

	// Calculate page range
	end := pq.PageOffset + pq.PageSize
	if end > total {
		end = total
	}

	// Get only the keys for current page
	pageKeys := keys[pq.PageOffset:end]

	// Batch fetch resources for current page
	resources, err := rs.GetByKeys(pageKeys)
	if err != nil {
		return nil, err
	}

	return coremodel.NewPageData(total, pq.PageOffset, pq.PageSize, resources), nil
}
