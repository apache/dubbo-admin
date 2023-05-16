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

	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/gin-gonic/gin"
)

func CreateConditionRule(c *gin.Context) {
	var routeDto model.ConditionRouteDto
	err := c.BindJSON(&routeDto)
	if err != nil {
		logger.Errorf("Parsing condition rule input error, err msg is: %s", err.Error())
		c.JSON(http.StatusBadRequest, false)
		return
	}

	err = routeService.CreateConditionRoute(routeDto)

	if err != nil {
		if _, ok := err.(*config.RuleExists); ok {
			logger.Infof("Condition rule for service %s already exists!", routeDto.Service)
			c.JSON(http.StatusOK, true)
		} else {
			logger.Errorf("Creating condition rule for service %s failed, err msg is: %s", routeDto.Service, err.Error())
			//fixme, return more information of the exact failure type.
			c.JSON(http.StatusInternalServerError, false)
		}
		return
	}
	c.JSON(http.StatusOK, true)
	return
}

func UpdateConditionRule(c *gin.Context) {
	var routeDto model.ConditionRouteDto
	err := c.BindJSON(&routeDto)
	if err != nil {
		panic(err)
	}
	id := c.Param("id")

	_, err = routeService.FindConditionRouteById(id)
	if err != nil {
		logger.Errorf("Check failed before trying to update condition rule for service %s , err msg is: %s", routeDto.Service, err.Error())
		c.JSON(http.StatusInternalServerError, false)
		return
	}

	err = routeService.UpdateConditionRoute(routeDto)

	if err != nil {
		logger.Errorf("Update condition rule for service %s failed, err msg is: %s", routeDto.Service, err.Error())
		c.JSON(http.StatusInternalServerError, false)
		return
	}
	c.JSON(http.StatusOK, true)
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
		logger.Errorf("Unsupported query type for condition rule, only application and service is available: %s", err.Error())
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}

	if err != nil {
		if _, ok := err.(*config.RuleNotFound); ok {
			logger.Infof("No condition rule for query parameters: service %s, application %s, group %s, version %s ", service, application, serviceGroup, serviceVersion)
			c.JSON(http.StatusOK, []model.ConditionRouteDto{})
		} else {
			logger.Errorf("Check condition rule detail failed, err msg is: %s", err.Error())
			c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, []model.ConditionRouteDto{routeDto})
}

func DetailConditionRoute(c *gin.Context) {
	id := c.Param("id")
	routeDto, err := routeService.FindConditionRouteById(id)
	if err != nil {
		logger.Errorf("Check condition rule detail with id %s failed, err msg is: %s", id, err.Error())
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, routeDto)
}

func DeleteConditionRoute(c *gin.Context) {
	id := c.Param("id")
	err := routeService.DeleteConditionRoute(id)
	if err != nil {
		logger.Errorf("Delete condition rule with id %s failed, err msg is: %s", id, err.Error())
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, true)
}

func EnableConditionRoute(c *gin.Context) {
	id := c.Param("id")
	err := routeService.EnableConditionRoute(id)
	if err != nil {
		logger.Errorf("Enable condition rule with id %s failed, err msg is: %s", id, err.Error())
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, true)
}

func DisableConditionRoute(c *gin.Context) {
	id := c.Param("id")
	err := routeService.DisableConditionRoute(id)
	if err != nil {
		logger.Errorf("Disable condition rule with id %s failed, err msg is: %s", id, err.Error())
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, true)
}
