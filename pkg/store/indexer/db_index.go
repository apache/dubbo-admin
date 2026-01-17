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

package indexer

import (
	"fmt"
	"sync"

	"gorm.io/gorm"
)

type DBIndex struct {
	db           *gorm.DB
	resourceKind string
	mu           sync.RWMutex
	indexers     Indexers
}

func NewDBIndex(db *gorm.DB, resourceKind string) *DBIndex {
	return &DBIndex{
		db:           db,
		resourceKind: resourceKind,
		indexers:     make(Indexers),
	}
}

func (d *DBIndex) AddIndexers(indexers Indexers) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for name, indexFunc := range indexers {
		if _, exists := d.indexers[name]; exists {
			return fmt.Errorf("indexer %s already exists", name)
		}
		d.indexers[name] = indexFunc
	}
	return nil
}

func (d *DBIndex) GetIndexers() Indexers {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make(Indexers, len(d.indexers))
	for k, v := range d.indexers {
		result[k] = v
	}
	return result
}

func (d *DBIndex) Add(indexName, indexValue, resourceKey string) error {
	model := IndexModel{
		IndexName:    indexName,
		IndexValue:   indexValue,
		ResourceKey:  resourceKey,
		ResourceKind: d.resourceKind,
	}
	return d.db.Create(&model).Error
}

func (d *DBIndex) Remove(indexName, indexValue, resourceKey string) error {
	return d.db.Where(
		"index_name = ? AND index_value = ? AND resource_key = ? AND resource_kind = ?",
		indexName, indexValue, resourceKey, d.resourceKind,
	).Delete(&IndexModel{}).Error
}

func (d *DBIndex) Query(query Query) ([]string, error) {
	var models []IndexModel
	tx := d.db.Where("index_name = ? AND resource_kind = ?", query.IndexName, d.resourceKind)

	switch query.Operator {
	case OperatorEquals:
		tx = tx.Where("index_value = ?", query.Value)
	case OperatorPrefix:
		tx = tx.Where("index_value LIKE ?", query.Value+"%")
	default:
		return nil, fmt.Errorf("unsupported operator: %s", query.Operator)
	}

	if err := tx.Find(&models).Error; err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(models))
	for _, model := range models {
		keys = append(keys, model.ResourceKey)
	}
	return keys, nil
}

func (d *DBIndex) Clear() {
	d.db.Where("resource_kind = ?", d.resourceKind).Delete(&IndexModel{})
}

// ListIndexFuncValues returns all distinct index values for a given index name
func (d *DBIndex) ListIndexFuncValues(indexName string) []string {
	var values []string
	d.db.Model(&IndexModel{}).
		Where("index_name = ? AND resource_kind = ?", indexName, d.resourceKind).
		Distinct("index_value").
		Pluck("index_value", &values)
	return values
}
