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

package v1alpha1

import (
	"strings"

	"github.com/apache/dubbo-admin/pkg/core/consts"
)

func GetOverridePath(key string) string {
	key = strings.Replace(key, "/", "*", -1)
	return key + consts.ConfiguratorRuleSuffix
}

func (d *DynamicConfig) ListUnGenConfigs() []*OverrideConfig {
	res := make([]*OverrideConfig, 0, len(d.Configs)/2+1)
	for _, config := range d.Configs {
		if !config.XGenerateByCp {
			res = append(res, config)
		}
	}
	return res
}

func (d *DynamicConfig) ListGenConfigs() []*OverrideConfig {
	res := make([]*OverrideConfig, 0, len(d.Configs)/2+1)
	for _, config := range d.Configs {
		if config.XGenerateByCp {
			res = append(res, config)
		}
	}
	return res
}

func (d *DynamicConfig) RangeConfigsToRemove(matchFunc func(conf *OverrideConfig) (IsRemove bool)) {
	if matchFunc == nil {
		return
	}
	newConf := make([]*OverrideConfig, 0, len(d.Configs)/2+1)
	for _, config := range d.Configs {
		if !matchFunc(config) {
			newConf = append(newConf, config)
		}
	}
	d.Configs = newConf
}

func (d *DynamicConfig) RangeConfig(f func(conf *OverrideConfig) (isStop bool)) {
	if f == nil {
		return
	}
	for _, config := range d.Configs {
		if f(config) {
			break
		}
	}
}
