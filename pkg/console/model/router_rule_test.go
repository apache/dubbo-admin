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

package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

func TestScriptRuleDetailRoundTripPreservesOmittedEnabled(t *testing.T) {
	source, err := meshresource.DecodeRule(meshresource.ScriptRouteKind, "mesh", "demo.script-router", `
configVersion: v3.0
scope: application
key: demo
type: javascript
script: return invokers;
`)
	require.NoError(t, err)

	input := GenScriptRuleResp(source.(*meshresource.ScriptRouteResource)).Data.(ScriptRuleInput)
	require.Nil(t, input.Enabled)
	body, err := json.Marshal(input)
	require.NoError(t, err)
	require.NotContains(t, string(body), `"enabled"`)

	restored := meshresource.NewScriptRouteResourceWithAttributes("demo.script-router", "mesh")
	restored.Spec = input.ToProto()
	meshresource.SetScriptRuleOptionalEnabled(restored, input.Enabled)
	raw, err := meshresource.EncodeRule(restored)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "enabled:")
}

func TestAffinityRuleDetailRoundTripPreservesOptionalBooleans(t *testing.T) {
	source, err := meshresource.DecodeRule(meshresource.AffinityRouteKind, "mesh", "demo.affinity-router", `
configVersion: v3.1
scope: application
key: demo
affinityAware:
  key: region
  ratio: 50
`)
	require.NoError(t, err)

	input := GenAffinityRuleResp(source.(*meshresource.AffinityRouteResource)).Data.(AffinityRuleInput)
	require.Nil(t, input.Runtime)
	require.Nil(t, input.Enabled)
	body, err := json.Marshal(input)
	require.NoError(t, err)
	require.NotContains(t, string(body), `"runtime"`)
	require.NotContains(t, string(body), `"enabled"`)

	restored := meshresource.NewAffinityRouteResourceWithAttributes("demo.affinity-router", "mesh")
	restored.Spec = input.ToProto()
	meshresource.SetAffinityRuleOptionalFields(restored, input.Runtime, input.Enabled)
	raw, err := meshresource.EncodeRule(restored)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "runtime:")
	require.NotContains(t, string(raw), "enabled:")
}

func TestScriptRuleDetailRoundTripPreservesExplicitFalse(t *testing.T) {
	source, err := meshresource.DecodeRule(meshresource.ScriptRouteKind, "mesh", "demo.script-router", `
configVersion: v3.0
scope: application
key: demo
enabled: false
type: javascript
script: return invokers;
`)
	require.NoError(t, err)

	input := GenScriptRuleResp(source.(*meshresource.ScriptRouteResource)).Data.(ScriptRuleInput)
	require.NotNil(t, input.Enabled)
	require.False(t, *input.Enabled)

	restored := meshresource.NewScriptRouteResourceWithAttributes("demo.script-router", "mesh")
	restored.Spec = input.ToProto()
	meshresource.SetScriptRuleOptionalEnabled(restored, input.Enabled)
	raw, err := meshresource.EncodeRule(restored)
	require.NoError(t, err)
	require.Contains(t, string(raw), "enabled: false")
}
