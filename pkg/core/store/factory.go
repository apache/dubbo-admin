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

package store

import (
	"fmt"

	storecfg "github.com/apache/dubbo-admin/pkg/config/store"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
)

var factoryRegistry = newStoreFactoryRegistry()

func RegisterFactory(f Factory) {
	factoryRegistry.Register(f)
}

func FactoryRegistry() Registry {
	return factoryRegistry
}

// Factory is the interface for create a specific type of ManagedResourceStore
type Factory interface {
	// Support returns true if the factory supports the given type in config
	Support(storecfg.Type) bool
	// New returns a new ManagedResourceStore for the model.ResourceKind using the given config
	New(model.ResourceKind, *storecfg.Config) (ManagedResourceStore, error)
}

type Registry interface {
	GetStoreFactory(storecfg.Type) (Factory, error)
}

type RegistryMutator interface {
	// Register registers a new factory
	Register(Factory)
}
type MutableRegistry interface {
	Registry
	RegistryMutator
}

var _ MutableRegistry = &storeFactoryRegistry{}

type storeFactoryRegistry struct {
	factories []Factory
}

func newStoreFactoryRegistry() MutableRegistry {
	return &storeFactoryRegistry{
		factories: make([]Factory, 0),
	}
}
func (s *storeFactoryRegistry) GetStoreFactory(t storecfg.Type) (Factory, error) {
	for _, factory := range s.factories {
		if factory.Support(t) {
			return factory, nil
		}
	}
	return nil, fmt.Errorf("store type %s not supported", t)
}

func (s *storeFactoryRegistry) Register(factory Factory) {
	s.factories = append(s.factories, factory)
}
