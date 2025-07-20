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
	"sync"

	"github.com/apache/dubbo-admin/pkg/core"
)

var log = core.Log.WithName("eventbus")

type subscriber struct {
	ch         chan Event
	predicates []Predicate
}

func NewEventBus(bufferSize uint) (EventBus, error) {
	return &eventBus{
		subscribers: map[string]subscriber{},
		bufferSize:  bufferSize,
	}, nil
}

type eventBus struct {
	mtx         sync.RWMutex
	subscribers map[string]subscriber
	bufferSize  uint
}

// Subscribe subscribes to a stream of events given Predicates
// Predicate should not block on I/O, otherwise the whole event bus can block.
// All predicates must pass for the event to enqueued.
func (b *eventBus) Subscribe(predicates ...Predicate) Listener {
	id := core.NewUUID()
	b.mtx.Lock()
	defer b.mtx.Unlock()

	events := make(chan Event, b.bufferSize)
	b.subscribers[id] = subscriber{
		ch:         events,
		predicates: predicates,
	}
	return &reader{
		events: events,
		close: func() {
			b.mtx.Lock()
			defer b.mtx.Unlock()
			delete(b.subscribers, id)
		},
	}
}

func (b *eventBus) Send(event Event) {
	b.mtx.RLock()
	defer b.mtx.RUnlock()
	for _, sub := range b.subscribers {
		matched := true
		for _, predicate := range sub.predicates {
			if !predicate(event) {
				matched = false
			}
		}
		if matched {
			select {
			case sub.ch <- event:
			default:
				log.Info("[WARNING] event is not sent because the channel is full. Ignoring event. Consider increasing buffer size using dubbo_EVENT_BUS_BUFFER_SIZE",
					"bufferSize", b.bufferSize,
					"event", event,
				)
			}
		}
	}
}

type reader struct {
	events chan Event
	close  func()
}

func (k *reader) Recv() <-chan Event {
	return k.events
}

func (k *reader) Close() {
	k.close()
}
