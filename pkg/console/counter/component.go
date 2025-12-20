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

package counter

import (
	"fmt"
	"math"

	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

const ComponentType runtime.ComponentType = "counter manager"

func init() {
	runtime.RegisterComponent(&managerComponent{})
}

type ManagerComponent interface {
	runtime.Component
	CounterManager() CounterManager
}

var _ ManagerComponent = &managerComponent{}

func (c *managerComponent) RequiredDependencies() []runtime.ComponentType {
	return []runtime.ComponentType{
		runtime.ResourceStore,
		runtime.EventBus, // Counter depends on EventBus to subscribe to events
	}
}

type managerComponent struct {
	manager CounterManager
}

func (c *managerComponent) Type() runtime.ComponentType {
	return ComponentType
}

func (c *managerComponent) Order() int {
	return math.MaxInt - 1
}

func (c *managerComponent) Init(runtime.BuilderContext) error {
	mgr := NewCounterManager()
	c.manager = mgr
	return nil
}

func (c *managerComponent) Start(rt runtime.Runtime, _ <-chan struct{}) error {
	component, err := rt.GetComponent(runtime.EventBus)
	if err != nil {
		return err
	}
	bus, ok := component.(events.EventBus)
	if !ok {
		return fmt.Errorf("component %s does not implement events.EventBus", runtime.EventBus)
	}
	return c.manager.Bind(bus)
}

func (c *managerComponent) CounterManager() CounterManager {
	return c.manager
}
