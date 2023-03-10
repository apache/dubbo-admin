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
	"github.com/apache/dubbo-admin/pkg/admin/util"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

type Tag struct {
	Name      string
	Addresses []string
}

type TagRouteDto struct {
	Tags []Tag `json:"tags" binding:"required"`

	Priority int  `json:"priority"`
	Enable   bool `json:"enable"`
	Force    bool `json:"force"`
	Runtime  bool `json:"runtime"`

	Application    string `json:"application" binding:"required"`
	Service        string `json:"service"`
	Id             string `json:"id"`
	ServiceVersion string `json:"serviceVersion"`
	ServiceGroup   string `json:"serviceGroup"`
}

type TagRoute struct {
	Priority int
	Enable   bool
	Force    bool
	Runtime  bool
	Key      string
	Tags     []Tag
}

func CreateRule(c *gin.Context) {
	var tagRouteDto TagRouteDto
	err := c.BindJSON(&tagRouteDto)
	if err != nil {
		panic(err)
	}

	id := getIdFromDto(tagRouteDto)
	path := getPath(id, constant.TagRoute)
	tagRoute := convertTagRouteToStore(tagRouteDto)
	tagRouteStr, err := util.DumpObject(tagRoute)
	if err != nil {
		panic(err)
	}

	println("path: " + path)
	println("tagRouteStr: " + tagRouteStr)

	err = config.SetConfig(path, tagRouteStr)
	if err != nil {
		panic(err)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": "success",
	})
}

func SearchRoutes(c *gin.Context) {
	application := c.Query("application")
	path := getPath(application, constant.TagRoute)
	cfg, err := config.GetConfig(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	if cfg != "" {
		var tagRoute TagRoute
		_ = util.LoadObject(cfg, &tagRoute)
		tagRouteDto := convertTagRouteToDto(tagRoute)
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"data": []TagRouteDto{tagRouteDto},
		})
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
	store.Key = tagRoute.Application
	store.Enable = tagRoute.Enable
	store.Force = tagRoute.Force
	store.Priority = tagRoute.Priority
	store.Runtime = tagRoute.Runtime
	store.Tags = tagRoute.Tags
	return store
}

func convertTagRouteToDto(tagRoute TagRoute) TagRouteDto {
	var dto TagRouteDto
	dto.Application = tagRoute.Key
	dto.Enable = tagRoute.Enable
	dto.Force = tagRoute.Force
	dto.Priority = tagRoute.Priority
	dto.Runtime = tagRoute.Runtime
	dto.Tags = tagRoute.Tags
	return dto
}

func getIdFromDto(baseDto TagRouteDto) string {
	if baseDto.Application != "" {
		return baseDto.Application
	}
	// id format: "${class}:${version}:${group}"
	return baseDto.Service + constant.Colon + baseDto.ServiceVersion + constant.Colon + baseDto.ServiceGroup
}
