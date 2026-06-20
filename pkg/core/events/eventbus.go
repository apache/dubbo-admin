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

	"github.com/duke-git/lancet/v2/convertor"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// SourceRegistryContextKey identifies which registry produced a resource event.
const SourceRegistryContextKey = "source-registry"

type Event interface {
	// Type returns the type of the event, see definitions in cache.DeltaType
	Type() cache.DeltaType
	// OldObj returns the old object, nil if old object doesn't exist in store
	OldObj() model.Resource
	// NewObj returns the new object, nil if event type is in [cache.Deleted]
	NewObj() model.Resource
	// Context returns read-only event metadata. Subscribers must not mutate the returned map.
	Context() map[string]string
	// String returns the string representation of the event
	String() string
}

type Subscriber interface {
	ResourceKind() model.ResourceKind

	Name() string

	ProcessEvent(event Event) error

	// AsyncEnabled controls whether this subscriber should be dispatched asynchronously.
	//
	// Return true when the subscriber may execute relatively slow logic and should not
	// block EventBus.Send(). In async mode, EventBus enqueues events into a buffered
	// channel and processes them in a dedicated drainer goroutine.
	//
	// Async dispatch is best-effort: when the subscriber queue is full, new events are
	// dropped with a warning log.
	AsyncEnabled() bool
}

type Subscribers []Subscriber

type ResourceChangedEvent struct {
	typ     cache.DeltaType
	oldObj  model.Resource
	newObj  model.Resource
	context map[string]string
}

var _ Event = &ResourceChangedEvent{}

func NewResourceChangedEvent(typ cache.DeltaType, oldObj model.Resource, newObj model.Resource) *ResourceChangedEvent {
	return &ResourceChangedEvent{
		typ:     typ,
		oldObj:  oldObj,
		newObj:  newObj,
		context: map[string]string{},
	}
}

func NewResourceChangedEventWithContext(typ cache.DeltaType, oldObj model.Resource,
	newObj model.Resource, ctx map[string]string) *ResourceChangedEvent {
	return &ResourceChangedEvent{
		typ:     typ,
		oldObj:  oldObj,
		newObj:  newObj,
		context: ctx,
	}
}

func (e *ResourceChangedEvent) Type() cache.DeltaType {
	return e.typ
}

func (e *ResourceChangedEvent) OldObj() model.Resource {
	return e.oldObj
}

func (e *ResourceChangedEvent) NewObj() model.Resource {
	return e.newObj
}

func (e *ResourceChangedEvent) Context() map[string]string {
	return e.context
}

func (e *ResourceChangedEvent) String() string {
	return fmt.Sprintf("\n[type]: %s, \n[oldObj]: %s, \n[newObj]: %s",
		e.typ, convertor.ToString(e.oldObj), convertor.ToString(e.newObj))
}

type Emitter interface {
	Send(event Event)
}

type SubscriptionManager interface {
	Subscribe(subscriber Subscriber) error
	Unsubscribe(subscriber Subscriber) error
}

type EventBus interface {
	Emitter
	SubscriptionManager
}
