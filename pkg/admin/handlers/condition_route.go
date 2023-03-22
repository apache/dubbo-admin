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
	"net/http"

	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/gin-gonic/gin"
)

func CreateConditionRule(c *gin.Context) {
	var routeDto model.ConditionRouteDto
	err := c.BindJSON(&routeDto)
	if err != nil {
		panic(err)
	}

	err = routeService.CreateConditionRoute(routeDto)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
}

func UpdateConditionRule(c *gin.Context) {
	var routeDto model.ConditionRouteDto
	err := c.BindJSON(&routeDto)
	if err != nil {
		panic(err)
	}

	err = routeService.UpdateConditionRoute(routeDto)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
}

func SearchConditionRoutes(c *gin.Context) {
	application := c.Query("application")
	service := c.Query("service")
	serviceVersion := c.Query("serviceVersion")
	serviceGroup := c.Query("serviceGroup")

	var routeDto model.ConditionRouteDto
	var err error
	crDto := model.ConditionRouteDto{}
	if application != "" {
		crDto.Application = application
		routeDto, err = routeService.FindConditionRoute(crDto)
	} else if service != "" {
		crDto.Service = service
		crDto.ServiceVersion = serviceVersion
		crDto.ServiceGroup = serviceGroup
		routeDto, err = routeService.FindConditionRoute(crDto)
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Either Service or application is required.",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": []model.ConditionRouteDto{routeDto},
	})
}

func DetailConditionRoute(c *gin.Context) {
	id := c.Param("id")
	routeDto, err := routeService.FindConditionRouteById(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": routeDto,
	})
}

func DeleteConditionRoute(c *gin.Context) {
	id := c.Param("id")
	err := routeService.DeleteConditionRoute(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": "success",
	})
}

func EnableConditionRoute(c *gin.Context) {
	id := c.Param("id")
	err := routeService.EnableConditionRoute(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": "success",
	})
}

func DisableConditionRoute(c *gin.Context) {
	id := c.Param("id")
	err := routeService.DisableConditionRoute(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": "success",
	})
}
