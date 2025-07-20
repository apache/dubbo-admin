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

package collection

import (
	"fmt"

	"github.com/apache/dubbo-admin/operator/pkg/config"
	"github.com/apache/dubbo-admin/operator/pkg/schema"
)

// Schemas contains metadata about configuration resources.
type Schemas struct {
	byCollection map[config.GroupVersionKind]schema.Schema
	byAddOrder   []schema.Schema
}

// FindByGroupVersionAliasesKind searches and returns the first schema with the given GVK,
// if not found, it will search for version aliases for the schema to see if there is a match.
func (s Schemas) FindByGroupVersionAliasesKind(gvk config.GroupVersionKind) (schema.Schema, bool) {
	for _, rs := range s.byAddOrder {
		for _, va := range rs.GroupVersionAliasKinds() {
			if va == gvk {
				return rs, true
			}
		}
	}
	return nil, false
}

// SchemasBuilder is a builder for the schemas type.
type SchemasBuilder struct {
	schemas Schemas
}

// NewSchemasBuilder returns a new instance of SchemasBuilder.
func NewSchemasBuilder() *SchemasBuilder {
	s := Schemas{
		byCollection: make(map[config.GroupVersionKind]schema.Schema),
	}
	return &SchemasBuilder{schemas: s}
}

// Add a new collection to the schemas.
func (b *SchemasBuilder) Add(s schema.Schema) error {
	if _, found := b.schemas.byCollection[s.GroupVersionKind()]; found {
		return fmt.Errorf("collection already exists: %v", s.GroupVersionKind())
	}
	b.schemas.byCollection[s.GroupVersionKind()] = s
	b.schemas.byAddOrder = append(b.schemas.byAddOrder, s)
	return nil
}

// MustAdd calls Add and panics if it fails.
func (b *SchemasBuilder) MustAdd(s schema.Schema) *SchemasBuilder {
	if err := b.Add(s); err != nil {
		panic(fmt.Sprintf("SchemasBuilder.MustAdd: %v", err))
	}
	return b
}

// Build a new schemas from this SchemasBuilder.
func (b *SchemasBuilder) Build() Schemas {
	s := b.schemas
	b.schemas = Schemas{}
	return s
}
