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
	"math"
	"reflect"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/core/controller"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

func init() {
	runtime.RegisterComponent(newDiscoveryComponent())
}

type Component interface {
	runtime.Component
	ResourceDiscovery
}

var _ Component = &discoveryComponent{}

type Informers []controller.Informer

type discoveryComponent struct {
	discoveryInformers map[string]Informers
}

func newDiscoveryComponent() Component {
	return &discoveryComponent{
		discoveryInformers: make(map[string]Informers),
	}
}

func (d *discoveryComponent) Type() runtime.ComponentType {
	return runtime.ResourceDiscovery
}

func (d *discoveryComponent) Order() int {
	return math.MaxInt - 2
}

func (d *discoveryComponent) Init(ctx runtime.BuilderContext) error {
	configs := ctx.Config().Discovery
	for _, cfg := range configs {
		informers, err := d.newInformers(cfg, ctx)
		if err != nil {
			return err
		}
		d.discoveryInformers[cfg.Name] = informers
	}
	return nil
}

func (d *discoveryComponent) Start(_ runtime.Runtime, ch <-chan struct{}) error {
	for name, informers := range d.discoveryInformers {
		for _, informer := range informers {
			go informer.Run(ch)
		}
		logger.Infof("resource discvoery %s has started succesfully", name)
	}
	return nil
}

func (d *discoveryComponent) Add(resource coremodel.Resource) error {
	//TODO implement me
	panic("implement me")
}

func (d *discoveryComponent) Update(resource coremodel.Resource) error {
	//TODO implement me
	panic("implement me")
}

func (d *discoveryComponent) Delete(resource coremodel.Resource) error {
	//TODO implement me
	panic("implement me")
}

func (d *discoveryComponent) newInformers(cfg *discovery.Config, ctx runtime.BuilderContext) (Informers, error) {
	factory, err := ListWatcherFactoryRegistry().GetListWatcherFactory(cfg.Type)
	if err != nil {
		return nil, err
	}
	lwList, err := factory.NewListWatchers(cfg)
	if err != nil {
		return nil, err
	}
	eventBusComponent, err := ctx.GetActivatedComponent(runtime.EventBus)
	if err != nil {
		return nil, err
	}
	emitter, ok := eventBusComponent.(events.Emitter)
	if !ok {
		return nil, bizerror.NewAssertionError("Emitter", reflect.TypeOf(eventBusComponent).Name())
	}
	storeComponent, err := ctx.GetActivatedComponent(runtime.ResourceStore)
	if err != nil {
		return nil, fmt.Errorf("can not retrieve store from runtime in engine %s, %w", cfg.Name, err)
	}
	storeRouter, ok := storeComponent.(store.Router)
	if !ok {
		return nil, bizerror.NewAssertionError("store.Router", reflect.TypeOf(storeComponent).Name())
	}
	var informers = make([]controller.Informer, len(lwList))
	for i, lw := range lwList {
		resourceStore, err := storeRouter.ResourceKindRoute(lw.ResourceKind())
		if err != nil {
			return nil, fmt.Errorf("cannot find store for resource kind %s", lw.ResourceKind())
		}
		informer := controller.NewInformerWithOptions(lw, emitter, resourceStore, controller.Options{ResyncPeriod: 0})
		informers[i] = informer
	}
	return informers, nil
}
