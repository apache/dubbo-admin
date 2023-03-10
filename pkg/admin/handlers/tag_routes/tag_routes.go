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

package tag_routes

import (
	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/util"
	"github.com/gin-gonic/gin"
	"strings"
)

type Tag struct {
	name      string
	addresses []string
}

type TagRouteDto struct {
	tags []Tag

	priority int
	enable   bool
	force    bool
	runtime  bool

	baseDto model.BaseDto
}

type TagRoute struct {
	priority int
	enable   bool
	force    bool
	runtime  bool
	key      string
	tags     []Tag
}

func CreateRule(c *gin.Context) {
	var tagRouteDto TagRouteDto
	err := c.BindJSON(&tagRouteDto)
	if err != nil {
		panic(err)
	}

	id := util.GetIdFromDto(tagRouteDto.baseDto)
	path := getPath(id, constant.TagRoute)
	tagRoute := convertTagRouteToStore(tagRouteDto)
	tagRouteStr, err := util.DumpObject(tagRoute)
	if err != nil {
		panic(err)
	}

	err = config.SetConfig(path, tagRouteStr)
	if err != nil {
		panic(err)
	}
}

func getPath(key string, routeType string) string {
	key = strings.ReplaceAll(key, "*", "/")
	if routeType == constant.ConditionRoute {
		return key + constant.ConditionRuleSuffix
	} else {
		return key + constant.TagRuleSuffix
	}
}

func convertTagRouteToStore(tagRoute TagRouteDto) TagRoute {
	var store TagRoute
	store.key = tagRoute.baseDto.Application
	store.enable = tagRoute.enable
	store.force = tagRoute.force
	store.priority = tagRoute.priority
	store.runtime = tagRoute.runtime
	store.tags = tagRoute.tags
	return store
}
