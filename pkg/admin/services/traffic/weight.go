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

package traffic

import (
	"strings"

	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"gopkg.in/yaml.v2"
)

type WeightService struct{}

// CreateOrUpdate create or update timeout rule
func (tm *WeightService) CreateOrUpdate(p *model.Percentage) error {
	key := services.GetOverridePath(p.GetKey())
	newRule := p.ToRule()

	err := createOrUpdateOverride(key, "provider", "weight", newRule)
	return err
}

func (tm *WeightService) Delete(p *model.Percentage) error {
	key := services.GetOverridePath(p.GetKey())
	err := removeFromOverride(key, "provider", "weight")
	if err != nil {
		return err
	}
	return nil
}

func (tm *WeightService) Search(p *model.Percentage) ([]*model.Percentage, error) {
	result := make([]*model.Percentage, 0)

	var con string
	if p.Service != "" && p.Service != "*" {
		con = p.GetKey()
	}

	list, err := services.GetRules(con, constant.ConfiguratorRuleSuffix)
	if err != nil {
		return result, err
	}

	for k, v := range list {
		k, _ = strings.CutSuffix(k, constant.ConfiguratorRuleSuffix)
		split := strings.Split(k, ":")
		percentage := &model.Percentage{
			Service: split[0],
			Weights: make([]model.Weight, 0),
		}
		if len(split) >= 2 {
			percentage.Version = split[1]
		}
		if len(split) >= 3 {
			percentage.Group = split[2]
		}

		override := &model.Override{}
		err = yaml.Unmarshal([]byte(v), override)
		if err != nil {
			return result, err
		}
		for _, c := range override.Configs {
			if c.Side == "provider" && c.Parameters["weight"] != nil {
				percentage.Weights = append(percentage.Weights, model.Weight{
					Weight: c.Parameters["weight"].(int),
					Match:  c.Match,
				})
			}
		}

		if len(percentage.Weights) > 0 {
			result = append(result, percentage)
		}
	}

	return result, nil
}
