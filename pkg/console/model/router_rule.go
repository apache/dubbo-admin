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
	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

// RouterRuleSearchResp is intentionally body-free; script text is returned
// only by the detail endpoint.
type RouterRuleSearchResp struct {
	CreateTime string `json:"createTime"`
	Enabled    bool   `json:"enabled"`
	RuleName   string `json:"ruleName"`
	Scope      string `json:"scope"`
}

// AffinityRuleInput is the public Console representation. affinityAware is
// mapped to the internal protobuf field named affinity.
type AffinityRuleInput struct {
	ConfigVersion string                   `json:"configVersion"`
	Scope         string                   `json:"scope"`
	Key           string                   `json:"key"`
	Runtime       *bool                    `json:"runtime,omitempty"`
	Enabled       *bool                    `json:"enabled,omitempty"`
	AffinityAware *meshproto.AffinityAware `json:"affinityAware"`
}

func (i *AffinityRuleInput) ToProto() *meshproto.AffinityRoute {
	return &meshproto.AffinityRoute{
		ConfigVersion: i.ConfigVersion,
		Scope:         i.Scope,
		Key:           i.Key,
		Runtime:       optionalBoolValue(i.Runtime),
		Enabled:       optionalBoolValue(i.Enabled),
		Affinity:      i.AffinityAware,
	}
}

func GenAffinityRuleResp(res *meshresource.AffinityRouteResource) *CommonResp {
	if res == nil || res.Spec == nil {
		return NewSuccessResp(nil)
	}
	data := res.Spec
	runtime, enabled := meshresource.AffinityRuleOptionalFields(res)
	return NewSuccessResp(AffinityRuleInput{
		ConfigVersion: data.ConfigVersion,
		Scope:         data.Scope,
		Key:           data.Key,
		Runtime:       runtime,
		Enabled:       enabled,
		AffinityAware: data.Affinity,
	})
}

type ScriptRuleInput struct {
	ConfigVersion string `json:"configVersion"`
	Scope         string `json:"scope"`
	Key           string `json:"key"`
	Enabled       *bool  `json:"enabled,omitempty"`
	Type          string `json:"type"`
	Script        string `json:"script"`
}

func (i *ScriptRuleInput) ToProto() *meshproto.ScriptRoute {
	return &meshproto.ScriptRoute{
		ConfigVersion: i.ConfigVersion,
		Scope:         i.Scope,
		Key:           i.Key,
		Enabled:       optionalBoolValue(i.Enabled),
		Type:          i.Type,
		Script:        i.Script,
	}
}

func GenScriptRuleResp(res *meshresource.ScriptRouteResource) *CommonResp {
	if res == nil || res.Spec == nil {
		return NewSuccessResp(nil)
	}
	data := res.Spec
	return NewSuccessResp(ScriptRuleInput{
		ConfigVersion: data.ConfigVersion,
		Scope:         data.Scope,
		Key:           data.Key,
		Enabled:       meshresource.ScriptRuleOptionalEnabled(res),
		Type:          data.Type,
		Script:        data.Script,
	})
}

func optionalBoolValue(value *bool) bool {
	return value != nil && *value
}
