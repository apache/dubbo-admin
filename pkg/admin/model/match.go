// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"regexp"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
)

type ConditionMatch struct {
	Address     AddressMatch    `json:"address" yaml:"address,omitempty"`
	Service     ListStringMatch `json:"service" yaml:"service,omitempty"`
	Application ListStringMatch `json:"application" yaml:"application,omitempty"`
	Param       []ParamMatch    `json:"param" yaml:"param,omitempty"`
}

type AddressMatch struct {
	Wildcard string `json:"wildcard" yaml:"wildcard,omitempty"`
	Cird     string `json:"cird" yaml:"cird,omitempty"`
	Exact    string `json:"exact" yaml:"exact,omitempty"`
}

func (m *AddressMatch) IsMatch(input string) bool {
	// FIXME depends on dubbo-go/common/MatchIpExpression()
	// if m.Cird != "" && input != "" || common.MatchIpExpression(m.Cird, input) {
	if m.Cird != "" && input != "" {
		return input == m.Cird
	} else if m.Wildcard != "" && input != "" {
		if constant.AnyHostValue == m.Wildcard || constant.AnyValue == m.Wildcard {
			return true
		}
		// FIXME depends on dubbo-go/common/IsMatchGlobPattern()
		// return common.IsMatchGlobPattern(m.Wildcard, input)
	} else if m.Exact != "" && input != "" {
		return input == m.Exact
	}
	return false
}

type ParamMatch struct {
	Key   string      `json:"key" yaml:"key"`
	Value StringMatch `json:"value" yaml:"value"`
}

func (m *ParamMatch) IsMatch(url *common.URL) bool {
	if m.Key == "" {
		return false
	}
	input := url.GetParam(m.Key, "")
	return input != "" && m.Value.IsMatch(input)
}

type ListStringMatch struct {
	Oneof []StringMatch `json:"oneof" yaml:"oneof,omitempty"`
}

func (l *ListStringMatch) IsMatch(input string) bool {
	for _, m := range l.Oneof {
		if m.IsMatch(input) {
			return true
		}
	}
	return false
}

type StringMatch struct {
	Exact    string `json:"exact" yaml:"exact,omitempty"`
	Prefix   string `json:"prefix" yaml:"prefix,omitempty"`
	Regex    string `json:"regex" yaml:"regex,omitempty"`
	Noempty  string `json:"noempty" yaml:"noempty,omitempty"`
	Empty    string `json:"empty" yaml:"empty,omitempty"`
	Wildcard string `json:"wildcard" yaml:"wildcard,omitempty"`
}

func (m *StringMatch) IsMatch(input string) bool {
	if m.Exact != "" && input != "" {
		return input == m.Exact
	} else if m.Prefix != "" && input != "" {
		return strings.HasPrefix(input, m.Prefix)
	} else if m.Regex != "" && input != "" {
		return regexp.MustCompile(m.Regex).MatchString(input)
	} else if m.Wildcard != "" && input != "" {
		// only supports "*"
		return input == m.Wildcard || constant.AnyValue == m.Wildcard
	} else if m.Empty != "" {
		return input == ""
	} else if m.Noempty != "" {
		return input != ""
	} else {
		return false
	}
}
