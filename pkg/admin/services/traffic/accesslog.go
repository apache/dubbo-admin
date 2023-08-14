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
	"strconv"

	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"gopkg.in/yaml.v2"
)

type AccesslogService struct{}

// CreateOrUpdate create or update timeout rule
func (tm *AccesslogService) CreateOrUpdate(a *model.Accesslog) error {
	key := services.GetOverridePath(a.Application)
	newRule := a.ToRule()

	var err error
	if a.Accesslog == "" {
		err = tm.Delete(a)
	} else {
		err = createOrUpdateOverride(key, "provider", "accesslog", newRule)
	}

	return err
}

func (tm *AccesslogService) Delete(a *model.Accesslog) error {
	key := services.GetOverridePath(a.Application)
	err2 := removeFromOverride(key, "provider", "accesslog")
	if err2 != nil {
		return err2
	}
	return nil
}

func (tm *AccesslogService) Search(a *model.Accesslog) ([]*model.Accesslog, error) {
	result := make([]*model.Accesslog, 0)

	list, err := services.GetRules(a.Application, constant.ConfiguratorRuleSuffix)
	if err != nil {
		return result, err
	}

	for _, v := range list {
		alv, err2 := getValue(v, "provider", "accesslog")
		if err2 != nil {
			return result, err2
		}

		override := &model.Override{}
		err = yaml.Unmarshal([]byte(v), override)
		if err != nil {
			return nil, err
		}

		if alv != nil {
			accesslog := &model.Accesslog{
				Application: override.Key,
			}
			if alvBool, ok := alv.(bool); ok {
				accesslog.Accesslog = strconv.FormatBool(alvBool)
			} else {
				accesslog.Accesslog = alv.(string)
			}
			result = append(result, accesslog)
		}
	}

	return result, nil
}
