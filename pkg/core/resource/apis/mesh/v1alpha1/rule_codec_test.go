/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0.
 */

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

func TestConditionRuleCodecRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		res  *ConditionRouteResource
	}{
		{
			name: "v3.0",
			res: conditionRouteForTest("demo.condition-router", &meshproto.ConditionRoute{
				ConfigVersion: "v3.0", Scope: "application", Key: "demo", Enabled: true,
				Conditions: []string{"env=gray => env=gray"},
			}),
		},
		{
			name: "v3.1",
			res: conditionRouteForTest("org.demo.Service:1.0.0:demo.condition-router", &meshproto.ConditionRoute{
				ConfigVersion: "v3.1", Scope: "service", Key: "org.demo.Service:1.0.0:demo", Enabled: true,
				ConditionRules: []*meshproto.ConditionRule{{
					From: &meshproto.ConditionRuleFrom{Match: "env=gray"},
					To: []*meshproto.ConditionRuleTo{
						{Match: "env=gray", Weight: 100},
						{Match: "env!=gray", Weight: 0},
					},
				}},
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := EncodeRule(tt.res)
			require.NoError(t, err)
			decoded, err := DecodeRule(ConditionRouteKind, tt.res.Mesh, tt.res.Name, string(raw))
			require.NoError(t, err)
			assert.Equal(t, tt.res.Spec, decoded.(*ConditionRouteResource).Spec)
			if tt.name == "v3.1" {
				assert.Contains(t, string(raw), "conditions:")
				assert.NotContains(t, string(raw), "conditionRules:")
				assert.Contains(t, string(raw), "weight: 0")
			}
		})
	}
}

func TestAffinityRuleCodecUsesAffinityAware(t *testing.T) {
	res := NewAffinityRouteResourceWithAttributes("demo.affinity-router", "default")
	res.Spec = &meshproto.AffinityRoute{
		ConfigVersion: "v3.1", Scope: "application", Key: "demo", Enabled: true,
		Affinity: &meshproto.AffinityAware{Key: "region", Ratio: 80},
	}
	raw, err := EncodeRule(res)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "affinityAware:")
	assert.NotContains(t, string(raw), "affinity:")
	decoded, err := DecodeRule(AffinityRouteKind, "default", res.Name, string(raw))
	require.NoError(t, err)
	assert.Equal(t, res.Spec, decoded.(*AffinityRouteResource).Spec)
}

func TestScriptRuleValidation(t *testing.T) {
	res := NewScriptRouteResourceWithAttributes("demo.script-router", "default")
	res.Spec = &meshproto.ScriptRoute{
		ConfigVersion: "v3.0", Scope: "application", Key: "demo", Enabled: true,
		Type: "javascript", Script: "return invokers;",
	}
	raw, err := EncodeRule(res)
	require.NoError(t, err)
	decoded, err := DecodeRule(ScriptRouteKind, "default", res.Name, string(raw))
	require.NoError(t, err)
	assert.Equal(t, res.Spec, decoded.(*ScriptRouteResource).Spec)

	res.Spec.Type = "lua"
	assert.Error(t, ValidateRule(res))
}

func TestValidateRuleRejectsInvalidRouterContracts(t *testing.T) {
	tests := []struct {
		name string
		res  coremodel.Resource
	}{
		{
			name: "affinity ratio",
			res: affinityRouteForTest("demo.affinity-router", &meshproto.AffinityRoute{
				ConfigVersion: "v3.1", Scope: "application", Key: "demo", Enabled: true,
				Affinity: &meshproto.AffinityAware{Key: "region", Ratio: 101},
			}),
		},
		{
			name: "affinity rule name",
			res: affinityRouteForTest("other.affinity-router", &meshproto.AffinityRoute{
				ConfigVersion: "v3.1", Scope: "application", Key: "demo", Enabled: true,
				Affinity: &meshproto.AffinityAware{Key: "region", Ratio: 80},
			}),
		},
		{
			name: "condition destination weight",
			res: conditionRouteForTest("demo.condition-router", &meshproto.ConditionRoute{
				ConfigVersion: "v3.1", Scope: "application", Key: "demo", Enabled: true,
				ConditionRules: []*meshproto.ConditionRule{{
					From: &meshproto.ConditionRuleFrom{Match: "method=SayHello"},
					To:   []*meshproto.ConditionRuleTo{{Match: "region=hangzhou", Weight: -1}},
				}},
			}),
		},
		{
			name: "condition empty destination",
			res: conditionRouteForTest("demo.condition-router", &meshproto.ConditionRoute{
				ConfigVersion: "v3.1", Scope: "application", Key: "demo", Enabled: true,
				ConditionRules: []*meshproto.ConditionRule{{
					From: &meshproto.ConditionRuleFrom{Match: "method=SayHello"},
				}},
			}),
		},
		{
			name: "script service scope",
			res: scriptRouteForTest("org.demo.Service:1.0.0:demo.script-router", &meshproto.ScriptRoute{
				ConfigVersion: "v3.0", Scope: "service", Key: "org.demo.Service:1.0.0:demo",
				Enabled: true, Type: "javascript", Script: "return invokers;",
			}),
		},
		{
			name: "script empty body",
			res: scriptRouteForTest("demo.script-router", &meshproto.ScriptRoute{
				ConfigVersion: "v3.0", Scope: "application", Key: "demo", Enabled: true,
				Type: "javascript", Script: " ",
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, ValidateRule(tt.res))
		})
	}
}

func TestRuleCodecPreservesExplicitDisabledState(t *testing.T) {
	res := NewScriptRouteResourceWithAttributes("demo.script-router", "default")
	res.Spec = &meshproto.ScriptRoute{ConfigVersion: "v3.0", Scope: "application", Key: "demo", Enabled: false, Type: "javascript", Script: "return invokers;"}
	raw, err := EncodeRule(res)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "enabled: false")
}

func TestRuleConvertersCreateDeleteTombstonesFromEmptyContent(t *testing.T) {
	tests := []struct {
		kind coremodel.ResourceKind
		fn   ToRuleResourceFunc
	}{
		{DynamicConfigKind, ToDynamicConfigResource},
		{ConditionRouteKind, ToConditionRouteResource},
		{TagRouteKind, ToTagRouteResource},
		{AffinityRouteKind, ToAffinityRouteResource},
		{ScriptRouteKind, ToScriptRouteResource},
	}
	for _, tt := range tests {
		r := tt.fn("default", "demo", "")
		require.NotNil(t, r)
		assert.Equal(t, tt.kind, r.ResourceKind())
		assert.Equal(t, "default/demo", r.ResourceKey())
	}
}

func TestDecodeRuleKeepsLegacyServiceKeyReadable(t *testing.T) {
	raw := `configVersion: v3.0
scope: service
key: org.apache.demo.DemoService
enabled: true
runtime: true
conditions:
  - => application=demo-provider
`
	r, err := DecodeRule(
		ConditionRouteKind,
		"default",
		"org.apache.demo.DemoService:1.0.0:demo.condition-router",
		raw,
	)
	require.NoError(t, err)
	assert.Equal(t, "org.apache.demo.DemoService", r.(*ConditionRouteResource).Spec.Key)
}

func conditionRouteForTest(name string, spec *meshproto.ConditionRoute) *ConditionRouteResource {
	r := NewConditionRouteResourceWithAttributes(name, "default")
	r.Spec = spec
	return r
}

func affinityRouteForTest(name string, spec *meshproto.AffinityRoute) *AffinityRouteResource {
	r := NewAffinityRouteResourceWithAttributes(name, "default")
	r.Spec = spec
	return r
}

func scriptRouteForTest(name string, spec *meshproto.ScriptRoute) *ScriptRouteResource {
	r := NewScriptRouteResourceWithAttributes(name, "default")
	r.Spec = spec
	return r
}
