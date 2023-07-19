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

package kube

import (
	"context"
	syncatomic "sync/atomic"
	"time"

	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/runtime/component"
	"go.uber.org/atomic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

type KubuLeaderElection struct {
	leader    int32
	namespace string
	name      string
	callbacks []component.LeaderCallbacks
	client    kubernetes.Interface
	ttl       time.Duration

	// Records which "cycle" the election is on. This is incremented each time an election is won and then lost
	// This is mostly just for testing
	cycle      *atomic.Int32
	electionID string
}

// Start will start leader election, calling all runFns when we become the leader.
func (l *KubuLeaderElection) Start(stop <-chan struct{}) {
	logger.Sugar().Info("starting Leader Elector")
	for {
		le, err := l.create()
		if err != nil {
			// This should never happen; errors are only from invalid input and the input is not user modifiable
			panic("KubuLeaderElection creation failed: " + err.Error())
		}
		l.cycle.Inc()
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-stop
			cancel()
		}()
		le.Run(ctx)
		select {
		case <-stop:
			// We were told to stop explicitly. Exit now
			return
		default:
			cancel()
			// Otherwise, we may have lost our lock. In practice, this is extremely rare; we need to have the lock, then lose it
			// Typically this means something went wrong, such as API server downtime, etc
			// If this does happen, we will start the cycle over again
			logger.Sugar().Errorf("Leader election cycle %v lost. Trying again", l.cycle.Load())
		}
	}
}

func (l *KubuLeaderElection) create() (*leaderelection.LeaderElector, error) {
	callbacks := leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			l.setLeader(true)
			for _, f := range l.callbacks {
				if f.OnStartedLeading != nil {
					go f.OnStartedLeading()
				}
			}
		},
		OnStoppedLeading: func() {
			logger.Sugar().Infof("leader election lock lost: %v", l.electionID)
			l.setLeader(false)
			for _, f := range l.callbacks {
				if f.OnStoppedLeading != nil {
					go f.OnStoppedLeading()
				}
			}
		},
	}
	lock, err := resourcelock.New(resourcelock.ConfigMapsLeasesResourceLock,
		l.namespace,
		l.electionID,
		l.client.CoreV1(),
		l.client.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity: l.name,
		},
	)
	if err != nil {
		return nil, err
	}
	return leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: l.ttl,
		RenewDeadline: l.ttl / 2,
		RetryPeriod:   l.ttl / 4,
		Callbacks:     callbacks,
		// When exits, the lease will be dropped. This is more likely to lead to a case where
		// to instances are both considered the leaders. As such, if this is intended to be use for mission-critical
		// usages (rather than avoiding duplication of work), this may need to be re-evaluated.
		ReleaseOnCancel: true,
	})
}

func (p *KubuLeaderElection) AddCallbacks(callbacks component.LeaderCallbacks) {
	p.callbacks = append(p.callbacks, callbacks)
}

func (p *KubuLeaderElection) IsLeader() bool {
	return syncatomic.LoadInt32(&(p.leader)) == 1
}

func (p *KubuLeaderElection) setLeader(leader bool) {
	var value int32 = 0
	if leader {
		value = 1
	}
	syncatomic.StoreInt32(&p.leader, value)
}

func NewLeaderElection(namespace, name, electionID string, client kubernetes.Interface) *KubuLeaderElection {
	if name == "" {
		name = "unknown"
	}
	return &KubuLeaderElection{
		namespace:  namespace,
		name:       name,
		electionID: electionID,
		client:     client,
		// Default to a 30s ttl. Overridable for tests
		ttl:   time.Second * 30,
		cycle: atomic.NewInt32(0),
	}
}
