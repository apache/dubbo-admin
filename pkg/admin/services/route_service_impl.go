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

package services

import (
	"fmt"
	"strings"

	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/util"
)

type RouteServiceImpl struct{}

func (t *RouteServiceImpl) CreateTagRoute(tagRoute model.TagRouteDto) error {
	id := util.BuildServiceKey(tagRoute.Base.Application, tagRoute.Base.Service, tagRoute.Base.ServiceVersion, tagRoute.Base.ServiceGroup)
	path := GetRoutePath(id, constant.TagRoute)
	store := convertTagRouteToStore(tagRoute)
	obj, _ := util.DumpObject(store)
	return config.Governance.SetConfig(path, obj)
}

func (t *RouteServiceImpl) UpdateTagRoute(tagRoute model.TagRouteDto) error {
	id := util.BuildServiceKey(tagRoute.Base.Application, tagRoute.Base.Service, tagRoute.Base.ServiceVersion, tagRoute.Base.ServiceGroup)
	path := GetRoutePath(id, constant.TagRoute)
	cfg, _ := config.Governance.GetConfig(path)
	if cfg == "" {
		return fmt.Errorf("tag route %s not found", id)
	}
	store := convertTagRouteToStore(tagRoute)
	obj, _ := util.DumpObject(store)
	return config.Governance.SetConfig(path, obj)
}

func (t *RouteServiceImpl) DeleteTagRoute(id string) error {
	path := GetRoutePath(id, constant.TagRoute)
	return config.Governance.DeleteConfig(path)
}

func (t *RouteServiceImpl) FindTagRoute(id string) (model.TagRouteDto, error) {
	path := GetRoutePath(id, constant.TagRoute)
	cfg, err := config.Governance.GetConfig(path)
	if cfg != "" {
		var tagRoute model.TagRoute
		_ = util.LoadObject(cfg, &tagRoute)
		return convertTagRouteToDto(tagRoute), nil
	}
	return model.TagRouteDto{}, err
}

func (t *RouteServiceImpl) EnableTagRoute(id string) error {
	path := GetRoutePath(id, constant.TagRoute)
	cfg, err := config.Governance.GetConfig(path)
	if cfg != "" {
		var tagRoute model.TagRoute
		_ = util.LoadObject(cfg, &tagRoute)
		tagRoute.Enabled = true
		obj, _ := util.DumpObject(tagRoute)
		return config.Governance.SetConfig(path, obj)
	}
	return err
}

func (t *RouteServiceImpl) DisableTagRoute(id string) error {
	path := GetRoutePath(id, constant.TagRoute)
	cfg, err := config.Governance.GetConfig(path)
	if cfg != "" {
		var tagRoute model.TagRoute
		_ = util.LoadObject(cfg, &tagRoute)
		tagRoute.Enabled = false
		obj, _ := util.DumpObject(tagRoute)
		return config.Governance.SetConfig(path, obj)
	}
	return err
}

func (t *RouteServiceImpl) CreateConditionRoute(conditionRouteDto model.ConditionRouteDto) error {
	id := util.BuildServiceKey(conditionRouteDto.Base.Application, conditionRouteDto.Base.Service, conditionRouteDto.Base.ServiceVersion, conditionRouteDto.Base.ServiceGroup)
	path := GetRoutePath(id, constant.ConditionRoute)
	existConfig, _ := config.Governance.GetConfig(path)

	var existRule model.ConditionRoute
	if existConfig != "" {
		_ = util.LoadObject(existConfig, &existRule)
	}
	store := convertConditionRouteToStore(existRule, conditionRouteDto)

	obj, _ := util.DumpObject(store)
	return config.Governance.SetConfig(path, obj)
}

func (t *RouteServiceImpl) UpdateConditionRoute(conditionRouteDto model.ConditionRouteDto) error {
	id := util.BuildServiceKey(conditionRouteDto.Base.Application, conditionRouteDto.Base.Service, conditionRouteDto.Base.ServiceVersion, conditionRouteDto.Base.ServiceGroup)
	path := GetRoutePath(id, constant.ConditionRoute)
	cfg, err := config.Governance.GetConfig(path)
	if err != nil {
		return err
	}
	if cfg == "" {
		return fmt.Errorf("no existing condition route for path: %s", path)
	}

	var existRule model.ConditionRoute
	_ = util.LoadObject(cfg, &existRule)
	store := convertConditionRouteToStore(existRule, conditionRouteDto)

	obj, _ := util.DumpObject(store)
	return config.Governance.SetConfig(path, obj)
}

func (t *RouteServiceImpl) DeleteConditionRoute(id string) error {
	path := GetRoutePath(id, constant.ConditionRoute)
	return config.Governance.DeleteConfig(path)
}

func (t *RouteServiceImpl) FindConditionRouteById(id string) (model.ConditionRouteDto, error) {
	path := GetRoutePath(id, constant.ConditionRoute)
	cfg, err := config.Governance.GetConfig(path)
	if err != nil {
		return model.ConditionRouteDto{}, err
	}
	if cfg != "" {
		var conditionRoute model.ConditionRoute
		_ = util.LoadObject(cfg, &conditionRoute)
		dto := convertConditionRouteToDto(conditionRoute)
		if dto.Service != "" {
			dto.Service = strings.ReplaceAll(dto.Service, "*", "/")
		}
		detachResult := detachId(id)
		if len(detachResult) > 1 {
			dto.ServiceVersion = detachResult[1]
		}
		if len(detachResult) > 2 {
			dto.ServiceGroup = detachResult[2]
		}
		dto.ID = id
		return dto, nil
	}
	return model.ConditionRouteDto{}, nil
}

func (t *RouteServiceImpl) FindConditionRoute(conditionRouteDto model.ConditionRouteDto) (model.ConditionRouteDto, error) {
	return t.FindConditionRouteById(util.BuildServiceKey(conditionRouteDto.Base.Application, conditionRouteDto.Base.Service, conditionRouteDto.Base.ServiceVersion, conditionRouteDto.Base.ServiceGroup))
}

func (t *RouteServiceImpl) EnableConditionRoute(id string) error {
	path := GetRoutePath(id, constant.ConditionRoute)
	cfg, err := config.Governance.GetConfig(path)
	if err != nil {
		return err
	}
	if cfg != "" {
		var conditionRoute model.ConditionRoute
		_ = util.LoadObject(cfg, &conditionRoute)
		conditionRoute.Enabled = true
		obj, _ := util.DumpObject(conditionRoute)
		return config.Governance.SetConfig(path, obj)
	}
	return fmt.Errorf("no existing condition route for path: %s", path)
}

func (t *RouteServiceImpl) DisableConditionRoute(id string) error {
	path := GetRoutePath(id, constant.ConditionRoute)
	cfg, err := config.Governance.GetConfig(path)
	if err != nil {
		return err
	}
	if cfg != "" {
		var conditionRoute model.ConditionRoute
		_ = util.LoadObject(cfg, &conditionRoute)
		conditionRoute.Enabled = false
		obj, _ := util.DumpObject(conditionRoute)
		return config.Governance.SetConfig(path, obj)
	}
	return fmt.Errorf("no existing condition route for path: %s", path)
}

func GetRoutePath(key string, routeType string) string {
	key = strings.ReplaceAll(key, "/", "*")
	if routeType == constant.ConditionRoute {
		return key + constant.ConditionRuleSuffix
	} else {
		return key + constant.TagRuleSuffix
	}
}

func convertTagRouteToStore(tagRoute model.TagRouteDto) model.TagRoute {
	var store model.TagRoute
	store.Key = tagRoute.Application
	store.Enabled = tagRoute.Enabled
	store.Force = tagRoute.Force
	store.Priority = tagRoute.Priority
	store.Runtime = tagRoute.Runtime
	store.Tags = tagRoute.Tags
	store.ConfigVersion = tagRoute.ConfigVersion
	return store
}

func convertTagRouteToDto(tagRoute model.TagRoute) model.TagRouteDto {
	var dto model.TagRouteDto
	dto.Application = tagRoute.Key
	dto.Enabled = tagRoute.Enabled
	dto.Force = tagRoute.Force
	dto.Priority = tagRoute.Priority
	dto.Runtime = tagRoute.Runtime
	dto.Tags = tagRoute.Tags
	dto.ConfigVersion = tagRoute.ConfigVersion
	return dto
}

func convertConditionRouteToStore(existRule model.ConditionRoute, conditionRouteDto model.ConditionRouteDto) model.ConditionRoute {
	if existRule.Key == "" || existRule.Scope == "" {
		existRule = model.ConditionRoute{}
		if conditionRouteDto.Application != "" {
			existRule.Key = conditionRouteDto.Application
			existRule.Scope = constant.Application
		} else {
			existRule.Key = strings.ReplaceAll(conditionRouteDto.Service, "/", "*")
			existRule.Scope = constant.Service
		}
	}
	existRule.Enabled = conditionRouteDto.Enabled
	existRule.Force = conditionRouteDto.Force
	existRule.Priority = conditionRouteDto.Priority
	existRule.Runtime = conditionRouteDto.Runtime
	existRule.Conditions = conditionRouteDto.Conditions
	existRule.ConfigVersion = conditionRouteDto.ConfigVersion
	return existRule
}

func convertConditionRouteToDto(conditionRoute model.ConditionRoute) model.ConditionRouteDto {
	var dto model.ConditionRouteDto
	if conditionRoute.Scope == constant.Application {
		dto.Application = conditionRoute.Key
	} else {
		dto.Service = conditionRoute.Key
	}
	dto.Enabled = conditionRoute.Enabled
	dto.Force = conditionRoute.Force
	dto.Priority = conditionRoute.Priority
	dto.Runtime = conditionRoute.Runtime
	dto.Conditions = conditionRoute.Conditions
	dto.ConfigVersion = conditionRoute.ConfigVersion
	return dto
}

func detachId(id string) []string {
	if strings.Contains(id, constant.Colon) {
		return strings.Split(id, constant.Colon)
	} else {
		return []string{id}
	}
}

func GetRules(con string, ruleType string) (map[string]string, error) {
	list := make(map[string]string)
	if con == "" || con == "*" {
		rules, err := config.Governance.GetList("dubbo")
		if _, ok := err.(*config.RuleNotFound); ok {
			logger.Infof("No rule found from config center, err msg is %s", err.Error())
			return list, nil
		}

		for k, v := range rules {
			if ruleType == "*" || strings.HasSuffix(k, ruleType) {
				list[k] = v
			}
		}
	} else {
		key := GetOverridePath(con)
		rule, err := config.Governance.GetConfig(key)
		if _, ok := err.(*config.RuleNotFound); ok {
			logger.Infof("No rule found from config center, err msg is %s", err.Error())
			return list, nil
		}
		list[key] = rule
	}
	return list, nil
}
