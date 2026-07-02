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
	"fmt"
	"math"
	"reflect"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	versioningcfg "github.com/apache/dubbo-admin/pkg/config/versioning"
	"github.com/apache/dubbo-admin/pkg/core/governor"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store"
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
	store   *ResourceStoreAdapter
}

func (c *component) Type() runtime.ComponentType {
	return ComponentType
}

func (c *component) Order() int {
	return math.MaxInt - 5
}

func (c *component) RequiredDependencies() []runtime.ComponentType {
	return []runtime.ComponentType{
		runtime.ResourceStore,
	}
}

func (c *component) Init(ctx runtime.BuilderContext) error {
	cfg := ctx.Config().RuleVersioning
	if cfg == nil {
		cfg = versioningcfg.Default()
	}

	storeComponent, err := ctx.GetActivatedComponent(runtime.ResourceStore)
	if err != nil {
		return err
	}
	storeRouter, ok := storeComponent.(store.Router)
	if !ok {
		return bizerror.NewAssertionError("store.Router", reflect.TypeOf(storeComponent).Name())
	}
	rvStore, err := storeRouter.ResourceKindRoute(meshresource.RuleVersionKind)
	if err != nil {
		return fmt.Errorf("failed to get RuleVersion store: %w", err)
	}
	if rvStore == nil {
		return fmt.Errorf("RuleVersion store not available - versioning requires resource store")
	}

	store := NewResourceStoreAdapter(rvStore)
	if err := store.ensureStores(); err != nil {
		return err
	}
	c.store = store
	c.service = NewService(cfg.MaxVersionsPerRule, store)
	logger.Infof("Using resource store for rule history (RuleVersion)")
	return nil
}

func (c *component) Start(rt runtime.Runtime, stop <-chan struct{}) error {
	cfg := rt.Config().RuleVersioning
	if cfg == nil {
		cfg = versioningcfg.Default()
	}
	ctx, cancel := contextWithStop(rt.AppContext(), stop)
	defer cancel()
	storeComponent, err := rt.GetComponent(runtime.ResourceStore)
	if err != nil {
		return err
	}
	storeRouter, ok := storeComponent.(store.Router)
	if !ok {
		return bizerror.NewAssertionError("store.Router", reflect.TypeOf(storeComponent).Name())
	}
	if err := c.bootstrapExistingRules(ctx, storeRouter, cfg.MaxVersionsPerRule); err != nil {
		logger.Warnf("rule history bootstrap failed: %v", err)
	}
	return nil
}

func (c *component) Service() *Service {
	return c.service
}

func (c *component) bootstrapExistingRules(ctx context.Context, storeRouter store.Router, maxVersions int64) error {
	for _, kind := range governor.RuleResourceKinds.Values() {
		if err := ctx.Err(); err != nil {
			return err
		}
		rs, err := storeRouter.ResourceKindRoute(kind)
		if err != nil {
			return err
		}
		if rs == nil {
			continue
		}
		resources, err := rs.GetByKeys(rs.ListKeys())
		if err != nil {
			return err
		}
		for _, res := range resources {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := RecordBootstrapState(ctx, c.store, maxVersions, res); err != nil {
				logger.Warnf("rule history bootstrap skipped for %s: %v", res.ResourceKey(), err)
			}
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
