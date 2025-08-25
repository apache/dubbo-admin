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
	"math"

	"github.com/apache/dubbo-admin/pkg/core/controller"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

func init() {
	runtime.RegisterComponent(newEngineComponent())
}

type Component interface {
	runtime.Component
	ResourceEngine
}

var _ Component = &engineComponent{}

type engineComponent struct {
	name      string
	informers []controller.Informer
}

func newEngineComponent() Component {
	return &engineComponent{
		informers: make([]controller.Informer, 0),
	}
}
func (b *engineComponent) Type() runtime.ComponentType {
	return runtime.ResourceEngine
}

func (b *engineComponent) Order() int {
	return math.MaxInt
}

func (b *engineComponent) Init(ctx runtime.BuilderContext) error {
	cfg := ctx.Config().Engine
	factory, err := FactoryRegistry().GetListWatcherFactory(cfg.Type)
	if err != nil {
		return err
	}
	lwList, err := factory.NewListWatchers(cfg)
	if err != nil {
		return err
	}
	for _, lw := range lwList {
		eventBusComponent, err := ctx.GetActivatedComponent(runtime.EventBus)
		if err != nil {
			return err
		}
		emitter, ok := eventBusComponent.(events.Emitter)
		if !ok {
			return fmt.Errorf("type assertion failed, event bus component in runtime is not an Emitter")
		}
		storeComponent, err := ctx.GetActivatedComponent(runtime.ResourceStore)
		if err != nil {
			return err
		}
		resourceStore, ok := storeComponent.(store.ResourceStore)
		if !ok {
			return fmt.Errorf("type assertion failed, resource store component in runtime is not a ResourceStore")
		}
		rk := lw.ResourceKind()
		newFunc, err := coremodel.ResourceSchemaRegistry().NewResourceFunc(rk)
		if err != nil {
			return err
		}
		informer := controller.NewInformerWithOptions(lw, emitter, resourceStore,
			newFunc(), controller.Options{ResyncPeriod: 0})
		b.informers = append(b.informers, informer)
	}
	b.name = cfg.Name
	logger.Infof("resource engine %s has been inited successfully", b.name)
	return nil
}

func (b *engineComponent) Start(_ runtime.Runtime, ch <-chan struct{}) error {
	for _, informer := range b.informers {
		go informer.Run(ch)
	}
	logger.Infof("resource engine %s has started successfully", b.name)
	return nil
}
