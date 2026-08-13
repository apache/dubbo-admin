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

package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apache/dubbo-admin/pkg/common/constants"
	"github.com/apache/dubbo-admin/pkg/console/model"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
)

func TestUpInsertServiceArgumentRouteConfigCreatesMissingConditionRule(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	req := model.BaseServiceReq{
		ServiceName: "org.apache.demo.DemoService",
		Version:     "1.0.0",
		Group:       "demo",
	}

	err := UpInsertServiceArgumentRouteConfig(ctx, req, model.ServiceArgumentRoute{
		Routes: []model.ServiceArgument{
			{
				Method: "sayHello",
				Conditions: []model.RouteCondition{
					{Index: "0", Relation: constants.Equal, Value: "foo"},
				},
				Destinations: []model.Destination{
					{
						Conditions: []model.DestinationCondition{
							{Tag: "region", Relation: constants.Equal, Value: "hangzhou"},
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	ruleName := "org.apache.demo.DemoService:1.0.0:demo.condition-router"
	current, exists, err := ctx.rm.GetByKey(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", ruleName))
	require.NoError(t, err)
	require.True(t, exists)
	conditionRule := current.(*meshresource.ConditionRouteResource)
	require.NotNil(t, conditionRule.Spec)
	assert.Equal(t, constants.ConfiguratorVersionV3, conditionRule.Spec.ConfigVersion)
	assert.Equal(t, constants.ScopeService, conditionRule.Spec.Scope)
	assert.Equal(t, "org.apache.demo.DemoService:1.0.0:demo", conditionRule.Spec.Key)
	assert.Equal(t, []string{"method=sayHello & arguments[0]=foo => region=hangzhou"}, conditionRule.Spec.Conditions)

	versions, err := ListRuleVersions(ctx, RuleRef{Kind: meshresource.ConditionRouteKind, Name: ruleName})
	require.NoError(t, err)
	require.Len(t, versions.Items, 1)
	assert.Equal(t, versioning.OperationCreate, versions.Items[0].Operation)
}

func TestUpInsertServiceArgumentRouteConfigUpdatesExistingConditionRule(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	req := model.BaseServiceReq{ServiceName: "org.apache.demo.DemoService"}
	ruleName := "org.apache.demo.DemoService::.condition-router"
	require.NoError(t, CreateConditionRule(ctx, conditionRule(ruleName, "=>region=$region")))

	err := UpInsertServiceArgumentRouteConfig(ctx, req, model.ServiceArgumentRoute{
		Routes: []model.ServiceArgument{
			{
				Method: "sayHello",
				Conditions: []model.RouteCondition{
					{Index: "0", Relation: constants.Equal, Value: "bar"},
				},
			},
		},
	})
	require.NoError(t, err)

	current, exists, err := ctx.rm.GetByKey(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", ruleName))
	require.NoError(t, err)
	require.True(t, exists)
	conditionRule := current.(*meshresource.ConditionRouteResource)
	assert.Equal(t, []string{"=>region=$region", "method=sayHello & arguments[0]=bar"}, conditionRule.Spec.Conditions)

	versions, err := ListRuleVersions(ctx, RuleRef{Kind: meshresource.ConditionRouteKind, Name: ruleName})
	require.NoError(t, err)
	require.Len(t, versions.Items, 2)
	assert.Equal(t, versioning.OperationUpdate, versions.Items[0].Operation)
}

func TestUpInsertServiceArgumentRouteConfigHandlesExistingRuleWithoutSpec(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	req := model.BaseServiceReq{ServiceName: "org.apache.demo.DemoService"}
	ruleName := "org.apache.demo.DemoService::.condition-router"
	res := meshresource.NewConditionRouteResourceWithAttributes(ruleName, "")
	res.Spec = nil
	require.NoError(t, ctx.stores[meshresource.ConditionRouteKind].Add(res))

	err := UpInsertServiceArgumentRouteConfig(ctx, req, model.ServiceArgumentRoute{
		Routes: []model.ServiceArgument{
			{Method: "sayHello"},
		},
	})
	require.NoError(t, err)

	current, exists, err := ctx.rm.GetByKey(meshresource.ConditionRouteKind, coremodel.BuildResourceKey("", ruleName))
	require.NoError(t, err)
	require.True(t, exists)
	conditionRule := current.(*meshresource.ConditionRouteResource)
	require.NotNil(t, conditionRule.Spec)
	assert.Equal(t, []string{"method=sayHello"}, conditionRule.Spec.Conditions)
}
