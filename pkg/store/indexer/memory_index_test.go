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

func TestNewMemoryIndex(t *testing.T) {
	idx := NewMemoryIndex()
	if idx == nil {
		t.Fatal("NewMemoryIndex returned nil")
	}
	if idx.indices == nil {
		t.Error("indices map is nil")
	}
	if idx.indexers == nil {
		t.Error("indexers map is nil")
	}
}

func TestMemoryIndexAddRemove(t *testing.T) {
	tests := []struct {
		name      string
		adds      []struct{ indexName, indexValue, resourceKey string }
		removes   []struct{ indexName, indexValue, resourceKey string }
		wantQuery Query
		wantCount int
	}{
		{
			name: "add single entry",
			adds: []struct{ indexName, indexValue, resourceKey string }{
				{"app_name", "dubbo-service", "key1"},
			},
			wantQuery: NewQuery("app_name", OperatorEquals, "dubbo-service"),
			wantCount: 1,
		},
		{
			name: "add and remove",
			adds: []struct{ indexName, indexValue, resourceKey string }{
				{"app_name", "dubbo-service", "key1"},
			},
			removes: []struct{ indexName, indexValue, resourceKey string }{
				{"app_name", "dubbo-service", "key1"},
			},
			wantQuery: NewQuery("app_name", OperatorEquals, "dubbo-service"),
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := NewMemoryIndex()
			for _, add := range tt.adds {
				idx.Add(add.indexName, add.indexValue, add.resourceKey)
			}
			for _, rm := range tt.removes {
				idx.Remove(rm.indexName, rm.indexValue, rm.resourceKey)
			}
			results, err := idx.Query(tt.wantQuery)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			if len(results) != tt.wantCount {
				t.Errorf("got %d results, want %d", len(results), tt.wantCount)
			}
		})
	}
}

func TestMemoryIndexQuery(t *testing.T) {
	tests := []struct {
		name      string
		adds      []struct{ indexName, indexValue, resourceKey string }
		query     Query
		wantCount int
	}{
		{
			name: "exact match",
			adds: []struct{ indexName, indexValue, resourceKey string }{
				{"app_name", "dubbo-service", "key1"},
				{"app_name", "dubbo-admin", "key2"},
			},
			query:     NewQuery("app_name", OperatorEquals, "dubbo-service"),
			wantCount: 1,
		},
		{
			name: "prefix match",
			adds: []struct{ indexName, indexValue, resourceKey string }{
				{"app_name", "dubbo-service-a", "key1"},
				{"app_name", "dubbo-service-b", "key2"},
				{"app_name", "dubbo-admin", "key3"},
			},
			query:     NewQuery("app_name", OperatorPrefix, "dubbo-service"),
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := NewMemoryIndex()
			for _, add := range tt.adds {
				idx.Add(add.indexName, add.indexValue, add.resourceKey)
			}
			results, err := idx.Query(tt.query)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			if len(results) != tt.wantCount {
				t.Errorf("got %d results, want %d", len(results), tt.wantCount)
			}
		})
	}
}
func TestMemoryIndexConcurrent(t *testing.T) {
	idx := NewMemoryIndex()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			idx.Add("app_name", "dubbo-service", "key1")
		}()
		go func() {
			defer wg.Done()
			idx.Query(NewQuery("app_name", OperatorPrefix, "dubbo"))
		}()
	}

	wg.Wait()
}
