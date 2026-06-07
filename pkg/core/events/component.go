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
	"sync/atomic"

	eventbusconfig "github.com/apache/dubbo-admin/pkg/config/eventbus"
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
var _ runtime.GracefulComponent = &eventBus{}

type eventBus struct {
	rwMutex       sync.RWMutex
	subscriberDir map[model.ResourceKind][]*subscriberState
	bufferSize    int
	started       atomic.Bool
	stopped       atomic.Bool
	wg            sync.WaitGroup
}

func (b *eventBus) RequiredDependencies() []runtime.ComponentType {
	return []runtime.ComponentType{} // EventBus has no dependencies
}

func (b *eventBus) Type() runtime.ComponentType {
	return runtime.EventBus
}

func (b *eventBus) Order() int {
	return math.MaxInt
}

func (b *eventBus) Init(ctx runtime.BuilderContext) error {
	b.subscriberDir = make(map[model.ResourceKind][]*subscriberState)
	if cfg := ctx.Config().EventBus; cfg != nil {
		b.bufferSize = int(cfg.BufferSize)
	} else {
		b.bufferSize = int(eventbusconfig.Default().BufferSize)
	}
	return nil
}

func (b *eventBus) Start(_ runtime.Runtime, _ <-chan struct{}) error {
	if !b.started.CompareAndSwap(false, true) {
		return fmt.Errorf("eventBus already started")
	}
	b.rwMutex.RLock()
	for _, states := range b.subscriberDir {
		for _, st := range states {
			if st.async {
				b.launchDrainer(st)
			}
		}
	}
	b.rwMutex.RUnlock()
	return nil
}

// Subscribe subscribes to a resource kind, ProcessEventFunc is synchronous which is used to avoid event loss
func (b *eventBus) Subscribe(subscriber Subscriber) error {
	b.rwMutex.Lock()
	defer b.rwMutex.Unlock()
	if b.stopped.Load() {
		return fmt.Errorf("eventBus already stopped")
	}
	rk := subscriber.ResourceKind()
	states, exists := b.subscriberDir[rk]
	if !exists {
		states = make([]*subscriberState, 0)
	}
	// check name if is unique
	for _, st := range states {
		if st.subscriber.Name() == subscriber.Name() {
			return fmt.Errorf("duplicated subscriber name %s, skipped subscribing", subscriber.Name())
		}
	}
	isAsync := false
	if b.bufferSize > 0 {
		if subscriber.AsyncEnabled() {
			isAsync = true
		}
	}
	state := &subscriberState{
		subscriber: subscriber,
		async:      isAsync,
	}
	if isAsync {
		state.ch = make(chan Event, b.bufferSize)
		state.done = make(chan struct{})
	}
	b.subscriberDir[rk] = append(states, state)
	if isAsync && b.started.Load() {
		b.launchDrainer(state)
	}
	return nil
}

func (b *eventBus) Unsubscribe(subscriber Subscriber) error {
	var asyncState *subscriberState
	var drainerRunning bool

	b.rwMutex.Lock()
	rk := subscriber.ResourceKind()
	name := subscriber.Name()
	states, exists := b.subscriberDir[rk]
	if !exists {
		b.rwMutex.Unlock()
		return fmt.Errorf("no subscriber for resource %s, skipped unsubscribing", rk)
	}
	found := false
	for i, st := range states {
		if st.subscriber.Name() == name {
			b.subscriberDir[rk] = append(states[:i], states[i+1:]...)
			if st.async && st.ch != nil {
				st.closed.Store(true)
				close(st.ch)
				asyncState = st
				drainerRunning = st.drainerStarted.Load()
			}
			found = true
			break
		}
	}
	b.rwMutex.Unlock()
	if !found {
		return fmt.Errorf("no subscriber named %s for resource %s, skipped unsubscribing", name, rk)
	}
	if asyncState != nil && drainerRunning {
		<-asyncState.done
	}
	return nil
}

func (b *eventBus) Send(event Event) {
	if b.stopped.Load() {
		return
	}

	b.rwMutex.RLock()
	defer b.rwMutex.RUnlock()
	var rk model.ResourceKind
	if event.NewObj() != nil {
		rk = event.NewObj().ResourceKind()
	} else if event.OldObj() != nil {
		rk = event.OldObj().ResourceKind()
	}
	states, exists := b.subscriberDir[rk]
	if !exists {
		logger.Debugf("no subscriber for resource %s, skipped sending event%v", rk, event)
		return
	}
	for _, st := range states {
		if st.async && !st.closed.Load() {
			select {
			case st.ch <- event:
			default:
				logger.Warnf("async subscriber %s channel full (cap=%d), event dropped: %v",
					st.subscriber.Name(), cap(st.ch), event)
			}
			continue
		}
		if !st.async {
			// TODO Do we need to support reprocess
			if err := st.subscriber.ProcessEvent(event); err != nil {
				logger.Errorf("failed to process event in %s, cause: %s, event: %v",
					st.subscriber.Name(), err.Error(), event)
			}
		}
	}
}

func (b *eventBus) WaitForDone() {
	b.stopped.Store(true)

	b.rwMutex.Lock()
	for _, states := range b.subscriberDir {
		for _, st := range states {
			if st.async && st.ch != nil && !st.closed.Load() {
				st.closed.Store(true)
				close(st.ch)
			}
		}
	}
	b.rwMutex.Unlock()

	b.wg.Wait()
}

func (b *eventBus) launchDrainer(st *subscriberState) {
	if b.stopped.Load() {
		return
	}
	if !st.drainerStarted.CompareAndSwap(false, true) {
		return
	}

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		defer close(st.done)
		for event := range st.ch {
			if err := st.subscriber.ProcessEvent(event); err != nil {
				logger.Errorf("async: failed to process event in %s, cause: %s, event: %v",
					st.subscriber.Name(), err.Error(), event)
			}
		}
	}()
}
