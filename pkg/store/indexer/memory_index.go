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
)

type MemoryIndex struct {
	mu       sync.RWMutex
	indices  map[string]*RadixTree
	indexers Indexers
}

func NewMemoryIndex() *MemoryIndex {
	return &MemoryIndex{
		indices:  make(map[string]*RadixTree),
		indexers: make(Indexers),
	}
}

func (m *MemoryIndex) AddIndexers(indexers Indexers) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, indexFunc := range indexers {
		if _, exists := m.indexers[name]; exists {
			return fmt.Errorf("indexer %s already exists", name)
		}
		m.indexers[name] = indexFunc
	}
	return nil
}

func (m *MemoryIndex) GetIndexers() Indexers {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(Indexers, len(m.indexers))
	for k, v := range m.indexers {
		result[k] = v
	}
	return result
}

func (m *MemoryIndex) Add(indexName, indexValue, resourceKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.indices[indexName] == nil {
		m.indices[indexName] = NewRadixTree()
	}

	m.indices[indexName].Insert(indexValue, resourceKey)
	return nil
}

func (m *MemoryIndex) Remove(indexName, indexValue, resourceKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.indices[indexName] == nil {
		return nil
	}

	m.indices[indexName].Delete(indexValue, resourceKey)
	return nil
}

func (m *MemoryIndex) Query(query Query) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tree := m.indices[query.IndexName]
	if tree == nil {
		return []string{}, nil
	}

	switch query.Operator {
	case OperatorEquals:
		return tree.ExactSearch(query.Value), nil
	case OperatorPrefix:
		return tree.SearchPrefix(query.Value), nil
	default:
		return nil, fmt.Errorf("unsupported operator: %s", query.Operator)
	}
}

func (m *MemoryIndex) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.indices = make(map[string]*RadixTree)
}

// ListIndexFuncValues returns all index values for a given index name
func (m *MemoryIndex) ListIndexFuncValues(indexName string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tree := m.indices[indexName]
	if tree == nil {
		return []string{}
	}

	return tree.GetAllIndexValues()
}
