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
