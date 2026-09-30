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

package v1alpha1

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

func TestAffinityRuleCodecUsesPublicAffinityAwareField(t *testing.T) {
	res := NewAffinityRouteResourceWithAttributes("demo.affinity-router", "mesh")
	res.Spec = &meshproto.AffinityRoute{
		ConfigVersion: "v3.1", Scope: "application", Key: "demo", Runtime: true,
		Enabled: false, Affinity: &meshproto.AffinityAware{Key: "region", Ratio: 0},
	}
	raw, err := EncodeRule(res)
	require.NoError(t, err)
	require.Contains(t, string(raw), "affinityAware:")
	require.NotContains(t, string(raw), "affinity:\n")
	require.Contains(t, string(raw), "enabled: false")
	decoded, err := DecodeRule(AffinityRouteKind, "mesh", res.Name, string(raw))
	require.NoError(t, err)
	require.Equal(t, res.Spec, decoded.(*AffinityRouteResource).Spec)
}

func TestScriptRuleCodecAndValidation(t *testing.T) {
	res := NewScriptRouteResourceWithAttributes("provider.script-router", "mesh")
	res.Spec = &meshproto.ScriptRoute{
		ConfigVersion: "v3.0", Scope: "application", Key: "provider",
		Enabled: true, Type: "javascript", Script: "return invokers;",
	}
	raw, err := EncodeRule(res)
	require.NoError(t, err)
	decoded, err := DecodeRule(ScriptRouteKind, "mesh", res.Name, string(raw))
	require.NoError(t, err)
	require.Equal(t, res.Spec, decoded.(*ScriptRouteResource).Spec)
	res.Spec.Type = "lua"
	require.Error(t, ValidateRule(res))
	res.Spec.Type = "javascript"
	res.Spec.Script = "  \n"
	require.Error(t, ValidateRule(res))
	res.Spec.Script = strings.Repeat("x", 64*1024+1)
	require.Error(t, ValidateRule(res))
}

func TestRouterRuleCodecPreservesOmittedBooleans(t *testing.T) {
	affinityYAML := "configVersion: v3.1\n" +
		"scope: application\n" +
		"key: demo\n" +
		"affinityAware:\n" +
		"  key: region\n" +
		"  ratio: 50\n"
	affinity, err := DecodeRule(AffinityRouteKind, "mesh", "demo.affinity-router", affinityYAML)
	require.NoError(t, err)
	affinityRaw, err := EncodeRule(affinity)
	require.NoError(t, err)
	require.NotContains(t, string(affinityRaw), "runtime:")
	require.NotContains(t, string(affinityRaw), "enabled:")
	copyRaw, err := EncodeRule(affinity.(*AffinityRouteResource).DeepCopyObject().(*AffinityRouteResource))
	require.NoError(t, err)
	require.NotContains(t, string(copyRaw), "runtime:")
	require.NotContains(t, string(copyRaw), "enabled:")

	scriptYAML := "configVersion: v3.0\n" +
		"scope: application\n" +
		"key: demo\n" +
		"type: javascript\n" +
		"script: return invokers;\n"
	script, err := DecodeRule(ScriptRouteKind, "mesh", "demo.script-router", scriptYAML)
	require.NoError(t, err)
	scriptRaw, err := EncodeRule(script)
	require.NoError(t, err)
	require.NotContains(t, string(scriptRaw), "enabled:")
}

func TestRouterRuleCodecPreservesExplicitFalseBooleans(t *testing.T) {
	affinityYAML := "configVersion: v3.1\n" +
		"scope: application\n" +
		"key: demo\n" +
		"runtime: false\n" +
		"enabled: false\n" +
		"affinityAware:\n" +
		"  key: region\n" +
		"  ratio: 50\n"
	affinity, err := DecodeRule(AffinityRouteKind, "mesh", "demo.affinity-router", affinityYAML)
	require.NoError(t, err)
	affinityRaw, err := EncodeRule(affinity)
	require.NoError(t, err)
	require.Contains(t, string(affinityRaw), "runtime: false")
	require.Contains(t, string(affinityRaw), "enabled: false")

	scriptYAML := "configVersion: v3.0\n" +
		"scope: application\n" +
		"key: demo\n" +
		"enabled: false\n" +
		"type: javascript\n" +
		"script: return invokers;\n"
	script, err := DecodeRule(ScriptRouteKind, "mesh", "demo.script-router", scriptYAML)
	require.NoError(t, err)
	scriptRaw, err := EncodeRule(script)
	require.NoError(t, err)
	require.Contains(t, string(scriptRaw), "enabled: false")
}

func TestRouterRuleCodecWritesFalseForNewResources(t *testing.T) {
	affinity := NewAffinityRouteResourceWithAttributes("demo.affinity-router", "mesh")
	affinity.Spec = &meshproto.AffinityRoute{
		ConfigVersion: "v3.1", Scope: "application", Key: "demo",
		Affinity: &meshproto.AffinityAware{Key: "region", Ratio: 50},
	}
	affinityRaw, err := EncodeRule(affinity)
	require.NoError(t, err)
	require.Contains(t, string(affinityRaw), "runtime: false")
	require.Contains(t, string(affinityRaw), "enabled: false")

	script := NewScriptRouteResourceWithAttributes("demo.script-router", "mesh")
	script.Spec = &meshproto.ScriptRoute{
		ConfigVersion: "v3.0", Scope: "application", Key: "demo",
		Type: "javascript", Script: "return invokers;",
	}
	scriptRaw, err := EncodeRule(script)
	require.NoError(t, err)
	require.Contains(t, string(scriptRaw), "enabled: false")
}

func TestDecodeRuleRejectsInvalidNonEmptyRules(t *testing.T) {
	tests := []struct {
		name         string
		kind         coremodel.ResourceKind
		resourceName string
		rule         string
	}{
		{
			name:         "affinity ratio",
			kind:         AffinityRouteKind,
			resourceName: "demo.affinity-router",
			rule: "configVersion: v3.1\n" +
				"scope: application\n" +
				"key: demo\n" +
				"affinityAware:\n" +
				"  key: region\n" +
				"  ratio: 101\n",
		},
		{
			name:         "affinity scope",
			kind:         AffinityRouteKind,
			resourceName: "demo.affinity-router",
			rule: "configVersion: v3.1\n" +
				"scope: method\n" +
				"key: demo\n" +
				"affinityAware:\n" +
				"  key: region\n" +
				"  ratio: 50\n",
		},
		{
			name:         "script type",
			kind:         ScriptRouteKind,
			resourceName: "demo.script-router",
			rule: "configVersion: v3.0\n" +
				"scope: application\n" +
				"key: demo\n" +
				"type: lua\n" +
				"script: return invokers;\n",
		},
		{
			name:         "empty script",
			kind:         ScriptRouteKind,
			resourceName: "demo.script-router",
			rule: "configVersion: v3.0\n" +
				"scope: application\n" +
				"key: demo\n" +
				"type: javascript\n" +
				"script: '  '\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeRule(tt.kind, "mesh", tt.resourceName, tt.rule)
			require.Error(t, err)
		})
	}
	require.Nil(t, ToAffinityRouteResource("mesh", "demo.affinity-router", tests[0].rule))
	require.Nil(t, ToScriptRouteResource("mesh", "demo.script-router", tests[2].rule))
}

func TestRouterRuleValidation(t *testing.T) {
	affinity := NewAffinityRouteResourceWithAttributes("provider.affinity-router", "mesh")
	affinity.Spec = &meshproto.AffinityRoute{
		ConfigVersion: "v3.0", Scope: "application", Key: "provider",
		Affinity: &meshproto.AffinityAware{Key: "region", Ratio: 50},
	}
	require.Error(t, ValidateRule(affinity))
	affinity.Spec.ConfigVersion = "v3.1"
	affinity.Spec.Affinity.Ratio = 101
	require.Error(t, ValidateRule(affinity))
}

func TestRouterRuleTombstones(t *testing.T) {
	a := ToAffinityRouteResource("mesh", "a.affinity-router", "")
	s := ToScriptRouteResource("mesh", "s.script-router", "")
	require.Equal(t, AffinityRouteKind, a.ResourceKind())
	require.Equal(t, ScriptRouteKind, s.ResourceKind())
}
