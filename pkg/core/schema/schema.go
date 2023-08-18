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

package schema

import (
	"fmt"

	"github.com/apache/dubbo-admin/pkg/core/schema/ast"
	"github.com/apache/dubbo-admin/pkg/core/schema/collection"
	"github.com/apache/dubbo-admin/pkg/core/schema/resource"
	resource2 "github.com/apache/dubbo-admin/pkg/core/tools/resource"
	"github.com/apache/dubbo-admin/pkg/core/validation"
	"github.com/google/go-cmp/cmp"
)

type Metadata struct {
	collections collection.Schemas
}

// AllCollections is all known collections
func (m *Metadata) AllCollections() collection.Schemas { return m.collections }

func (m *Metadata) Equal(o *Metadata) bool {
	return cmp.Equal(m.collections, o.collections)
}

// ParseAndBuild parses the given metadata file and returns the strongly typed schema.
func ParseAndBuild(yamlText string) (*Metadata, error) {
	mast, err := ast.Parse(yamlText)
	if err != nil {
		return nil, err
	}

	return Build(mast)
}

// Build strongly-typed Metadata from parsed AST.
func Build(astm *ast.Metadata) (*Metadata, error) {
	resourceKey := func(group, kind string) string {
		return group + "/" + kind
	}

	resources := make(map[string]resource.Schema)
	for i, ar := range astm.Resources {
		if ar.Kind == "" {
			return nil, fmt.Errorf("resource %d missing type", i)
		}
		if ar.Plural == "" {
			return nil, fmt.Errorf("resource %d missing type", i)
		}
		if ar.Version == "" {
			return nil, fmt.Errorf("resource %d missing type", i)
		}
		if ar.Proto == "" {
			return nil, fmt.Errorf("resource %d missing type", i)
		}
		if ar.Validate == "" {
			validateFn := "Validate" + resource2.CamelCase(ar.Kind)
			if !validation.IsValidateFunc(validateFn) {
				validateFn = "EmptyValidate"
			}
			ar.Validate = validateFn
		}
		validateFn := validation.GetValidateFunc(ar.Validate)
		if validateFn == nil {
			return nil, fmt.Errorf("failed locating proto validation function %s", ar.Validate)
		}

		r := resource.Builder{
			ClusterScoped: ar.ClusterScoped,
			Kind:          ar.Kind,
			Plural:        ar.Plural,
			Group:         ar.Group,
			Version:       ar.Version,
			Proto:         ar.Proto,
			ValidateProto: validateFn,
		}.BuildNoValidate()

		key := resourceKey(ar.Group, ar.Kind)
		if _, ok := resources[key]; ok {
			return nil, fmt.Errorf("found duplicate resource for resource (%s)", key)
		}
		resources[key] = r
	}

	cBuilder := collection.NewSchemasBuilder()
	for _, c := range astm.Collections {
		key := resourceKey(c.Group, c.Kind)
		r, found := resources[key]
		if !found {
			return nil, fmt.Errorf("failed locating resource (%s) for collection %s", key, c.Name)
		}

		s, err := collection.Builder{
			Name:     c.Name,
			Resource: r,
		}.Build()
		if err != nil {
			return nil, err
		}

		if err = cBuilder.Add(s); err != nil {
			return nil, err
		}
	}

	collections := cBuilder.Build()

	return &Metadata{
		collections: collections,
	}, nil
}
