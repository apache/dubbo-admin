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

package versioning

import (
	"testing"

	"github.com/stretchr/testify/require"

	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

func TestNormalizeResourceDistinguishesOmittedAndExplicitFalseScriptEnabled(t *testing.T) {
	omitted := decodeScriptRule(t, `
configVersion: v3.0
scope: application
key: demo
type: javascript
script: return invokers;
`)
	explicitFalse := decodeScriptRule(t, `
configVersion: v3.0
scope: application
key: demo
enabled: false
type: javascript
script: return invokers;
`)

	omittedHash, omittedSnapshot, err := NormalizeResource(omitted)
	require.NoError(t, err)
	explicitHash, explicitSnapshot, err := NormalizeResource(explicitFalse)
	require.NoError(t, err)
	require.NotEqual(t, omittedHash, explicitHash)
	require.NotContains(t, omittedSnapshot, `"omittedFields"`)
	require.NotContains(t, explicitSnapshot, `"omittedFields"`)

	restored, err := ResourceFromSpecJSON(meshresource.ScriptRouteKind, "mesh", "demo.script-router", omittedSnapshot)
	require.NoError(t, err)
	raw, err := meshresource.EncodeRule(restored)
	require.NoError(t, err)
	require.Contains(t, string(raw), "enabled: true")

	restored, err = ResourceFromSpecJSON(meshresource.ScriptRouteKind, "mesh", "demo.script-router", explicitSnapshot)
	require.NoError(t, err)
	raw, err = meshresource.EncodeRule(restored)
	require.NoError(t, err)
	require.Contains(t, string(raw), "enabled: false")
}

func TestAffinitySnapshotPreservesOptionalRuntimeAndEnabled(t *testing.T) {
	omitted, err := meshresource.DecodeRule(meshresource.AffinityRouteKind, "mesh", "demo.affinity-router", `
configVersion: v3.1
scope: application
key: demo
affinityAware:
  key: region
  ratio: 50
`)
	require.NoError(t, err)
	explicitFalse, err := meshresource.DecodeRule(meshresource.AffinityRouteKind, "mesh", "demo.affinity-router", `
configVersion: v3.1
scope: application
key: demo
runtime: false
enabled: false
affinityAware:
  key: region
  ratio: 50
`)
	require.NoError(t, err)

	omittedHash, snapshot, err := NormalizeResource(omitted)
	require.NoError(t, err)
	explicitHash, explicitSnapshot, err := NormalizeResource(explicitFalse)
	require.NoError(t, err)
	require.NotEqual(t, omittedHash, explicitHash)
	require.Contains(t, snapshot, `"omittedFields":["runtime","enabled"]`)
	require.NotContains(t, explicitSnapshot, `"omittedFields"`)

	restored, err := ResourceFromSpecJSON(meshresource.AffinityRouteKind, "mesh", "demo.affinity-router", snapshot)
	require.NoError(t, err)
	raw, err := meshresource.EncodeRule(restored)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "runtime:")
	require.NotContains(t, string(raw), "enabled:")

	restored, err = ResourceFromSpecJSON(meshresource.AffinityRouteKind, "mesh", "demo.affinity-router", explicitSnapshot)
	require.NoError(t, err)
	raw, err = meshresource.EncodeRule(restored)
	require.NoError(t, err)
	require.Contains(t, string(raw), "runtime: false")
	require.Contains(t, string(raw), "enabled: false")
}

func TestResourceFromSpecJSONAcceptsLegacyScriptSnapshots(t *testing.T) {
	resource, err := ResourceFromSpecJSON(meshresource.ScriptRouteKind, "mesh", "demo.script-router", `{
"configVersion":"v3.0","scope":"application","key":"demo","type":"javascript","script":"return invokers;"
}`)
	require.NoError(t, err)
	raw, err := meshresource.EncodeRule(resource)
	require.NoError(t, err)
	require.Contains(t, string(raw), "enabled: false")
}

func decodeScriptRule(t *testing.T, rule string) *meshresource.ScriptRouteResource {
	t.Helper()
	resource, err := meshresource.DecodeRule(meshresource.ScriptRouteKind, "mesh", "demo.script-router", rule)
	require.NoError(t, err)
	return resource.(*meshresource.ScriptRouteResource)
}
