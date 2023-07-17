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

// LeaderCallbacks defines callbacks for events from LeaderElector
// It is guaranteed that each methods will be executed from the same goroutine, so only one method can be run at once.
type LeaderCallbacks struct {
	OnStartedLeading func()
	OnStoppedLeading func()
}

type LeaderElector interface {
	AddCallbacks(LeaderCallbacks)
	// IsLeader should be used for diagnostic reasons (metrics/API info), because there may not be any leader elector for a short period of time.
	// Use Callbacks to write logic to execute when Leader is elected.
	IsLeader() bool

	// Start blocks until the channel is closed or an error occurs.
	Start(stop <-chan struct{})
}
