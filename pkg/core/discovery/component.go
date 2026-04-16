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
	"context"
	"fmt"
	"math"
	"reflect"
	"sync/atomic"

	"github.com/duke-git/lancet/v2/slice"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/config/engine"
	storecfg "github.com/apache/dubbo-admin/pkg/config/store"
	"github.com/apache/dubbo-admin/pkg/core/controller"
	"github.com/apache/dubbo-admin/pkg/core/discovery/subscriber"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/leader"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
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
	configs             []*discovery.Config
	informers           map[string]Informers
	subscribers         []events.Subscriber
	subscriptionMgr     events.SubscriptionManager
	leaderElection      *leader.LeaderElection
	needsLeaderElection bool
	subscribed          atomic.Bool
}

func (d *discoveryComponent) RequiredDependencies() []runtime.ComponentType {
	return []runtime.ComponentType{
		runtime.EventBus,      // Discovery needs EventBus for event emission
		runtime.ResourceStore, // Discovery needs Store for resource storage
	}
}

func newDiscoveryComponent() Component {
	return &discoveryComponent{
		informers:   make(map[string]Informers),
		subscribers: make([]events.Subscriber, 0),
	}
}

func (d *discoveryComponent) Type() runtime.ComponentType {
	return runtime.ResourceDiscovery
}

func (d *discoveryComponent) Order() int {
	return math.MaxInt - 2
}

func (d *discoveryComponent) Init(ctx runtime.BuilderContext) error {
	eventBusComponent, err := ctx.GetActivatedComponent(runtime.EventBus)
	if err != nil {
		return bizerror.New(bizerror.UnknownError,
			fmt.Sprintf("can not retrieve event bus from runtime in discovery, %s", err))
	}
	eventBus, ok := eventBusComponent.(events.EventBus)
	if !ok {
		return bizerror.NewAssertionError("EventBus", reflect.TypeOf(eventBusComponent).Name())
	}
	d.subscriptionMgr = eventBus
	storeComponent, err := ctx.GetActivatedComponent(runtime.ResourceStore)
	if err != nil {
		return bizerror.New(bizerror.UnknownError,
			fmt.Sprintf("can not retrieve store from runtime in discovery, %s", err))
	}
	storeRouter, ok := storeComponent.(store.Router)
	if !ok {
		return bizerror.NewAssertionError("store.Router", reflect.TypeOf(storeComponent).Name())
	}
	d.configs = ctx.Config().Discovery
	for _, cfg := range d.configs {
		informers, err := d.initInformers(cfg, storeRouter, eventBus)
		if err != nil {
			return err
		}
		d.informers[cfg.ID] = informers
	}
	err = d.initSubscribes(storeRouter, eventBus, ctx.Config().Engine)
	if err != nil {
		return err
	}

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
		logger.Warnf("discovery: failed to generate holder ID, skipping leader election: %v", err)
		return nil
	}
	le := leader.NewLeaderElection(db, runtime.ResourceDiscovery, holderID)
	if err := le.EnsureTable(); err != nil {
		logger.Warnf("discovery: failed to ensure leader lease table: %v", err)
		return nil
	}
	d.leaderElection = le
	d.needsLeaderElection = true
	logger.Infof("discovery: leader election initialized (holder: %s)", holderID)
	return nil
}

func (d *discoveryComponent) Start(_ runtime.Runtime, ch <-chan struct{}) error {
	if !d.needsLeaderElection {
		return d.startBusinessLogic(ch)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-ch
		cancel()
	}()

	var leaderStopCh chan struct{}

	d.leaderElection.RunLeaderElection(ctx, ch,
		func() { // onStartLeading: create a fresh stopCh for this leadership term
			leaderStopCh = make(chan struct{})
			logger.Infof("discovery: became leader, starting business logic")
			if err := d.startBusinessLogic(leaderStopCh); err != nil {
				logger.Errorf("discovery: failed to start business logic: %v", err)
			}
		},
		func() { // onStopLeading: stop informers from the current term
			logger.Warnf("discovery: lost leadership, stopping business logic")
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
func (d *discoveryComponent) startBusinessLogic(stopCh <-chan struct{}) error {
	// 1. subscribe resource changed events (only once for the process lifetime)
	if !d.subscribed.Load() {
		for _, sub := range d.subscribers {
			err := d.subscriptionMgr.Subscribe(sub)
			if err != nil {
				return bizerror.Wrap(err, bizerror.EventError,
					fmt.Sprintf("subscriber %s can not subscribe resource changed events", sub.Name()))
			}
		}
		d.subscribed.Store(true)
	}
	// 2. start informers
	for name, informers := range d.informers {
		for _, informer := range informers {
			go informer.Run(stopCh)
		}
		logger.Infof("resource discovery %s has started successfully", name)
	}
	return nil
}

func (d *discoveryComponent) initInformers(cfg *discovery.Config, storeRouter store.Router, eventBus events.EventBus) (Informers, error) {
	factory, err := ListWatcherFactoryRegistry().GetListWatcherFactory(cfg.Type)
	if err != nil {
		return nil, err
	}
	lwList, err := factory.NewListWatchers(cfg)
	if err != nil {
		return nil, err
	}
	var informers = make([]controller.Informer, len(lwList))
	for i, lw := range lwList {
		resourceStore, err := storeRouter.ResourceKindRoute(lw.ResourceKind())
		if err != nil {
			return nil, fmt.Errorf("cannot find store for resource kind %s", lw.ResourceKind())
		}
		informer := controller.NewInformerWithOptions(lw, eventBus, resourceStore, keyFunc, controller.Options{ResyncPeriod: 0})
		informers[i] = informer
	}
	return informers, nil
}
func keyFunc(obj interface{}) (string, error) {
	if r, ok := obj.(coremodel.Resource); ok {
		return r.ResourceKey(), nil
	}
	return "", bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
}

func (d *discoveryComponent) initSubscribes(storeRouter store.Router, emitter events.Emitter, engineConfig *engine.Config) error {
	instanceStore, err := storeRouter.ResourceKindRoute(meshresource.InstanceKind)
	if err != nil {
		return bizerror.Wrap(err, bizerror.StoreError,
			fmt.Sprintf("can not find store for resource kind %s", meshresource.InstanceKind))
	}
	rtInstanceStore, err := storeRouter.ResourceKindRoute(meshresource.RuntimeInstanceKind)
	if err != nil {
		return bizerror.Wrap(err, bizerror.StoreError,
			fmt.Sprintf("can not find store for resource kind %s", meshresource.RuntimeInstanceKind))
	}
	rpcInstanceSub := subscriber.NewRPCInstanceEventSubscriber(instanceStore, rtInstanceStore, emitter, engineConfig)

	appStore, err := storeRouter.ResourceKindRoute(meshresource.ApplicationKind)
	if err != nil {
		return bizerror.Wrap(err, bizerror.StoreError,
			fmt.Sprintf("can not find store for resource kind %s", meshresource.ApplicationKind))
	}
	serviceStore, err := storeRouter.ResourceKindRoute(meshresource.ServiceKind)
	if err != nil {
		return bizerror.Wrap(err, bizerror.StoreError,
			fmt.Sprintf("can not find store for resource kind %s", meshresource.ServiceKind))
	}
	serviceProviderMetadataStore, err := storeRouter.ResourceKindRoute(meshresource.ServiceProviderMetadataKind)
	if err != nil {
		return bizerror.Wrap(err, bizerror.StoreError,
			fmt.Sprintf("can not find store for resource kind %s", meshresource.ServiceProviderMetadataKind))
	}
	serviceConsumerMetadataSub := subscriber.NewServiceConsumerMetadataEventSubscriber(appStore, emitter)
	serviceProviderMetadataSub := subscriber.NewServiceProviderMetadataEventSubscriber(
		appStore, serviceStore, serviceProviderMetadataStore, emitter)
	instanceSub := subscriber.NewInstanceEventSubscriber(appStore, instanceStore, emitter)
	d.subscribers = append(d.subscribers, rpcInstanceSub, serviceConsumerMetadataSub, serviceProviderMetadataSub, instanceSub)

	// if there is a nacos discovery, a NacosServiceEventSubscriber is needed
	_, hasNacosDiscovery := slice.FindBy(d.configs, func(index int, item *discovery.Config) bool {
		return item.Type == discovery.Nacos2
	})
	if hasNacosDiscovery {
		nacosServiceSub := subscriber.NewNacosServiceEventSubscriber(emitter, storeRouter)
		d.subscribers = append(d.subscribers, nacosServiceSub)
	}
	// if there is a zk discovery, a ZKMetadataEventSubscriber and ZKConfigEventSubscriber is needed
	_, hasZkDiscovery := slice.FindBy(d.configs, func(index int, item *discovery.Config) bool {
		return item.Type == discovery.Zookeeper
	})
	if hasZkDiscovery {
		zkMetadataSub := subscriber.NewZKMetadataEventSubscriber(emitter, storeRouter)
		zkConfigSub := subscriber.NewZKConfigEventSubscriber(emitter, storeRouter)
		d.subscribers = append(d.subscribers, zkMetadataSub, zkConfigSub)
	}
	return nil
}
