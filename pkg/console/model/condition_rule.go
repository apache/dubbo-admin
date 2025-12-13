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
	"strings"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type SearchConditionRuleReq struct {
	coremodel.PageReq

	Keywords string `json:"keywords"`
}

func (s *SearchConditionRuleReq) PageRequest() coremodel.PageReq {
	return s.PageReq
}

func NewSearchConditionRuleReq() *SearchConditionRuleReq {
	return &SearchConditionRuleReq{
		PageReq: coremodel.PageReq{PageSize: 15},
	}
}

type ConditionRuleSearchResp struct {
	CreateTime string `json:"createTime"`
	Enabled    bool   `json:"enabled"`
	RuleName   string `json:"ruleName"`
	Scope      string `json:"scope"`
}

type ConditionRuleResp struct {
	Conditions    []string `json:"conditions"`
	ConfigVersion string   `json:"configVersion"`
	Enabled       bool     `json:"enabled"`
	Key           string   `json:"key"`
	Runtime       bool     `json:"runtime"`
	Scope         string   `json:"scope"`
}

type ServiceArgumentRoute struct {
	Routes []ServiceArgument `json:"routes"`
}

type ServiceArgument struct {
	Conditions   []RouteCondition `json:"conditions"`
	Destinations []Destination    `json:"destinations"`
	Method       string           `json:"method"`
}

func (s *ServiceArgument) toFrom() *meshproto.ConditionRuleFrom {
	res := "method=" + s.Method
	if len(s.Conditions) != 0 {
		for i := 0; len(s.Conditions) > i; i++ {
			res += " & " + s.Conditions[i].string()
		}
	}
	return &meshproto.ConditionRuleFrom{
		Match: res,
	}
}

func (s *ServiceArgument) toTo() []*meshproto.ConditionRuleTo {
	res := make([]*meshproto.ConditionRuleTo, 0, len(s.Destinations))
	for _, destination := range s.Destinations {
		match := ""
		for _, condition := range destination.Conditions {
			if match == "" {
				match += condition.string()
			} else {
				match += " & " + condition.string()
			}
		}
		res = append(res, &meshproto.ConditionRuleTo{
			Match:  match,
			Weight: destination.Weight,
		})
	}
	return res
}

type RouteCondition struct {
	Index    string `json:"index"`
	Relation string `json:"relation"`
	Value    string `json:"value"`
}

func (r *RouteCondition) string() string {
	if r.Relation == constants.Equal {
		return "arguments[" + r.Index + "]" + constants.Equal + r.Value
	} else {
		return "arguments[" + r.Index + "]" + constants.NotEqual + r.Value
	}
}

type Destination struct {
	Conditions []DestinationCondition `json:"conditions"`
	Weight     int32                  `json:"weight"`
}

type DestinationCondition struct {
	Relation string `json:"relation"`
	Tag      string `json:"tag"`
	Value    string `json:"value"`
}

func (d *DestinationCondition) string() string {
	if d.Relation == constants.Equal {
		return d.Tag + constants.Equal + d.Value
	} else {
		return d.Tag + constants.NotEqual + d.Value
	}
}

func (s *ServiceArgumentRoute) ToConditionV3x1Condition() []*meshproto.ConditionRule {
	res := make([]*meshproto.ConditionRule, 0, len(s.Routes))
	for _, route := range s.Routes {
		res = append(res, &meshproto.ConditionRule{
			From: route.toFrom(),
			To:   route.toTo(),
		})
	}
	return res
}

func ConditionV3x1ToServiceArgumentRoute(mesh []*meshproto.ConditionRule) *ServiceArgumentRoute {
	res := &ServiceArgumentRoute{
		Routes: make([]ServiceArgument, 0, len(mesh)),
	}
	for i := range mesh {
		method, _ := mesh[i].IsMatchMethod()
		cond := ServiceArgument{
			Conditions:   matchValueToRouteCondition(mesh[i].From.Match),
			Destinations: make([]Destination, 0, len(mesh[i].To)),
			Method:       method,
		}
		for _, to := range mesh[i].To {
			cond.Destinations = append(cond.Destinations, Destination{
				Conditions: matchValueToDestinationCondition(to.Match),
				Weight:     to.Weight,
			})
		}
		res.Routes = append(res.Routes, cond)
	}
	return res
}

func matchValueToRouteCondition(val string) []RouteCondition {
	subsets := strings.Split(val, "&")
	res := make([]RouteCondition, 0, len(subsets))
	for _, subset := range subsets {
		if index := strings.Index(subset, constants.NotEqual); index != -1 {
			res = append(res, RouteCondition{
				Index:    strings.Trim(subset[:index], " "),
				Relation: constants.NotEqual,
				Value:    strings.Trim(subset[index+len(constants.NotEqual):], " "),
			})
		} else if index := strings.Index(subset, constants.Equal); index != -1 {
			res = append(res, RouteCondition{
				Index:    strings.Trim(subset[:index], " "),
				Relation: constants.Equal,
				Value:    strings.Trim(subset[index+len(constants.Equal):], " "),
			})
		}
	}
	return res
}

func matchValueToDestinationCondition(val string) []DestinationCondition {
	subsets := strings.Split(val, "&")
	res := make([]DestinationCondition, 0, len(subsets))
	for _, subset := range subsets {
		if index := strings.Index(subset, constants.NotEqual); index != -1 {
			res = append(res, DestinationCondition{
				Tag:      strings.Trim(subset[:index], " "),
				Relation: constants.NotEqual,
				Value:    strings.Trim(subset[index+len(constants.NotEqual):], " "),
			})
		} else if index := strings.Index(subset, constants.Equal); index != -1 {
			res = append(res, DestinationCondition{
				Tag:      strings.Trim(subset[:index], " "),
				Relation: constants.Equal,
				Value:    strings.Trim(subset[index+len(constants.Equal):], " "),
			})
		}
	}
	return res
}

func GenConditionRuleToResp(data *meshproto.ConditionRoute) *CommonResp {
	if data == nil {
		return NewSuccessResp(nil)
	}
	if pb := data.ToConditionRouteV3(); pb != nil {
		return NewSuccessResp(ConditionRuleResp{
			Conditions:    pb.Conditions,
			ConfigVersion: pb.ConfigVersion,
			Enabled:       pb.Enabled,
			Key:           pb.Key,
			Runtime:       pb.Runtime,
			Scope:         pb.Scope,
		})
	} else if pb := data.ToConditionRouteV3x1(); pb != nil {
		res := ConditionRuleV3X1{
			Conditions:    make([]Condition, 0, len(pb.Conditions)),
			ConfigVersion: "v3.1",
			Enabled:       pb.Enabled,
			Force:         pb.Force,
			Key:           pb.Key,
			Runtime:       pb.Runtime,
			Scope:         pb.Scope,
		}
		for _, condition := range pb.Conditions {
			resCondition := Condition{
				From: ConditionFrom{Match: condition.From.Match},
				To:   make([]ConditionTo, 0, len(condition.To)),
			}
			for _, to := range condition.To {
				resCondition.To = append(resCondition.To, ConditionTo{
					Match:  to.Match,
					Weight: to.Weight,
				})
			}
			res.Conditions = append(res.Conditions, resCondition)
		}
		return NewSuccessResp(res)
	} else {
		return NewErrorResp("invalid condition rule")
	}
}

type ConditionRuleV3X1 struct {
	Conditions    []Condition `json:"conditions"`
	ConfigVersion string      `json:"configVersion"`
	Enabled       bool        `json:"enabled"`
	Force         bool        `json:"force"`
	Key           string      `json:"key"`
	Runtime       bool        `json:"runtime"`
	Scope         string      `json:"scope"`
}

type AffinityAware struct {
	Enabled bool   `json:"enabled"`
	Key     string `json:"key"`
}

type Condition struct {
	From ConditionFrom `json:"from"`
	To   []ConditionTo `json:"to"`
}

type ConditionFrom struct {
	Match string `json:"match"`
}

type ConditionTo struct {
	Match  string `json:"match"`
	Weight int32  `json:"weight"`
}
