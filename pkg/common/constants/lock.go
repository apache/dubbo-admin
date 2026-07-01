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

package constants

import "time"

const (
	// DefaultLockTimeout is the default timeout for distributed lock operations
	// This timeout applies to lock acquisition, renewal, and release operations
	DefaultLockTimeout = 30 * time.Second

	// DefaultAutoRenewThreshold is the TTL threshold above which auto-renewal is enabled
	// Locks with TTL longer than this value will be automatically renewed
	DefaultAutoRenewThreshold = 10 * time.Second

	// DefaultUnlockTimeout is the timeout for unlock operations in deferred cleanup
	DefaultUnlockTimeout = 5 * time.Second

	// DefaultRenewTimeout is the timeout for lock renewal operations
	DefaultRenewTimeout = 5 * time.Second

	// DefaultLockRetryInterval is the interval between lock acquisition retry attempts
	DefaultLockRetryInterval = 100 * time.Millisecond

	// DefaultCleanupInterval is the interval for periodic expired lock cleanup
	DefaultCleanupInterval = 5 * time.Minute

	// DefaultCleanupTimeout is the timeout for cleanup operations
	DefaultCleanupTimeout = 30 * time.Second
)

// Lock key prefixes for different resource types
const (
	// TagRouteKeyPrefix is the prefix for tag route lock keys
	TagRouteKeyPrefix = "tag_route"

	// ConfiguratorRuleKeyPrefix is the prefix for configurator rule lock keys
	ConfiguratorRuleKeyPrefix = "configurator_rule"

	// ConditionRuleKeyPrefix is the prefix for condition rule lock keys
	ConditionRuleKeyPrefix = "condition_rule"
)
