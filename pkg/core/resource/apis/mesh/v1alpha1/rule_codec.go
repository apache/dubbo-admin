/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0.
 */

package v1alpha1

import (
	"fmt"
	"strings"

	"sigs.k8s.io/yaml"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type conditionRouteV31YAML struct {
	ConfigVersion string              `json:"configVersion"`
	Priority      int32               `json:"priority,omitempty"`
	Enabled       bool                `json:"enabled"`
	Force         bool                `json:"force"`
	Runtime       bool                `json:"runtime"`
	Key           string              `json:"key"`
	Scope         string              `json:"scope"`
	Conditions    []conditionRuleYAML `json:"conditions"`
}

type conditionRouteV30YAML struct {
	ConfigVersion string   `json:"configVersion"`
	Priority      int32    `json:"priority,omitempty"`
	Enabled       bool     `json:"enabled"`
	Force         bool     `json:"force"`
	Runtime       bool     `json:"runtime"`
	Key           string   `json:"key"`
	Scope         string   `json:"scope"`
	Conditions    []string `json:"conditions"`
}

type conditionRuleYAML struct {
	From conditionRuleFromYAML `json:"from"`
	To   []conditionRuleToYAML `json:"to"`
}

type conditionRuleFromYAML struct {
	Match string `json:"match"`
}

type conditionRuleToYAML struct {
	Match  string `json:"match"`
	Weight int32  `json:"weight"`
}

type affinityRouteYAML struct {
	ConfigVersion string             `json:"configVersion"`
	Scope         string             `json:"scope"`
	Key           string             `json:"key"`
	Runtime       bool               `json:"runtime"`
	Enabled       bool               `json:"enabled"`
	AffinityAware *affinityAwareYAML `json:"affinityAware"`
}

type affinityAwareYAML struct {
	Key   string `json:"key"`
	Ratio int32  `json:"ratio"`
}

type scriptRouteYAML struct {
	ConfigVersion string `json:"configVersion"`
	Scope         string `json:"scope"`
	Key           string `json:"key"`
	Enabled       bool   `json:"enabled"`
	Type          string `json:"type"`
	Script        string `json:"script"`
}

type tagRouteYAML struct {
	Priority      int32            `json:"priority,omitempty"`
	Enabled       bool             `json:"enabled"`
	Runtime       bool             `json:"runtime"`
	Key           string           `json:"key"`
	ConfigVersion string           `json:"configVersion"`
	Force         bool             `json:"force"`
	Tags          []*meshproto.Tag `json:"tags"`
}

type dynamicConfigYAML struct {
	Key           string                      `json:"key"`
	Scope         string                      `json:"scope"`
	ConfigVersion string                      `json:"configVersion"`
	Enabled       bool                        `json:"enabled"`
	Configs       []*meshproto.OverrideConfig `json:"configs"`
}

// EncodeRule serializes an internal rule resource to the public Dubbo dynamic
// configuration YAML contract.
func EncodeRule(r coremodel.Resource) ([]byte, error) {
	if err := ValidateRule(r); err != nil {
		return nil, err
	}
	var value any = r.ResourceSpec()
	switch typed := r.(type) {
	case *ConditionRouteResource:
		if typed.Spec.ConfigVersion == constants.ConfiguratorVersionV3x1 {
			value = conditionRouteV31YAML{
				ConfigVersion: typed.Spec.ConfigVersion,
				Priority:      typed.Spec.Priority,
				Enabled:       typed.Spec.Enabled,
				Force:         typed.Spec.Force,
				Runtime:       typed.Spec.Runtime,
				Key:           typed.Spec.Key,
				Scope:         typed.Spec.Scope,
				Conditions:    conditionRulesToYAML(typed.Spec.ConditionRules),
			}
		} else {
			value = conditionRouteV30YAML{
				ConfigVersion: typed.Spec.ConfigVersion, Priority: typed.Spec.Priority,
				Enabled: typed.Spec.Enabled, Force: typed.Spec.Force, Runtime: typed.Spec.Runtime,
				Key: typed.Spec.Key, Scope: typed.Spec.Scope, Conditions: typed.Spec.Conditions,
			}
		}
	case *AffinityRouteResource:
		var affinity *affinityAwareYAML
		if typed.Spec.Affinity != nil {
			affinity = &affinityAwareYAML{Key: typed.Spec.Affinity.Key, Ratio: typed.Spec.Affinity.Ratio}
		}
		value = affinityRouteYAML{
			ConfigVersion: typed.Spec.ConfigVersion,
			Scope:         typed.Spec.Scope,
			Key:           typed.Spec.Key,
			Runtime:       typed.Spec.Runtime,
			Enabled:       typed.Spec.Enabled,
			AffinityAware: affinity,
		}
	case *ScriptRouteResource:
		value = scriptRouteYAML{
			ConfigVersion: typed.Spec.ConfigVersion,
			Scope:         typed.Spec.Scope,
			Key:           typed.Spec.Key,
			Enabled:       typed.Spec.Enabled,
			Type:          typed.Spec.Type,
			Script:        typed.Spec.Script,
		}
	case *TagRouteResource:
		value = tagRouteYAML{
			Priority:      typed.Spec.Priority,
			Enabled:       typed.Spec.Enabled,
			Runtime:       typed.Spec.Runtime,
			Key:           typed.Spec.Key,
			ConfigVersion: typed.Spec.ConfigVersion,
			Force:         typed.Spec.Force,
			Tags:          typed.Spec.Tags,
		}
	case *DynamicConfigResource:
		value = dynamicConfigYAML{
			Key:           typed.Spec.Key,
			Scope:         typed.Spec.Scope,
			ConfigVersion: typed.Spec.ConfigVersion,
			Enabled:       typed.Spec.Enabled,
			Configs:       typed.Spec.Configs,
		}
	}
	data, err := yaml.Marshal(value)
	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.YamlError, "failed to marshal rule")
	}
	return data, nil
}

// DecodeRule parses public Dubbo dynamic configuration YAML without losing
// version-specific fields.
func DecodeRule(kind coremodel.ResourceKind, mesh, name, data string) (coremodel.Resource, error) {
	var r coremodel.Resource
	switch kind {
	case DynamicConfigKind:
		r = NewDynamicConfigResourceWithAttributes(name, mesh)
	case TagRouteKind:
		r = NewTagRouteResourceWithAttributes(name, mesh)
	case ScriptRouteKind:
		r = NewScriptRouteResourceWithAttributes(name, mesh)
	case AffinityRouteKind:
		external := &affinityRouteYAML{}
		if err := yaml.Unmarshal([]byte(data), external); err != nil {
			return nil, ruleYAMLError(name, err)
		}
		r = NewAffinityRouteResourceWithAttributes(name, mesh)
		var affinity *meshproto.AffinityAware
		if external.AffinityAware != nil {
			affinity = &meshproto.AffinityAware{Key: external.AffinityAware.Key, Ratio: external.AffinityAware.Ratio}
		}
		r.(*AffinityRouteResource).Spec = &meshproto.AffinityRoute{
			ConfigVersion: external.ConfigVersion,
			Scope:         external.Scope, Key: external.Key, Runtime: external.Runtime,
			Enabled: external.Enabled, Affinity: affinity,
		}
	case ConditionRouteKind:
		version := struct {
			ConfigVersion string `json:"configVersion"`
		}{}
		if err := yaml.Unmarshal([]byte(data), &version); err != nil {
			return nil, ruleYAMLError(name, err)
		}
		r = NewConditionRouteResourceWithAttributes(name, mesh)
		if version.ConfigVersion == constants.ConfiguratorVersionV3x1 {
			external := &conditionRouteV31YAML{}
			if err := yaml.Unmarshal([]byte(data), external); err != nil {
				return nil, ruleYAMLError(name, err)
			}
			r.(*ConditionRouteResource).Spec = &meshproto.ConditionRoute{
				ConfigVersion: external.ConfigVersion, Priority: external.Priority,
				Enabled: external.Enabled, Force: external.Force, Runtime: external.Runtime,
				Key: external.Key, Scope: external.Scope, ConditionRules: conditionRulesFromYAML(external.Conditions),
			}
		} else if err := yaml.Unmarshal([]byte(data), r.(*ConditionRouteResource).Spec); err != nil {
			return nil, ruleYAMLError(name, err)
		}
	default:
		return nil, bizerror.New(bizerror.InvalidArgument, fmt.Sprintf("unsupported rule kind %s", kind))
	}
	if kind != AffinityRouteKind && kind != ConditionRouteKind {
		if err := yaml.Unmarshal([]byte(data), r.ResourceSpec()); err != nil {
			return nil, ruleYAMLError(name, err)
		}
	}
	return r, nil
}

func conditionRulesToYAML(rules []*meshproto.ConditionRule) []conditionRuleYAML {
	result := make([]conditionRuleYAML, 0, len(rules))
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		from := conditionRuleFromYAML{}
		if rule.From != nil {
			from.Match = rule.From.Match
		}
		to := make([]conditionRuleToYAML, 0, len(rule.To))
		for _, destination := range rule.To {
			if destination != nil {
				to = append(to, conditionRuleToYAML{Match: destination.Match, Weight: destination.Weight})
			}
		}
		result = append(result, conditionRuleYAML{From: from, To: to})
	}
	return result
}

func conditionRulesFromYAML(rules []conditionRuleYAML) []*meshproto.ConditionRule {
	result := make([]*meshproto.ConditionRule, 0, len(rules))
	for _, rule := range rules {
		to := make([]*meshproto.ConditionRuleTo, 0, len(rule.To))
		for _, destination := range rule.To {
			to = append(to, &meshproto.ConditionRuleTo{Match: destination.Match, Weight: destination.Weight})
		}
		result = append(result, &meshproto.ConditionRule{From: &meshproto.ConditionRuleFrom{Match: rule.From.Match}, To: to})
	}
	return result
}

func ruleYAMLError(name string, err error) error {
	return bizerror.Wrap(err, bizerror.YamlError, fmt.Sprintf("invalid rule YAML %s", name))
}

// ValidateRule verifies the external contract before a governor writes it.
func ValidateRule(r coremodel.Resource) error {
	if r == nil || r.ResourceSpec() == nil {
		return invalidRule("rule spec is required")
	}
	name := r.ResourceMeta().Name
	var scope, key, version, suffix string
	switch typed := r.(type) {
	case *DynamicConfigResource:
		scope, key, version, suffix = typed.Spec.Scope, typed.Spec.Key, typed.Spec.ConfigVersion, constants.ConfiguratorRuleDotSuffix
	case *TagRouteResource:
		key, version, suffix = typed.Spec.Key, typed.Spec.ConfigVersion, constants.TagRuleDotSuffix
		if name != key+suffix {
			return invalidRule("tag rule name must be <application>.tag-router")
		}
		return validateVersion(version)
	case *ConditionRouteResource:
		scope, key, version, suffix = typed.Spec.Scope, typed.Spec.Key, typed.Spec.ConfigVersion, constants.ConditionRuleDotSuffix
		if version == constants.ConfiguratorVersionV3x1 {
			if len(typed.Spec.Conditions) != 0 || len(typed.Spec.ConditionRules) == 0 {
				return invalidRule("v3.1 condition rules require structured conditions only")
			}
			for _, condition := range typed.Spec.ConditionRules {
				if condition == nil || condition.From == nil || strings.TrimSpace(condition.From.Match) == "" || len(condition.To) == 0 {
					return invalidRule("each v3.1 condition requires from.match and at least one destination")
				}
				for _, to := range condition.To {
					if to == nil || strings.TrimSpace(to.Match) == "" || to.Weight < 0 || to.Weight > 100 {
						return invalidRule("condition destinations require match and weight in [0, 100]")
					}
				}
			}
		} else if len(typed.Spec.ConditionRules) != 0 {
			return invalidRule("v3.0 condition rules cannot contain structured conditions")
		}
	case *AffinityRouteResource:
		scope, key, version, suffix = typed.Spec.Scope, typed.Spec.Key, typed.Spec.ConfigVersion, constants.AffinityRuleDotSuffix
		if typed.Spec.Affinity == nil || strings.TrimSpace(typed.Spec.Affinity.Key) == "" || typed.Spec.Affinity.Ratio < 0 || typed.Spec.Affinity.Ratio > 100 {
			return invalidRule("affinityAware.key is required and ratio must be in [0, 100]")
		}
	case *ScriptRouteResource:
		scope, key, version, suffix = typed.Spec.Scope, typed.Spec.Key, typed.Spec.ConfigVersion, constants.ScriptRuleDotSuffix
		if scope != constants.ScopeApplication {
			return invalidRule("script rules only support application scope")
		}
		if typed.Spec.Type != constants.ScriptTypeJavaScript {
			return invalidRule("script type must be javascript")
		}
		if strings.TrimSpace(typed.Spec.Script) == "" || len(typed.Spec.Script) > constants.MaxScriptRuleSize {
			return invalidRule(fmt.Sprintf("script must be non-empty and no larger than %d bytes", constants.MaxScriptRuleSize))
		}
	default:
		return invalidRule(fmt.Sprintf("unsupported rule kind %s", r.ResourceKind()))
	}
	if err := validateVersion(version); err != nil {
		return err
	}
	if scope != constants.ScopeApplication && scope != constants.ScopeService {
		return invalidRule("scope must be application or service")
	}
	if strings.TrimSpace(key) == "" || name != key+suffix {
		return invalidRule(fmt.Sprintf("rule name must be key%s", suffix))
	}
	return nil
}

func validateVersion(version string) error {
	if version != constants.ConfiguratorVersionV3 && version != constants.ConfiguratorVersionV3x1 {
		return invalidRule("configVersion must be v3.0 or v3.1")
	}
	return nil
}

func invalidRule(message string) error {
	return bizerror.New(bizerror.InvalidArgument, message)
}

func toRuleResource(kind coremodel.ResourceKind, mesh, name, data string) coremodel.Resource {
	if strings.TrimSpace(data) == "" {
		switch kind {
		case DynamicConfigKind:
			return NewDynamicConfigResourceWithAttributes(name, mesh)
		case TagRouteKind:
			return NewTagRouteResourceWithAttributes(name, mesh)
		case ConditionRouteKind:
			return NewConditionRouteResourceWithAttributes(name, mesh)
		case AffinityRouteKind:
			return NewAffinityRouteResourceWithAttributes(name, mesh)
		case ScriptRouteKind:
			return NewScriptRouteResourceWithAttributes(name, mesh)
		}
	}
	r, err := DecodeRule(kind, mesh, name, data)
	if err != nil {
		return nil
	}
	return r
}

func ToAffinityRouteResource(mesh, name, data string) coremodel.Resource {
	return toRuleResource(AffinityRouteKind, mesh, name, data)
}

func ToScriptRouteResource(mesh, name, data string) coremodel.Resource {
	return toRuleResource(ScriptRouteKind, mesh, name, data)
}
