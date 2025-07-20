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

package runtime

type ComponentType = string

const (
	Console           ComponentType = "console"
	ResourceManager   ComponentType = "resource manager"
	ResourceStore   ComponentType = "resource store"
	ResourceEngine    ComponentType = "resource engine"
	ResourceDiscovery ComponentType = "resource discovery"
)

var CoreComponentTypes = []ComponentType{Console, ResourceManager, ResourceStore, ResourceEngine, ResourceDiscovery}

type ComponentSubType = string

const DefaultComponentSubType = "default"

// Component defines a process that will be run in the application
// Component should be designed in such a way that it can be stopped by stop channel and started again (for example when instance is reelected for a leader).
type Component interface {
	// Type returns the type of the component
	Type() ComponentType
	// SubType returns the subtype of the component
	SubType()  ComponentSubType
	// Order indicates the order of the component during bootstrap, the bigger will be started first
	Order() int
	// Init initializes the component
	Init(ctx BuilderContext) error
	// Start blocks until the channel is closed or an error occurs.
	// The component will stop running when the channel is closed.
	Start(Runtime ,<-chan struct{}) error
}

// GracefulComponent is a component that supports waiting until it's finished.
// It's useful if there is cleanup logic that has to be executed before the process exits
// (i.e. sending SIGTERM signals to subprocesses started by this component).
type GracefulComponent interface {
	Component

	// WaitForDone blocks until all components are done.
	// If a component was not started (i.e. leader components on non-leader CP) it returns immediately.
	WaitForDone()
}

type ComponentManager interface {
	// Add adds a component to the manager.
	Add(... Component)

	// Start starts all components.
	Start( <-chan struct{}) error
}



