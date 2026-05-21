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
	"fmt"
	"math"
	"time"

	versioningcfg "github.com/apache/dubbo-admin/pkg/config/versioning"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/governor"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"gorm.io/gorm"
)

const ComponentType runtime.ComponentType = "rule versioning"

func init() {
	runtime.RegisterComponent(&component{})
}

type Component interface {
	runtime.Component
	Service() Service
}

type component struct {
	service     Service
	store       Store
	subscribers []*Subscriber
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
	cfg := ctx.Config().Versioning
	if cfg == nil {
		cfg = versioningcfg.Default()
	}
	hints := NewAdminHintRegistry()
	storeComponent, err := ctx.GetActivatedComponent(runtime.ResourceStore)
	if err != nil {
		return err
	}
	store := Store(NewMemoryStore())
	if sc, ok := storeComponent.(interface {
		GetDB() (*gorm.DB, bool)
	}); ok {
		if db, exists := sc.GetDB(); exists {
			gormStore := NewGormStore(db)
			if err := gormStore.AutoMigrate(); err != nil {
				return err
			}
			store = gormStore
		}
	}
	c.store = store
	c.service = NewServiceWithRollbackWait(
		cfg.Enabled,
		cfg.MaxVersionsPerRule,
		time.Duration(cfg.CoalesceWindowMs)*time.Millisecond,
		time.Duration(cfg.AdminHintTTLSec)*time.Second,
		time.Duration(cfg.RollbackWaitMs)*time.Millisecond,
		store,
		hints,
	)
	if !cfg.Enabled {
		return nil
	}
	eventBusComponent, err := ctx.GetActivatedComponent(runtime.EventBus)
	if err != nil {
		return err
	}
	bus, ok := eventBusComponent.(events.EventBus)
	if !ok {
		return fmt.Errorf("component %s does not implement events.EventBus", runtime.EventBus)
	}
	for _, kind := range governor.RuleResourceKinds.Values() {
		sub := NewSubscriber(kind, store, hints, cfg.MaxVersionsPerRule, time.Duration(cfg.CoalesceWindowMs)*time.Millisecond)
		if err := bus.Subscribe(sub); err != nil {
			return err
		}
		c.subscribers = append(c.subscribers, sub)
	}
	return nil
}

func (c *component) Start(rt runtime.Runtime, _ <-chan struct{}) error {
	cfg := rt.Config().Versioning
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
	for _, kind := range governor.RuleResourceKinds.Values() {
		resources, err := rm.List(kind)
		if err != nil {
			return err
		}
		for _, res := range resources {
			if err := RecordBootstrap(c.store, cfg.MaxVersionsPerRule, res); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *component) Service() Service {
	return c.service
}
