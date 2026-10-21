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
	"fmt"
	"strings"

	"sigs.k8s.io/yaml"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// The rule protobufs use scalar bools, so annotations retain field presence
// from config-center YAML without changing generated resource types. Metadata
// is copied and persisted by the existing resource store implementations.
const (
	affinityRuntimeOmittedAnnotation = "dubbo.apache.org/affinity-runtime-omitted"
)

// The protobuf field is named affinity for historical/internal reasons. The
// public Dubbo 3 contract is affinityAware, so never marshal the protobuf
// message directly for these resources.
type affinityRouteYAML struct {
	ConfigVersion string             `json:"configVersion"`
	Scope         string             `json:"scope"`
	Key           string             `json:"key"`
	Runtime       *bool              `json:"runtime,omitempty"`
	Enabled       *bool              `json:"enabled,omitempty"`
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
	Enabled       *bool  `json:"enabled,omitempty"`
	Type          string `json:"type"`
	Script        string `json:"script"`
}

// EncodeRule serializes a rule resource using the public Dubbo YAML contract.
// Existing rule kinds use MarshalRuleSpec so their explicit router booleans
// survive protobuf's omitempty tags. Affinity and Script use explicit codecs
// because their public field names and boolean defaults differ from the
// internal protobuf shape. Their Enabled fields are always written because
// consumers do not safely handle omitted values.
func EncodeRule(r coremodel.Resource) ([]byte, error) {
	if err := ValidateRule(r); err != nil {
		return nil, err
	}
	var value any
	switch typed := r.(type) {
	case *AffinityRouteResource:
		var aware *affinityAwareYAML
		if typed.Spec.Affinity != nil {
			aware = &affinityAwareYAML{Key: typed.Spec.Affinity.Key, Ratio: typed.Spec.Affinity.Ratio}
		}
		value = affinityRouteYAML{
			ConfigVersion: typed.Spec.ConfigVersion,
			Scope:         typed.Spec.Scope,
			Key:           typed.Spec.Key,
			Runtime:       optionalRuleBool(typed.Spec.Runtime, typed.Annotations, affinityRuntimeOmittedAnnotation),
			Enabled:       &typed.Spec.Enabled,
			AffinityAware: aware,
		}
	case *ScriptRouteResource:
		value = scriptRouteYAML{
			ConfigVersion: typed.Spec.ConfigVersion,
			Scope:         typed.Spec.Scope,
			Key:           typed.Spec.Key,
			// Dubbo-Go's ScriptRouter dereferences Enabled, so Script rules must
			// always publish this field. Unlike Affinity rules, enabled is not
			// optional in the Script rule contract.
			Enabled: &typed.Spec.Enabled,
			Type:    typed.Spec.Type,
			Script:  typed.Spec.Script,
		}
	default:
		if r == nil || r.ResourceSpec() == nil {
			return nil, invalidRule("rule spec is required")
		}
		return MarshalRuleSpec(r.ResourceSpec())
	}
	data, err := yaml.Marshal(value)
	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.YamlError, "failed to marshal rule")
	}
	return data, nil
}

// DecodeRule parses a public rule YAML document into a typed resource. It is
// used by both registry subscribers and governor read-back paths.
func DecodeRule(kind coremodel.ResourceKind, mesh, name, data string) (coremodel.Resource, error) {
	if strings.TrimSpace(data) == "" {
		switch kind {
		case AffinityRouteKind:
			return NewAffinityRouteResourceWithAttributes(name, mesh), nil
		case ScriptRouteKind:
			return NewScriptRouteResourceWithAttributes(name, mesh), nil
		}
	}
	switch kind {
	case AffinityRouteKind:
		external := affinityRouteYAML{}
		if err := yaml.Unmarshal([]byte(data), &external); err != nil {
			return nil, ruleYAMLError(name, err)
		}
		res := NewAffinityRouteResourceWithAttributes(name, mesh)
		if external.AffinityAware != nil {
			res.Spec.Affinity = &meshproto.AffinityAware{
				Key:   external.AffinityAware.Key,
				Ratio: external.AffinityAware.Ratio,
			}
		}
		res.Spec.ConfigVersion = external.ConfigVersion
		res.Spec.Scope = external.Scope
		res.Spec.Key = external.Key
		res.Spec.Enabled = true
		if external.Runtime != nil {
			res.Spec.Runtime = *external.Runtime
			clearRuleBooleanOmitted(&res.Annotations, affinityRuntimeOmittedAnnotation)
		} else {
			markRuleBooleanOmitted(&res.Annotations, affinityRuntimeOmittedAnnotation)
		}
		if external.Enabled != nil {
			res.Spec.Enabled = *external.Enabled
		}
		if err := ValidateRule(res); err != nil {
			return nil, err
		}
		return res, nil
	case ScriptRouteKind:
		external := scriptRouteYAML{}
		if err := yaml.Unmarshal([]byte(data), &external); err != nil {
			return nil, ruleYAMLError(name, err)
		}
		res := NewScriptRouteResourceWithAttributes(name, mesh)
		res.Spec.ConfigVersion = external.ConfigVersion
		res.Spec.Scope = external.Scope
		res.Spec.Key = external.Key
		// Treat old configurations which omit enabled as enabled. This keeps
		// Console's next write compatible with ScriptRouter by making the
		// normalized value explicit in the published YAML.
		res.Spec.Enabled = true
		if external.Enabled != nil {
			res.Spec.Enabled = *external.Enabled
		}
		res.Spec.Type = external.Type
		res.Spec.Script = external.Script
		if err := ValidateRule(res); err != nil {
			return nil, err
		}
		return res, nil
	default:
		res, err := coremodel.ResourceSchemaRegistry().NewResourceFunc(kind)
		if err != nil {
			return nil, bizerror.New(bizerror.InvalidArgument, fmt.Sprintf("unsupported rule kind %s", kind))
		}
		r := res()
		if err := yaml.Unmarshal([]byte(data), r.ResourceSpec()); err != nil {
			return nil, ruleYAMLError(name, err)
		}
		return r, nil
	}
}

func optionalRuleBool(value bool, annotations map[string]string, omittedAnnotation string) *bool {
	if !value && annotations[omittedAnnotation] == "true" {
		return nil
	}
	return &value
}

// AffinityRuleOptionalFields returns the public Affinity values. Runtime is
// optional, while Enabled is normalized because an omitted enabled field can
// silently disable the rule in consumers.
func AffinityRuleOptionalFields(res *AffinityRouteResource) (runtime *bool, enabled *bool) {
	if res == nil || res.Spec == nil {
		return nil, nil
	}
	return optionalRuleBool(res.Spec.Runtime, res.Annotations, affinityRuntimeOmittedAnnotation),
		&res.Spec.Enabled
}

// SetAffinityRuleOptionalFields applies public Affinity values. Runtime keeps
// its field-presence metadata, but a missing Enabled value is normalized to
// true before the rule can be published.
func SetAffinityRuleOptionalFields(res *AffinityRouteResource, runtime, enabled *bool) {
	if res == nil || res.Spec == nil {
		return
	}
	setOptionalRuleBool(&res.Spec.Runtime, &res.Annotations, affinityRuntimeOmittedAnnotation, runtime)
	normalizedEnabled := true
	if enabled != nil {
		normalizedEnabled = *enabled
	}
	res.Spec.Enabled = normalizedEnabled
}

// ScriptRuleOptionalEnabled returns the normalized Script enabled value.
// ScriptRouter requires this field, so an omitted value is normalized to true.
func ScriptRuleOptionalEnabled(res *ScriptRouteResource) *bool {
	if res == nil || res.Spec == nil {
		return nil
	}
	return &res.Spec.Enabled
}

// SetScriptRuleOptionalEnabled applies the public enabled field. A missing
// input is normalized to true so all Script rule writes remain safe for
// ScriptRouter consumers.
func SetScriptRuleOptionalEnabled(res *ScriptRouteResource, enabled *bool) {
	if res == nil || res.Spec == nil {
		return
	}
	normalized := true
	if enabled != nil {
		normalized = *enabled
	}
	res.Spec.Enabled = normalized
}

func setOptionalRuleBool(value *bool, annotations *map[string]string, omittedAnnotation string, input *bool) {
	if input == nil {
		*value = false
		markRuleBooleanOmitted(annotations, omittedAnnotation)
		return
	}
	*value = *input
	clearRuleBooleanOmitted(annotations, omittedAnnotation)
}

func markRuleBooleanOmitted(annotations *map[string]string, key string) {
	if *annotations == nil {
		*annotations = make(map[string]string)
	}
	(*annotations)[key] = "true"
}

func clearRuleBooleanOmitted(annotations *map[string]string, key string) {
	delete(*annotations, key)
	if len(*annotations) == 0 {
		*annotations = nil
	}
}

func ruleYAMLError(name string, err error) error {
	return bizerror.Wrap(err, bizerror.YamlError, fmt.Sprintf("invalid rule YAML %s", name))
}

// ValidateRule enforces the part of the Admin/runtime contract that can be
// checked before a write reaches a configuration center.
func ValidateRule(r coremodel.Resource) error {
	if r == nil {
		return invalidRule("rule spec is required")
	}
	switch typed := r.(type) {
	case *AffinityRouteResource:
		if typed == nil || typed.Spec == nil {
			return invalidRule("rule spec is required")
		}
		if typed.Spec.ConfigVersion != constants.ConfiguratorVersionV3x1 {
			return invalidRule("affinity configVersion must be v3.1")
		}
		if typed.Spec.Scope != constants.ScopeApplication && typed.Spec.Scope != constants.ScopeService {
			return invalidRule("affinity scope must be application or service")
		}
		if strings.TrimSpace(typed.Spec.Key) == "" || typed.Name != typed.Spec.Key+constants.AffinityRuleDotSuffix {
			return invalidRule("affinity rule name must be key.affinity-router")
		}
		if typed.Spec.Affinity == nil || strings.TrimSpace(typed.Spec.Affinity.Key) == "" {
			return invalidRule("affinityAware.key is required")
		}
		if typed.Spec.Affinity.Ratio < 0 || typed.Spec.Affinity.Ratio > 100 {
			return invalidRule("affinityAware.ratio must be in [0, 100]")
		}
		return nil
	case *ScriptRouteResource:
		if typed == nil || typed.Spec == nil {
			return invalidRule("rule spec is required")
		}
		if typed.Spec.ConfigVersion != constants.ConfiguratorVersionV3 && typed.Spec.ConfigVersion != constants.ConfiguratorVersionV3x1 {
			return invalidRule("script configVersion must be v3.0 or v3.1")
		}
		if typed.Spec.Scope != constants.ScopeApplication {
			return invalidRule("script rules only support application scope")
		}
		if strings.TrimSpace(typed.Spec.Key) == "" || typed.Name != typed.Spec.Key+constants.ScriptRuleDotSuffix {
			return invalidRule("script rule name must be key.script-router")
		}
		if typed.Spec.Type != constants.ScriptTypeJavaScript {
			return invalidRule("script type must be javascript")
		}
		if strings.TrimSpace(typed.Spec.Script) == "" {
			return invalidRule("script must be non-empty")
		}
		if len([]byte(typed.Spec.Script)) > constants.MaxScriptRuleSize {
			return invalidRule(fmt.Sprintf("script must not exceed %d bytes", constants.MaxScriptRuleSize))
		}
		return nil
	default:
		if r.ResourceSpec() == nil {
			return invalidRule("rule spec is required")
		}
		return nil
	}
}

func invalidRule(message string) error {
	return bizerror.New(bizerror.InvalidArgument, message)
}

// ToRuleResourceFunc adapters used by configuration-center lister/watchers.
// Empty content is a tombstone and must still produce a typed resource so the
// informer can emit a delete event.
func ToAffinityRouteResource(mesh, name, data string) coremodel.Resource {
	if strings.TrimSpace(data) == "" {
		return NewAffinityRouteResourceWithAttributes(name, mesh)
	}
	r, err := DecodeRule(AffinityRouteKind, mesh, name, data)
	if err != nil {
		return nil
	}
	return r
}

func ToScriptRouteResource(mesh, name, data string) coremodel.Resource {
	if strings.TrimSpace(data) == "" {
		return NewScriptRouteResourceWithAttributes(name, mesh)
	}
	r, err := DecodeRule(ScriptRouteKind, mesh, name, data)
	if err != nil {
		return nil
	}
	return r
}
