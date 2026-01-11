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

package counter

import (
	"fmt"
	"math"

	"github.com/apache/dubbo-admin/pkg/core/events"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	resmodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

const ComponentType runtime.ComponentType = "counter manager"

func init() {
	runtime.RegisterComponent(&managerComponent{})
}

type ManagerComponent interface {
	runtime.Component
	CounterManager() CounterManager
}

var _ ManagerComponent = &managerComponent{}

func (c *managerComponent) RequiredDependencies() []runtime.ComponentType {
	return []runtime.ComponentType{
		runtime.ResourceStore,
		runtime.EventBus,
	}
}

type managerComponent struct {
	manager CounterManager
}

func (c *managerComponent) Type() runtime.ComponentType {
	return ComponentType
}

func (c *managerComponent) Order() int {
	return math.MaxInt - 1
}

func (c *managerComponent) Init(runtime.BuilderContext) error {
	mgr := NewCounterManager()
	c.manager = mgr
	return nil
}

func (c *managerComponent) Start(rt runtime.Runtime, _ <-chan struct{}) error {
	storeComponent, err := rt.GetComponent(runtime.ResourceStore)
	if err != nil {
		return err
	}
	storeRouter, ok := storeComponent.(store.Router)
	if !ok {
		return fmt.Errorf("component %s does not implement store.Router", runtime.ResourceStore)
	}

	c.initializeCountsFromStore(storeRouter)

	component, err := rt.GetComponent(runtime.EventBus)
	if err != nil {
		return err
	}
	bus, ok := component.(events.EventBus)
	if !ok {
		return fmt.Errorf("component %s does not implement events.EventBus", runtime.EventBus)
	}
	return c.manager.Bind(bus)
}

func (c *managerComponent) initializeCountsFromStore(storeRouter store.Router) error {
	if err := c.initializeResourceCount(storeRouter, meshresource.InstanceKind); err != nil {
		return fmt.Errorf("failed to initialize instance count: %w", err)
	}

	if err := c.initializeResourceCount(storeRouter, meshresource.ApplicationKind); err != nil {
		return fmt.Errorf("failed to initialize application count: %w", err)
	}

	if err := c.initializeResourceCount(storeRouter, meshresource.ServiceProviderMetadataKind); err != nil {
		return fmt.Errorf("failed to initialize service provider metadata count: %w", err)
	}

	return nil
}

func (c *managerComponent) initializeResourceCount(storeRouter store.Router, kind resmodel.ResourceKind) error {
	resourceStore, err := storeRouter.ResourceKindRoute(kind)
	if err != nil {
		return err
	}

	allResources := resourceStore.List()

	meshCounts := make(map[string]int64)
	meshDistributions := make(map[string]map[string]int64)

	for _, obj := range allResources {
		resource, ok := obj.(resmodel.Resource)
		if !ok {
			continue
		}

		mesh := resource.ResourceMesh()
		if mesh == "" {
			mesh = "default"
		}

		meshCounts[mesh]++

		if kind == meshresource.InstanceKind {
			instance, ok := resource.(*meshresource.InstanceResource)
			if ok && instance.Spec != nil {
				protocol := instance.Spec.GetProtocol()
				if protocol != "" {
					if meshDistributions[mesh] == nil {
						meshDistributions[mesh] = make(map[string]int64)
					}
					meshDistributions[mesh]["protocol:"+protocol]++
				}

				releaseVersion := instance.Spec.GetReleaseVersion()
				if releaseVersion != "" {
					if meshDistributions[mesh] == nil {
						meshDistributions[mesh] = make(map[string]int64)
					}
					meshDistributions[mesh]["release:"+releaseVersion]++
				}

				if meshDistributions[mesh] == nil {
					meshDistributions[mesh] = make(map[string]int64)
				}
				meshDistributions[mesh]["discovery:"+mesh]++
			}
		}
	}

	cm := c.manager.(*counterManager)

	if counter, exists := cm.simpleCounters[kind]; exists {
		for mesh, count := range meshCounts {
			for i := int64(0); i < count; i++ {
				counter.Increment(mesh)
			}
		}
	}

	if kind == meshresource.InstanceKind {
		for mesh, distributions := range meshDistributions {
			for key, count := range distributions {
				if len(key) > 9 && key[:9] == "protocol:" {
					if cfg := cm.getDistributionConfig(kind, ProtocolCounter); cfg != nil {
						for i := int64(0); i < count; i++ {
							cfg.counter.Increment(mesh, key[9:])
						}
					}
				} else if len(key) > 8 && key[:8] == "release:" {
					if cfg := cm.getDistributionConfig(kind, ReleaseCounter); cfg != nil {
						for i := int64(0); i < count; i++ {
							cfg.counter.Increment(mesh, key[8:])
						}
					}
				} else if len(key) > 11 && key[:11] == "discovery:" {
					if cfg := cm.getDistributionConfig(kind, DiscoveryCounter); cfg != nil {
						for i := int64(0); i < count; i++ {
							cfg.counter.Increment(mesh, key[11:])
						}
					}
				}
			}
		}
	}

	return nil
}

func (c *managerComponent) CounterManager() CounterManager {
	return c.manager
}
