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
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&IndexModel{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func TestNewDBIndex(t *testing.T) {
	db := setupTestDB(t)
	idx := NewDBIndex(db, "Service")
	if idx == nil {
		t.Fatal("NewDBIndex returned nil")
	}
	if idx.db == nil {
		t.Error("db is nil")
	}
	if idx.resourceKind != "Service" {
		t.Errorf("resourceKind = %s, want Service", idx.resourceKind)
	}
	if idx.indexers == nil {
		t.Error("indexers map is nil")
	}
}

func TestDBIndexAddRemove(t *testing.T) {
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
			db := setupTestDB(t)
			idx := NewDBIndex(db, "Service")
			for _, add := range tt.adds {
				if err := idx.Add(add.indexName, add.indexValue, add.resourceKey); err != nil {
					t.Fatalf("Add failed: %v", err)
				}
			}
			for _, rm := range tt.removes {
				if err := idx.Remove(rm.indexName, rm.indexValue, rm.resourceKey); err != nil {
					t.Fatalf("Remove failed: %v", err)
				}
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

func TestDBIndexQuery(t *testing.T) {
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
			db := setupTestDB(t)
			idx := NewDBIndex(db, "Service")
			for _, add := range tt.adds {
				if err := idx.Add(add.indexName, add.indexValue, add.resourceKey); err != nil {
					t.Fatalf("Add failed: %v", err)
				}
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

func TestDBIndexClear(t *testing.T) {
	db := setupTestDB(t)
	idx := NewDBIndex(db, "Service")

	idx.Add("app_name", "dubbo-service", "key1")
	idx.Add("app_name", "dubbo-admin", "key2")

	idx.Clear()

	results, err := idx.Query(NewQuery("app_name", OperatorPrefix, "dubbo"))
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results after Clear, want 0", len(results))
	}
}

func TestDBIndexResourceKindIsolation(t *testing.T) {
	db := setupTestDB(t)
	serviceIdx := NewDBIndex(db, "Service")
	instanceIdx := NewDBIndex(db, "Instance")

	serviceIdx.Add("app_name", "dubbo-service", "service-key1")
	instanceIdx.Add("app_name", "dubbo-service", "instance-key1")

	serviceResults, err := serviceIdx.Query(NewQuery("app_name", OperatorEquals, "dubbo-service"))
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(serviceResults) != 1 || serviceResults[0] != "service-key1" {
		t.Errorf("Service index got %v, want [service-key1]", serviceResults)
	}

	instanceResults, err := instanceIdx.Query(NewQuery("app_name", OperatorEquals, "dubbo-service"))
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(instanceResults) != 1 || instanceResults[0] != "instance-key1" {
		t.Errorf("Instance index got %v, want [instance-key1]", instanceResults)
	}
}
