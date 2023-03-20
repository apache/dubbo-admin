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
	"strings"

	"github.com/apache/dubbo-admin/pkg/admin/constant"
)

type Override struct {
	Key           string           `json:"key" yaml:"key"`
	Scope         string           `json:"scope" yaml:"scope"`
	ConfigVersion string           `json:"configVersion" yaml:"configVersion"`
	Enabled       bool             `json:"enabled" yaml:"enabled"`
	Configs       []OverrideConfig `json:"configs" yaml:"configs"`
}

type OverrideConfig struct {
	Side              string            `json:"side" yaml:"side"`
	Addresses         []string          `json:"addresses" yaml:"addresses"`
	ProviderAddresses []string          `json:"providerAddresses" yaml:"providerAddresses"`
	Parameters        map[string]string `json:"parameters" yaml:"parameters"`
	Applications      []string          `json:"applications" yaml:"applications"`
	Services          []string          `json:"services" yaml:"services"`
	Type              string            `json:"type" yaml:"type"`
	Enabled           bool              `json:"enabled" yaml:"enabled"`
	Match             ConditionMatch    `json:"match" yaml:"match"`
}

type ConditionMatch struct {
	Address     StringMatch  `json:"address" yaml:"address"`
	Service     StringMatch  `json:"service" yaml:"service"`
	Application StringMatch  `json:"application" yaml:"application"`
	Param       []ParamMatch `json:"param" yaml:"param"`
}

type ParamMatch struct {
	Key   string      `json:"key" yaml:"key"`
	Value StringMatch `json:"value" yaml:"value"`
}

type StringMatch struct {
	Exact  string `json:"exact" yaml:"exact"`
	Prefix string `json:"prefix" yaml:"prefix"`
	Regex  string `json:"regex" yaml:"regex"`
}

func (o *Override) ToDynamicConfig() *DynamicConfig {
	d := &DynamicConfig{}
	d.ConfigVersion = o.ConfigVersion

	configs := make([]OverrideConfig, 0, len(o.Configs))
	for _, c := range o.Configs {
		if c.Type == "" {
			configs = append(configs, c)
		}
	}

	if len(configs) == 0 {
		return nil
	}

	d.Configs = configs

	if o.Scope == constant.ApplicationKey {
		d.Application = o.Key
	} else {
		d.Service = o.Key
	}

	d.Enabled = o.Enabled
	return d
}

type OldOverride struct {
	Entity
	Service     string
	Address     string
	Enabled     bool
	Application string
	Params      string
}

func (o *OldOverride) SetParamsByOverrideConfig(config OverrideConfig) {
	parameters := config.Parameters
	var params strings.Builder

	for key, value := range parameters {
		param := key + "=" + value
		params.WriteString(param)
		params.WriteString("&")
	}

	p := params.String()
	if p != "" {
		if p[len(p)-1] == '&' {
			p = p[:len(p)-1]
		}
	}
	o.Params = p
}
