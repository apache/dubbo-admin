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

package mysql

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"

	storecfg "github.com/apache/dubbo-admin/pkg/config/store"
	"gorm.io/gorm"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/store/dbcommon"
)

func init() {
	store.RegisterFactory(&mysqlStoreFactory{})
}

type mysqlStoreFactory struct{}

var _ store.Factory = &mysqlStoreFactory{}

func (f *mysqlStoreFactory) Support(s storecfg.Type) bool {
	return s == storecfg.MySQL
}

func (f *mysqlStoreFactory) New(kind model.ResourceKind, cfg *storecfg.Config) (store.ManagedResourceStore, error) {
	return NewMySQLStore(kind, cfg.Address)
}

type mysqlStore struct {
	pool        *dbcommon.ConnectionPool
	kind        model.ResourceKind
	address     string
	indexers    cache.Indexers
	indexerLock sync.RWMutex
	// In-memory index: map[indexName]map[indexedValue]set[resourceKey]
	indices     map[string]map[string]map[string]struct{}
	indicesLock sync.RWMutex
	stopCh      chan struct{}
}

var _ store.ManagedResourceStore = &mysqlStore{}

func NewMySQLStore(kind model.ResourceKind, address string) (store.ManagedResourceStore, error) {
	return &mysqlStore{
		kind:     kind,
		address:  address,
		indexers: cache.Indexers{},
		indices:  make(map[string]map[string]map[string]struct{}),
		stopCh:   make(chan struct{}),
	}, nil
}

func (ms *mysqlStore) Init(_ runtime.BuilderContext) error {
	// Get or create MySQL connection pool
	pool, err := GetOrCreateMySQLPool(ms.address, dbcommon.DefaultConnectionPoolConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize mysql connection pool: %w", err)
	}
	ms.pool = pool

	// Perform table migration
	db := ms.pool.GetDB()
	modelForMigration := &dbcommon.ResourceModel{ResourceKind: ms.kind.ToString()}
	if err := db.AutoMigrate(modelForMigration); err != nil {
		return fmt.Errorf("failed to migrate schema for %s: %w", ms.kind.ToString(), err)
	}

	logger.Infof("MySQL store initialized for resource kind: %s", ms.kind.ToString())
	return nil
}

func (ms *mysqlStore) Start(_ runtime.Runtime, stopCh <-chan struct{}) error {
	logger.Infof("MySQL store started for resource kind: %s", ms.kind.ToString())

	// Monitor stop channel for graceful shutdown in a goroutine
	go func() {
		<-stopCh
		logger.Infof("MySQL store for %s received stop signal, initiating graceful shutdown", ms.kind.ToString())

		// Close the internal stop channel to signal any ongoing operations
		close(ms.stopCh)

		// Decrement the reference count and potentially close the connection pool
		if ms.pool != nil {
			if err := ms.pool.Close(); err != nil {
				logger.Errorf("Failed to close MySQL connection pool for %s: %v", ms.kind.ToString(), err)
			} else {
				logger.Infof("MySQL store for %s shutdown completed", ms.kind.ToString())
			}
		}
	}()

	return nil
}

func (ms *mysqlStore) Add(obj interface{}) error {
	resource, ok := obj.(model.Resource)
	if !ok {
		return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
	}

	if resource.ResourceKind() != ms.kind {
		return fmt.Errorf("resource kind mismatch: expected %s, got %s", ms.kind, resource.ResourceKind())
	}

	var count int64
	db := ms.pool.GetDB()
	err := db.Model(&dbcommon.ResourceModel{}).
		Where("resource_key = ?", resource.ResourceKey()).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return store.ErrorResourceAlreadyExists(
			resource.ResourceKind().ToString(),
			resource.ResourceMeta().Name,
			resource.MeshName(),
		)
	}

	m, err := dbcommon.FromResource(resource)
	if err != nil {
		return err
	}

	if err := db.Create(m).Error; err != nil {
		return err
	}

	// Update indices after successful DB operation
	ms.updateIndicesForResource(resource, nil)

	return nil
}

func (ms *mysqlStore) Update(obj interface{}) error {
	resource, ok := obj.(model.Resource)
	if !ok {
		return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
	}

	if resource.ResourceKind() != ms.kind {
		return fmt.Errorf("resource kind mismatch: expected %s, got %s", ms.kind, resource.ResourceKind())
	}

	// Get old resource for index update
	oldResource, exists, err := ms.GetByKey(resource.ResourceKey())
	if err != nil {
		return err
	}
	if !exists {
		return store.ErrorResourceNotFound(
			resource.ResourceKind().ToString(),
			resource.ResourceMeta().Name,
			resource.MeshName(),
		)
	}

	m, err := dbcommon.FromResource(resource)
	if err != nil {
		return err
	}

	db := ms.pool.GetDB()
	result := db.Model(&dbcommon.ResourceModel{}).
		Where("resource_key = ?", resource.ResourceKey()).
		Updates(map[string]interface{}{
			"data":       m.Data,
			"updated_at": m.UpdatedAt,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return store.ErrorResourceNotFound(
			resource.ResourceKind().ToString(),
			resource.ResourceMeta().Name,
			resource.MeshName(),
		)
	}

	// Update indices: remove old and add new
	ms.updateIndicesForResource(resource, oldResource.(model.Resource))

	return nil
}

func (ms *mysqlStore) Delete(obj interface{}) error {
	resource, ok := obj.(model.Resource)
	if !ok {
		return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
	}

	db := ms.pool.GetDB()
	result := db.Where("resource_key = ?", resource.ResourceKey()).
		Delete(&dbcommon.ResourceModel{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return store.ErrorResourceNotFound(
			resource.ResourceKind().ToString(),
			resource.ResourceMeta().Name,
			resource.MeshName(),
		)
	}

	// Remove from indices
	ms.removeFromIndices(resource)

	return nil
}

func (ms *mysqlStore) List() []interface{} {
	var models []dbcommon.ResourceModel
	db := ms.pool.GetDB()
	if err := db.Where("resource_kind = ?", ms.kind.ToString()).Find(&models).Error; err != nil {
		logger.Errorf("failed to list resources: %v", err)
		return []interface{}{}
	}

	result := make([]interface{}, 0, len(models))
	for _, m := range models {
		resource, err := m.ToResource()
		if err != nil {
			logger.Errorf("failed to deserialize resource: %v", err)
			continue
		}
		result = append(result, resource)
	}
	return result
}

func (ms *mysqlStore) ListKeys() []string {
	var keys []string
	db := ms.pool.GetDB()
	db.Model(&dbcommon.ResourceModel{}).
		Where("resource_kind = ?", ms.kind.ToString()).
		Pluck("resource_key", &keys)
	return keys
}

func (ms *mysqlStore) Get(obj interface{}) (item interface{}, exists bool, err error) {
	resource, ok := obj.(model.Resource)
	if !ok {
		return nil, false, bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
	}
	return ms.GetByKey(resource.ResourceKey())
}

func (ms *mysqlStore) GetByKey(key string) (item interface{}, exists bool, err error) {
	var m dbcommon.ResourceModel
	db := ms.pool.GetDB()
	result := db.Where("resource_key = ? AND resource_kind = ?", key, ms.kind.ToString()).
		First(&m)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, result.Error
	}

	resource, err := m.ToResource()
	if err != nil {
		return nil, false, err
	}

	return resource, true, nil
}

func (ms *mysqlStore) Replace(list []interface{}, _ string) error {
	db := ms.pool.GetDB()
	return db.Transaction(func(tx *gorm.DB) error {
		// Delete all existing records for this resource kind
		if err := tx.Where("resource_kind = ?", ms.kind.ToString()).Delete(&dbcommon.ResourceModel{}).Error; err != nil {
			return err
		}

		// Clear all indices
		ms.clearIndices()

		// Return early if list is empty
		if len(list) == 0 {
			return nil
		}

		// Convert all resources to dbcommon.ResourceModel
		models := make([]*dbcommon.ResourceModel, 0, len(list))
		resources := make([]model.Resource, 0, len(list))
		for _, obj := range list {
			resource, ok := obj.(model.Resource)
			if !ok {
				return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
			}

			m, err := dbcommon.FromResource(resource)
			if err != nil {
				return err
			}
			models = append(models, m)
			resources = append(resources, resource)
		}

		// Batch insert all models at once
		// GORM will automatically split into multiple batches if needed
		if err := tx.CreateInBatches(models, 100).Error; err != nil {
			return err
		}

		// Rebuild indices for all resources
		for _, resource := range resources {
			ms.updateIndicesForResource(resource, nil)
		}

		return nil
	})
}

func (ms *mysqlStore) Resync() error {
	return nil
}

func (ms *mysqlStore) Index(indexName string, obj interface{}) ([]interface{}, error) {
	ms.indexerLock.RLock()
	indexFunc, exists := ms.indexers[indexName]
	ms.indexerLock.RUnlock()

	if !exists {
		return nil, fmt.Errorf("index %s does not exist", indexName)
	}

	indexValues, err := indexFunc(obj)
	if err != nil {
		return nil, err
	}

	if len(indexValues) == 0 {
		return []interface{}{}, nil
	}

	return ms.findByIndex(indexName, indexValues[0])
}

func (ms *mysqlStore) IndexKeys(indexName, indexedValue string) ([]string, error) {
	ms.indexerLock.RLock()
	_, exists := ms.indexers[indexName]
	ms.indexerLock.RUnlock()

	if !exists {
		return nil, fmt.Errorf("index %s does not exist", indexName)
	}

	resources, err := ms.findByIndex(indexName, indexedValue)
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(resources))
	for _, obj := range resources {
		if resource, ok := obj.(model.Resource); ok {
			keys = append(keys, resource.ResourceKey())
		}
	}

	return keys, nil
}

func (ms *mysqlStore) ListIndexFuncValues(indexName string) []string {
	ms.indexerLock.RLock()
	_, exists := ms.indexers[indexName]
	ms.indexerLock.RUnlock()

	if !exists {
		return []string{}
	}

	resources := ms.List()
	valueSet := make(map[string]struct{})

	for _, obj := range resources {
		if indexFunc, ok := ms.indexers[indexName]; ok {
			values, err := indexFunc(obj)
			if err == nil {
				for _, value := range values {
					valueSet[value] = struct{}{}
				}
			}
		}
	}

	result := make([]string, 0, len(valueSet))
	for value := range valueSet {
		result = append(result, value)
	}

	return result
}

func (ms *mysqlStore) ByIndex(indexName, indexedValue string) ([]interface{}, error) {
	ms.indexerLock.RLock()
	_, exists := ms.indexers[indexName]
	ms.indexerLock.RUnlock()

	if !exists {
		return nil, fmt.Errorf("index %s does not exist", indexName)
	}

	return ms.findByIndex(indexName, indexedValue)
}

func (ms *mysqlStore) GetIndexers() cache.Indexers {
	ms.indexerLock.RLock()
	defer ms.indexerLock.RUnlock()

	result := make(cache.Indexers, len(ms.indexers))
	for k, v := range ms.indexers {
		result[k] = v
	}
	return result
}

func (ms *mysqlStore) AddIndexers(newIndexers cache.Indexers) error {
	ms.indexerLock.Lock()
	defer ms.indexerLock.Unlock()

	for name, indexFunc := range newIndexers {
		if _, exists := ms.indexers[name]; exists {
			return fmt.Errorf("indexer %s already exists", name)
		}
		ms.indexers[name] = indexFunc
	}

	return nil
}

func (ms *mysqlStore) GetByKeys(keys []string) ([]model.Resource, error) {
	if len(keys) == 0 {
		return []model.Resource{}, nil
	}

	var models []dbcommon.ResourceModel
	db := ms.pool.GetDB()
	err := db.Where("resource_key IN ? AND resource_kind = ?", keys, ms.kind.ToString()).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	resources := make([]model.Resource, 0, len(models))
	for _, m := range models {
		resource, err := m.ToResource()
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (ms *mysqlStore) ListByIndexes(indexes map[string]string) ([]model.Resource, error) {
	keys, err := ms.getKeysByIndexes(indexes)
	if err != nil {
		return nil, err
	}

	resources, err := ms.GetByKeys(keys)
	if err != nil {
		return nil, err
	}

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].ResourceKey() < resources[j].ResourceKey()
	})

	return resources, nil
}

func (ms *mysqlStore) PageListByIndexes(indexes map[string]string, pq model.PageReq) (*model.PageData[model.Resource], error) {
	keys, err := ms.getKeysByIndexes(indexes)
	if err != nil {
		return nil, err
	}

	sort.Strings(keys)
	total := len(keys)

	if pq.PageOffset >= total {
		return model.NewPageData(total, pq.PageOffset, pq.PageSize, []model.Resource{}), nil
	}

	end := pq.PageOffset + pq.PageSize
	if end > total {
		end = total
	}

	pageKeys := keys[pq.PageOffset:end]
	resources, err := ms.GetByKeys(pageKeys)
	if err != nil {
		return nil, err
	}

	return model.NewPageData(total, pq.PageOffset, pq.PageSize, resources), nil
}

func (ms *mysqlStore) findByIndex(indexName, indexedValue string) ([]interface{}, error) {
	ms.indexerLock.RLock()
	_, indexExists := ms.indexers[indexName]
	ms.indexerLock.RUnlock()

	if !indexExists {
		return nil, fmt.Errorf("index %s does not exist", indexName)
	}

	// Get resource keys from in-memory index
	ms.indicesLock.RLock()
	var keys []string
	if ms.indices[indexName] != nil && ms.indices[indexName][indexedValue] != nil {
		keys = make([]string, 0, len(ms.indices[indexName][indexedValue]))
		for key := range ms.indices[indexName][indexedValue] {
			keys = append(keys, key)
		}
	}
	ms.indicesLock.RUnlock()

	if len(keys) == 0 {
		return []interface{}{}, nil
	}

	// Fetch resources from DB by keys
	resources, err := ms.GetByKeys(keys)
	if err != nil {
		return nil, err
	}

	// Convert to []interface{}
	result := make([]interface{}, len(resources))
	for i, resource := range resources {
		result[i] = resource
	}

	return result, nil
}

func (ms *mysqlStore) getKeysByIndexes(indexes map[string]string) ([]string, error) {
	if len(indexes) == 0 {
		return ms.ListKeys(), nil
	}

	var keySet map[string]struct{}
	first := true

	for indexName, indexValue := range indexes {
		keys, err := ms.IndexKeys(indexName, indexValue)
		if err != nil {
			return nil, err
		}

		if first {
			keySet = make(map[string]struct{}, len(keys))
			for _, key := range keys {
				keySet[key] = struct{}{}
			}
			first = false
		} else {
			nextSet := make(map[string]struct{}, len(keys))
			for _, key := range keys {
				if _, exists := keySet[key]; exists {
					nextSet[key] = struct{}{}
				}
			}
			keySet = nextSet
		}
	}

	result := make([]string, 0, len(keySet))
	for key := range keySet {
		result = append(result, key)
	}

	return result, nil
}

// updateIndicesForResource updates in-memory indices when a resource is added or updated
// If oldResource is nil, it means this is an add operation (only add to indices)
// If oldResource is not nil, it means this is an update operation (remove old, add new)
func (ms *mysqlStore) updateIndicesForResource(newResource model.Resource, oldResource model.Resource) {
	ms.indexerLock.RLock()
	indexers := ms.indexers
	ms.indexerLock.RUnlock()

	ms.indicesLock.Lock()
	defer ms.indicesLock.Unlock()

	// Remove old resource from indices if this is an update
	if oldResource != nil {
		for indexName, indexFunc := range indexers {
			oldValues, err := indexFunc(oldResource)
			if err != nil {
				continue
			}
			for _, oldValue := range oldValues {
				if ms.indices[indexName] != nil && ms.indices[indexName][oldValue] != nil {
					delete(ms.indices[indexName][oldValue], oldResource.ResourceKey())
					// Clean up empty maps
					if len(ms.indices[indexName][oldValue]) == 0 {
						delete(ms.indices[indexName], oldValue)
					}
				}
			}
		}
	}

	// Add new resource to indices
	for indexName, indexFunc := range indexers {
		newValues, err := indexFunc(newResource)
		if err != nil {
			continue
		}

		// Ensure index exists
		if ms.indices[indexName] == nil {
			ms.indices[indexName] = make(map[string]map[string]struct{})
		}

		for _, newValue := range newValues {
			// Ensure value map exists
			if ms.indices[indexName][newValue] == nil {
				ms.indices[indexName][newValue] = make(map[string]struct{})
			}
			// Add resource key to the set
			ms.indices[indexName][newValue][newResource.ResourceKey()] = struct{}{}
		}
	}
}

// removeFromIndices removes a resource from all in-memory indices
func (ms *mysqlStore) removeFromIndices(resource model.Resource) {
	ms.indexerLock.RLock()
	indexers := ms.indexers
	ms.indexerLock.RUnlock()

	ms.indicesLock.Lock()
	defer ms.indicesLock.Unlock()

	for indexName, indexFunc := range indexers {
		values, err := indexFunc(resource)
		if err != nil {
			continue
		}

		for _, value := range values {
			if ms.indices[indexName] != nil && ms.indices[indexName][value] != nil {
				delete(ms.indices[indexName][value], resource.ResourceKey())
				// Clean up empty maps
				if len(ms.indices[indexName][value]) == 0 {
					delete(ms.indices[indexName], value)
				}
			}
		}
	}
}

// clearIndices clears all in-memory indices
func (ms *mysqlStore) clearIndices() {
	ms.indicesLock.Lock()
	defer ms.indicesLock.Unlock()
	ms.indices = make(map[string]map[string]map[string]struct{})
}
