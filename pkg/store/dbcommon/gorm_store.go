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
	"errors"
	"fmt"
	"reflect"
	"sort"

	"gorm.io/gorm"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

// GormStore is a GORM-backed store implementation for Dubbo resources
// It uses GORM for database operations and maintains in-memory indices for fast lookups
// This implementation is database-agnostic and works with any GORM-supported database
type GormStore struct {
	pool    *ConnectionPool // Shared connection pool with reference counting
	kind    model.ResourceKind
	address string
	indices *Index // In-memory index with thread-safe operations
	stopCh  chan struct{}
}

var _ store.ManagedResourceStore = &GormStore{}

// NewGormStore creates a new GORM store for the specified resource kind
func NewGormStore(kind model.ResourceKind, address string, pool *ConnectionPool) *GormStore {
	return &GormStore{
		kind:    kind,
		address: address,
		pool:    pool,
		indices: NewIndex(),
		stopCh:  make(chan struct{}),
	}
}

// Init initializes the GORM store by migrating the schema and rebuilding indices
func (gs *GormStore) Init(_ runtime.BuilderContext) error {
	// Perform table migration
	db := gs.pool.GetDB()
	// Use Scopes to set the table name dynamically for migration
	if err := db.Scopes(TableScope(gs.kind.ToString())).AutoMigrate(&ResourceModel{}); err != nil {
		return fmt.Errorf("failed to migrate schema for %s: %w", gs.kind.ToString(), err)
	}

	// Migrate resource_indices table (shared across all resource kinds)
	if err := db.AutoMigrate(&ResourceIndexModel{}); err != nil {
		return fmt.Errorf("failed to migrate resource_indices: %w", err)
	}

	// Register indexers for the resource kind
	indexers := index.IndexersRegistry().Indexers(gs.kind)
	if err := gs.AddIndexers(indexers); err != nil {
		return err
	}

	// Rebuild indices from existing data in the database
	if err := gs.rebuildIndices(); err != nil {
		return fmt.Errorf("failed to rebuild indices for %s: %w", gs.kind.ToString(), err)
	}

	logger.Infof("GORM store initialized for resource kind: %s", gs.kind.ToString())
	return nil
}

// Start starts the GORM store and monitors for shutdown signal
func (gs *GormStore) Start(_ runtime.Runtime, stopCh <-chan struct{}) error {
	logger.Infof("GORM store started for resource kind: %s", gs.kind.ToString())

	// Monitor stop channel for graceful shutdown in a goroutine
	go func() {
		<-stopCh
		logger.Infof("GORM store for %s received stop signal, initiating graceful shutdown", gs.kind.ToString())

		// Close the internal stop channel to signal any ongoing operations
		close(gs.stopCh)

		// Decrement the reference count and potentially close the connection pool
		if gs.pool != nil {
			if err := gs.pool.Close(); err != nil {
				logger.Errorf("Failed to close connection pool for %s: %v", gs.kind.ToString(), err)
			} else {
				logger.Infof("GORM store for %s shutdown completed", gs.kind.ToString())
			}
		}
	}()

	return nil
}

// Add inserts a new resource into the database
func (gs *GormStore) Add(obj interface{}) error {
	resource, ok := obj.(model.Resource)
	if !ok {
		return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
	}

	if resource.ResourceKind() != gs.kind {
		return fmt.Errorf("resource kind mismatch: expected %s, got %s", gs.kind, resource.ResourceKind())
	}

	var count int64
	db := gs.pool.GetDB()
	err := db.Scopes(TableScope(gs.kind.ToString())).Model(&ResourceModel{}).
		Where("resource_key = ?", resource.ResourceKey()).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return store.ErrorResourceAlreadyExists(
			resource.ResourceKind().ToString(),
			resource.ResourceMeta().Name,
			resource.ResourceMesh(),
		)
	}

	m, err := FromResource(resource)
	if err != nil {
		return err
	}

	if err := db.Scopes(TableScope(gs.kind.ToString())).Create(m).Error; err != nil {
		return err
	}

	// Update indices after successful DB operation
	gs.indices.UpdateResource(resource, nil)

	// Persist index entries to DB
	if err := gs.persistIndexEntries(resource, nil); err != nil {
		logger.Warnf("failed to persist index entries for %s: %v", resource.ResourceKey(), err)
	}

	return nil
}

// Update modifies an existing resource in the database
func (gs *GormStore) Update(obj interface{}) error {
	resource, ok := obj.(model.Resource)
	if !ok {
		return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
	}

	if resource.ResourceKind() != gs.kind {
		return fmt.Errorf("resource kind mismatch: expected %s, got %s", gs.kind, resource.ResourceKind())
	}

	// Get old resource for index update
	oldResource, exists, err := gs.GetByKey(resource.ResourceKey())
	if err != nil {
		return err
	}
	if !exists {
		return store.ErrorResourceNotFound(
			resource.ResourceKind().ToString(),
			resource.ResourceMeta().Name,
			resource.ResourceMesh(),
		)
	}

	m, err := FromResource(resource)
	if err != nil {
		return err
	}

	db := gs.pool.GetDB()
	result := db.Scopes(TableScope(gs.kind.ToString())).Model(&ResourceModel{}).
		Where("resource_key = ?", resource.ResourceKey()).
		Updates(map[string]interface{}{
			"name": m.Name,
			"mesh": m.Mesh,
			"data": m.Data,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return store.ErrorResourceNotFound(
			resource.ResourceKind().ToString(),
			resource.ResourceMeta().Name,
			resource.ResourceMesh(),
		)
	}

	// Update indices: remove old and add new
	gs.indices.UpdateResource(resource, oldResource.(model.Resource))

	// Persist index entries to DB
	if err := gs.persistIndexEntries(resource, oldResource.(model.Resource)); err != nil {
		logger.Warnf("failed to persist index entries for %s: %v", resource.ResourceKey(), err)
	}

	return nil
}

// Delete removes a resource from the database
func (gs *GormStore) Delete(obj interface{}) error {
	resource, ok := obj.(model.Resource)
	if !ok {
		return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
	}

	db := gs.pool.GetDB()
	result := db.Scopes(TableScope(gs.kind.ToString())).
		Where("resource_key = ?", resource.ResourceKey()).
		Delete(&ResourceModel{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return store.ErrorResourceNotFound(
			resource.ResourceKind().ToString(),
			resource.ResourceMeta().Name,
			resource.ResourceMesh(),
		)
	}

	// Remove from indices
	gs.indices.RemoveResource(resource)

	// Delete index entries from DB
	if err := gs.deleteIndexEntries(resource.ResourceKey()); err != nil {
		logger.Warnf("failed to delete index entries for %s: %v", resource.ResourceKey(), err)
	}

	return nil
}

// List returns all resources of the configured kind from the database
func (gs *GormStore) List() []interface{} {
	var models []ResourceModel
	db := gs.pool.GetDB()
	if err := db.Scopes(TableScope(gs.kind.ToString())).Model(&ResourceModel{}).Find(&models).Error; err != nil {
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

// ListKeys returns all resource keys of the configured kind from the database
func (gs *GormStore) ListKeys() []string {
	var keys []string
	db := gs.pool.GetDB()
	if err := db.Scopes(TableScope(gs.kind.ToString())).Model(&ResourceModel{}).Pluck("resource_key", &keys).Error; err != nil {
		logger.Errorf("failed to list keys: %v", err)
		return []string{}
	}
	return keys
}

// Get retrieves a resource by its object reference
func (gs *GormStore) Get(obj interface{}) (item interface{}, exists bool, err error) {
	resource, ok := obj.(model.Resource)
	if !ok {
		return nil, false, bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
	}
	return gs.GetByKey(resource.ResourceKey())
}

// GetByKey retrieves a resource by its unique key
func (gs *GormStore) GetByKey(key string) (item interface{}, exists bool, err error) {
	var m ResourceModel
	db := gs.pool.GetDB()
	result := db.Scopes(TableScope(gs.kind.ToString())).
		Where("resource_key = ?", key).
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

// Replace atomically replaces all resources in the database with the provided list
func (gs *GormStore) Replace(list []interface{}, _ string) error {
	db := gs.pool.GetDB()
	return db.Transaction(func(tx *gorm.DB) error {
		// Delete all existing records for this resource kind
		if err := tx.Scopes(TableScope(gs.kind.ToString())).
			Delete(&ResourceModel{}, "1=1").Error; err != nil {
			return err
		}

		// Delete all index entries for this resource kind
		if err := tx.Where("resource_kind = ?", gs.kind.ToString()).
			Delete(&ResourceIndexModel{}).Error; err != nil {
			return err
		}

		// Clear all indices
		gs.clearIndices()

		// Return early if list is empty
		if len(list) == 0 {
			return nil
		}

		// Convert all resources to ResourceModel
		models := make([]*ResourceModel, 0, len(list))
		resources := make([]model.Resource, 0, len(list))
		for _, obj := range list {
			resource, ok := obj.(model.Resource)
			if !ok {
				return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
			}

			m, err := FromResource(resource)
			if err != nil {
				return err
			}
			models = append(models, m)
			resources = append(resources, resource)
		}

		// Batch insert all models at once
		if err := tx.Scopes(TableScope(gs.kind.ToString())).CreateInBatches(models, 100).Error; err != nil {
			return err
		}

		// Rebuild indices for all resources
		for _, resource := range resources {
			gs.indices.UpdateResource(resource, nil)
		}

		// Persist all index entries in bulk
		var indexEntries []ResourceIndexModel
		indexers := gs.indices.GetIndexers()
		for _, resource := range resources {
			for indexName, indexFunc := range indexers {
				values, err := indexFunc(resource)
				if err != nil {
					continue
				}
				for _, v := range values {
					indexEntries = append(indexEntries, ResourceIndexModel{
						ResourceKind: gs.kind.ToString(),
						IndexName:    indexName,
						IndexValue:   v,
						ResourceKey:  resource.ResourceKey(),
					})
				}
			}
		}

		if len(indexEntries) > 0 {
			if err := tx.CreateInBatches(&indexEntries, 100).Error; err != nil {
				logger.Warnf("failed to persist index entries during replace: %v", err)
			}
		}

		return nil
	})
}

func (gs *GormStore) Resync() error {
	return nil
}

func (gs *GormStore) Index(indexName string, obj interface{}) ([]interface{}, error) {
	if !gs.indices.IndexExists(indexName) {
		return nil, fmt.Errorf("index %s does not exist", indexName)
	}

	indexFunc := gs.indices.GetIndexers()[indexName]
	indexValues, err := indexFunc(obj)
	if err != nil {
		return nil, err
	}

	if len(indexValues) == 0 {
		return []interface{}{}, nil
	}

	return gs.findByIndex(indexName, indexValues[0])
}

func (gs *GormStore) IndexKeys(indexName, indexedValue string) ([]string, error) {
	if !gs.indices.IndexExists(indexName) {
		return nil, fmt.Errorf("index %s does not exist", indexName)
	}

	resources, err := gs.findByIndex(indexName, indexedValue)
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

func (gs *GormStore) ListIndexFuncValues(indexName string) []string {
	if !gs.indices.IndexExists(indexName) {
		return []string{}
	}

	return gs.indices.ListIndexFuncValues(indexName)
}

func (gs *GormStore) ByIndex(indexName, indexedValue string) ([]interface{}, error) {
	if !gs.indices.IndexExists(indexName) {
		return nil, fmt.Errorf("index %s does not exist", indexName)
	}

	return gs.findByIndex(indexName, indexedValue)
}

func (gs *GormStore) GetIndexers() cache.Indexers {
	return gs.indices.GetIndexers()
}

func (gs *GormStore) AddIndexers(newIndexers cache.Indexers) error {
	return gs.indices.AddIndexers(newIndexers)
}

func (gs *GormStore) GetByKeys(keys []string) ([]model.Resource, error) {
	if len(keys) == 0 {
		return []model.Resource{}, nil
	}

	var models []ResourceModel
	db := gs.pool.GetDB()
	err := db.Scopes(TableScope(gs.kind.ToString())).Model(&ResourceModel{}).
		Where("resource_key IN ?", keys).
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

func (gs *GormStore) ListByIndexes(indexes []index.IndexCondition) ([]model.Resource, error) {
	keys, err := gs.getKeysByIndexes(indexes)
	if err != nil {
		return nil, err
	}

	resources, err := gs.GetByKeys(keys)
	if err != nil {
		return nil, err
	}

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].ResourceKey() < resources[j].ResourceKey()
	})

	return resources, nil
}

func (gs *GormStore) PageListByIndexes(indexes []index.IndexCondition, pq model.PageReq) (*model.PageData[model.Resource], error) {
	keys, err := gs.getKeysByIndexes(indexes)
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
	resources, err := gs.GetByKeys(pageKeys)
	if err != nil {
		return nil, err
	}

	return model.NewPageData(total, pq.PageOffset, pq.PageSize, resources), nil
}

func (gs *GormStore) findByIndex(indexName, indexedValue string) ([]interface{}, error) {
	if !gs.indices.IndexExists(indexName) {
		return nil, fmt.Errorf("index %s does not exist", indexName)
	}

	// Get resource keys from in-memory index
	keys := gs.indices.GetKeys(indexName, indexedValue)

	if len(keys) == 0 {
		return []interface{}{}, nil
	}

	// Fetch resources from DB by keys
	resources, err := gs.GetByKeys(keys)
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

func (gs *GormStore) getKeysByIndexes(indexes []index.IndexCondition) ([]string, error) {
	if len(indexes) == 0 {
		return gs.ListKeys(), nil
	}

	var keySet map[string]struct{}
	first := true

	for _, condition := range indexes {
		var keys []string
		var err error
		switch condition.Operator {
		case index.Equals:
			keys, err = gs.IndexKeys(condition.IndexName, condition.Value)
		case index.HasPrefix:
			keys, err = gs.getKeysByPrefixFromDB(condition.IndexName, condition.Value)
		default:
			return nil, bizerror.New(bizerror.InvalidArgument, "operator not yet supported: "+string(condition.Operator))
		}
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

// clearIndices clears all in-memory indices
func (gs *GormStore) clearIndices() {
	gs.indices.Clear()
}

// persistIndexEntries writes index entries for a resource to the database
// If oldResource is not nil, first deletes old entries, then inserts new ones
func (gs *GormStore) persistIndexEntries(resource model.Resource, oldResource model.Resource) error {
	db := gs.pool.GetDB()

	// Delete old entries if updating
	if oldResource != nil {
		if err := db.Where("resource_key = ?", oldResource.ResourceKey()).Delete(&ResourceIndexModel{}).Error; err != nil {
			return err
		}
	}

	// Get all index entries for this resource
	indexers := gs.indices.GetIndexers()
	var entries []ResourceIndexModel
	for indexName, indexFunc := range indexers {
		values, err := indexFunc(resource)
		if err != nil {
			continue
		}
		for _, v := range values {
			entries = append(entries, ResourceIndexModel{
				ResourceKind: gs.kind.ToString(),
				IndexName:    indexName,
				IndexValue:   v,
				ResourceKey:  resource.ResourceKey(),
			})
		}
	}

	if len(entries) == 0 {
		return nil
	}

	return db.Create(&entries).Error
}

// deleteIndexEntries removes all index entries for a resource key
func (gs *GormStore) deleteIndexEntries(resourceKey string) error {
	db := gs.pool.GetDB()
	return db.Where("resource_key = ?", resourceKey).Delete(&ResourceIndexModel{}).Error
}

// getKeysByPrefixFromDB retrieves resource keys matching a prefix from the database
func (gs *GormStore) getKeysByPrefixFromDB(indexName, prefix string) ([]string, error) {
	db := gs.pool.GetDB()
	var entries []ResourceIndexModel
	err := db.Where("resource_kind = ? AND index_name = ? AND index_value LIKE ?",
		gs.kind.ToString(), indexName, prefix+"%").
		Find(&entries).Error
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(entries))
	seen := make(map[string]struct{})
	for _, e := range entries {
		if _, ok := seen[e.ResourceKey]; !ok {
			keys = append(keys, e.ResourceKey)
			seen[e.ResourceKey] = struct{}{}
		}
	}
	return keys, nil
}

// rebuildIndices rebuilds all in-memory indices from existing database records
// First tries to rebuild from resource_indices table for efficiency
// Falls back to rebuilding from resource table if resource_indices is empty
func (gs *GormStore) rebuildIndices() error {
	// Clear existing indices first
	gs.clearIndices()

	// Try to load from resource_indices table (persisted indices)
	var entries []ResourceIndexModel
	db := gs.pool.GetDB()
	if err := db.Where("resource_kind = ?", gs.kind.ToString()).Find(&entries).Error; err != nil {
		logger.Warnf("failed to load from resource_indices for %s: %v", gs.kind.ToString(), err)
		return gs.rebuildIndicesFromResources()
	}

	// If resource_indices table has entries, rebuild from there
	if len(entries) > 0 {
		for _, entry := range entries {
			gs.indices.AddEntry(entry.IndexName, entry.IndexValue, entry.ResourceKey)
		}
		logger.Infof("Rebuilt indices for %s from resource_indices table: %d entries", gs.kind.ToString(), len(entries))
		return nil
	}

	// Fallback: rebuild from resources table
	return gs.rebuildIndicesFromResources()
}

// rebuildIndicesFromResources is the original implementation, now used as fallback
// It scans all resources and rebuilds both in-memory indices and persisted indices
func (gs *GormStore) rebuildIndicesFromResources() error {
	// Load all resources from the database
	var models []ResourceModel
	db := gs.pool.GetDB()
	if err := db.Scopes(TableScope(gs.kind.ToString())).Model(&ResourceModel{}).Find(&models).Error; err != nil {
		return fmt.Errorf("failed to load resources for index rebuild: %w", err)
	}

	// Rebuild indices for all resources
	for _, m := range models {
		resource, err := m.ToResource()
		if err != nil {
			logger.Errorf("failed to deserialize resource during index rebuild: %v", err)
			continue
		}
		// Add resource to indices (nil for oldResource since this is initial load)
		gs.indices.UpdateResource(resource, nil)

		// Also persist to resource_indices
		if err := gs.persistIndexEntries(resource, nil); err != nil {
			logger.Warnf("failed to persist index entries during rebuild: %v", err)
		}
	}

	logger.Infof("Rebuilt indices for %s from resources table: loaded %d resources", gs.kind.ToString(), len(models))
	return nil
}
