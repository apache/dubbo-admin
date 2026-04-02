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
	"context"
	"fmt"
	"math"
	"reflect"
	"sync/atomic"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	enginecfg "github.com/apache/dubbo-admin/pkg/config/engine"
	storecfg "github.com/apache/dubbo-admin/pkg/config/store"
	"github.com/apache/dubbo-admin/pkg/core/controller"
	"github.com/apache/dubbo-admin/pkg/core/engine/subscriber"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/leader"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"k8s.io/client-go/tools/cache"
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
	name                string
	storeRouter         store.Router
	informers           []controller.Informer
	subscriptionManager events.SubscriptionManager
	subscribers         []events.Subscriber
	leaderElection      *leader.LeaderElection
	needsLeaderElection bool
	subscribed          atomic.Bool
}

func newEngineComponent() Component {
	return &engineComponent{
		informers:   make([]controller.Informer, 0),
		subscribers: make([]events.Subscriber, 0),
	}
}

func (e *engineComponent) RequiredDependencies() []runtime.ComponentType {
	return []runtime.ComponentType{
		runtime.EventBus,
		runtime.ResourceStore,
	}
}

func (e *engineComponent) Type() runtime.ComponentType {
	return runtime.ResourceEngine
}

func (e *engineComponent) Order() int {
	return math.MaxInt - 3
}

func (e *engineComponent) Init(ctx runtime.BuilderContext) error {
	cfg := ctx.Config().Engine
	e.name = cfg.ID
	eventBusComponent, err := ctx.GetActivatedComponent(runtime.EventBus)
	if err != nil {
		return fmt.Errorf("can not retrieve event bus from runtime in engine %s, %w", e.name, err)
	}
	eventBus, ok := eventBusComponent.(events.EventBus)
	if !ok {
		return bizerror.NewAssertionError("EventBus", reflect.TypeOf(eventBusComponent).Name())
	}
	e.subscriptionManager = eventBus
	storeComponent, err := ctx.GetActivatedComponent(runtime.ResourceStore)
	if err != nil {
		return fmt.Errorf("can not retrieve store from runtime in engine %s, %w", e.name, err)
	}
	storeRouter, ok := storeComponent.(store.Router)
	if !ok {
		return bizerror.NewAssertionError("store.Router", reflect.TypeOf(storeComponent).Name())
	}
	e.storeRouter = storeRouter
	if err = e.initInformers(cfg, eventBus); err != nil {
		return fmt.Errorf("init informer failed, %w", err)
	}
	if err = e.initSubscribers(eventBus); err != nil {
		return fmt.Errorf("init subscribers failed, %w", err)
	}

	defer logger.Infof("resource engine %s has been inited successfully", e.name)

	// Memory store runs single-replica; leader election is not needed.
	if ctx.Config().Store.Type == storecfg.Memory {
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
		logger.Warnf("engine: failed to generate holder ID, skipping leader election: %v", err)
		return nil
	}
	le := leader.NewLeaderElection(db, runtime.ResourceEngine, holderID)
	if err := le.EnsureTable(); err != nil {
		logger.Warnf("engine: failed to ensure leader lease table: %v", err)
		return nil
	}
	e.leaderElection = le
	e.needsLeaderElection = true
	logger.Infof("engine: leader election initialized (holder: %s)", holderID)
	return nil
}

func (e *engineComponent) initInformers(cfg *enginecfg.Config, emitter events.Emitter) error {
	factory, err := FactoryRegistry().GetListWatcherFactory(cfg.Type)
	if err != nil {
		return err
	}
	lwList, err := factory.NewListWatchers(cfg)
	if err != nil {
		return err
	}
	for _, lw := range lwList {
		rk := lw.ResourceKind()
		rs, err := e.storeRouter.ResourceKindRoute(rk)
		if err != nil {
			return fmt.Errorf("can not find store for resource kind %s, %w", rk, err)
		}
		informer := controller.NewInformerWithOptions(lw, emitter, rs, resolveInformerKeyFunc(lw), controller.Options{ResyncPeriod: 0})
		if lw.TransformFunc() != nil {
			err = informer.SetTransform(lw.TransformFunc())
			if err != nil {
				return fmt.Errorf("can not set transform for informer of resource kind %s, %w", rk, err)
			}
		}
		e.informers = append(e.informers, informer)
		logger.Infof("resource engine %s has added informer for resource kind %s", e.name, rk)
	}
	return nil
}

func resolveInformerKeyFunc(lw controller.ResourceListerWatcher) cache.KeyFunc {
	if provider, ok := lw.(controller.ResourceKeyProvider); ok {
		return provider.KeyFunc()
	}
	return cache.MetaNamespaceKeyFunc
}

func (e *engineComponent) initSubscribers(eventbus events.EventBus) error {
	rs, err := e.storeRouter.ResourceKindRoute(meshresource.InstanceKind)
	if err != nil {
		return fmt.Errorf("can not find store for resource kind %s, %w", meshresource.InstanceKind, err)
	}
	runtimeInstanceSub := subscriber.NewRuntimeInstanceEventSubscriber(rs, eventbus)
	e.subscribers = append(e.subscribers, runtimeInstanceSub)
	return nil
}

func (e *engineComponent) Start(_ runtime.Runtime, ch <-chan struct{}) error {
	if !e.needsLeaderElection {
		return e.startBusinessLogic(ch)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-ch
		cancel()
	}()

	var leaderStopCh chan struct{}

	e.leaderElection.RunLeaderElection(ctx, ch,
		func() { // onStartLeading: create a fresh stopCh for this leadership term
			leaderStopCh = make(chan struct{})
			logger.Infof("engine: became leader, starting business logic")
			if err := e.startBusinessLogic(leaderStopCh); err != nil {
				logger.Errorf("engine: failed to start business logic: %v", err)
			}
		},
		func() { // onStopLeading: stop informers from the current term
			logger.Warnf("engine: lost leadership, stopping business logic")
			if leaderStopCh != nil {
				close(leaderStopCh)
				leaderStopCh = nil
			}
		},
	)

	return nil
}

// startBusinessLogic starts subscribers and informers using the provided stopCh.
// When stopCh is closed all informer goroutines will exit.
func (e *engineComponent) startBusinessLogic(stopCh <-chan struct{}) error {
	// 1. subscribe resource changed events (only once for the process lifetime)
	if !e.subscribed.Load() {
		for _, sub := range e.subscribers {
			if err := e.subscriptionManager.Subscribe(sub); err != nil {
				return fmt.Errorf("could not subscribe %s to eventbus, %w", sub.Name(), err)
			}
		}
		e.subscribed.Store(true)
	}
	// 2. start informers
	for _, informer := range e.informers {
		go informer.Run(stopCh)
	}
	logger.Infof("resource engine %s has started successfully", e.name)
	return nil
}
