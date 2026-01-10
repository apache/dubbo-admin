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

package governor

import (
	"fmt"

	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

var factoryRegistry = newGovernorFactoryRegistry()

func RegisterFactory(f Factory) {
	factoryRegistry.Register(f)
}

func FactoryRegistry() Registry {
	return factoryRegistry
}

// Factory is the interface for create a specific type of RuleGovernor
type Factory interface {
	// Support returns true if the factory supports the given type in config
	Support(t discoverycfg.Type) bool
	// New returns a new RuleGovernor for the mesh using the given config and other components
	New(mesh string, config *discoverycfg.Config, router store.Router, emitter events.Emitter) (RuleGovernor, error)
}

type Registry interface {
	GetGovernorFactory(discoverycfg.Type) (Factory, error)
}

type RegistryMutator interface {
	// Register registers a new factory
	Register(Factory)
}

type MutableRegistry interface {
	Registry
	RegistryMutator
}

var _ MutableRegistry = &governorFactoryRegistry{}

type governorFactoryRegistry struct {
	factories []Factory
}

func newGovernorFactoryRegistry() MutableRegistry {
	return &governorFactoryRegistry{
		factories: make([]Factory, 0),
	}
}

func (g *governorFactoryRegistry) GetGovernorFactory(t discoverycfg.Type) (Factory, error) {
	for _, factory := range g.factories {
		if factory.Support(t) {
			return factory, nil
		}
	}
	return nil, fmt.Errorf("governor type %s not supported", t)
}

func (g *governorFactoryRegistry) Register(factory Factory) {
	g.factories = append(g.factories, factory)
}
