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

package lock

import (
	"fmt"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

var registry = newLockFactoryRegistry()

// RegisterLockFactory registers a Lock factory
func RegisterLockFactory(f Factory) {
	registry.Register(f)
}

// LockFactoryRegistry returns the global factory registry
func LockFactoryRegistry() Registry {
	return registry
}

// Factory defines the factory interface for creating Locks
type Factory interface {
	// Support determines whether the factory supports creating a Lock from a given context
	// Determines this by inspecting the components in the context
	Support(ctx runtime.BuilderContext) bool

	// NewLock creates a Lock instance from the BuilderContext
	// The factory decides for itself how to extract dependencies from the context
	NewLock(ctx runtime.BuilderContext) (Lock, error)
}

// Registry defines the query interface for the factory registry
type Registry interface {
	// GetSupportedFactory returns the first supported factory
	GetSupportedFactory(ctx runtime.BuilderContext) (Factory, error)
	// GetAllSupportedFactories returns all supported factories
	GetAllSupportedFactories(ctx runtime.BuilderContext) []Factory
}

// RegistryMutator defines the interface for modifying the factory registry
type RegistryMutator interface {
	Register(Factory)
}

// MutableRegistry combines query and modification interfaces
type MutableRegistry interface {
	Registry
	RegistryMutator
}

var _ MutableRegistry = &lockFactoryRegistry{}

type lockFactoryRegistry struct {
	factories []Factory
}

func newLockFactoryRegistry() MutableRegistry {
	return &lockFactoryRegistry{
		factories: make([]Factory, 0),
	}
}

func (r *lockFactoryRegistry) GetSupportedFactory(ctx runtime.BuilderContext) (Factory, error) {
	for _, factory := range r.factories {
		if factory.Support(ctx) {
			return factory, nil
		}
	}
	return nil, fmt.Errorf("no supported lock factory found")
}

func (r *lockFactoryRegistry) GetAllSupportedFactories(ctx runtime.BuilderContext) []Factory {
	supported := make([]Factory, 0)
	for _, factory := range r.factories {
		if factory.Support(ctx) {
			supported = append(supported, factory)
		}
	}
	return supported
}

func (r *lockFactoryRegistry) Register(factory Factory) {
	r.factories = append(r.factories, factory)
}
