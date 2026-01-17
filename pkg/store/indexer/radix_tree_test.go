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
	"sync"
	"testing"
)

func TestNewRadixTree(t *testing.T) {
	tree := NewRadixTree()
	if tree == nil {
		t.Fatal("NewRadixTree returned nil")
	}
	if tree.Size() != 0 {
		t.Errorf("Expected size 0, got %d", tree.Size())
	}
	if tree.root == nil {
		t.Error("Root node is nil")
	}
}

func TestInsert(t *testing.T) {
	tests := []struct {
		name     string
		inserts  []struct{ index, key string }
		wantSize int
	}{
		{
			name:     "single insert",
			inserts:  []struct{ index, key string }{{"dubbo-service", "key1"}},
			wantSize: 1,
		},
		{
			name: "multiple inserts",
			inserts: []struct{ index, key string }{
				{"dubbo-service-a", "key1"},
				{"dubbo-service-b", "key2"},
				{"dubbo-admin", "key3"},
			},
			wantSize: 3,
		},
		{
			name: "duplicate insert",
			inserts: []struct{ index, key string }{
				{"dubbo-service", "key1"},
				{"dubbo-service", "key1"},
			},
			wantSize: 1,
		},
		{
			name: "same index different keys",
			inserts: []struct{ index, key string }{
				{"dubbo-service", "key1"},
				{"dubbo-service", "key2"},
			},
			wantSize: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := NewRadixTree()
			for _, ins := range tt.inserts {
				tree.Insert(ins.index, ins.key)
			}
			if got := tree.Size(); got != tt.wantSize {
				t.Errorf("Size() = %d, want %d", got, tt.wantSize)
			}
		})
	}
}

func TestSearchPrefix(t *testing.T) {
	tests := []struct {
		name      string
		inserts   []struct{ index, key string }
		prefix    string
		wantCount int
	}{
		{
			name: "basic prefix match",
			inserts: []struct{ index, key string }{
				{"dubbo-service-a", "key1"},
				{"dubbo-service-b", "key2"},
				{"dubbo-admin", "key3"},
			},
			prefix:    "dubbo-service",
			wantCount: 2,
		},
		{
			name: "empty prefix returns all",
			inserts: []struct{ index, key string }{
				{"dubbo-service", "key1"},
				{"dubbo-admin", "key2"},
			},
			prefix:    "",
			wantCount: 2,
		},
		{
			name: "no match",
			inserts: []struct{ index, key string }{
				{"dubbo-service", "key1"},
			},
			prefix:    "spring",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := NewRadixTree()
			for _, ins := range tt.inserts {
				tree.Insert(ins.index, ins.key)
			}
			results := tree.SearchPrefix(tt.prefix)
			if got := len(results); got != tt.wantCount {
				t.Errorf("SearchPrefix() returned %d results, want %d", got, tt.wantCount)
			}
		})
	}
}

func TestExactSearch(t *testing.T) {
	tests := []struct {
		name      string
		inserts   []struct{ index, key string }
		searchKey string
		wantCount int
		wantKey   string
	}{
		{
			name: "exact match",
			inserts: []struct{ index, key string }{
				{"dubbo-service", "key1"},
				{"dubbo-service-a", "key2"},
			},
			searchKey: "dubbo-service",
			wantCount: 1,
			wantKey:   "key1",
		},
		{
			name: "no match",
			inserts: []struct{ index, key string }{
				{"dubbo-service", "key1"},
			},
			searchKey: "dubbo-service-a",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := NewRadixTree()
			for _, ins := range tt.inserts {
				tree.Insert(ins.index, ins.key)
			}
			results := tree.ExactSearch(tt.searchKey)
			if got := len(results); got != tt.wantCount {
				t.Errorf("ExactSearch() returned %d results, want %d", got, tt.wantCount)
			}
			if tt.wantCount > 0 && len(results) > 0 && results[0] != tt.wantKey {
				t.Errorf("ExactSearch() returned key %s, want %s", results[0], tt.wantKey)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name          string
		inserts       []struct{ index, key string }
		deleteIndex   string
		deleteKey     string
		wantDeleted   bool
		wantSize      int
		wantRemaining string
	}{
		{
			name:        "basic delete",
			inserts:     []struct{ index, key string }{{"dubbo-service", "key1"}},
			deleteIndex: "dubbo-service",
			deleteKey:   "key1",
			wantDeleted: true,
			wantSize:    0,
		},
		{
			name: "delete one of multiple keys",
			inserts: []struct{ index, key string }{
				{"dubbo-service", "key1"},
				{"dubbo-service", "key2"},
			},
			deleteIndex:   "dubbo-service",
			deleteKey:     "key1",
			wantDeleted:   true,
			wantSize:      1,
			wantRemaining: "key2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := NewRadixTree()
			for _, ins := range tt.inserts {
				tree.Insert(ins.index, ins.key)
			}

			deleted := tree.Delete(tt.deleteIndex, tt.deleteKey)
			if deleted != tt.wantDeleted {
				t.Errorf("Delete() = %v, want %v", deleted, tt.wantDeleted)
			}
			if got := tree.Size(); got != tt.wantSize {
				t.Errorf("Size() = %d, want %d", got, tt.wantSize)
			}

			if tt.wantRemaining != "" {
				results := tree.ExactSearch(tt.deleteIndex)
				if len(results) != 1 || results[0] != tt.wantRemaining {
					t.Errorf("Expected remaining key %s, got %v", tt.wantRemaining, results)
				}
			}
		})
	}
}

func TestConcurrentInsert(t *testing.T) {
	tree := NewRadixTree()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tree.Insert("dubbo-service", "key")
		}(i)
	}

	wg.Wait()
	if tree.Size() != 1 {
		t.Errorf("Expected size 1 after concurrent inserts, got %d", tree.Size())
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	tree := NewRadixTree()
	tree.Insert("dubbo-service-a", "key1")
	tree.Insert("dubbo-service-b", "key2")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			tree.SearchPrefix("dubbo-service")
		}()
		go func() {
			defer wg.Done()
			tree.Insert("dubbo-admin", "key3")
		}()
	}

	wg.Wait()
}
