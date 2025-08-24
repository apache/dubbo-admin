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
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"k8s.io/client-go/tools/cache"
)

type Event interface {
	Type() cache.DeltaType
	OldObj() model.Resource
	NewObj() model.Resource
}

type ProcessEventFunc func(event Event) error

type ResourceChangedEvent struct {
	typ    cache.DeltaType
	oldObj model.Resource
	newObj model.Resource
}

func NewResourceChangedEvent(typ cache.DeltaType, oldObj model.Resource, newObj model.Resource) *ResourceChangedEvent {
	return &ResourceChangedEvent{
		typ:    typ,
		oldObj: oldObj,
		newObj: newObj,
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

type Emitter interface {
	Send(event Event)
}

type SubscriptionManager interface {
	Subscribe(rk model.ResourceKind, name string, process ProcessEventFunc) error
	Unsubscribe(rk model.ResourceKind, name string) error
}

type EventBus interface {
	Emitter
	SubscriptionManager
}
