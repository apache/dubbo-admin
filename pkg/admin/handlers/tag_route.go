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

package handlers

import (
	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/logger"
	"net/http"
	"strings"

	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/gin-gonic/gin"
)

var routeService services.RouteService = &services.RouteServiceImpl{}

func CreateRule(c *gin.Context) {
	var tagRouteDto model.TagRouteDto
	err := c.BindJSON(&tagRouteDto)
	if err != nil {
		logger.Errorf("Parsing tag rule input error, err msg is: %s", err.Error())
		c.JSON(http.StatusBadRequest, false)
		return
	}

	err = routeService.CreateTagRoute(tagRouteDto)

	if err != nil {
		if _, ok := err.(*config.RuleExists); ok {
			logger.Infof("Condition rule for service %s already exists!", tagRouteDto.Service)
		} else {
			logger.Infof("Create tag rule for service %s failed, err msg is %s", tagRouteDto.Service, err.Error())
			c.JSON(http.StatusInternalServerError, false)
		}
		return
	}
	c.JSON(http.StatusOK, true)
}

func UpdateRule(c *gin.Context) {
	var tagRouteDto model.TagRouteDto
	err := c.BindJSON(&tagRouteDto)
	if err != nil {
		panic(err)
	}
	id := c.Param("id")
	id = strings.ReplaceAll(id, "*", "/")

	_, err = routeService.FindTagRoute(id)
	if err != nil {
		logger.Errorf("Check failed before trying to update condition rule for service %s , err msg is: %s", tagRouteDto.Service, err.Error())
		c.JSON(http.StatusInternalServerError, false)
		return
	}

	err = routeService.UpdateTagRoute(tagRouteDto)

	if err != nil {
		logger.Errorf("Update tag rule for service %s failed, err msg is: %s", tagRouteDto.Service, err.Error())
		c.JSON(http.StatusInternalServerError, false)
		return
	}
	c.JSON(http.StatusOK, true)
}

func SearchRoutes(c *gin.Context) {
	application := c.Query("application")

	tagRoute, err := routeService.FindTagRoute(application)
	if err != nil {
		if _, ok := err.(*config.RuleNotFound); ok {
			logger.Infof("No tag rule for query parameters: application %s", application)
			c.JSON(http.StatusOK, []model.TagRouteDto{})
		} else {
			logger.Errorf("Check tag rule detail failed, err msg is: %s", err.Error())
			c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, []model.TagRouteDto{tagRoute})
}

func DetailRoute(c *gin.Context) {
	id := c.Param("id")
	id = strings.ReplaceAll(id, "*", "/")

	tagRoute, err := routeService.FindTagRoute(id)
	if err != nil {
		logger.Errorf("Check tag rule detail with id %s failed, err msg is: %s", id, err.Error())
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, tagRoute)
}

func DeleteRoute(c *gin.Context) {
	id := c.Param("id")
	id = strings.ReplaceAll(id, "*", "/")

	err := routeService.DeleteTagRoute(id)
	if err != nil {
		logger.Errorf("Delete tag rule with id %s failed, err msg is: %s", id, err.Error())
		c.JSON(http.StatusInternalServerError, false)
		return
	}
	c.JSON(http.StatusOK, true)
}

func EnableRoute(c *gin.Context) {
	id := c.Param("id")
	id = strings.ReplaceAll(id, "*", "/")

	err := routeService.EnableTagRoute(id)
	if err != nil {
		logger.Errorf("Enable tag rule with id %s failed, err msg is: %s", id, err.Error())
		c.JSON(http.StatusInternalServerError, false)
		return
	}
	c.JSON(http.StatusOK, true)
}

func DisableRoute(c *gin.Context) {
	id := c.Param("id")
	id = strings.ReplaceAll(id, "*", "/")

	err := routeService.DisableTagRoute(id)
	if err != nil {
		logger.Errorf("Disable tag rule with id %s failed, err msg is: %s", id, err.Error())
		c.JSON(http.StatusInternalServerError, false)
		return
	}
	c.JSON(http.StatusOK, true)
}
