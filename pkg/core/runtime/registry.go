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

package runtime

import (
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
)

var global = NewRegistry()

func ComponentRegistry() Registry {
	return global
}

func RegisterComponent(component Component) {
	if err := global.Register(component); err != nil {
		panic(err)
	}
}

type Registry interface {
	Get(typ ComponentType, subType ComponentSubType) (Component, error)
	Console(subType ComponentSubType) (Component, error)
	ResourceStore(subType ComponentSubType) (Component, error)
	ResourceManager(subType ComponentSubType) (Component, error)
	ResourceDiscovery(subType ComponentSubType) (Component, error)
	ResourceEngine(subType ComponentSubType)	 (Component, error)
}

type RegistryMutator interface {
	Register(Component) error
}

type MutableRegistry interface {
	Registry
	RegistryMutator
}

func NewRegistry() MutableRegistry {
	return &componentRegistry{
		directory: make(map[ComponentType][]Component),
	}
}

var _ MutableRegistry = &componentRegistry{}

type componentRegistry struct {
	directory map[ComponentType][]Component
}

func (r *componentRegistry) Console(subType ComponentSubType) (Component, error) {
	return r.Get(Console, subType)
}

func (r *componentRegistry) ResourceManager(subType ComponentSubType) (Component, error) {
	return r.Get(ResourceManager, subType)
}

func (r *componentRegistry) ResourceStore(subType ComponentSubType) (Component, error) {
	return r.Get(ResourceStore, subType)
}

func (r *componentRegistry) ResourceDiscovery(subType ComponentSubType) (Component, error) {
	return r.Get(ResourceDiscovery, subType)
}

func (r *componentRegistry) ResourceEngine(subType ComponentSubType) (Component, error) {
	return r.Get(ResourceEngine, subType)
}

func (r *componentRegistry) Register(component Component) error {
	components, ok := r.directory[component.Type()]
	if !ok {
		// if already registered
		if slices.ContainsFunc(components, func(c Component) bool {
			return c.SubType() == component.SubType()
		}) {
			return componentAlreadyRegisteredError(component.Type(), component.SubType())
		}
		// if not registered
		r.directory[component.Type()] = append(components, component)
		return nil
	}
	r.directory[component.SubType()] = []Component{component}
	return nil
}

func (r *componentRegistry) Get(typ ComponentType, subType ComponentSubType) (Component, error) {
	components, ok := r.directory[typ]
	if !ok {
		return nil, noSuchComponentError(typ, subType)
	}
	index := slices.IndexFunc(components, func(component Component) bool {
		return component.Type() == typ && component.SubType() == subType
	})
	if index == -1 {
		return nil, noSuchComponentError(typ, subType)
	}
	return components[index], nil
}

func noSuchComponentError(typ ComponentType, subType ComponentSubType) error {
	return errors.Errorf("there is no available component registered with type %q and subtype %q", typ, subType)
}

func componentAlreadyRegisteredError(typ ComponentType, subType ComponentSubType) error {
	return errors.Errorf("component %q with subType %q has already been registered", typ, subType)
}
