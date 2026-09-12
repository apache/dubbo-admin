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

// The protobuf field is named affinity for historical/internal reasons. The
// public Dubbo 3 contract is affinityAware, so never marshal the protobuf
// message directly for these resources.
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

// EncodeRule serializes a rule resource using the public Dubbo YAML contract.
// Existing rule kinds fall back to their historical protobuf/YAML encoding;
// Affinity and Script use explicit codecs because their public field names and
// boolean defaults differ from the internal protobuf shape.
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
			Runtime:       typed.Spec.Runtime,
			Enabled:       typed.Spec.Enabled,
			AffinityAware: aware,
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
	default:
		if r == nil || r.ResourceSpec() == nil {
			return nil, invalidRule("rule spec is required")
		}
		value = r.ResourceSpec()
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
		res.Spec.Runtime = external.Runtime
		res.Spec.Enabled = external.Enabled
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
		res.Spec.Enabled = external.Enabled
		res.Spec.Type = external.Type
		res.Spec.Script = external.Script
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

func ruleYAMLError(name string, err error) error {
	return bizerror.Wrap(err, bizerror.YamlError, fmt.Sprintf("invalid rule YAML %s", name))
}

// ValidateRule enforces the part of the Admin/runtime contract that can be
// checked before a write reaches a configuration center.
func ValidateRule(r coremodel.Resource) error {
	if r == nil || r.ResourceSpec() == nil {
		return invalidRule("rule spec is required")
	}
	switch typed := r.(type) {
	case *AffinityRouteResource:
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
