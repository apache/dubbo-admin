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

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
)

func TestAffinityRuleLifecycleAndRollback(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	key := "demo-provider"
	name := key + ".affinity-router"
	initial := affinityRouterRule(key, 50)

	require.NoError(t, CreateAffinityRuleWithOptions(ctx, initial, RuleMutationOptions{Author: "admin"}))
	require.NoError(t, UpdateAffinityRuleWithOptions(ctx, affinityRouterRule(key, 80), RuleMutationOptions{Author: "admin"}))

	versions, err := ListRuleVersions(ctx, RuleRef{Kind: meshresource.AffinityRouteKind, Name: name})
	require.NoError(t, err)
	require.Len(t, versions.Items, 2)
	assert.Equal(t, versioning.OperationUpdate, versions.Items[0].Operation)
	assert.Equal(t, versioning.OperationCreate, versions.Items[1].Operation)

	_, err = RollbackRuleVersion(ctx, RuleRef{Kind: meshresource.AffinityRouteKind, Name: name}, versions.Items[1].VersionNo, "restore affinity", "admin")
	require.NoError(t, err)
	actual, err := GetAffinityRule(ctx, name, "")
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.EqualValues(t, 50, actual.Spec.Affinity.Ratio)

	require.NoError(t, DeleteAffinityRuleWithOptions(ctx, name, "", RuleMutationOptions{Author: "admin"}))
}

func TestScriptRuleLifecycleAndRollback(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	key := "demo-provider"
	name := key + ".script-router"
	initial := scriptRouterRule(key, "return [invokers[0]];")

	require.NoError(t, CreateScriptRuleWithOptions(ctx, initial, RuleMutationOptions{Author: "admin"}))
	require.NoError(t, UpdateScriptRuleWithOptions(ctx, scriptRouterRule(key, "return [invokers[1]];"), RuleMutationOptions{Author: "admin"}))

	versions, err := ListRuleVersions(ctx, RuleRef{Kind: meshresource.ScriptRouteKind, Name: name})
	require.NoError(t, err)
	require.Len(t, versions.Items, 2)
	assert.Equal(t, versioning.OperationUpdate, versions.Items[0].Operation)
	assert.Equal(t, versioning.OperationCreate, versions.Items[1].Operation)

	_, err = RollbackRuleVersion(ctx, RuleRef{Kind: meshresource.ScriptRouteKind, Name: name}, versions.Items[1].VersionNo, "restore script", "admin")
	require.NoError(t, err)
	actual, err := GetScriptRule(ctx, name, "")
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, initial.Spec.Script, actual.Spec.Script)

	require.NoError(t, DeleteScriptRuleWithOptions(ctx, name, "", RuleMutationOptions{Author: "admin"}))
}

func TestRouterRuleMutationRejectsInvalidRuntimeContract(t *testing.T) {
	ctx := setupRollbackTestEnv(t)
	invalid := scriptRouterRule("demo-provider", "return invokers;")
	invalid.Spec.Type = "groovy"

	err := CreateScriptRuleWithOptions(ctx, invalid, RuleMutationOptions{Author: "admin"})
	require.Error(t, err)
	versions, historyErr := ListRuleVersions(ctx, RuleRef{Kind: meshresource.ScriptRouteKind, Name: invalid.Name})
	require.NoError(t, historyErr)
	assert.Empty(t, versions.Items)
}

func affinityRouterRule(key string, ratio int32) *meshresource.AffinityRouteResource {
	res := meshresource.NewAffinityRouteResourceWithAttributes(key+".affinity-router", "")
	res.Spec = &meshproto.AffinityRoute{
		ConfigVersion: "v3.1",
		Scope:         "application",
		Key:           key,
		Runtime:       true,
		Enabled:       true,
		Affinity:      &meshproto.AffinityAware{Key: "region", Ratio: ratio},
	}
	return res
}

func scriptRouterRule(key, script string) *meshresource.ScriptRouteResource {
	res := meshresource.NewScriptRouteResourceWithAttributes(key+".script-router", "")
	res.Spec = &meshproto.ScriptRoute{
		ConfigVersion: "v3.0",
		Scope:         "application",
		Key:           key,
		Enabled:       true,
		Type:          "javascript",
		Script:        script,
	}
	return res
}
