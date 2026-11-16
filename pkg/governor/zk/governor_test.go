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

package zk

import (
	"testing"

	"github.com/stretchr/testify/require"

	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

func TestRuleConfigPathUsesGroupedPathForEveryRuleKind(t *testing.T) {
	require.Equal(t, "/dubbo/config/dubbo", ruleConfigGroupPath())

	resources := []coremodel.Resource{
		meshresource.NewDynamicConfigResourceWithAttributes("demo.configurators", "mesh"),
		meshresource.NewConditionRouteResourceWithAttributes("demo.condition-router", "mesh"),
		meshresource.NewTagRouteResourceWithAttributes("demo.tag-router", "mesh"),
		meshresource.NewAffinityRouteResourceWithAttributes("demo.affinity-router", "mesh"),
		meshresource.NewScriptRouteResourceWithAttributes("demo.script-router", "mesh"),
	}
	for _, resource := range resources {
		require.Equal(t, "/dubbo/config/dubbo/"+resource.ResourceMeta().Name, ruleConfigPath(resource))
	}
}
