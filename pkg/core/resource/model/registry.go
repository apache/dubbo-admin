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

package model

import (
	"fmt"

	"github.com/duke-git/lancet/v2/maputil"
)

var registry = newRegistry()

func ResourceSchemaRegistry() Registry {
	return registry
}

func RegisterResourceSchema(kind ResourceKind, newFunc NewResourceFunc, newListFunc NewResourceListFunc) {
	registry.Register(kind, newFunc, newListFunc)
}

type NewResourceFunc func() Resource

type NewResourceListFunc func() ResourceList

type Registry interface {
	// AllResourceKinds returns all registered resource resourceDir
	AllResourceKinds() []ResourceKind
	// NewResourceFunc returns the new resource function for a resource kind
	NewResourceFunc(kind ResourceKind) (NewResourceFunc, error)
	// NewResourceListFunc returns the new resource list function for a resource kind
	NewResourceListFunc(kind ResourceKind) (NewResourceListFunc, error)
}

type RegistryMutator interface {
	// Register registers a resource kind, if a resource kind has been registered before, it will be ignored
	Register(kind ResourceKind, newFunc NewResourceFunc, newListFunc NewResourceListFunc)
}

type MutableRegistry interface {
	Registry
	RegistryMutator
}

var _ MutableRegistry = &resourceSchemaRegistry{}

func newRegistry() MutableRegistry {
	return &resourceSchemaRegistry{
		resourceDir: make(map[ResourceKind]resourceSchema),
	}
}

type resourceSchema struct {
	rk          ResourceKind
	newFunc     NewResourceFunc
	newListFunc NewResourceListFunc
}

type resourceSchemaRegistry struct {
	resourceDir map[ResourceKind]resourceSchema
}

func (r *resourceSchemaRegistry) AllResourceKinds() []ResourceKind {
	return maputil.Keys(r.resourceDir)
}

func (r *resourceSchemaRegistry) Register(kind ResourceKind, newFunc NewResourceFunc, newListFunc NewResourceListFunc) {
	r.resourceDir[kind] = resourceSchema{
		rk:          kind,
		newFunc:     newFunc,
		newListFunc: newListFunc,
	}
}

func (r *resourceSchemaRegistry) NewResourceFunc(kind ResourceKind) (NewResourceFunc, error) {
	if rs, exists := r.resourceDir[kind]; exists {
		return rs.newFunc, nil
	}
	return nil, fmt.Errorf("there is no new resource func for %s", kind)
}

func (r *resourceSchemaRegistry) NewResourceListFunc(kind ResourceKind) (NewResourceListFunc, error) {
	if rs, exists := r.resourceDir[kind]; exists {
		return rs.newListFunc, nil
	}
	return nil, fmt.Errorf("there is no new resource list func for %s", kind)
}
