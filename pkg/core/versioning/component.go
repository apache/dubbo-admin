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

package versioning

import (
	"context"
	"errors"
	"fmt"
	"math"

	versioningcfg "github.com/apache/dubbo-admin/pkg/config/versioning"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/governor"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

const ComponentType runtime.ComponentType = "rule versioning"

func init() {
	runtime.RegisterComponent(&component{})
}

type Component interface {
	runtime.Component
	Service() *Service
}

type component struct {
	service *Service
	store   Store
	lock    lock.Lock
}

func (c *component) Type() runtime.ComponentType {
	return ComponentType
}

func (c *component) Order() int {
	return math.MaxInt - 5
}

func (c *component) RequiredDependencies() []runtime.ComponentType {
	return []runtime.ComponentType{
		runtime.EventBus,
		lock.DistributedLockComponent,
		runtime.ResourceStore,
		runtime.ResourceManager,
	}
}

func (c *component) Init(ctx runtime.BuilderContext) error {
	cfg := ctx.Config().RuleVersioning
	if cfg == nil {
		cfg = versioningcfg.Default()
	}

	if !cfg.Enabled {
		// If versioning is disabled, no need to set up store
		c.service = NewService(false, 0, nil)
		return nil
	}

	// Get ResourceManager for ResourceStoreAdapter
	rmComponent, err := ctx.GetActivatedComponent(runtime.ResourceManager)
	if err != nil {
		return err
	}
	rm := rmComponent.(manager.ResourceManagerComponent).ResourceManager()

	// Get resource stores for each versioning resource kind. ResourceStore is
	// routed by kind, so RuleVersion, RuleIntent, and RuleMeta cannot share one
	// store instance.
	rvStore, err := rm.GetStore(meshresource.RuleVersionKind)
	if err != nil {
		return fmt.Errorf("failed to get RuleVersion store: %w", err)
	}
	if rvStore == nil {
		return fmt.Errorf("RuleVersion store not available - versioning requires resource store")
	}
	intentStore, err := rm.GetStore(meshresource.RuleIntentKind)
	if err != nil {
		return fmt.Errorf("failed to get RuleIntent store: %w", err)
	}
	if intentStore == nil {
		return fmt.Errorf("RuleIntent store not available - versioning requires resource store")
	}
	metaStore, err := rm.GetStore(meshresource.RuleMetaKind)
	if err != nil {
		return fmt.Errorf("failed to get RuleMeta store: %w", err)
	}
	if metaStore == nil {
		return fmt.Errorf("RuleMeta store not available - versioning requires resource store")
	}

	store := NewResourceStoreAdapter(rvStore, intentStore, metaStore)
	lockComponent, err := ctx.GetActivatedComponent(lock.DistributedLockComponent)
	if err != nil {
		return fmt.Errorf("rule versioning requires a lock component when enabled: %w", err)
	}
	lockComp, ok := lockComponent.(*lock.Component)
	if !ok {
		return fmt.Errorf("component %s does not implement lock component", lock.DistributedLockComponent)
	}
	lockMgr := lockComp.GetLock()
	if lockMgr == nil {
		return fmt.Errorf("rule versioning requires an available lock implementation when enabled")
	}
	c.store = store
	c.lock = lockMgr
	c.service = NewService(
		cfg.Enabled,
		cfg.MaxVersionsPerRule,
		store,
	)
	logger.Infof("Using resource store for rule versioning (RuleVersion, RuleIntent, RuleMeta)")

	eventBusComponent, err := ctx.GetActivatedComponent(runtime.EventBus)
	if err != nil {
		return err
	}
	bus, ok := eventBusComponent.(events.EventBus)
	if !ok {
		return fmt.Errorf("component %s does not implement events.EventBus", runtime.EventBus)
	}
	for _, kind := range governor.RuleResourceKinds.Values() {
		sub := NewSubscriber(kind, store, cfg.MaxVersionsPerRule, lockMgr)
		if err := bus.Subscribe(sub); err != nil {
			return err
		}
	}
	return nil
}

func (c *component) Start(rt runtime.Runtime, stop <-chan struct{}) error {
	cfg := rt.Config().RuleVersioning
	if cfg == nil {
		cfg = versioningcfg.Default()
	}
	if !cfg.Enabled {
		return nil
	}
	rmComp, err := rt.GetComponent(runtime.ResourceManager)
	if err != nil {
		return err
	}
	rm := rmComp.(manager.ResourceManagerComponent).ResourceManager()
	// Repair open intents left by crashes or failed mutations.
	// Why: If admin crashed after creating an intent but before the subscriber
	// committed it, the intent stays PENDING forever and blocks future writes.
	// Repair attempts to commit intents whose desired state matches actual state.
	if err := c.repairOpenIntents(rm); err != nil {
		return err
	}
	// Bootstrap: record initial version for all existing rules.
	// Why: Versioning was just enabled or this is the first startup. Existing rules
	// have no version history. Recording a BOOTSTRAP version establishes a baseline
	// so future mutations have a proper "before" state to diff against.
	for _, kind := range governor.RuleResourceKinds.Values() {
		// Get the store for this kind and list all resources
		rs, err := rm.GetStore(kind)
		if err != nil {
			return err
		}
		if rs == nil {
			// Store not available (e.g., in test), skip bootstrap for this kind
			continue
		}
		keys := rs.ListKeys()
		resources, err := rs.GetByKeys(keys)
		if err != nil {
			return err
		}
		for _, res := range resources {
			if err := RecordBootstrapLocked(c.store, cfg.MaxVersionsPerRule, res.ResourceKind(), res.ResourceKey(), rm, c.lock); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *component) Service() *Service {
	return c.service
}

func (c *component) repairOpenIntents(rm manager.ResourceManager) error {
	intents, err := c.store.ListOpenIntents()
	if err != nil {
		return err
	}
	for _, intent := range intents {
		err = withRuleVersionLock(c.lock, intent.RuleKind, intent.ResourceKey, func(leaseCtx context.Context) error {
			freshIntent, err := c.service.GetIntent(intent.ID)
			if err != nil {
				return err
			}
			current, exists, err := rm.GetByKey(freshIntent.RuleKind, freshIntent.ResourceKey)
			if err != nil {
				return err
			}
			_, err = c.service.FinalizeMutation(leaseCtx, freshIntent, current, !exists)
			return err
		})
		if err != nil {
			if errors.Is(err, ErrVersionIntentNotFound) {
				continue
			}
			if errors.Is(err, ErrVersionIntentPending) || errors.Is(err, ErrIntentOutcomeMismatch) {
				logger.Warnf("rule version intent %d cannot be repaired automatically for %s: %v", intent.ID, intent.ResourceKey, err)
				continue
			}
			return err
		}
	}
	return nil
}
