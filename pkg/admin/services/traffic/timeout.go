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
	"fmt"
	"strconv"
	"strings"

	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
)

type TimeoutService struct{}

// CreateOrUpdate create or update timeout rule
func (tm *TimeoutService) CreateOrUpdate(t *model.Timeout) error {
	key := services.GetOverridePath(t.GetKey())
	newRule := t.ToRule()

	err := createOrUpdateOverride(key, "consumer", "timeout", newRule)
	return err
}

func (tm *TimeoutService) Delete(t *model.Timeout) error {
	key := services.GetOverridePath(t.GetKey())
	err2 := removeFromOverride(key, "consumer", "timeout")
	if err2 != nil {
		return err2
	}
	return nil
}

func (tm *TimeoutService) Search(t *model.Timeout) ([]*model.Timeout, error) {
	result := make([]*model.Timeout, 0)

	var con string
	if t.Service != "" && t.Service != "*" {
		con = t.GetKey()
	}

	list, err := services.GetRules(con, constant.ConfiguratorRuleSuffix)
	if err != nil {
		return result, err
	}

	for k, v := range list {
		k, _ = strings.CutSuffix(k, constant.ConfiguratorRuleSuffix)
		split := strings.Split(k, ":")

		t := &model.Timeout{
			Service: split[0],
		}
		if len(split) >= 2 {
			t.Version = split[1]
		}
		if len(split) >= 3 {
			t.Group = split[2]
		}

		tv, err2 := getValue(v, "consumer", "timeout")
		if err2 != nil {
			return result, err2
		}

		if tv != nil {
			if tvStr, ok := tv.(string); ok {
				tvInt, err := strconv.Atoi(tvStr)
				if err != nil {
					logger.Error(fmt.Sprintf("Error parsing timeout rule %s", v), err)
					return result, err
				}
				t.Timeout = tvInt
			} else {
				t.Timeout = tv.(int)
			}
			result = append(result, t)
		}
	}

	return result, nil
}
