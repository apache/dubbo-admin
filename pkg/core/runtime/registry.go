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
)

var registry = NewRegistry()

func ComponentRegistry() Registry {
	return registry
}

func RegisterComponent(component Component) {
	if err := registry.Register(component); err != nil {
		panic(err)
	}
}

type Registry interface {
	Get(typ ComponentType) (Component, error)
	EventBus() (Component, error)
	Console() (Component, error)
	ResourceStore() (Component, error)
	ResourceManager() (Component, error)
	ResourceDiscovery() (Component, error)
	ResourceEngine() (Component, error)
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
		directory: make(map[ComponentType]Component),
	}
}

var _ MutableRegistry = &componentRegistry{}

type componentRegistry struct {
	directory map[ComponentType]Component
}

func (r *componentRegistry) EventBus() (Component, error) {
	return r.Get(EventBus)
}

func (r *componentRegistry) Console() (Component, error) {
	return r.Get(Console)
}

func (r *componentRegistry) ResourceManager() (Component, error) {
	return r.Get(ResourceManager)
}

func (r *componentRegistry) ResourceStore() (Component, error) {
	return r.Get(ResourceStore)
}

func (r *componentRegistry) ResourceDiscovery() (Component, error) {
	return r.Get(ResourceDiscovery)
}

func (r *componentRegistry) ResourceEngine() (Component, error) {
	return r.Get(ResourceEngine)
}

func (r *componentRegistry) Register(component Component) error {
	_, ok := r.directory[component.Type()]
	if ok {
		return componentAlreadyRegisteredError(component.Type())
	}
	r.directory[component.Type()] = component
	return nil
}

func (r *componentRegistry) Get(typ ComponentType) (Component, error) {
	component, ok := r.directory[typ]
	if !ok {
		return nil, noSuchComponentError(typ)
	}
	return component, nil
}

func noSuchComponentError(typ ComponentType) error {
	return errors.Errorf("there is no available component registered with type %q", typ)
}

func componentAlreadyRegisteredError(typ ComponentType) error {
	return errors.Errorf("component %q has already been registered", typ)
}
