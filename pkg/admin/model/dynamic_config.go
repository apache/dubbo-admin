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

import "github.com/apache/dubbo-admin/pkg/admin/constant"

type DynamicConfig struct {
	Base
	ConfigVersion string           `json:"configVersion"`
	Enabled       bool             `json:"enabled"`
	Configs       []OverrideConfig `json:"configs"`
}

func (d *DynamicConfig) ToOverride() *Override {
	o := &Override{}
	if d.Application != "" {
		o.Scope = constant.ApplicationKey
		o.Key = d.Application
	} else {
		o.Scope = constant.Service
		o.Key = d.Service
	}
	o.ConfigVersion = d.ConfigVersion
	o.Enabled = d.Enabled
	o.Configs = d.Configs

	return o
}

func (d *DynamicConfig) ToOldOverride() []*OldOverride {
	result := []*OldOverride{}
	configs := d.Configs
	for _, config := range configs {
		if constant.Configs.Contains(config.Type) {
			continue
		}
		apps := config.Applications
		addresses := config.Addresses
		for _, address := range addresses {
			if len(apps) > 0 {
				for _, app := range apps {
					o := &OldOverride{
						Service: d.Service,
						Address: address,
						Enabled: d.Enabled,
					}
					o.SetParamsByOverrideConfig(config)
					o.Application = app
					result = append(result, o)
				}
			} else {
				o := &OldOverride{
					Service: d.Service,
					Address: address,
					Enabled: d.Enabled,
				}
				o.SetParamsByOverrideConfig(config)
				result = append(result, o)
			}
		}
	}
	return result
}
