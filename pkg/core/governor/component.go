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

package governor

import (
	"fmt"
	"math"
	"reflect"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/core/events"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

func init() {
	runtime.RegisterComponent(newGovernorComponent())
}

type Router interface {
	ResourceRoute(r coremodel.Resource) (RuleGovernor, error)
	ResourceMeshRoute(mesh string) (RuleGovernor, error)
}

type Component interface {
	runtime.Component
	Router
}

var _ Component = &ruleGovernorComponent{}

type ruleGovernorComponent struct {
	configs   []*discovery.Config
	governors map[string]RuleGovernor
}

func newGovernorComponent() Component {
	return &ruleGovernorComponent{
		governors: make(map[string]RuleGovernor),
	}
}

func (g *ruleGovernorComponent) Type() runtime.ComponentType {
	return runtime.RuleGovernor
}

func (g *ruleGovernorComponent) Order() int {
	return math.MaxInt - 2
}

func (g *ruleGovernorComponent) RequiredDependencies() []runtime.ComponentType {
	return []runtime.ComponentType{
		runtime.EventBus,
		runtime.ResourceStore,
	}
}
func (g *ruleGovernorComponent) Init(ctx runtime.BuilderContext) error {
	eventBusComponent, err := ctx.GetActivatedComponent(runtime.EventBus)
	if err != nil {
		return bizerror.New(bizerror.UnknownError,
			fmt.Sprintf("can not retrieve event bus from runtime in discovery, %s", err))
	}
	eventBus, ok := eventBusComponent.(events.EventBus)
	if !ok {
		return bizerror.NewAssertionError("EventBus", reflect.TypeOf(eventBusComponent).Name())
	}

	storeComponent, err := ctx.GetActivatedComponent(runtime.ResourceStore)
	if err != nil {
		return bizerror.New(bizerror.UnknownError,
			fmt.Sprintf("can not retrieve store from runtime in discovery, %s", err))
	}
	storeRouter, ok := storeComponent.(store.Router)
	if !ok {
		return bizerror.NewAssertionError("store.Router", reflect.TypeOf(storeComponent).Name())
	}
	g.configs = ctx.Config().Discovery
	for _, cfg := range g.configs {
		factory, err := FactoryRegistry().GetGovernorFactory(cfg.Type)
		if err != nil {
			return err
		}
		governor, err := factory.New(cfg.ID, cfg, storeRouter, eventBus)
		if err != nil {
			return err
		}
		g.governors[cfg.ID] = governor
	}
	return nil
}

func (g *ruleGovernorComponent) Start(_ runtime.Runtime, _ <-chan struct{}) error {
	return nil
}

func (g *ruleGovernorComponent) ResourceRoute(r coremodel.Resource) (RuleGovernor, error) {
	return g.ResourceMeshRoute(r.ResourceMesh())
}

func (g *ruleGovernorComponent) ResourceMeshRoute(mesh string) (RuleGovernor, error) {
	if governor, exists := g.governors[mesh]; exists {
		return governor, nil
	}
	return nil, bizerror.New(bizerror.GovernorError, fmt.Sprintf("mesh %s not found for governor", mesh))
}
