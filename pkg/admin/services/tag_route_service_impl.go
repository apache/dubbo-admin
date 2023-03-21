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

	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/util"
)

type TagRoutesServiceImpl struct {
	GovernanceConfig config.GovernanceConfig
}

func (t *TagRoutesServiceImpl) CreateTagRoute(tagRoute model.TagRouteDto) error {
	id := getIdFromDto(tagRoute)
	path := getTagRoutePath(id, constant.TagRoute)
	store := convertTagRouteToStore(tagRoute)
	obj, _ := util.DumpObject(store)
	return t.GovernanceConfig.SetConfig(path, obj)
}

func (t *TagRoutesServiceImpl) UpdateTagRoute(tagRoute model.TagRouteDto) error {
	id := getIdFromDto(tagRoute)
	path := getTagRoutePath(id, constant.TagRoute)
	cfg, _ := t.GovernanceConfig.GetConfig(path)
	if cfg == "" {
		return fmt.Errorf("tag route %s not found", id)
	}
	store := convertTagRouteToStore(tagRoute)
	obj, _ := util.DumpObject(store)
	return t.GovernanceConfig.SetConfig(path, obj)
}

func (t *TagRoutesServiceImpl) DeleteTagRoute(id string) error {
	path := getTagRoutePath(id, constant.TagRoute)
	return t.GovernanceConfig.DeleteConfig(path)
}

func (t *TagRoutesServiceImpl) FindTagRoute(id string) (model.TagRouteDto, error) {
	path := getTagRoutePath(id, constant.TagRoute)
	cfg, err := t.GovernanceConfig.GetConfig(path)
	if cfg != "" {
		var tagRoute model.TagRoute
		_ = util.LoadObject(cfg, &tagRoute)
		return convertTagRouteToDto(tagRoute), nil
	}
	return model.TagRouteDto{}, err
}

func (t *TagRoutesServiceImpl) EnableTagRoute(id string) error {
	path := getTagRoutePath(id, constant.TagRoute)
	cfg, err := t.GovernanceConfig.GetConfig(path)
	if cfg != "" {
		var tagRoute model.TagRoute
		_ = util.LoadObject(cfg, &tagRoute)
		tagRoute.Enabled = true
		obj, _ := util.DumpObject(tagRoute)
		return t.GovernanceConfig.SetConfig(path, obj)
	}
	return err
}

func (t *TagRoutesServiceImpl) DisableTagRoute(id string) error {
	path := getTagRoutePath(id, constant.TagRoute)
	cfg, err := t.GovernanceConfig.GetConfig(path)
	if cfg != "" {
		var tagRoute model.TagRoute
		_ = util.LoadObject(cfg, &tagRoute)
		tagRoute.Enabled = false
		obj, _ := util.DumpObject(tagRoute)
		return t.GovernanceConfig.SetConfig(path, obj)
	}
	return err
}

func getTagRoutePath(key string, routeType string) string {
	key = strings.ReplaceAll(key, "*", "/")
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
	return dto
}

func getIdFromDto(baseDto model.TagRouteDto) string {
	if baseDto.Application != "" {
		return baseDto.Application
	}
	// id format: "${class}:${version}:${group}"
	return baseDto.Service + constant.Colon + baseDto.ConfigVersion + constant.Colon + baseDto.ServiceGroup
}
