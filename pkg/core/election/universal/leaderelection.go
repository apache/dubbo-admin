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

package universal

import (
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/runtime/component"
	syncatomic "sync/atomic"
)

type LeaderElection struct {
	callbacks []component.LeaderCallbacks
	leader    int32
}

// Start will start leader election, calling all runFns when we become the leader.
func (l *LeaderElection) Start(stop <-chan struct{}) {
	logger.Sugar().Info("starting Leader Elector")

}

func (p *LeaderElection) AddCallbacks(callbacks component.LeaderCallbacks) {
	p.callbacks = append(p.callbacks, callbacks)
}

func (p *LeaderElection) IsLeader() bool {
	return syncatomic.LoadInt32(&(p.leader)) == 1
}

func (p *LeaderElection) setLeader(leader bool) {
	var value int32 = 0
	if leader {
		value = 1
	}
	syncatomic.StoreInt32(&p.leader, value)
}

func NewLeaderElection() *LeaderElection {
	return &LeaderElection{}
}
