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
	"strconv"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

// GormStore is a GORM-backed store implementation for Dubbo resources
// It uses GORM for database operations and persists all indices to the resource_indices table
// This implementation is database-agnostic and works with any GORM-supported database
type GormStore struct {
	pool     *ConnectionPool // Shared connection pool with reference counting
	kind     model.ResourceKind
	address  string
	indexers cache.Indexers // Index functions for creating indices
	mu       sync.RWMutex   // Protects indexers
	stopCh   chan struct{}
}

var _ store.ManagedResourceStore = &GormStore{}
var _ store.ConditionalResourceStore = &GormStore{}
var _ store.CascadingResourceStore = &GormStore{}
var _ store.DependentResourceStore = &GormStore{}

// NewGormStore creates a new GORM store for the specified resource kind
func NewGormStore(kind model.ResourceKind, address string, pool *ConnectionPool) *GormStore {
	return &GormStore{
		kind:     kind,
		address:  address,
		pool:     pool,
		indexers: make(cache.Indexers),
		stopCh:   make(chan struct{}),
	}
}

// Init initializes the GORM store by migrating the schema and registering indexers
func (gs *GormStore) Init(_ runtime.BuilderContext) error {
	// Perform table migration
	db := gs.pool.GetDB()
	// Use Scopes to set the table name dynamically for migration. Only resource
	// kinds that opt into conditional writes receive the version column.
	migrationModel := any(&ResourceModel{})
	if store.IsVersionedResourceKind(gs.kind) {
		migrationModel = &VersionedResourceModel{}
	}
	if err := db.Scopes(TableScope(gs.kind.ToString())).AutoMigrate(migrationModel); err != nil {
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

	// Backfill resource_indices from existing ResourceModel rows so that
	// ListByIndexes/HasPrefix work correctly after restarts or upgrades.
	if err := gs.backfillIndices(); err != nil {
		return fmt.Errorf("failed to backfill indices for %s: %w", gs.kind.ToString(), err)
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
	if versioned, ok := resource.(store.VersionedResource); ok {
		candidate := versioned.DeepCopyObject().(store.VersionedResource)
		if err := store.PrepareInitialVersion(candidate); err != nil {
			return err
		}
		resource = candidate
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

	err = db.Transaction(func(tx *gorm.DB) error {
		modelToCreate := any(m)
		if _, ok := resource.(store.VersionedResource); ok {
			modelToCreate = &VersionedResourceModel{ResourceModel: *m, Version: 1}
		}
		if err := tx.Scopes(TableScope(gs.kind.ToString())).Create(modelToCreate).Error; err != nil {
			return err
		}
		if err := gs.persistIndexEntriesTx(tx, resource, nil); err != nil {
			return fmt.Errorf("failed to persist index entries for %s: %w", resource.ResourceKey(), err)
		}
		return nil
	})
	if err == nil {
		if versioned, ok := obj.(store.VersionedResource); ok {
			versioned.SetResourceVersion(resource.ResourceMeta().ResourceVersion)
		}
	}
	return err
}

func (gs *GormStore) AddWithParent(parent model.Resource, parentStore store.ResourceStore, child model.Resource) error {
	serverStore, ok := parentStore.(*GormStore)
	if !ok || serverStore.pool != gs.pool || child.ResourceKind() != gs.kind || parent.ResourceKind() != serverStore.kind {
		return &store.PreconditionError{Reason: "parent and dependent must share a Gorm connection pool"}
	}
	candidate := child.DeepCopyObject().(model.Resource)
	if err := store.PrepareInitialVersion(candidate); err != nil {
		return err
	}
	m, err := FromResource(candidate)
	if err != nil {
		return err
	}
	err = gs.pool.GetDB().Transaction(func(tx *gorm.DB) error {
		var server VersionedResourceModel
		err := tx.Scopes(TableScope(serverStore.kind.ToString())).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("resource_key = ?", parent.ResourceKey()).First(&server).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return store.ErrorResourceNotFound(parent.ResourceKind().ToString(), parent.ResourceMeta().Name, parent.ResourceMesh())
		}
		if err != nil {
			return err
		}
		if err := tx.Scopes(TableScope(gs.kind.ToString())).Create(&VersionedResourceModel{ResourceModel: *m, Version: 1}).Error; err != nil {
			return err
		}
		return gs.persistIndexEntriesTx(tx, candidate, nil)
	})
	if err == nil {
		child.(store.VersionedResource).SetResourceVersion(candidate.ResourceMeta().ResourceVersion)
	}
	return err
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
	if _, ok := resource.(store.VersionedResource); ok {
		return &store.PreconditionError{Reason: "versioned resources require CompareAndSwap"}
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
	return db.Transaction(func(tx *gorm.DB) error {
		result := tx.Scopes(TableScope(gs.kind.ToString())).Model(&ResourceModel{}).
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

		if err := gs.persistIndexEntriesTx(tx, resource, oldResource.(model.Resource)); err != nil {
			return fmt.Errorf("failed to persist index entries for %s: %w", resource.ResourceKey(), err)
		}
		return nil
	})
}

// Delete removes a resource from the database
func (gs *GormStore) Delete(obj interface{}) error {
	resource, ok := obj.(model.Resource)
	if !ok {
		return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
	}
	if _, ok := resource.(store.VersionedResource); ok {
		return &store.PreconditionError{Reason: "versioned resources require CompareAndDelete"}
	}

	db := gs.pool.GetDB()
	return db.Transaction(func(tx *gorm.DB) error {
		result := tx.Scopes(TableScope(gs.kind.ToString())).
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

		return tx.Where("resource_kind = ? AND resource_key = ?", gs.kind.ToString(), resource.ResourceKey()).
			Delete(&ResourceIndexModel{}).Error
	})
}

func (gs *GormStore) CompareAndSwap(obj model.Resource, expectedVersion string) error {
	if obj.ResourceKind() != gs.kind {
		return fmt.Errorf("resource kind mismatch: expected %s, got %s", gs.kind, obj.ResourceKind())
	}
	nextVersion, err := store.NextResourceVersion(obj, expectedVersion)
	if err != nil {
		return err
	}
	expected, _ := strconv.ParseUint(expectedVersion, 10, 64)

	db := gs.pool.GetDB()
	err = db.Transaction(func(tx *gorm.DB) error {
		var current VersionedResourceModel
		err := tx.Scopes(TableScope(gs.kind.ToString())).
			Where("resource_key = ?", obj.ResourceKey()).
			First(&current).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return store.ErrorResourceNotFound(obj.ResourceKind().ToString(), obj.ResourceMeta().Name, obj.ResourceMesh())
		}
		if err != nil {
			return err
		}
		if current.Version != expected {
			return store.ErrorResourceConflict(obj.ResourceKind().ToString(), obj.ResourceMeta().Name, obj.ResourceMesh())
		}

		oldResource, err := current.ResourceModel.ToResource()
		if err != nil {
			return err
		}
		candidate, ok := obj.DeepCopyObject().(model.Resource)
		if !ok {
			return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj.DeepCopyObject()).Name())
		}
		candidateVersioned, ok := candidate.(store.VersionedResource)
		if !ok {
			return &store.PreconditionError{Reason: "resource does not support conditional mutations"}
		}
		candidateVersioned.SetResourceVersion(nextVersion)
		updated, err := FromResource(candidate)
		if err != nil {
			return err
		}

		result := tx.Scopes(TableScope(gs.kind.ToString())).Model(&VersionedResourceModel{}).
			Where("resource_key = ? AND version = ?", obj.ResourceKey(), expected).
			Updates(map[string]interface{}{
				"name":    updated.Name,
				"mesh":    updated.Mesh,
				"data":    updated.Data,
				"version": expected + 1,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return store.ErrorResourceConflict(obj.ResourceKind().ToString(), obj.ResourceMeta().Name, obj.ResourceMesh())
		}
		if err := gs.persistIndexEntriesTx(tx, candidate, oldResource); err != nil {
			return fmt.Errorf("failed to persist index entries for %s: %w", candidate.ResourceKey(), err)
		}
		return nil
	})
	if err == nil {
		obj.(store.VersionedResource).SetResourceVersion(nextVersion)
	}
	return err
}

func (gs *GormStore) CompareAndDelete(obj model.Resource, expectedVersion string) error {
	if obj.ResourceKind() != gs.kind {
		return fmt.Errorf("resource kind mismatch: expected %s, got %s", gs.kind, obj.ResourceKind())
	}
	if _, err := store.NextResourceVersion(obj, expectedVersion); err != nil {
		return err
	}
	expected, _ := strconv.ParseUint(expectedVersion, 10, 64)

	db := gs.pool.GetDB()
	return db.Transaction(func(tx *gorm.DB) error {
		result := tx.Scopes(TableScope(gs.kind.ToString())).
			Where("resource_key = ? AND version = ?", obj.ResourceKey(), expected).
			Delete(&VersionedResourceModel{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			var count int64
			if err := tx.Scopes(TableScope(gs.kind.ToString())).Model(&VersionedResourceModel{}).
				Where("resource_key = ?", obj.ResourceKey()).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return store.ErrorResourceNotFound(obj.ResourceKind().ToString(), obj.ResourceMeta().Name, obj.ResourceMesh())
			}
			return store.ErrorResourceConflict(obj.ResourceKind().ToString(), obj.ResourceMeta().Name, obj.ResourceMesh())
		}
		return tx.Where("resource_kind = ? AND resource_key = ?", gs.kind.ToString(), obj.ResourceKey()).
			Delete(&ResourceIndexModel{}).Error
	})
}

func (gs *GormStore) CompareAndDeleteWithDependents(obj model.Resource, expectedVersion string, dependents store.ResourceStore, conditions []index.IndexCondition) error {
	childStore, ok := dependents.(*GormStore)
	if !ok || childStore.pool != gs.pool || childStore == gs || obj.ResourceKind() != gs.kind {
		return &store.PreconditionError{Reason: "parent and dependent must use distinct Gorm stores sharing one connection pool"}
	}
	if len(conditions) != 2 || conditions[0].IndexName != index.ByMeshIndex || conditions[0].Value != obj.ResourceMesh() || conditions[1].IndexName != index.ByMCPCredentialServerID || conditions[1].Value != obj.ResourceMeta().Name {
		return &store.PreconditionError{Reason: "dependent query must be scoped to server mesh and ID"}
	}
	if _, err := store.NextResourceVersion(obj, expectedVersion); err != nil {
		return err
	}
	expected, _ := strconv.ParseUint(expectedVersion, 10, 64)
	return gs.pool.GetDB().Transaction(func(tx *gorm.DB) error {
		var current VersionedResourceModel
		err := tx.Scopes(TableScope(gs.kind.ToString())).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("resource_key = ?", obj.ResourceKey()).First(&current).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return store.ErrorResourceNotFound(obj.ResourceKind().ToString(), obj.ResourceMeta().Name, obj.ResourceMesh())
		}
		if err != nil {
			return err
		}
		if current.Version != expected {
			return store.ErrorResourceConflict(obj.ResourceKind().ToString(), obj.ResourceMeta().Name, obj.ResourceMesh())
		}
		var children []VersionedResourceModel
		keysQuery := tx.Model(&ResourceIndexModel{}).Select("resource_key").Where(
			"resource_kind = ? AND index_name = ? AND index_value = ?", childStore.kind.ToString(), index.ByMCPCredentialServerID, obj.ResourceMeta().Name)
		if err := tx.Scopes(TableScope(childStore.kind.ToString())).Where("mesh = ? AND resource_key IN (?)", obj.ResourceMesh(), keysQuery).
			Find(&children).Error; err != nil {
			return err
		}
		if len(children) > 0 {
			keys := make([]string, len(children))
			for i := range children {
				keys[i] = children[i].ResourceKey
			}
			if err := tx.Scopes(TableScope(childStore.kind.ToString())).Where("resource_key IN ?", keys).Delete(&VersionedResourceModel{}).Error; err != nil {
				return err
			}
			if err := tx.Where("resource_kind = ? AND resource_key IN ?", childStore.kind.ToString(), keys).Delete(&ResourceIndexModel{}).Error; err != nil {
				return err
			}
		}
		result := tx.Scopes(TableScope(gs.kind.ToString())).Where("resource_key = ? AND version = ?", obj.ResourceKey(), expected).
			Delete(&VersionedResourceModel{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return store.ErrorResourceConflict(obj.ResourceKind().ToString(), obj.ResourceMeta().Name, obj.ResourceMesh())
		}
		return tx.Where("resource_kind = ? AND resource_key = ?", gs.kind.ToString(), obj.ResourceKey()).Delete(&ResourceIndexModel{}).Error
	})
}

// List returns all resources of the configured kind from the database
func (gs *GormStore) List() []interface{} {
	db := gs.pool.GetDB()
	result := make([]interface{}, 0)
	if store.IsVersionedResourceKind(gs.kind) {
		var models []VersionedResourceModel
		if err := db.Scopes(TableScope(gs.kind.ToString())).Model(&VersionedResourceModel{}).Find(&models).Error; err != nil {
			logger.Errorf("failed to list resources: %v", err)
			return result
		}
		for _, m := range models {
			resource, err := m.ResourceModel.ToResourceWithVersion(m.Version)
			if err != nil {
				logger.Errorf("failed to deserialize resource: %v", err)
				continue
			}
			result = append(result, resource)
		}
		return result
	}
	var models []ResourceModel
	if err := db.Scopes(TableScope(gs.kind.ToString())).Model(&ResourceModel{}).Find(&models).Error; err != nil {
		logger.Errorf("failed to list resources: %v", err)
		return result
	}
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
	if store.IsVersionedResourceKind(gs.kind) {
		var m VersionedResourceModel
		result := gs.pool.GetDB().Scopes(TableScope(gs.kind.ToString())).
			Where("resource_key = ?", key).
			First(&m)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, false, nil
			}
			return nil, false, result.Error
		}
		resource, err := m.ResourceModel.ToResourceWithVersion(m.Version)
		if err != nil {
			return nil, false, err
		}
		return resource, true, nil
	}
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
	if store.IsVersionedResourceKind(gs.kind) {
		return &store.PreconditionError{Reason: "versioned resources require conditional mutations"}
	}
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

		// Persist all index entries in bulk
		var indexEntries []ResourceIndexModel
		indexers := gs.GetIndexers()
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
						Operator:     string(index.Equals),
					})
				}
			}
		}

		if len(indexEntries) > 0 {
			if err := tx.CreateInBatches(&indexEntries, 100).Error; err != nil {
				return fmt.Errorf("failed to persist index entries during replace: %w", err)
			}
		}

		return nil
	})
}

func (gs *GormStore) Resync() error {
	return nil
}

func (gs *GormStore) Index(indexName string, obj interface{}) ([]interface{}, error) {
	if !gs.IndexExists(indexName) {
		return nil, fmt.Errorf("index %s does not exist", indexName)
	}

	indexers := gs.GetIndexers()
	indexFunc := indexers[indexName]
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
	if !gs.IndexExists(indexName) {
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
	if !gs.IndexExists(indexName) {
		return []string{}
	}

	var values []string
	db := gs.pool.GetDB()
	if err := db.Model(&ResourceIndexModel{}).
		Where("resource_kind = ? AND index_name = ?", gs.kind.ToString(), indexName).
		Distinct("index_value").
		Pluck("index_value", &values).Error; err != nil {
		logger.Errorf("failed to list index func values for %s: %v", indexName, err)
		return []string{}
	}
	return values
}

func (gs *GormStore) ByIndex(indexName, indexedValue string) ([]interface{}, error) {
	if !gs.IndexExists(indexName) {
		return nil, fmt.Errorf("index %s does not exist", indexName)
	}

	return gs.findByIndex(indexName, indexedValue)
}

func (gs *GormStore) GetIndexers() cache.Indexers {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	result := make(cache.Indexers, len(gs.indexers))
	for k, v := range gs.indexers {
		result[k] = v
	}
	return result
}

func (gs *GormStore) AddIndexers(newIndexers cache.Indexers) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	for name, indexFunc := range newIndexers {
		if _, exists := gs.indexers[name]; exists {
			return fmt.Errorf("indexer %s already exists", name)
		}
		gs.indexers[name] = indexFunc
	}

	return nil
}

func (gs *GormStore) GetByKeys(keys []string) ([]model.Resource, error) {
	if len(keys) == 0 {
		return []model.Resource{}, nil
	}

	db := gs.pool.GetDB()
	resources := make([]model.Resource, 0, len(keys))
	if store.IsVersionedResourceKind(gs.kind) {
		var models []VersionedResourceModel
		if err := db.Scopes(TableScope(gs.kind.ToString())).Model(&VersionedResourceModel{}).
			Where("resource_key IN ?", keys).Find(&models).Error; err != nil {
			return nil, err
		}
		byKey := make(map[string]model.Resource, len(models))
		for _, m := range models {
			resource, err := m.ResourceModel.ToResourceWithVersion(m.Version)
			if err != nil {
				return nil, err
			}
			byKey[m.ResourceKey] = resource
		}
		for _, key := range keys {
			if resource, ok := byKey[key]; ok {
				resources = append(resources, resource)
			}
		}
		return resources, nil
	}
	var models []ResourceModel
	if err := db.Scopes(TableScope(gs.kind.ToString())).Model(&ResourceModel{}).
		Where("resource_key IN ?", keys).Find(&models).Error; err != nil {
		return nil, err
	}
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
	if !gs.IndexExists(indexName) {
		return nil, fmt.Errorf("index %s does not exist", indexName)
	}

	// Get resource keys from database index
	db := gs.pool.GetDB()
	var entries []ResourceIndexModel
	err := db.Where("resource_kind = ? AND index_name = ? AND index_value = ?",
		gs.kind.ToString(), indexName, indexedValue).
		Find(&entries).Error
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return []interface{}{}, nil
	}

	// Collect unique resource keys
	keys := make([]string, 0, len(entries))
	seen := make(map[string]struct{})
	for _, e := range entries {
		if _, ok := seen[e.ResourceKey]; !ok {
			keys = append(keys, e.ResourceKey)
			seen[e.ResourceKey] = struct{}{}
		}
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

// persistIndexEntriesTx writes index entries for a resource within an existing transaction.
// If oldResource is not nil, first deletes its old entries scoped by resource_kind.
func (gs *GormStore) persistIndexEntriesTx(tx *gorm.DB, resource model.Resource, oldResource model.Resource) error {
	if oldResource != nil {
		if err := tx.Where("resource_kind = ? AND resource_key = ?", gs.kind.ToString(), oldResource.ResourceKey()).
			Delete(&ResourceIndexModel{}).Error; err != nil {
			return err
		}
	}

	indexers := gs.GetIndexers()
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
				Operator:     string(index.Equals),
			})
		}
	}

	if len(entries) == 0 {
		return nil
	}

	return tx.Create(&entries).Error
}

// backfillIndices rebuilds resource_indices from existing ResourceModel rows.
// Called at Init time so that ListByIndexes works correctly after restarts or upgrades.
func (gs *GormStore) backfillIndices() error {
	db := gs.pool.GetDB()

	// Load all existing resources for this kind
	var models []ResourceModel
	if err := db.Scopes(TableScope(gs.kind.ToString())).Find(&models).Error; err != nil {
		return err
	}
	if len(models) == 0 {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// Drop stale index rows for this kind and rebuild from scratch
		if err := tx.Where("resource_kind = ?", gs.kind.ToString()).
			Delete(&ResourceIndexModel{}).Error; err != nil {
			return err
		}

		indexers := gs.GetIndexers()
		var entries []ResourceIndexModel
		for _, m := range models {
			resource, err := m.ToResource()
			if err != nil {
				logger.Warnf("backfillIndices: failed to deserialize resource %s: %v", m.ResourceKey, err)
				continue
			}
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
						Operator:     string(index.Equals),
					})
				}
			}
		}

		if len(entries) == 0 {
			return nil
		}
		return tx.CreateInBatches(&entries, 100).Error
	})
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

// IndexExists checks if an indexer with the given name exists
func (gs *GormStore) IndexExists(indexName string) bool {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	_, exists := gs.indexers[indexName]
	return exists
}

// Pool returns the connection pool for this store
// Used by other components (e.g., leader election) that need direct DB access
// Returns interface{} to satisfy the poolProvider interface in pkg/core/store
func (gs *GormStore) Pool() interface{} {
	return gs.pool
}
