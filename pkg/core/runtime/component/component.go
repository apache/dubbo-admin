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

package component

import (
	"errors"
	"sync"

	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/tools/channels"
)

// Component defines a process that will be run in the application
// Component should be designed in such a way that it can be stopped by stop channel and started again (for example when instance is reelected for a leader).
type Component interface {
	// Start blocks until the channel is closed or an error occurs.
	// The component will stop running when the channel is closed.
	Start(<-chan struct{}) error

	// NeedLeaderElection indicates if component should be run only by one instance of Control Plane even with many Control Plane replicas.
	NeedLeaderElection() bool
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

// Component of Kuma, i.e. gRPC Server, HTTP server, reconciliation loop.
var _ Component = ComponentFunc(nil)

type ComponentFunc func(<-chan struct{}) error

func (f ComponentFunc) NeedLeaderElection() bool {
	return false
}

func (f ComponentFunc) Start(stop <-chan struct{}) error {
	return f(stop)
}

var _ Component = LeaderComponentFunc(nil)

type LeaderComponentFunc func(<-chan struct{}) error

func (f LeaderComponentFunc) NeedLeaderElection() bool {
	return true
}

func (f LeaderComponentFunc) Start(stop <-chan struct{}) error {
	return f(stop)
}

type Manager interface {
	// Add registers a component, i.e. gRPC Server, HTTP server, reconciliation loop.
	Add(...Component) error

	// Start starts registered components and blocks until the Stop channel is closed.
	// Returns an error if there is an error starting any component.
	// If there are any GracefulComponent, it waits until all components are done.
	Start(<-chan struct{}) error
}

var _ Manager = &manager{}

func NewManager(leaderElector LeaderElector) Manager {
	return &manager{
		leaderElector: leaderElector,
	}
}

var LeaderComponentAddAfterStartErr = errors.New("cannot add leader component after component manager is started")

type manager struct {
	leaderElector LeaderElector

	sync.Mutex // protects access to fields below
	components []Component
	started    bool
	stopCh     <-chan struct{}
	errCh      chan error
}

func (cm *manager) Add(c ...Component) error {
	cm.Lock()
	defer cm.Unlock()
	cm.components = append(cm.components, c...)
	if cm.started {
		// start component if it's added in runtime after Start is called.
		for _, component := range c {
			if component.NeedLeaderElection() {
				return LeaderComponentAddAfterStartErr
			}
			go func(c Component, stopCh <-chan struct{}, errCh chan error) {
				if err := c.Start(stopCh); err != nil {
					errCh <- err
				}
			}(component, cm.stopCh, cm.errCh)
		}
	}
	return nil
}

func (cm *manager) waitForDone() {
	// limitation: waitForDone does not wait for components added after Start() is called.
	// This is ok for now, because it's used only in context of Kuma DP where we are not adding components in runtime.
	for _, c := range cm.components {
		if gc, ok := c.(GracefulComponent); ok {
			gc.WaitForDone()
		}
	}
}

func (cm *manager) Start(stop <-chan struct{}) error {
	errCh := make(chan error)

	cm.Lock()
	cm.startNonLeaderComponents(stop, errCh)
	cm.started = true
	cm.stopCh = stop
	cm.errCh = errCh
	cm.Unlock()
	// this has to be called outside of lock because it can be leader at any time, and it locks in leader callbacks.
	cm.startLeaderComponents(stop, errCh)

	defer cm.waitForDone()
	select {
	case <-stop:
		return nil
	case err := <-errCh:
		return err
	}
}

func (cm *manager) startNonLeaderComponents(stop <-chan struct{}, errCh chan error) {
	for _, component := range cm.components {
		if !component.NeedLeaderElection() {
			go func(c Component) {
				if err := c.Start(stop); err != nil {
					errCh <- err
				}
			}(component)
		}
	}
}

func (cm *manager) startLeaderComponents(stop <-chan struct{}, errCh chan error) {
	// leader stop channel needs to be stored in atomic because it will be written by leader elector goroutine
	// and read by the last goroutine in this function.
	// we need separate channel for leader components because they can be restarted
	mutex := sync.Mutex{}
	leaderStopCh := make(chan struct{})
	closeLeaderCh := func() {
		mutex.Lock()
		defer mutex.Unlock()
		if !channels.IsClosed(leaderStopCh) {
			close(leaderStopCh)
		}
	}

	cm.leaderElector.AddCallbacks(LeaderCallbacks{
		OnStartedLeading: func() {
			logger.Sugar().Info("leader acquired")
			mutex.Lock()
			defer mutex.Unlock()
			leaderStopCh = make(chan struct{})

			cm.Lock()
			defer cm.Unlock()
			for _, component := range cm.components {
				if component.NeedLeaderElection() {
					go func(c Component) {
						if err := c.Start(leaderStopCh); err != nil {
							errCh <- err
						}
					}(component)
				}
			}
		},
		OnStoppedLeading: func() {
			logger.Sugar().Info("leader lost")
			closeLeaderCh()
		},
	})
	go cm.leaderElector.Start(stop)
	go func() {
		<-stop
		closeLeaderCh()
	}()
}
