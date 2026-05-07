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
	"github.com/apache/dubbo-admin/pkg/common/constants"
)

type TagRuleSearchResp struct {
	CreateTime string `json:"createTime,omitempty"`
	Enabled    bool   `json:"enabled,omitempty"`
	RuleName   string `json:"ruleName,omitempty"`
}

type TagRuleResp struct {
	ConfigVersion string           `json:"configVersion"`
	Enabled       bool             `json:"enabled"`
	Key           string           `json:"key"`
	Runtime       bool             `json:"runtime"`
	Scope         string           `json:"scope"`
	Tags          []RespTagElement `json:"tags"`
}

type RespTagElement struct {
	Addresses []string     `json:"addresses,omitempty"`
	Match     []ParamMatch `json:"match,omitempty"`
	Name      string       `json:"name"`
}

func GenTagRouteResp(pb *meshproto.TagRoute) *CommonResp {
	if pb == nil {
		return NewSuccessResp(nil)
	} else {
		return NewSuccessResp(TagRuleResp{
			ConfigVersion: pb.ConfigVersion,
			Enabled:       pb.Enabled,
			Key:           pb.Key,
			Runtime:       pb.Runtime,
			Scope:         constants.ScopeApplication,
			Tags:          tagToRespTagElement(pb.Tags),
		})
	}
}

func tagToRespTagElement(tags []*meshproto.Tag) []RespTagElement {
	res := make([]RespTagElement, 0, len(tags))
	if tags != nil {
		for _, tag := range tags {
			res = append(res, RespTagElement{
				Addresses: tag.Addresses,
				Match:     paramMatchToRespParamMatch(tag.Match),
				Name:      tag.Name,
			})
		}
	}
	return res
}
