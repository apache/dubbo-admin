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

import meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"

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
	Runtime       bool                     `json:"runtime"`
	Enabled       bool                     `json:"enabled"`
	AffinityAware *meshproto.AffinityAware `json:"affinityAware"`
}

func (i *AffinityRuleInput) ToProto() *meshproto.AffinityRoute {
	return &meshproto.AffinityRoute{
		ConfigVersion: i.ConfigVersion,
		Scope:         i.Scope,
		Key:           i.Key,
		Runtime:       i.Runtime,
		Enabled:       i.Enabled,
		Affinity:      i.AffinityAware,
	}
}

func GenAffinityRuleResp(data *meshproto.AffinityRoute) *CommonResp {
	if data == nil {
		return NewSuccessResp(nil)
	}
	return NewSuccessResp(AffinityRuleInput{
		ConfigVersion: data.ConfigVersion,
		Scope:         data.Scope,
		Key:           data.Key,
		Runtime:       data.Runtime,
		Enabled:       data.Enabled,
		AffinityAware: data.Affinity,
	})
}

type ScriptRuleInput struct {
	ConfigVersion string `json:"configVersion"`
	Scope         string `json:"scope"`
	Key           string `json:"key"`
	Enabled       bool   `json:"enabled"`
	Type          string `json:"type"`
	Script        string `json:"script"`
}

func (i *ScriptRuleInput) ToProto() *meshproto.ScriptRoute {
	return &meshproto.ScriptRoute{
		ConfigVersion: i.ConfigVersion,
		Scope:         i.Scope,
		Key:           i.Key,
		Enabled:       i.Enabled,
		Type:          i.Type,
		Script:        i.Script,
	}
}

func GenScriptRuleResp(data *meshproto.ScriptRoute) *CommonResp {
	if data == nil {
		return NewSuccessResp(nil)
	}
	return NewSuccessResp(ScriptRuleInput{
		ConfigVersion: data.ConfigVersion,
		Scope:         data.Scope,
		Key:           data.Key,
		Enabled:       data.Enabled,
		Type:          data.Type,
		Script:        data.Script,
	})
}
