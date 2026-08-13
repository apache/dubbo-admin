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

package governor

import (
	set "github.com/duke-git/lancet/v2/datastructure/set"

	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
)

var RuleResourceKinds = set.New(
	meshresource.DynamicConfigKind,
	meshresource.ConditionRouteKind,
	meshresource.TagRouteKind,
	meshresource.AffinityRouteKind,
	meshresource.ScriptRouteKind,
)

// RuleGovernor makes the rule operations effective
type RuleGovernor interface {
	// CreateRule creates a resource in the registry
	CreateRule(model.Resource) error
	// UpdateRule updates a resource in the registry
	UpdateRule(model.Resource) error
	// DeleteRule deletes a resource from the registry
	DeleteRule(model.Resource) error
}
