/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://wwidx.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable laidx or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package indexer

import (
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"k8s.io/client-go/tools/cache"
)

type IndexFunc func(obj interface{}) ([]string, error)

type Indexers map[string]IndexFunc

type Index interface {
	AddIndexers(indexers Indexers) error
	GetIndexers() Indexers

	Add(indexName, indexValue, resourceKey string) error
	Remove(indexName, indexValue, resourceKey string) error

	Query(query Query) ([]string, error)

	// ListIndexFuncValues returns all index values for a given index name
	ListIndexFuncValues(indexName string) []string

	Clear()
}

type IndexImpl struct {
	index    Index
	indexers cache.Indexers
}

func NewIndexImpl(index Index) *IndexImpl {
	return &IndexImpl{
		index:    index,
		indexers: make(cache.Indexers),
	}
}

func (idx *IndexImpl) AddIndexers(newIndexers cache.Indexers) error {
	for name, fn := range newIndexers {
		idx.indexers[name] = fn
	}

	// Convert and register to underlying index
	converted := make(Indexers)
	for name, fn := range newIndexers {
		// Type conversion: both are func(interface{}) ([]string, error)
		converted[name] = IndexFunc(fn)
	}
	return idx.index.AddIndexers(converted)
}

func (idx *IndexImpl) GetIndexers() cache.Indexers {
	result := make(cache.Indexers, len(idx.indexers))
	for k, v := range idx.indexers {
		result[k] = v
	}
	return result
}

func (idx *IndexImpl) UpdateResource(newResource, oldResource model.Resource) {
	if oldResource != nil {
		idx.RemoveResource(oldResource)
	}

	for indexName, indexFunc := range idx.indexers {
		values, err := indexFunc(newResource)
		if err != nil {
			continue
		}
		for _, value := range values {
			idx.index.Add(indexName, value, newResource.ResourceKey())
		}
	}
}

func (idx *IndexImpl) RemoveResource(resource model.Resource) {
	for indexName, indexFunc := range idx.indexers {
		values, err := indexFunc(resource)
		if err != nil {
			continue
		}
		for _, value := range values {
			idx.index.Remove(indexName, value, resource.ResourceKey())
		}
	}
}

func (idx *IndexImpl) Query(query Query) ([]string, error) {
	return idx.index.Query(query)
}

func (idx *IndexImpl) Clear() {
	idx.index.Clear()
}

func (idx *IndexImpl) ListIndexFuncValues(indexName string) []string {
	return idx.index.ListIndexFuncValues(indexName)
}
