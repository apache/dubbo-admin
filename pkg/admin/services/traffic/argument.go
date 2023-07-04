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
	"github.com/apache/dubbo-admin/pkg/admin/util"
	"gopkg.in/yaml.v2"
)

type ArgumentService struct{}

// CreateOrUpdate create or update timeout rule
func (tm *ArgumentService) CreateOrUpdate(a *model.Argument) error {
	key := services.GetRoutePath(util.ColonSeparatedKey(a.Service, a.Group, a.Version), constant.ConditionRoute)
	newRule := a.ToRule()

	err := createOrUpdateCondition(key, newRule)
	return err
}

func (tm *ArgumentService) Delete(a *model.Argument) error {
	key := services.GetRoutePath(util.ColonSeparatedKey(a.Service, a.Group, a.Version), constant.ConditionRoute)
	err2 := removeCondition(key, a.Rule, model.RegionAdminIdentifier)
	if err2 != nil {
		return err2
	}
	return nil
}

func (tm *ArgumentService) Search(a *model.Argument) ([]*model.Argument, error) {
	result := make([]*model.Argument, 0)

	var con string
	if a.Service != "" && a.Service != "*" {
		con = util.ColonSeparatedKey(a.Service, a.Group, a.Version)
	}

	list, err := services.GetRules(con, constant.ConditionRuleSuffix)
	if err != nil {
		return result, err
	}

	for k, v := range list {
		k, _ = strings.CutSuffix(k, constant.ConditionRuleSuffix)
		split := strings.Split(k, ":")
		argument := &model.Argument{
			Service: split[0],
		}

		route := &model.ConditionRoute{}
		err = yaml.Unmarshal([]byte(v), route)
		if err != nil {
			return result, err
		}
		for _, c := range route.Conditions {
			// fixme, regex match
			if i := strings.Index(c, model.ArgumentAdminIdentifier); i > 0 {
				argument.Rule = strings.TrimSpace(c[0:i])
				break
			}
		}

		if argument.Rule != "" {
			result = append(result, argument)
		}
	}

	return result, nil
}
