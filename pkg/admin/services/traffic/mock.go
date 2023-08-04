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
)

type MockService struct{}

// CreateOrUpdate create or update timeout rule
func (tm *MockService) CreateOrUpdate(m *model.Mock) error {
	key := services.GetOverridePath(m.GetKey())
	newRule := m.ToRule()

	err := createOrUpdateOverride(key, "consumer", "mock", newRule)
	return err
}

func (tm *MockService) Delete(m *model.Mock) error {
	key := services.GetOverridePath(m.GetKey())
	err2 := removeFromOverride(key, "consumer", "mock")
	if err2 != nil {
		return err2
	}
	return nil
}

func (tm *MockService) Search(m *model.Mock) ([]*model.Mock, error) {
	result := make([]*model.Mock, 0)

	var con string
	if m.Service != "" && m.Service != "*" {
		con = m.GetKey()
	}
	list, err := services.GetRules(con, constant.ConfiguratorRuleSuffix)
	if err != nil {
		return result, err
	}

	for k, v := range list {
		k, _ = strings.CutSuffix(k, constant.ConfiguratorRuleSuffix)
		split := strings.Split(k, ":")
		mock := &model.Mock{
			Service: split[0],
		}
		if len(split) >= 2 {
			mock.Version = split[1]
		}
		if len(split) >= 3 {
			mock.Group = split[2]
		}

		mv, err2 := getValue(v, "consumer", "mock")
		if err2 != nil {
			return result, err2
		}
		if mv != nil {
			mock.Mock = mv.(string)
			result = append(result, mock)
		}
	}

	return result, nil
}
