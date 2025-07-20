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
	"github.com/pkg/errors"
)

type Event interface{}

type Op int

const (
	Create Op = iota
	Update
	Delete
)

type ResourceChangedEvent struct {
	Operation Op
	Kind      string
	Key       string
	TenantID  string
}

type TriggerInsightsComputationEvent struct {
	TenantID string
}

var ListenerStoppedErr = errors.New("listener closed")

type Listener interface {
	Recv() <-chan Event
	Close()
}

func NewNeverListener() Listener {
	return &neverRecvListener{}
}

type neverRecvListener struct{}

func (*neverRecvListener) Recv() <-chan Event {
	return nil
}

func (*neverRecvListener) Close() {
}

type Predicate = func(event Event) bool

type Emitter interface {
	Send(Event)
}

type ListenerFactory interface {
	Subscribe(...Predicate) Listener
}

type EventBus interface {
	Emitter
	ListenerFactory
}
