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

package engine

import (
	"fmt"

	enginecfg "github.com/apache/dubbo-admin/pkg/config/engine"
	"github.com/apache/dubbo-admin/pkg/core/controller"
)

var factoryRegistry = newListWatchFactoryRegistry()

func RegisterFactory(f Factory) {
	factoryRegistry.Register(f)
}

func FactoryRegistry() Registry {
	return factoryRegistry
}

// Factory defines if a specific engine type is supported and how to create ListWatchers
type Factory interface {
	Support(enginecfg.Type) bool
	NewListWatchers(*enginecfg.Config) ([]controller.ResourceListerWatcher, error)
}

type Registry interface {
	GetListWatcherFactory(enginecfg.Type) (Factory, error)
}

type RegistryMutator interface {
	// Register a new engine factory
	Register(Factory)
}

type MutableRegistry interface {
	Registry
	RegistryMutator
}

var _ MutableRegistry = &listWatchFactoryRegistry{}

type listWatchFactoryRegistry struct {
	factories []Factory
}

func newListWatchFactoryRegistry() MutableRegistry {
	return &listWatchFactoryRegistry{
		factories: make([]Factory, 0),
	}
}

func (e *listWatchFactoryRegistry) Register(factory Factory) {
	e.factories = append(e.factories, factory)
}

func (e *listWatchFactoryRegistry) GetListWatcherFactory(t enginecfg.Type) (Factory, error) {
	for _, factory := range e.factories {
		if factory.Support(t) {
			return factory, nil
		}
	}
	return nil, fmt.Errorf("engine type %s not supported", t)
}
