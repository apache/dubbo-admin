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
	"fmt"
	"strings"

	"github.com/apache/dubbo-admin/pkg/admin/constant"
)

type Override struct {
	Key           string           `json:"key"`
	Scope         string           `json:"scope"`
	ConfigVersion string           `json:"configVersion"`
	Enabled       bool             `json:"enabled"`
	Configs       []OverrideConfig `json:"configs"`
}

type OverrideConfig struct {
	Side              string                 `json:"side"`
	Addresses         []string               `json:"addresses"`
	ProviderAddresses []string               `json:"providerAddresses"`
	Parameters        map[string]interface{} `json:"parameters"`
	Applications      []string               `json:"applications"`
	Services          []string               `json:"services"`
	Type              string                 `json:"type"`
	Enabled           bool                   `json:"enabled"`
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
	Service     string `json:"service"`
	Address     string `json:"address"`
	Enabled     bool   `json:"enabled"`
	Application string `json:"application"`
	Params      string `json:"params"`
}

func (o *OldOverride) SetParamsByOverrideConfig(config OverrideConfig) {
	parameters := config.Parameters
	var params strings.Builder

	for key, value := range parameters {
		param := key + "=" + fmt.Sprintf("%v", value)
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
