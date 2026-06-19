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
	"encoding/base64"
	"strings"

	"github.com/apache/dubbo-admin/pkg/common/constants"
)

// BuildLockKey constructs a lock key from a prefix and parts
func BuildLockKey(prefix string, parts ...string) string {
	segments := make([]string, 0, len(parts)+1)
	segments = append(segments, encodeLockPart(prefix))
	for _, part := range parts {
		segments = append(segments, encodeLockPart(part))
	}
	return strings.Join(segments, ":")
}

// BuildRuleVersioningLockKey constructs the canonical per-rule lock key used by
// console writes, rollback, bootstrap, repair, retention, and subscriber commits.
func BuildRuleVersioningLockKey(kind, mesh, name string) string {
	return BuildLockKey(constants.RuleVersioningKeyPrefix, kind, mesh, name)
}

// BuildTagRouteLockKey constructs a lock key for tag route operations
func BuildTagRouteLockKey(mesh, name string) string {
	return BuildRuleVersioningLockKey("TagRoute", mesh, name)
}

// BuildConfiguratorRuleLockKey constructs a lock key for configurator rule operations
func BuildConfiguratorRuleLockKey(mesh, name string) string {
	return BuildRuleVersioningLockKey("DynamicConfig", mesh, name)
}

// BuildConditionRuleLockKey constructs a lock key for condition rule operations
func BuildConditionRuleLockKey(mesh, name string) string {
	return BuildRuleVersioningLockKey("ConditionRoute", mesh, name)
}

func encodeLockPart(part string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(part))
}
