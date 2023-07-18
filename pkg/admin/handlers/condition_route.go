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

	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/gin-gonic/gin"
)

// CreateConditionRule create a new condition rule
// @Summary      Create a new condition rule
// @Description  Create a new condition rule
// @Tags         ConditionRule
// @Accept       json
// @Produce      json
// @Param        env       path  string          		  false  "environment"       default(dev)
// @Param        routeDto  body  model.ConditionRouteDto  true   "Condition Rule Input"
// @Success      200  {boolean} true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/route/condition [post]
func CreateConditionRule(c *gin.Context) {
	var routeDto model.ConditionRouteDto
	err := c.BindJSON(&routeDto)
	if err != nil {
		logger.Errorf("Parsing condition rule input error, err msg is: %s", err.Error())
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}

	err = routeService.CreateConditionRoute(routeDto)

	if err != nil {
		if _, ok := err.(*config.RuleExists); ok {
			logger.Infof("Condition rule for service %s already exists!", routeDto.Service)
			c.JSON(http.StatusOK, true)
		} else {
			logger.Errorf("Creating condition rule for service %s failed, err msg is: %s", routeDto.Service, err.Error())
			c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, true)
	return
}

// UpdateConditionRule update condition rule
// @Summary      Update condition rule
// @Description  Update condition rule
// @Tags         ConditionRule
// @Accept       json
// @Produce      json
// @Param        env       path  string          		  false  "environment"       default(dev)
// @Param        id        path  string          		  true   "Condition Rule Id"
// @Param        routeDto  body  model.ConditionRouteDto  true   "Condition Rule Input"
// @Success      200  {boolean} true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/route/condition/{id} [post]
func UpdateConditionRule(c *gin.Context) {
	var routeDto model.ConditionRouteDto
	err := c.BindJSON(&routeDto)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}
	id := c.Param("id")

	_, err = routeService.FindConditionRouteById(id)
	if err != nil {
		logger.Errorf("Check failed before trying to update condition rule for service %s , err msg is: %s", routeDto.Service, err.Error())
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		return
	}

	err = routeService.UpdateConditionRoute(routeDto)

	if err != nil {
		logger.Errorf("Update condition rule for service %s failed, err msg is: %s", routeDto.Service, err.Error())
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, true)
}

// SearchConditionRoutes search condition rule with key word
// @Summary      Search condition rule
// @Description  Search condition rule
// @Tags         ConditionRule
// @Accept       json
// @Produce      json
// @Param        env       			path     string     false  "environment"       default(dev)
// @Param        application        query    string     false  "application and service must not left empty at the same time"
// @Param        service     		query    string     false  "application and service must not left empty at the same time"
// @Param        serviceVersion     query    string     false  "version of dubbo service"
// @Param        serviceGroup       query    string     false  "group of dubbo service"
// @Success      200  {object}  []model.ConditionRouteDto
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/route/condition [get]
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

// DetailConditionRoute show the detail of one specified condition rule
// @Summary      Show the detail of one specified condition rule
// @Description  Show the detail of one specified condition rule
// @Tags         ConditionRule
// @Accept       json
// @Produce      json
// @Param        env       path  string          		  false  "environment"       default(dev)
// @Param        id        path  string          		  true   "Condition Rule Id"
// @Success      200  {object}  model.ConditionRouteDto
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/route/condition/{id} [get]
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

// DeleteConditionRoute delete the specified condition rule
// @Summary      Delete the specified condition rule
// @Description  Delete the specified condition rule
// @Tags         ConditionRule
// @Accept       json
// @Produce      json
// @Param        env       path  string          		  false  "environment"       default(dev)
// @Param        id        path  string          		  true   "Condition Rule Id"
// @Success      200  {boolean} true
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/route/condition/{id} [delete]
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

// EnableConditionRoute Enable the specified condition rule
// @Summary      Enable the specified condition rule
// @Description  Enable the specified condition rule
// @Tags         ConditionRule
// @Accept       json
// @Produce      json
// @Param        env       path  string          		  false  "environment"       default(dev)
// @Param        id        path  string          		  true   "Condition Rule Id"
// @Success      200  {boolean} true
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/route/condition/enable/{id} [put]
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

// DisableConditionRoute Disable the specified condition rule
// @Summary      Disable the specified condition rule
// @Description  Disable the specified condition rule
// @Tags         ConditionRule
// @Accept       json
// @Produce      json
// @Param        env       path  string          		  false  "environment"       default(dev)
// @Param        id        path  string          		  true   "Condition Rule Id"
// @Success      200  {boolean} true
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/route/condition/disable/{id} [put]
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
