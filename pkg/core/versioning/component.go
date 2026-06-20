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
	"time"

	versioningcfg "github.com/apache/dubbo-admin/pkg/config/versioning"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/governor"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
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
	service           *Service
	store             Store
	lock              lock.Lock
	reconcileRequests chan struct{}
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
		runtime.ResourceStore,
		runtime.ResourceManager,
	}
}

func (c *component) Init(ctx runtime.BuilderContext) error {
	cfg := ctx.Config().RuleVersioning
	if cfg == nil {
		cfg = versioningcfg.Default()
	}

	rmComponent, err := ctx.GetActivatedComponent(runtime.ResourceManager)
	if err != nil {
		return err
	}
	rm := rmComponent.(manager.ResourceManagerComponent).ResourceManager()

	// Get resource stores for each versioning resource kind. ResourceStore is
	// routed by kind, so RuleVersion and RuleIntent cannot share one store
	// instance.
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

	store := NewResourceStoreAdapter(rvStore, intentStore)
	if err := store.ensureStores(); err != nil {
		return err
	}
	lockComponent, err := ctx.GetActivatedComponent(lock.DistributedLockComponent)
	if err != nil {
		return fmt.Errorf("rule versioning requires a lock component: %w", err)
	}
	lockComp, ok := lockComponent.(*lock.Component)
	if !ok {
		return fmt.Errorf("component %s does not implement lock component", lock.DistributedLockComponent)
	}
	lockMgr := lockComp.GetLock()
	if lockMgr == nil {
		return fmt.Errorf("rule versioning requires an available lock implementation")
	}
	c.store = store
	c.lock = lockMgr
	c.reconcileRequests = make(chan struct{}, 1)
	c.service = NewService(
		cfg.MaxVersionsPerRule,
		store,
	)
	logger.Infof("Using resource store for rule versioning (RuleVersion, RuleIntent)")

	eventBusComponent, err := ctx.GetActivatedComponent(runtime.EventBus)
	if err != nil {
		return err
	}
	bus, ok := eventBusComponent.(events.EventBus)
	if !ok {
		return fmt.Errorf("component %s does not implement events.EventBus", runtime.EventBus)
	}
	for _, kind := range governor.RuleResourceKinds.Values() {
		sub := NewSubscriber(kind, store, cfg.MaxVersionsPerRule, lockMgr, ctx.AppContext(), c.requestReconcile)
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
	startCtx, cancel := contextWithStop(rt.AppContext(), stop)
	defer cancel()
	rmComp, err := rt.GetComponent(runtime.ResourceManager)
	if err != nil {
		return err
	}
	rm := rmComp.(manager.ResourceManagerComponent).ResourceManager()
	// Startup repair resolves durable intents left by crashes before bootstrap
	// records current rules, so a stale intent cannot fence all later writes.
	if err := c.repairOpenIntents(startCtx, rm); err != nil {
		return err
	}
	if err := c.cleanupTerminalIntents(startCtx); err != nil {
		return err
	}
	if err := c.bootstrapExistingRules(startCtx, rm, cfg.MaxVersionsPerRule); err != nil {
		return err
	}
	c.startReconcileLoop(rt.AppContext(), stop, rm, cfg.MaxVersionsPerRule)
	return nil
}

func (c *component) Service() *Service {
	return c.service
}

func (c *component) bootstrapExistingRules(ctx context.Context, rm manager.ResourceManager, maxVersions int64) error {
	currentKeys := make(map[coremodel.ResourceKind]map[string]struct{})
	for _, kind := range governor.RuleResourceKinds.Values() {
		if err := ctx.Err(); err != nil {
			return err
		}
		rs, err := rm.GetStore(kind)
		if err != nil {
			return err
		}
		if rs == nil {
			continue
		}
		keys := rs.ListKeys()
		currentKeys[kind] = make(map[string]struct{}, len(keys))
		for _, key := range keys {
			currentKeys[kind][key] = struct{}{}
		}
		resources, err := rs.GetByKeys(keys)
		if err != nil {
			return err
		}
		for _, res := range resources {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := RecordBootstrapLocked(ctx, c.store, maxVersions, res.ResourceKind(), res.ResourceKey(), rm, c.lock); err != nil {
				return err
			}
		}
	}
	return c.reconcileDeletedRules(ctx, rm, currentKeys)
}

func (c *component) reconcileDeletedRules(ctx context.Context, rm manager.ResourceManager, currentKeys map[coremodel.ResourceKind]map[string]struct{}) error {
	for _, kind := range governor.RuleResourceKinds.Values() {
		if err := ctx.Err(); err != nil {
			return err
		}
		latest, err := c.store.ListLatestVersions(kind)
		if err != nil {
			return err
		}
		for _, head := range latest {
			if err := ctx.Err(); err != nil {
				return err
			}
			if head.Operation == OperationDelete {
				continue
			}
			if _, exists := currentKeys[kind][head.ResourceKey]; exists {
				continue
			}
			err := withRuleVersionLock(ctx, c.lock, kind, head.ResourceKey, func(leaseCtx context.Context) error {
				intent, err := c.store.OpenIntent(kind, head.ResourceKey)
				if err != nil {
					return err
				}
				if intent != nil {
					return nil
				}
				current, exists, err := rm.GetByKey(kind, head.ResourceKey)
				if err != nil {
					return err
				}
				if exists {
					return nil
				}
				_, err = c.service.ReconcileActualState(leaseCtx, kind, head.ResourceKey, current, true, "system:reconcile")
				return err
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *component) requestReconcile() {
	if c == nil || c.reconcileRequests == nil {
		return
	}
	select {
	case c.reconcileRequests <- struct{}{}:
	default:
	}
}

func (c *component) startReconcileLoop(parent context.Context, stop <-chan struct{}, rm manager.ResourceManager, maxVersions int64) {
	if c.reconcileRequests == nil {
		c.reconcileRequests = make(chan struct{}, 1)
	}
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-parent.Done():
				return
			case <-c.reconcileRequests:
			case <-ticker.C:
			}
			ctx, cancel := contextWithStop(parent, stop)
			if err := c.bootstrapExistingRules(ctx, rm, maxVersions); err != nil {
				logger.Warnf("rule version current-state reconcile failed: %v", err)
			}
			cancel()
		}
	}()
}

func (c *component) repairOpenIntents(ctx context.Context, rm manager.ResourceManager) error {
	intents, err := c.store.ListOpenIntents()
	if err != nil {
		return err
	}
	for _, intent := range intents {
		if err := ctx.Err(); err != nil {
			return err
		}
		err = withRuleVersionLock(ctx, c.lock, intent.RuleKind, intent.ResourceKey, func(leaseCtx context.Context) error {
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

func (c *component) cleanupTerminalIntents(ctx context.Context) error {
	intents, err := c.store.ListTerminalIntents()
	if err != nil {
		return err
	}
	for _, intent := range intents {
		if err := ctx.Err(); err != nil {
			return err
		}
		err = withRuleVersionLock(ctx, c.lock, intent.RuleKind, intent.ResourceKey, func(leaseCtx context.Context) error {
			if err := lock.CheckLease(leaseCtx); err != nil {
				return err
			}
			return c.store.CleanupIntent(intent.ID, intent.Status)
		})
		if err != nil {
			if errors.Is(err, ErrVersionIntentNotFound) {
				continue
			}
			if errors.Is(err, ErrVersionNotFound) && intent.Status == IntentStatusCommitted {
				return fmt.Errorf("%w: terminal committed intent %d has no RuleVersion", ErrVersionLedgerCorrupt, intent.ID)
			}
			return err
		}
	}
	return nil
}

func contextWithStop(parent context.Context, stop <-chan struct{}) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	go func() {
		select {
		case <-stop:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}
