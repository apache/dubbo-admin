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
	"math"
	"reflect"

	"gorm.io/gorm"

	"github.com/apache/dubbo-admin/pkg/core/leader"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

func init() {
	runtime.RegisterComponent(newStoreComponent())
}

type Router interface {
	ResourceRoute(coremodel.Resource) (ResourceStore, error)
	ResourceKindRoute(k coremodel.ResourceKind) (ResourceStore, error)
}

// poolProvider is an internal interface for stores that provide DB access
// This avoids circular imports by not referencing dbcommon directly
type poolProvider interface {
	Pool() interface{} // Returns *ConnectionPool, but we don't type it to avoid import
}

// The Component interface is composed of both functional interfaces and lifecycle interfaces
type Component interface {
	runtime.Component
	Router
}

var _ Component = &storeComponent{}

// storeComponent combine the stores and route Resource a request to an appropriate store
type storeComponent struct {
	// stores every resource corresponds to a ResourceStore
	stores map[coremodel.ResourceKind]ManagedResourceStore
}

// Compile-time check that storeComponent implements leader.DBSource interface
var _ leader.DBSource = &storeComponent{}

func (sc *storeComponent) RequiredDependencies() []runtime.ComponentType {
	return []runtime.ComponentType{
		runtime.EventBus, // Store may need EventBus for event emission
	}
}

func newStoreComponent() *storeComponent {
	return &storeComponent{
		stores: make(map[coremodel.ResourceKind]ManagedResourceStore),
	}
}

func (sc *storeComponent) Type() runtime.ComponentType {
	return runtime.ResourceStore
}

func (sc *storeComponent) Order() int {
	return math.MaxInt - 1
}

func (sc *storeComponent) Init(ctx runtime.BuilderContext) error {
	// 1. retrieve store config and choose the corresponding store factory
	cfg := ctx.Config().Store
	factory, err := FactoryRegistry().GetStoreFactory(cfg.Type)
	if err != nil {
		return err
	}
	// 2. create store for each resource kind
	for _, kind := range coremodel.ResourceSchemaRegistry().AllResourceKinds() {
		store, err := factory.New(kind, cfg)
		if err != nil {
			return err
		}
		sc.stores[kind] = store
		if err = store.Init(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (sc *storeComponent) Start(rt runtime.Runtime, ch <-chan struct{}) error {
	for _, store := range sc.stores {
		if err := store.Start(rt, ch); err != nil {
			return err
		}
	}
	return nil
}

func (sc *storeComponent) ResourceRoute(r coremodel.Resource) (ResourceStore, error) {
	return sc.ResourceKindRoute(r.ResourceKind())
}

func (sc *storeComponent) ResourceKindRoute(k coremodel.ResourceKind) (ResourceStore, error) {
	if store, exists := sc.stores[k]; exists {
		return store, nil
	}
	return nil, fmt.Errorf("%s is not supported by store yet", k)

}

// GetDB returns the shared DB connection if the underlying store is DB-backed
// Implements the leader.DBSource interface
func (sc *storeComponent) GetDB() (*gorm.DB, bool) {
	// Try to get DB from any store that has a Pool() method (all GormStores share the same ConnectionPool)
	for _, store := range sc.stores {
		pp, ok := store.(poolProvider)
		if !ok {
			continue
		}
		pool := pp.Pool()
		if pool == nil {
			continue
		}
		// Use reflection to call GetDB() on the pool to avoid importing dbcommon
		poolVal := reflect.ValueOf(pool)
		getDBMethod := poolVal.MethodByName("GetDB")
		if !getDBMethod.IsValid() {
			continue
		}
		result := getDBMethod.Call(nil)
		if len(result) > 0 {
			if db, ok := result[0].Interface().(*gorm.DB); ok {
				return db, true
			}
		}
	}
	return nil, false
}
