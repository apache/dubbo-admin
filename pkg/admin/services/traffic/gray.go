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
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"gopkg.in/yaml.v2"
)

type GrayService struct{}

// CreateOrUpdate create or update timeout rule
func (tm *GrayService) CreateOrUpdate(g *model.Gray) error {
	key := services.GetRoutePath(g.Application, constant.TagRuleSuffix)
	newRule := g.ToRule()

	err := createOrUpdateTag(key, newRule)
	return err
}

func (tm *GrayService) Delete(g *model.Gray) error {
	key := services.GetRoutePath(g.Application, constant.TagRuleSuffix)
	err2 := deleteTag(key)
	if err2 != nil {
		return err2
	}
	return nil
}

func (tm *GrayService) Search(g *model.Gray) ([]*model.Gray, error) {
	result := make([]*model.Gray, 0)

	list, err := services.GetRules(g.Application, constant.TagRuleSuffix)
	if err != nil {
		return result, err
	}

	for _, v := range list {
		route := &model.TagRoute{}
		err = yaml.Unmarshal([]byte(v), route)
		if err != nil {
			return result, err
		}

		if len(route.Tags) > 0 {
			gray := &model.Gray{
				Application: route.Key,
			}
			gray.Tags = route.Tags
			result = append(result, gray)
		}
	}

	return result, nil
}
