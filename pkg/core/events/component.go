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

package events

import (
	"fmt"
	"math"
	"sync"

	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

func init() {
	runtime.RegisterComponent(&eventBus{})
}

type EventBusComponent interface {
	EventBus
	runtime.Component
}

var _ EventBusComponent = &eventBus{}

type eventBus struct {
	rwMutex       sync.RWMutex
	subscriberDir map[model.ResourceKind]Subscribers
}

func (b *eventBus) Type() runtime.ComponentType {
	return runtime.EventBus
}

func (b *eventBus) Order() int {
	return math.MaxInt
}

func (b *eventBus) Init(_ runtime.BuilderContext) error {
	b.subscriberDir = make(map[model.ResourceKind]Subscribers)
	return nil
}

func (b *eventBus) Start(_ runtime.Runtime, _ <-chan struct{}) error {
	return nil
}

// Subscribe subscribes to a resource kind, ProcessEventFunc is synchronous which is used to avoid event loss
func (b *eventBus) Subscribe(subscriber Subscriber) error {
	b.rwMutex.Lock()
	defer b.rwMutex.Unlock()
	subs, exists := b.subscriberDir[subscriber.ResourceKind()]
	if !exists {
		subs = make(Subscribers, 0)
	}
	// check name if is unique
	for _, sub := range subs {
		if sub.Name() == subscriber.Name() {
			return fmt.Errorf("duplicated subscriber name %s, skipped subscribing", subscriber.Name())
		}
	}
	b.subscriberDir[subscriber.ResourceKind()] = append(subs, subscriber)
	return nil
}

func (b *eventBus) Unsubscribe(subscriber Subscriber) error {
	b.rwMutex.Lock()
	defer b.rwMutex.Unlock()
	rk := subscriber.ResourceKind()
	name := subscriber.Name()
	subs, exists := b.subscriberDir[rk]
	if !exists {
		return fmt.Errorf("no subscriber for resource %s, skipped unsubscribing", rk)
	}
	for i, sub := range subs {
		if sub.Name() == name {
			b.subscriberDir[rk] = append(subs[:i], subs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("no subscriber named %s for resource %s, skipped unsubscribing", name, rk)
}

func (b *eventBus) Send(event Event) {
	b.rwMutex.RLock()
	defer b.rwMutex.RUnlock()
	var rk model.ResourceKind
	if event.NewObj() != nil {
		rk = event.NewObj().ResourceKind()
	} else if event.OldObj() != nil {
		rk = event.OldObj().ResourceKind()
	}
	subs, exists := b.subscriberDir[rk]
	if !exists {
		logger.Infof("no subscriber for resource %s, skipped sending event%v", rk, event)
		return
	}
	for _, sub := range subs {
		// TODO Do we need to support reprocess
		if err := sub.ProcessEvent(event); err != nil {
			logger.Errorf("failed to process event in %s, cause: %s, event: %v", sub.Name(), err.Error(), event)
		}
	}
}
