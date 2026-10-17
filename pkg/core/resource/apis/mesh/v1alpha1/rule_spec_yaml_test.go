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
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

func TestMarshalRuleSpecPreservesRouterBooleans(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, tc := range []struct {
			name string
			spec coremodel.ResourceSpec
		}{
			{"condition", &meshproto.ConditionRoute{
				ConfigVersion: "v3.0", Key: "demo-provider", Scope: "application",
				Enabled: enabled, Force: false, Runtime: false,
				Conditions: []string{"=>host!=192.0.2.1"},
			}},
			{"tag", &meshproto.TagRoute{
				ConfigVersion: "v3.0", Key: "demo-provider",
				Enabled: enabled, Force: false, Runtime: false,
				Tags: []*meshproto.Tag{{Name: "gray", Addresses: []string{"192.0.2.1:20880"}}},
			}},
		} {
			name := tc.name + "/disabled"
			if enabled {
				name = tc.name + "/enabled"
			}
			t.Run(name, func(t *testing.T) {
				raw, err := MarshalRuleSpec(tc.spec)
				require.NoError(t, err)
				var fields map[string]any
				require.NoError(t, yaml.Unmarshal(raw, &fields))
				for field, want := range map[string]bool{"enabled": enabled, "force": false, "runtime": false} {
					require.Contains(t, fields, field, "%s must not be omitted: %s", field, raw)
					require.Equal(t, want, fields[field])
				}
				require.Equal(t, "demo-provider", fields["key"])
				switch tc.spec.(type) {
				case *meshproto.ConditionRoute:
					var got meshproto.ConditionRoute
					require.NoError(t, yaml.Unmarshal(raw, &got))
					require.Equal(t, tc.spec, &got)
				case *meshproto.TagRoute:
					var got meshproto.TagRoute
					require.NoError(t, yaml.Unmarshal(raw, &got))
					require.Equal(t, tc.spec, &got)
				}
			})
		}
	}
}

func TestMarshalRuleSpecOtherKindsUnchanged(t *testing.T) {
	spec := &meshproto.AffinityRoute{ConfigVersion: "v3.1", Enabled: false, Runtime: false}
	want, err := yaml.Marshal(spec)
	require.NoError(t, err)
	got, err := MarshalRuleSpec(spec)
	require.NoError(t, err)
	require.Equal(t, string(want), string(got))

	var typedNil *meshproto.TagRoute
	want, err = yaml.Marshal(typedNil)
	require.NoError(t, err)
	got, err = MarshalRuleSpec(typedNil)
	require.NoError(t, err)
	require.Equal(t, string(want), string(got))

	want, err = yaml.Marshal(nil)
	require.NoError(t, err)
	got, err = MarshalRuleSpec(nil)
	require.NoError(t, err)
	require.Equal(t, string(want), string(got))
}
