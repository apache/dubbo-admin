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

type RegionService struct{}

// CreateOrUpdate create or update timeout rule
func (tm *RegionService) CreateOrUpdate(r *model.Region) error {
	key := services.GetRoutePath(r.GetKey(), constant.ConditionRoute)
	newRule := r.ToRule()

	var err error
	if r.Rule == "" {
		err = tm.Delete(r)
	} else {
		err = createOrUpdateCondition(key, newRule)
	}

	return err
}

func (tm *RegionService) Delete(r *model.Region) error {
	key := services.GetRoutePath(r.GetKey(), constant.ConditionRoute)
	err2 := removeCondition(key, r.Rule, model.RegionAdminIdentifier)
	if err2 != nil {
		return err2
	}
	return nil
}

func (tm *RegionService) Search(r *model.Region) ([]*model.Region, error) {
	result := make([]*model.Region, 0)

	var con string
	if r.Service != "" && r.Service != "*" {
		con = r.GetKey()
	}

	list, err := services.GetRules(con, constant.ConditionRuleSuffix)
	if err != nil {
		return result, err
	}

	for k, v := range list {
		k, _ = strings.CutSuffix(k, constant.ConditionRuleSuffix)
		split := strings.Split(k, ":")
		region := &model.Region{
			Service: split[0],
		}
		if len(split) >= 2 {
			region.Version = split[1]
		}
		if len(split) >= 3 {
			region.Group = split[2]
		}

		route := &model.ConditionRoute{}
		err = yaml.Unmarshal([]byte(v), route)
		if err != nil {
			return result, err
		}
		for _, c := range route.Conditions {
			// fixme, regex match
			if strings.Contains(c, model.RegionAdminIdentifier) {
				i := strings.Index(c, "=$")
				if i > 3 {
					region.Rule = strings.TrimSpace(c[3:i])
					break
				}
			}
		}

		if region.Rule != "" {
			result = append(result, region)
		}
	}

	return result, nil
}
