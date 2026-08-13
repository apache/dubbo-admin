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
	Mesh     string `form:"mesh"`
	Keywords string `form:"keywords"`
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
	Force         bool     `json:"force"`
	Key           string   `json:"key"`
	Priority      int32    `json:"priority"`
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

// ToExpression Convert ServiceArgument to expression string
func (sa *ServiceArgument) ToExpression() string {
	expression := "method=" + sa.Method

	// Add route conditions
	for _, condition := range sa.Conditions {
		expression += " & " + condition.string()
	}

	// Add destinations if any
	if len(sa.Destinations) > 0 {
		expression += " => "

		// Process each destination
		for destIdx, destination := range sa.Destinations {
			for condIdx, condition := range destination.Conditions {
				if destIdx > 0 || condIdx > 0 {
					expression += " & "
				}
				expression += condition.string()
			}

			// If there are multiple destinations, separate them (using comma as separator)
			if destIdx < len(sa.Destinations)-1 {
				expression += ", "
			}
		}
	}

	return expression
}

func (sa *ServiceArgument) toFrom() *meshproto.ConditionRuleFrom {
	res := "method=" + sa.Method
	if len(sa.Conditions) != 0 {
		for i := 0; len(sa.Conditions) > i; i++ {
			res += " & " + sa.Conditions[i].string()
		}
	}
	return &meshproto.ConditionRuleFrom{
		Match: res,
	}
}

func (sa *ServiceArgument) toTo() []*meshproto.ConditionRuleTo {
	res := make([]*meshproto.ConditionRuleTo, 0, len(sa.Destinations))
	for _, destination := range sa.Destinations {
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

// ParseConditionExpression converts an expression string to a ServiceArgument struct
// Expression format: "method=methodName & condition1 => destination1"
func ParseConditionExpression(expression string) ServiceArgument {
	// Split into from and to parts (separated by "=>")
	parts := strings.Split(expression, "=>")
	fromPart := strings.TrimSpace(parts[0])
	toPart := ""
	if len(parts) > 1 {
		toPart = strings.TrimSpace(parts[1])
	}

	// Parse from part to get method and conditions
	method, conditions := parseFromPart(fromPart)

	// Parse to part to get destinations
	destinations := parseToPart(toPart)

	return ServiceArgument{
		Method:       method,
		Conditions:   conditions,
		Destinations: destinations,
	}
}

// parseFromPart parses the from part containing method and arguments conditions
func parseFromPart(fromPart string) (string, []RouteCondition) {
	conditions := []RouteCondition{}
	method := ""

	// Split conditions by "&"
	subConditions := strings.Split(fromPart, "&")

	for _, condition := range subConditions {
		condition = strings.TrimSpace(condition)

		// Check if it's a method condition
		if strings.HasPrefix(condition, "method=") {
			method = strings.TrimPrefix(condition, "method=")
			method = strings.TrimSpace(method)
		} else {
			// Parse arguments condition
			routeCond := parseRouteCondition(condition)
			if routeCond.Index != "" {
				conditions = append(conditions, routeCond)
			}
		}
	}

	return method, conditions
}

// parseRouteCondition parses a single route condition
func parseRouteCondition(condition string) RouteCondition {
	var relation string
	var relationOp string

	// Check if it's "=" or "!="
	if index := strings.Index(condition, constants.NotEqual); index != -1 {
		relation = constants.NotEqual
		relationOp = constants.NotEqual
	} else if index := strings.Index(condition, constants.Equal); index != -1 {
		relation = constants.Equal
		relationOp = constants.Equal
	} else {
		return RouteCondition{} // Invalid condition
	}

	// Find the position of the relation
	index := strings.Index(condition, relationOp)
	leftPart := strings.TrimSpace(condition[:index])
	rightPart := strings.TrimSpace(condition[index+len(relationOp):])

	// Parse arguments[index] format
	if strings.HasPrefix(leftPart, "arguments[") && strings.HasSuffix(leftPart, "]") {
		argIndex := leftPart[10 : len(leftPart)-1] // Extract index within brackets
		return RouteCondition{
			Index:    argIndex,
			Relation: relation,
			Value:    rightPart,
		}
	}

	// If not arguments format, return generic condition
	return RouteCondition{
		Index:    leftPart,
		Relation: relation,
		Value:    rightPart,
	}
}

// parseToPart parses the to part (destination conditions)
func parseToPart(toPart string) []Destination {
	if toPart == "" {
		return []Destination{}
	}

	// Split multiple conditions by "&"
	conditions := strings.Split(toPart, "&")
	destinationConditions := []DestinationCondition{}

	for _, condition := range conditions {
		condition = strings.TrimSpace(condition)
		destCond := parseDestinationCondition(condition)
		if destCond.Tag != "" {
			destinationConditions = append(destinationConditions, destCond)
		}
	}

	return []Destination{
		{
			Conditions: destinationConditions,
			Weight:     0, // Default weight is 0, can be adjusted as needed
		},
	}
}

// parseDestinationCondition parses destination condition
func parseDestinationCondition(condition string) DestinationCondition {
	var relation string
	var relationOp string

	// Check if it's "=" or "!="
	if index := strings.Index(condition, constants.NotEqual); index != -1 {
		relation = constants.NotEqual
		relationOp = constants.NotEqual
	} else if index := strings.Index(condition, constants.Equal); index != -1 {
		relation = constants.Equal
		relationOp = constants.Equal
	} else {
		return DestinationCondition{} // Invalid condition
	}

	// Find the position of the relation
	index := strings.Index(condition, relationOp)
	leftPart := strings.TrimSpace(condition[:index])
	rightPart := strings.TrimSpace(condition[index+len(relationOp):])

	return DestinationCondition{
		Tag:      leftPart,
		Relation: relation,
		Value:    rightPart,
	}
}
