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

package postgres

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
	store.RegisterFactory(&postgresStoreFactory{})
}

type postgresStoreFactory struct{}

var _ store.Factory = &postgresStoreFactory{}

func (f *postgresStoreFactory) Support(s storecfg.Type) bool {
	return s == storecfg.Postgres
}

func (f *postgresStoreFactory) New(kind model.ResourceKind, cfg *storecfg.Config) (store.ManagedResourceStore, error) {
	return NewPostgresStore(kind, cfg.Address)
}

type postgresStore struct {
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

var _ store.ManagedResourceStore = &postgresStore{}

func NewPostgresStore(kind model.ResourceKind, address string) (store.ManagedResourceStore, error) {
	return &postgresStore{
		kind:     kind,
		address:  address,
		indexers: cache.Indexers{},
		indices:  make(map[string]map[string]map[string]struct{}),
		stopCh:   make(chan struct{}),
	}, nil
}

func (ps *postgresStore) Init(_ runtime.BuilderContext) error {
	// Get or create PostgreSQL connection pool
	pool, err := GetOrCreatePostgresPool(ps.address, dbcommon.DefaultConnectionPoolConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize postgres connection pool: %w", err)
	}
	ps.pool = pool

	// Perform table migration
	db := ps.pool.GetDB()
	modelForMigration := &dbcommon.ResourceModel{ResourceKind: ps.kind.ToString()}
	if err := db.AutoMigrate(modelForMigration); err != nil {
		return fmt.Errorf("failed to migrate schema for %s: %w", ps.kind.ToString(), err)
	}

	logger.Infof("PostgreSQL store initialized for resource kind: %s", ps.kind.ToString())
	return nil
}

func (ps *postgresStore) Start(_ runtime.Runtime, stopCh <-chan struct{}) error {
	logger.Infof("PostgreSQL store started for resource kind: %s", ps.kind.ToString())

	// Monitor stop channel for graceful shutdown in a goroutine
	go func() {
		<-stopCh
		logger.Infof("PostgreSQL store for %s received stop signal, initiating graceful shutdown", ps.kind.ToString())

		// Close the internal stop channel to signal any ongoing operations
		close(ps.stopCh)

		// Decrement the reference count and potentially close the connection pool
		if ps.pool != nil {
			if err := ps.pool.Close(); err != nil {
				logger.Errorf("Failed to close PostgreSQL connection pool for %s: %v", ps.kind.ToString(), err)
			} else {
				logger.Infof("PostgreSQL store for %s shutdown completed", ps.kind.ToString())
			}
		}
	}()

	return nil
}

func (ps *postgresStore) Add(obj interface{}) error {
	resource, ok := obj.(model.Resource)
	if !ok {
		return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
	}

	if resource.ResourceKind() != ps.kind {
		return fmt.Errorf("resource kind mismatch: expected %s, got %s", ps.kind, resource.ResourceKind())
	}

	var count int64
	db := ps.pool.GetDB()
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
	ps.updateIndicesForResource(resource, nil)

	return nil
}

func (ps *postgresStore) Update(obj interface{}) error {
	resource, ok := obj.(model.Resource)
	if !ok {
		return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
	}

	if resource.ResourceKind() != ps.kind {
		return fmt.Errorf("resource kind mismatch: expected %s, got %s", ps.kind, resource.ResourceKind())
	}

	// Get old resource for index update
	oldResource, exists, err := ps.GetByKey(resource.ResourceKey())
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

	db := ps.pool.GetDB()
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
	ps.updateIndicesForResource(resource, oldResource.(model.Resource))

	return nil
}

func (ps *postgresStore) Delete(obj interface{}) error {
	resource, ok := obj.(model.Resource)
	if !ok {
		return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
	}

	db := ps.pool.GetDB()
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
	ps.removeFromIndices(resource)

	return nil
}

func (ps *postgresStore) List() []interface{} {
	var models []dbcommon.ResourceModel
	db := ps.pool.GetDB()
	if err := db.Where("resource_kind = ?", ps.kind.ToString()).Find(&models).Error; err != nil {
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

func (ps *postgresStore) ListKeys() []string {
	var keys []string
	db := ps.pool.GetDB()
	db.Model(&dbcommon.ResourceModel{}).
		Where("resource_kind = ?", ps.kind.ToString()).
		Pluck("resource_key", &keys)
	return keys
}

func (ps *postgresStore) Get(obj interface{}) (item interface{}, exists bool, err error) {
	resource, ok := obj.(model.Resource)
	if !ok {
		return nil, false, bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
	}
	return ps.GetByKey(resource.ResourceKey())
}

func (ps *postgresStore) GetByKey(key string) (item interface{}, exists bool, err error) {
	var m dbcommon.ResourceModel
	db := ps.pool.GetDB()
	result := db.Where("resource_key = ? AND resource_kind = ?", key, ps.kind.ToString()).
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

func (ps *postgresStore) Replace(list []interface{}, _ string) error {
	db := ps.pool.GetDB()
	return db.Transaction(func(tx *gorm.DB) error {
		// Delete all existing records for this resource kind
		if err := tx.Where("resource_kind = ?", ps.kind.ToString()).Delete(&dbcommon.ResourceModel{}).Error; err != nil {
			return err
		}

		// Clear all indices
		ps.clearIndices()

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
			ps.updateIndicesForResource(resource, nil)
		}

		return nil
	})
}

func (ps *postgresStore) Resync() error {
	return nil
}

func (ps *postgresStore) Index(indexName string, obj interface{}) ([]interface{}, error) {
	ps.indexerLock.RLock()
	indexFunc, exists := ps.indexers[indexName]
	ps.indexerLock.RUnlock()

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

	return ps.findByIndex(indexName, indexValues[0])
}

func (ps *postgresStore) IndexKeys(indexName, indexedValue string) ([]string, error) {
	ps.indexerLock.RLock()
	_, exists := ps.indexers[indexName]
	ps.indexerLock.RUnlock()

	if !exists {
		return nil, fmt.Errorf("index %s does not exist", indexName)
	}

	resources, err := ps.findByIndex(indexName, indexedValue)
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

func (ps *postgresStore) ListIndexFuncValues(indexName string) []string {
	ps.indexerLock.RLock()
	_, exists := ps.indexers[indexName]
	ps.indexerLock.RUnlock()

	if !exists {
		return []string{}
	}

	resources := ps.List()
	valueSet := make(map[string]struct{})

	for _, obj := range resources {
		if indexFunc, ok := ps.indexers[indexName]; ok {
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

func (ps *postgresStore) ByIndex(indexName, indexedValue string) ([]interface{}, error) {
	ps.indexerLock.RLock()
	_, exists := ps.indexers[indexName]
	ps.indexerLock.RUnlock()

	if !exists {
		return nil, fmt.Errorf("index %s does not exist", indexName)
	}

	return ps.findByIndex(indexName, indexedValue)
}

func (ps *postgresStore) GetIndexers() cache.Indexers {
	ps.indexerLock.RLock()
	defer ps.indexerLock.RUnlock()

	result := make(cache.Indexers, len(ps.indexers))
	for k, v := range ps.indexers {
		result[k] = v
	}
	return result
}

func (ps *postgresStore) AddIndexers(newIndexers cache.Indexers) error {
	ps.indexerLock.Lock()
	defer ps.indexerLock.Unlock()

	for name, indexFunc := range newIndexers {
		if _, exists := ps.indexers[name]; exists {
			return fmt.Errorf("indexer %s already exists", name)
		}
		ps.indexers[name] = indexFunc
	}

	return nil
}

func (ps *postgresStore) GetByKeys(keys []string) ([]model.Resource, error) {
	if len(keys) == 0 {
		return []model.Resource{}, nil
	}

	var models []dbcommon.ResourceModel
	db := ps.pool.GetDB()
	err := db.Where("resource_key IN ? AND resource_kind = ?", keys, ps.kind.ToString()).
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

func (ps *postgresStore) ListByIndexes(indexes map[string]string) ([]model.Resource, error) {
	keys, err := ps.getKeysByIndexes(indexes)
	if err != nil {
		return nil, err
	}

	resources, err := ps.GetByKeys(keys)
	if err != nil {
		return nil, err
	}

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].ResourceKey() < resources[j].ResourceKey()
	})

	return resources, nil
}

func (ps *postgresStore) PageListByIndexes(indexes map[string]string, pq model.PageReq) (*model.PageData[model.Resource], error) {
	keys, err := ps.getKeysByIndexes(indexes)
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
	resources, err := ps.GetByKeys(pageKeys)
	if err != nil {
		return nil, err
	}

	return model.NewPageData(total, pq.PageOffset, pq.PageSize, resources), nil
}

func (ps *postgresStore) findByIndex(indexName, indexedValue string) ([]interface{}, error) {
	ps.indexerLock.RLock()
	_, indexExists := ps.indexers[indexName]
	ps.indexerLock.RUnlock()

	if !indexExists {
		return nil, fmt.Errorf("index %s does not exist", indexName)
	}

	// Get resource keys from in-memory index
	ps.indicesLock.RLock()
	var keys []string
	if ps.indices[indexName] != nil && ps.indices[indexName][indexedValue] != nil {
		keys = make([]string, 0, len(ps.indices[indexName][indexedValue]))
		for key := range ps.indices[indexName][indexedValue] {
			keys = append(keys, key)
		}
	}
	ps.indicesLock.RUnlock()

	if len(keys) == 0 {
		return []interface{}{}, nil
	}

	// Fetch resources from DB by keys
	resources, err := ps.GetByKeys(keys)
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

func (ps *postgresStore) getKeysByIndexes(indexes map[string]string) ([]string, error) {
	if len(indexes) == 0 {
		return ps.ListKeys(), nil
	}

	var keySet map[string]struct{}
	first := true

	for indexName, indexValue := range indexes {
		keys, err := ps.IndexKeys(indexName, indexValue)
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
func (ps *postgresStore) updateIndicesForResource(newResource model.Resource, oldResource model.Resource) {
	ps.indexerLock.RLock()
	indexers := ps.indexers
	ps.indexerLock.RUnlock()

	ps.indicesLock.Lock()
	defer ps.indicesLock.Unlock()

	// Remove old resource from indices if this is an update
	if oldResource != nil {
		for indexName, indexFunc := range indexers {
			oldValues, err := indexFunc(oldResource)
			if err != nil {
				continue
			}
			for _, oldValue := range oldValues {
				if ps.indices[indexName] != nil && ps.indices[indexName][oldValue] != nil {
					delete(ps.indices[indexName][oldValue], oldResource.ResourceKey())
					// Clean up empty maps
					if len(ps.indices[indexName][oldValue]) == 0 {
						delete(ps.indices[indexName], oldValue)
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
		if ps.indices[indexName] == nil {
			ps.indices[indexName] = make(map[string]map[string]struct{})
		}

		for _, newValue := range newValues {
			// Ensure value map exists
			if ps.indices[indexName][newValue] == nil {
				ps.indices[indexName][newValue] = make(map[string]struct{})
			}
			// Add resource key to the set
			ps.indices[indexName][newValue][newResource.ResourceKey()] = struct{}{}
		}
	}
}

// removeFromIndices removes a resource from all in-memory indices
func (ps *postgresStore) removeFromIndices(resource model.Resource) {
	ps.indexerLock.RLock()
	indexers := ps.indexers
	ps.indexerLock.RUnlock()

	ps.indicesLock.Lock()
	defer ps.indicesLock.Unlock()

	for indexName, indexFunc := range indexers {
		values, err := indexFunc(resource)
		if err != nil {
			continue
		}

		for _, value := range values {
			if ps.indices[indexName] != nil && ps.indices[indexName][value] != nil {
				delete(ps.indices[indexName][value], resource.ResourceKey())
				// Clean up empty maps
				if len(ps.indices[indexName][value]) == 0 {
					delete(ps.indices[indexName], value)
				}
			}
		}
	}
}

// clearIndices clears all in-memory indices
func (ps *postgresStore) clearIndices() {
	ps.indicesLock.Lock()
	defer ps.indicesLock.Unlock()
	ps.indices = make(map[string]map[string]map[string]struct{})
}
