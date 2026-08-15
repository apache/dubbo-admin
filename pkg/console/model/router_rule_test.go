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

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConditionRuleInputUsesVersionSpecificConditions(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		conditions string
		wantLegacy int
		wantV31    int
	}{
		{name: "v3.0", version: "v3.0", conditions: `["=> application=demo"]`, wantLegacy: 1},
		{name: "v3.1", version: "v3.1", conditions: `[{"from":{"match":"method=SayHello"},"to":[{"match":"application=demo","weight":0}]}]`, wantV31: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &ConditionRuleInput{
				ConfigVersion: tt.version,
				Conditions:    json.RawMessage(tt.conditions),
			}
			got, err := input.ToProto()
			require.NoError(t, err)
			assert.Len(t, got.Conditions, tt.wantLegacy)
			assert.Len(t, got.ConditionRules, tt.wantV31)
			if tt.wantV31 != 0 {
				assert.Zero(t, got.ConditionRules[0].To[0].Weight)
			}
		})
	}
}

func TestConditionRuleInputRejectsCrossVersionConditionShape(t *testing.T) {
	_, err := (&ConditionRuleInput{
		ConfigVersion: "v3.1",
		Conditions:    json.RawMessage(`["=> application=demo"]`),
	}).ToProto()
	assert.Error(t, err)
}

func TestGenConditionRuleToRespPreservesV31Fields(t *testing.T) {
	rule := &meshproto.ConditionRoute{
		ConfigVersion: constants.ConfiguratorVersionV3x1,
		Priority:      7,
		Enabled:       true,
		Force:         true,
		Runtime:       true,
		Key:           "org.apache.dubbo.quickstart.Greeter:1.0.0:demo",
		Scope:         constants.ScopeService,
		ConditionRules: []*meshproto.ConditionRule{
			{
				From: &meshproto.ConditionRuleFrom{Match: "method=SayHello"},
				To:   []*meshproto.ConditionRuleTo{{Match: "application=quickstart-provider", Weight: 100}},
			},
		},
	}

	encoded, err := json.Marshal(GenConditionRuleToResp(rule))
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"code": "Success",
		"message": "success",
		"data": {
			"configVersion": "v3.1",
			"priority": 7,
			"enabled": true,
			"force": true,
			"runtime": true,
			"key": "org.apache.dubbo.quickstart.Greeter:1.0.0:demo",
			"scope": "service",
			"conditions": [{
				"from": {"match": "method=SayHello"},
				"to": [{"match": "application=quickstart-provider", "weight": 100}]
			}]
		}
	}`, string(encoded))
}
