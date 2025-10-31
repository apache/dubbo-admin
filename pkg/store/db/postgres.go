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

package db

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"

	storecfg "github.com/apache/dubbo-admin/pkg/config/store"
	"github.com/go-logr/logr"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/log"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store"
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
	db          *gorm.DB
	kind        model.ResourceKind
	indexers    cache.Indexers
	indexerLock sync.RWMutex
	logger      logr.Logger
}

var _ store.ManagedResourceStore = &postgresStore{}

func NewPostgresStore(kind model.ResourceKind, address string) (store.ManagedResourceStore, error) {
	db, err := gorm.Open(postgres.Open(address), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := db.AutoMigrate(&ResourceModel{}); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}

	return &postgresStore{
		db:       db,
		kind:     kind,
		indexers: cache.Indexers{},
		logger:   log.NewLogger(log.InfoLevel).WithName("postgres-store"),
	}, nil
}

func (ps *postgresStore) Init(_ runtime.BuilderContext) error {
	return nil
}

func (ps *postgresStore) Start(_ runtime.Runtime, _ <-chan struct{}) error {
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
	err := ps.db.Model(&ResourceModel{}).
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

	m, err := FromResource(resource)
	if err != nil {
		return err
	}

	return ps.db.Create(m).Error
}

func (ps *postgresStore) Update(obj interface{}) error {
	resource, ok := obj.(model.Resource)
	if !ok {
		return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
	}

	if resource.ResourceKind() != ps.kind {
		return fmt.Errorf("resource kind mismatch: expected %s, got %s", ps.kind, resource.ResourceKind())
	}

	m, err := FromResource(resource)
	if err != nil {
		return err
	}

	result := ps.db.Model(&ResourceModel{}).
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

	return nil
}

func (ps *postgresStore) Delete(obj interface{}) error {
	resource, ok := obj.(model.Resource)
	if !ok {
		return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
	}

	result := ps.db.Where("resource_key = ?", resource.ResourceKey()).
		Delete(&ResourceModel{})

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

	return nil
}

func (ps *postgresStore) List() []interface{} {
	var models []ResourceModel
	if err := ps.db.Where("resource_kind = ?", ps.kind.ToString()).Find(&models).Error; err != nil {
		ps.logger.Error(err, "failed to list resources")
		return []interface{}{}
	}

	result := make([]interface{}, 0, len(models))
	for _, m := range models {
		resource, err := m.ToResource()
		if err != nil {
			ps.logger.Error(err, "failed to deserialize resource")
			continue
		}
		result = append(result, resource)
	}
	return result
}

func (ps *postgresStore) ListKeys() []string {
	var keys []string
	ps.db.Model(&ResourceModel{}).
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
	var m ResourceModel
	result := ps.db.Where("resource_key = ? AND resource_kind = ?", key, ps.kind.ToString()).
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
	return ps.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("resource_kind = ?", ps.kind.ToString()).Delete(&ResourceModel{}).Error; err != nil {
			return err
		}

		for _, obj := range list {
			resource, ok := obj.(model.Resource)
			if !ok {
				return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
			}

			m, err := FromResource(resource)
			if err != nil {
				return err
			}

			if err := tx.Create(m).Error; err != nil {
				return err
			}
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

	var models []ResourceModel
	err := ps.db.Where("resource_key IN ? AND resource_kind = ?", keys, ps.kind.ToString()).
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
	indexFunc := ps.indexers[indexName]
	ps.indexerLock.RUnlock()

	allResources := ps.List()
	result := make([]interface{}, 0)

	for _, obj := range allResources {
		values, err := indexFunc(obj)
		if err != nil {
			continue
		}

		for _, value := range values {
			if value == indexedValue {
				result = append(result, obj)
				break
			}
		}
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
