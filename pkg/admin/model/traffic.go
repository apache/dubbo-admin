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

	"github.com/apache/dubbo-admin/pkg/admin/util"
)

const (
	RegionAdminIdentifier   string = " & region_admin_rule!=false"
	ArgumentAdminIdentifier string = " & arg_admin_rule!=false"
)

type Timeout struct {
	Service string `json:"service" binding:"required"`
	Group   string `json:"group"`
	Version string `json:"version"`
	Timeout int    `json:"timeout" binding:"required"`
}

func (t Timeout) ToRule() Override {
	return Override{
		Key:           util.ServiceKey(t.Service, t.Group, t.Version),
		Scope:         "service",
		ConfigVersion: "v3.0",
		Enabled:       true,
		Configs: []OverrideConfig{{
			Side:       "consumer",
			Enabled:    true,
			Parameters: map[string]interface{}{"timeout": t.Timeout},
		}},
	}
}

func (t Timeout) GetKey() string {
	interfaze := util.GetInterface(t.Service)
	group := util.GetGroup(t.Service)
	version := util.GetVersion(t.Service)
	return util.ColonSeparatedKey(interfaze, group, version)
}

type Retry struct {
	Service string `json:"service" binding:"required"`
	Group   string `json:"group"`
	Version string `json:"version"`
	Retry   int    `json:"retry" binding:"required"`
}

func (t Retry) ToRule() Override {
	return Override{
		Key:           t.Service,
		Scope:         "service",
		ConfigVersion: "v3.0",
		Enabled:       true,
		Configs: []OverrideConfig{{
			Side:       "consumer",
			Parameters: map[string]interface{}{"retries": t.Retry},
		}},
	}
}

func (r Retry) GetKey() string {
	interfaze := util.GetInterface(r.Service)
	group := util.GetGroup(r.Service)
	version := util.GetVersion(r.Service)
	return util.ColonSeparatedKey(interfaze, group, version)
}

type Accesslog struct {
	Application string `json:"application" binding:"required"`
	Accesslog   string `json:"accesslog"`
}

func (t Accesslog) ToRule() Override {
	return Override{
		Key:           t.Application,
		Scope:         "application",
		ConfigVersion: "v3.0",
		Enabled:       true,
		Configs: []OverrideConfig{{
			Side:       "provider",
			Parameters: map[string]interface{}{"accesslog": t.Accesslog},
		}},
	}
}

type Region struct {
	Service string `json:"service" binding:"required"`
	Group   string `json:"group"`
	Version string `json:"version"`
	Rule    string `json:"rule"`
}

func (r Region) ToRule() ConditionRoute {
	return ConditionRoute{
		Enabled:       true,
		Force:         false,
		Runtime:       true,
		Key:           r.Service,
		Scope:         "service",
		ConfigVersion: "v3.0",
		Conditions:    []string{strings.Join([]string{"=> ", r.Rule, "=$", r.Rule, RegionAdminIdentifier}, "")},
	}
}

func (r Region) GetKey() string {
	interfaze := util.GetInterface(r.Service)
	group := util.GetGroup(r.Service)
	version := util.GetVersion(r.Service)
	return util.ColonSeparatedKey(interfaze, group, version)
}

type Gray struct {
	Application string `json:"application" binding:"required"`
	Tags        []Tag  `json:"tags" binding:"required"`
}

func (g Gray) ToRule() TagRoute {
	return TagRoute{
		Enabled:       true,
		Force:         true,
		Runtime:       true,
		Key:           g.Application,
		ConfigVersion: "v3.0",
		Tags:          g.Tags,
	}
}

type Argument struct {
	Service string `json:"service" binding:"required"`
	Group   string `json:"group"`
	Version string `json:"version"`
	Rule    string `json:"rule" binding:"required"`
}

func (r Argument) ToRule() ConditionRoute {
	return ConditionRoute{
		Enabled:       true,
		Force:         true,
		Runtime:       true,
		Key:           r.Service,
		Scope:         "service",
		ConfigVersion: "v3.0",
		Conditions:    []string{r.Rule + ArgumentAdminIdentifier},
	}
}

func (a Argument) GetKey() string {
	interfaze := util.GetInterface(a.Service)
	group := util.GetGroup(a.Service)
	version := util.GetVersion(a.Service)
	return util.ColonSeparatedKey(interfaze, group, version)
}

type Percentage struct {
	Service string   `json:"service" binding:"required"`
	Group   string   `json:"group"`
	Version string   `json:"version"`
	Weights []Weight `json:"weights" binding:"required"`
}
type Weight struct {
	Weight int            `json:"weight" binding:"required"`
	Match  ConditionMatch `json:"match"  binding:"required"`
}

func (p Percentage) ToRule() Override {
	configs := make([]OverrideConfig, len(p.Weights))
	for _, weight := range p.Weights {
		configs = append(configs, OverrideConfig{
			Side:       "provider",
			Match:      weight.Match,
			Parameters: map[string]interface{}{"weight": weight.Weight},
		})
	}
	return Override{
		Key:           p.Service,
		Scope:         "service",
		ConfigVersion: "v3.0",
		Enabled:       true,
		Configs:       configs,
	}
}

func (p Percentage) GetKey() string {
	interfaze := util.GetInterface(p.Service)
	group := util.GetGroup(p.Service)
	version := util.GetVersion(p.Service)
	return util.ColonSeparatedKey(interfaze, group, version)
}

type Mock struct {
	Service string `json:"service" binding:"required"`
	Group   string `json:"group"`
	Version string `json:"version"`
	Mock    string `json:"mock" binding:"required"`
}

func (m Mock) ToRule() Override {
	return Override{
		Key:           m.Service,
		Scope:         "service",
		ConfigVersion: "v3.0",
		Enabled:       true,
		Configs: []OverrideConfig{{
			Side:       "consumer",
			Parameters: map[string]interface{}{"mock": m.Mock},
		}},
	}
}

func (m Mock) GetKey() string {
	interfaze := util.GetInterface(m.Service)
	group := util.GetGroup(m.Service)
	version := util.GetVersion(m.Service)
	return util.ColonSeparatedKey(interfaze, group, version)
}

type Host struct {
	Condition string `json:"condition" binding:"required"`
	Host      string `json:"host" binding:"required"`
}
