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
	"encoding/json"
	"strings"
	"time"
	"unicode"

	"gorm.io/gorm"

	"github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// ResourceModel is the database model for storing Dubbo resources
// It uses dynamic table naming based on ResourceKind to improve query performance
// Note: TableName() method is intentionally removed as GORM caches it.
// Use TableScope() instead for dynamic table names.
type ResourceModel struct {
	ID           uint      `gorm:"primarykey"`                             // Auto-incrementing primary key
	ResourceKey  string    `gorm:"type:varchar(255);uniqueIndex;not null"` // Unique identifier for the resource
	ResourceKind string    `gorm:"not null"`                               // Type of resource (e.g., "Application", "ServiceProviderMapping")
	Name         string    `gorm:"index;not null"`                         // Resource name, indexed for fast lookups
	Mesh         string    `gorm:"index;not null"`                         // Mesh identifier, indexed for filtering by mesh
	Data         []byte    `gorm:"type:text;not null"`                     // JSON-encoded resource data
	CreatedAt    time.Time `gorm:"autoCreateTime"`                         // Automatically set on creation
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`                         // Automatically updated on modification
}

// TableNameForKind returns the table name for a given ResourceKind
// Uses dynamic table naming: each ResourceKind gets its own table
// e.g., "Application" -> "resources_application", "ServiceProviderMapping" -> "resources_service_provider_mapping"
func TableNameForKind(kind string) string {
	if kind == "" {
		return "resources"
	}
	// Convert ResourceKind to snake_case table name with "resources_" prefix
	return "resources_" + toSnakeCase(kind)
}

// TableScope returns a GORM scope function that sets the table name dynamically
// This is the recommended approach for dynamic table names as TableName() is cached by GORM
// Usage: db.Scopes(TableScope(kind)).Find(&models)
func TableScope(kind string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Table(TableNameForKind(kind))
	}
}

// toSnakeCase converts a string to snake_case
// e.g., "ServiceProviderMapping" -> "service_provider_mapping", "RPCInstance" -> "rpc_instance"
func toSnakeCase(s string) string {
	var result strings.Builder
	runes := []rune(s)

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		if unicode.IsUpper(r) {
			// Add underscore before uppercase letter if:
			// 1. Not at the beginning
			// 2. Previous char is lowercase or
			// 3. Next char exists and is lowercase (for handling acronyms like "RPCInstance")
			if i > 0 {
				prevIsLower := unicode.IsLower(runes[i-1])
				nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])

				if prevIsLower || nextIsLower {
					result.WriteRune('_')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToResource converts the database model back to a Resource object
// Unmarshals the JSON data and returns the typed resource
func (rm *ResourceModel) ToResource() (model.Resource, error) {
	newFunc, err := model.ResourceSchemaRegistry().NewResourceFunc(model.ResourceKind(rm.ResourceKind))
	if err != nil {
		return nil, err
	}
	resource := newFunc()
	if err := json.Unmarshal(rm.Data, resource); err != nil {
		return nil, err
	}
	return resource, nil
}

// FromResource converts a Resource object to a database model
// Marshals the resource to JSON and populates the model fields
func FromResource(resource model.Resource) (*ResourceModel, error) {
	data, err := json.Marshal(resource)
	if err != nil {
		return nil, err
	}

	return &ResourceModel{
		ResourceKey:  resource.ResourceKey(),
		ResourceKind: resource.ResourceKind().ToString(),
		Name:         resource.ResourceMeta().Name,
		Mesh:         resource.MeshName(),
		Data:         data,
	}, nil
}
