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
	"context"
	"fmt"
	"math"
	"sync/atomic"

	storecfg "github.com/apache/dubbo-admin/pkg/config/store"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/leader"
	"github.com/apache/dubbo-admin/pkg/core/logger"
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
	manager             CounterManager
	leaderElection      *leader.LeaderElection
	needsLeaderElection bool
	isLeader            atomic.Bool
	bound               atomic.Bool
	storeRouter         store.Router
	bus                 events.EventBus
}

func (c *managerComponent) Type() runtime.ComponentType {
	return ComponentType
}

func (c *managerComponent) Order() int {
	return math.MaxInt - 1
}

func (c *managerComponent) Init(ctx runtime.BuilderContext) error {
	mgr := NewCounterManager()
	c.manager = mgr

	// Memory store runs single-replica; leader election is not needed.
	if ctx.Config().Store.Type == storecfg.Memory {
		return nil
	}

	storeComponent, err := ctx.GetActivatedComponent(runtime.ResourceStore)
	if err != nil {
		logger.Warnf("counter: failed to get ResourceStore component, skipping leader election: %v", err)
		return nil
	}
	dbSrc, ok := storeComponent.(leader.DBSource)
	if !ok {
		return nil
	}
	db, hasDB := dbSrc.GetDB()
	if !hasDB {
		return nil
	}
	holderID, err := leader.GenerateHolderID()
	if err != nil {
		logger.Warnf("counter: failed to generate holder ID, skipping leader election: %v", err)
		return nil
	}
	le := leader.NewLeaderElection(db, string(ComponentType), holderID)
	if err := le.EnsureTable(); err != nil {
		logger.Warnf("counter: failed to ensure leader lease table: %v", err)
		return nil
	}
	c.leaderElection = le
	c.needsLeaderElection = true
	logger.Infof("counter: leader election initialized (holder: %s)", holderID)
	return nil
}

func (c *managerComponent) Start(rt runtime.Runtime, ch <-chan struct{}) error {
	storeComponent, err := rt.GetComponent(runtime.ResourceStore)
	if err != nil {
		return err
	}
	storeRouter, ok := storeComponent.(store.Router)
	if !ok {
		return fmt.Errorf("component %s does not implement store.Router", runtime.ResourceStore)
	}
	c.storeRouter = storeRouter

	component, err := rt.GetComponent(runtime.EventBus)
	if err != nil {
		return err
	}
	bus, ok := component.(events.EventBus)
	if !ok {
		return fmt.Errorf("component %s does not implement events.EventBus", runtime.EventBus)
	}
	c.bus = bus

	if !c.needsLeaderElection {
		return c.startBusinessLogic()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-ch
		cancel()
	}()

	c.leaderElection.RunLeaderElection(ctx, ch,
		func() { // onStartLeading
			logger.Infof("counter: became leader, starting business logic")
			c.isLeader.Store(true)
			if err := c.startBusinessLogic(); err != nil {
				logger.Errorf("counter: failed to start business logic: %v", err)
			}
		},
		func() { // onStopLeading
			logger.Warnf("counter: lost leadership, resetting counters")
			c.isLeader.Store(false)
			c.manager.Reset()
		},
	)

	return nil
}

// startBusinessLogic initializes counts from store and binds to EventBus.
// When re-elected, it resets and re-initializes counts; Bind is called only once.
func (c *managerComponent) startBusinessLogic() error {
	c.manager.Reset()
	// Wire up leader guard so event handler skips processing when not leader.
	if c.needsLeaderElection {
		cm := c.manager.(*counterManager)
		cm.isLeader = &c.isLeader
	}
	if err := c.initializeCountsFromStore(c.storeRouter); err != nil {
		logger.Warnf("Failed to initialize counter manager from store: %v", err)
	}
	if !c.bound.Load() {
		if err := c.manager.Bind(c.bus); err != nil {
			return err
		}
		c.bound.Store(true)
	}
	return nil
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
	cm := c.manager.(*counterManager)

	for _, obj := range allResources {
		resource, ok := obj.(resmodel.Resource)
		if !ok {
			continue
		}

		mesh := resource.ResourceMesh()
		if mesh == "" {
			mesh = "default"
		}

		if counter, exists := cm.simpleCounters[kind]; exists {
			counter.Increment(mesh)
		}

		if kind == meshresource.InstanceKind {
			instance, ok := resource.(*meshresource.InstanceResource)
			if ok && instance.Spec != nil {
				protocol := instance.Spec.GetProtocol()
				if protocol != "" {
					if cfg := cm.getDistributionConfig(kind, ProtocolCounter); cfg != nil {
						cfg.counter.Increment(mesh, protocol)
					}
				}

				releaseVersion := instance.Spec.GetReleaseVersion()
				if releaseVersion != "" {
					if cfg := cm.getDistributionConfig(kind, ReleaseCounter); cfg != nil {
						cfg.counter.Increment(mesh, releaseVersion)
					}
				}

				if cfg := cm.getDistributionConfig(kind, DiscoveryCounter); cfg != nil {
					cfg.counter.Increment(mesh, mesh)
				}
			}
		}
	}

	return nil
}

func (c *managerComponent) CounterManager() CounterManager {
	return c.manager
}
