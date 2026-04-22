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

package discovery

import (
	"fmt"

	"github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/core/controller"
)

var registry = newDiscoveryFactoryRegistry()

func RegisterListWatcherFactory(f Factory) {
	registry.Register(f)
}

func ListWatcherFactoryRegistry() Registry {
	return registry
}

// Factory creates informers for the given type
type Factory interface {
	// Support returns true if the factory can create ListWatchers for the given discovery type
	Support(discovery.Type) bool
	// NewListWatchers creates series of list watchers for the given discovery type
	NewListWatchers(config *discovery.Config) ([]controller.ResourceListerWatcher, error)
}

type Registry interface {
	GetListWatcherFactory(discovery.Type) (Factory, error)
}

type RegistryMutator interface {
	Register(Factory)
}

type MutableRegistry interface {
	Registry
	RegistryMutator
}

var _ MutableRegistry = &discoveryRegistry{}

type discoveryRegistry struct {
	factories []Factory
}

func newDiscoveryFactoryRegistry() MutableRegistry {
	return &discoveryRegistry{
		factories: make([]Factory, 0),
	}
}

func (d *discoveryRegistry) GetListWatcherFactory(t discovery.Type) (Factory, error) {
	for _, factory := range d.factories {
		if factory.Support(t) {
			return factory, nil
		}
	}
	return nil, fmt.Errorf("discovery type %s not supported", t)
}

func (d *discoveryRegistry) Register(factory Factory) {
	d.factories = append(d.factories, factory)
}
