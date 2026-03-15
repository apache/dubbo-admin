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

package lock

import (
	"fmt"

	"github.com/apache/dubbo-admin/pkg/common/constants"
)

// BuildLockKey constructs a lock key from a prefix and parts
func BuildLockKey(prefix string, parts ...string) string {
	key := prefix
	for _, part := range parts {
		key += ":" + part
	}
	return key
}

// BuildTagRouteLockKey constructs a lock key for tag route operations
func BuildTagRouteLockKey(mesh, name string) string {
	return fmt.Sprintf("%s:%s:%s", constants.TagRouteKeyPrefix, mesh, name)
}

// BuildConfiguratorRuleLockKey constructs a lock key for configurator rule operations
func BuildConfiguratorRuleLockKey(mesh, name string) string {
	return fmt.Sprintf("%s:%s:%s", constants.ConfiguratorRuleKeyPrefix, mesh, name)
}

// BuildConditionRuleLockKey constructs a lock key for condition rule operations
func BuildConditionRuleLockKey(mesh, name string) string {
	return fmt.Sprintf("%s:%s:%s", constants.ConditionRuleKeyPrefix, mesh, name)
}
