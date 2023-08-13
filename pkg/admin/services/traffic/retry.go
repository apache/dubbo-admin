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
	"github.com/apache/dubbo-admin/pkg/admin/util"
)

type RetryService struct{}

// CreateOrUpdate create or update timeout rule
func (tm *RetryService) CreateOrUpdate(r *model.Retry) error {
	key := services.GetOverridePath(util.ColonSeparatedKey(r.Service, r.Group, r.Version))
	newRule := r.ToRule()

	err := createOrUpdateOverride(key, "consumer", "retries", newRule)
	return err
}

func (tm *RetryService) Delete(r *model.Retry) error {
	key := services.GetOverridePath(util.ColonSeparatedKey(r.Service, r.Group, r.Version))
	err2 := removeFromOverride(key, "consumer", "retries")
	if err2 != nil {
		return err2
	}
	return nil
}

func (tm *RetryService) Search(r *model.Retry) ([]*model.Retry, error) {
	result := make([]*model.Retry, 0)

	var con string
	if r.Service != "" && r.Service != "*" {
		con = util.ColonSeparatedKey(r.Service, r.Group, r.Version)
	}

	list, err := services.GetRules(con, constant.ConfiguratorRuleSuffix)
	if err != nil {
		return result, err
	}

	for k, v := range list {
		k, _ = strings.CutSuffix(k, constant.ConfiguratorRuleSuffix)
		split := strings.Split(k, ":")
		retry := &model.Retry{
			Service: split[0],
		}
		if len(split) >= 2 {
			retry.Version = split[1]
		}
		if len(split) >= 3 {
			retry.Group = split[2]
		}

		rv, err2 := getValue(v, "consumer", "retries")
		if err2 != nil {
			return result, err2
		}
		if rv != nil {
			if rvStr, ok := rv.(string); ok {
				rvInt, err := strconv.Atoi(rvStr)
				if err != nil {
					logger.Error(fmt.Sprintf("Error parsing retry rule %s", v), err)
					return result, err
				}
				retry.Retry = rvInt
			} else {
				retry.Retry = rv.(int)
			}
			result = append(result, retry)
		}
	}

	return result, nil
}
