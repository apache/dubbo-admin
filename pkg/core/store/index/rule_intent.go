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

package index

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

const (
	ByRuleIntentParentAndStatus = "ByRuleIntentParentAndStatus"
	ByRuleIntentIDIndexName     = "ByRuleIntentID"
	ByRuleIntentStatusIndexName = "ByRuleIntentStatus"
)

func init() {
	RegisterIndexers(meshresource.RuleIntentKind, map[string]cache.IndexFunc{
		ByRuleIntentParentAndStatus: byRuleIntentParentAndStatus,
		ByRuleIntentIDIndexName:     byRuleIntentID,
		ByRuleIntentStatusIndexName: byRuleIntentStatus,
	})
}

// byRuleIntentParentAndStatus indexes RuleIntent resources by parent rule and status.
// Index key format: "<parent_kind>/<parent_mesh>/<parent_name>/<status>"
//
// This enables efficient queries like:
// - Find pending intent for a specific rule: "ConditionRoute/default/my-rule/PENDING"
// - Find applied intents for a rule: "ConditionRoute/default/my-rule/APPLIED"
//
// Used by versioning.ResourceStoreAdapter to avoid full table scans when looking up intents.
func byRuleIntentParentAndStatus(obj interface{}) ([]string, error) {
	intent, ok := obj.(*meshresource.RuleIntentResource)
	if !ok || intent.Spec == nil {
		return nil, nil
	}

	key := fmt.Sprintf("%s/%s/%s/%s",
		intent.Spec.ParentRuleKind,
		intent.Spec.ParentRuleMesh,
		intent.Spec.ParentRuleName,
		intent.Spec.Status,
	)
	return []string{key}, nil
}

// byRuleIntentID indexes RuleIntent resources by the numeric ID suffix in
// their resource name. The suffix parsing intentionally matches versioning's
// extractIDFromIntentName behavior.
func byRuleIntentID(obj interface{}) ([]string, error) {
	intent, ok := obj.(*meshresource.RuleIntentResource)
	if !ok {
		return nil, nil
	}
	idx := strings.LastIndex(intent.Name, "-")
	if idx == -1 || idx == len(intent.Name)-1 {
		logger.Warnf("skipping malformed RuleIntent name in id index: %s", intent.Name)
		return nil, nil
	}
	id, err := strconv.ParseInt(intent.Name[idx+1:], 10, 64)
	if err != nil {
		logger.Warnf("skipping malformed RuleIntent name in id index: %s", intent.Name)
		return nil, nil
	}
	return []string{strconv.FormatInt(id, 10)}, nil
}

func byRuleIntentStatus(obj interface{}) ([]string, error) {
	intent, ok := obj.(*meshresource.RuleIntentResource)
	if !ok || intent.Spec == nil || intent.Spec.Status == "" {
		return nil, nil
	}
	return []string{intent.Spec.Status}, nil
}
