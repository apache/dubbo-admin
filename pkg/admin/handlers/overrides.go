// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handlers

import (
	"net/http"

	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/apache/dubbo-admin/pkg/admin/util"
	"github.com/gin-gonic/gin"
)

var overrideServiceImpl services.OverrideService = &services.OverrideServiceImpl{}

// CreateOverride create a new override rule
// @Summary      Create a new override rule
// @Description  Create a new override rule
// @Tags         OverrideRule
// @Accept       json
// @Produce      json
// @Param        env       		path  string          		   false  "environment"       default(dev)
// @Param        dynamicConfig  body  model.DynamicConfig      true   "Override Rule Input"
// @Success      201  {boolean} true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/override [post]
func CreateOverride(c *gin.Context) {
	var dynamicConfig *model.DynamicConfig
	if err := c.ShouldBindJSON(&dynamicConfig); err != nil {
		logger.Errorf("Error parsing override rule input when trying to create override rule, err msg is %s.", err.Error())
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}

	serviceName := dynamicConfig.Service
	application := dynamicConfig.Application
	if serviceName == "" && application == "" {
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: "service or application must not be empty"})
		return
	}
	// TODO: providerService.findVersionInApplication(application).equals("2.6")
	// if application != "" && providerService.findVersionInApplication(application).equals("2.6") {
	// 	c.JSON(http.StatusBadRequest, errors.New("dubbo 2.6 does not support application scope dynamic config"))
	// 	return
	// }
	err := overrideServiceImpl.SaveOverride(dynamicConfig)
	if err != nil {
		if _, ok := err.(*config.RuleExists); ok {
			logger.Infof("Override rule already exists!")
		} else {
			logger.Infof("Override rule create failed, err msg is %s.", err.Error())
			c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, true)
}

// UpdateOverride update override rule
// @Summary      Update override rule
// @Description  Update override rule
// @Tags         OverrideRule
// @Accept       json
// @Produce      json
// @Param        env       path  string          		  false  "environment"       default(dev)
// @Param        id        path  string          		  true   "Override Rule Id"
// @Param        dynamicConfig  body  model.DynamicConfig  true   "Override Rule Input"
// @Success      200  {boolean} true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/override/{id} [post]
func UpdateOverride(c *gin.Context) {
	id := c.Param("id")
	// env := c.Param("env")
	var dynamicConfig model.DynamicConfig
	if err := c.ShouldBindJSON(&dynamicConfig); err != nil {
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}

	old, err := overrideServiceImpl.FindOverride(id)
	if err != nil {
		logger.Errorf("Check failed before trying to update override rule for service %s , err msg is: %s", dynamicConfig.Service, err.Error())
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}
	if old == nil {
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: "override not found"})
		return
	}
	if err := overrideServiceImpl.UpdateOverride(&dynamicConfig); err != nil {
		logger.Errorf("Update tag rule for service %s failed, err msg is: %s", dynamicConfig.Service, err.Error())
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, true)
}

// SearchOverride search override rule with key word
// @Summary      Search override rule
// @Description  Search override rule
// @Tags         OverrideRule
// @Accept       json
// @Produce      json
// @Param        env       			path     string     false  "environment"       default(dev)
// @Param        application        query    string     false  "application and service must not left empty at the same time"
// @Param        service     		query    string     false  "application and service must not left empty at the same time"
// @Param        serviceVersion     query    string     false  "version of dubbo service"
// @Param        serviceGroup       query    string     false  "group of dubbo service"
// @Success      200  {object}  []model.DynamicConfig
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/override [get]
func SearchOverride(c *gin.Context) {
	service := c.DefaultQuery("service", "")
	application := c.DefaultQuery("application", "")
	serviceVersion := c.DefaultQuery("serviceVersion", "")
	serviceGroup := c.DefaultQuery("serviceGroup", "")

	var override *model.DynamicConfig
	result := make([]*model.DynamicConfig, 0)
	var err error
	if service != "" {
		id := util.BuildServiceKey("", service, serviceGroup, serviceVersion)
		override, err = overrideServiceImpl.FindOverride(id)
	} else if application != "" {
		override, err = overrideServiceImpl.FindOverride(application)
	} else {
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: "Either Service or application is required!"})
		return
	}

	if err != nil {
		if _, ok := err.(*config.RuleNotFound); ok {
			logger.Infof("No override rule found for query parameters: application %s, service %, group %, version %", application, service, serviceGroup, serviceVersion)
			c.JSON(http.StatusOK, []*model.DynamicConfig{})
		} else {
			logger.Errorf("Check override rule failed, err msg is: %s", err.Error())
			c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		}
	}

	if override != nil {
		result = append(result, override)
	}
	c.JSON(http.StatusOK, result)
}

// DetailOverride show the detail of one specified rule
// @Summary      Show the detail of one specified rule
// @Description  Show the detail of one specified rule
// @Tags         OverrideRule
// @Accept       json
// @Produce      json
// @Param        env       path  string          		  false  "environment"       default(dev)
// @Param        id        path  string          		  true   "rule id"
// @Success      200  {object}  model.DynamicConfig
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/override/{id} [get]
func DetailOverride(c *gin.Context) {
	id := c.Param("id")
	override, err := overrideServiceImpl.FindOverride(id)
	if err != nil {
		logger.Errorf("Check override rule detail with id %s failed, err msg is: %s", id, err.Error())
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		return
	}
	if override == nil {
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: "Unknown ID!"})
		return
	}
	c.JSON(http.StatusOK, override)
}

// EnableOverride Enable the specified rule
// @Summary      Enable the specified rule
// @Description  Enable the specified rule
// @Tags         OverrideRule
// @Accept       json
// @Produce      json
// @Param        env       path  string          		  false  "environment"       default(dev)
// @Param        id        path  string          		  true   "rule id"
// @Success      200  {boolean} true
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/override/enable/{id} [put]
func EnableOverride(c *gin.Context) {
	id := c.Param("id")
	err := overrideServiceImpl.EnableOverride(id)
	if err != nil {
		logger.Errorf("Enable override rule with id %s failed, err msg is: %s", id, err.Error())
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, true)
}

// DeleteOverride delete the specified rule
// @Summary      Delete the specified rule
// @Description  Delete the specified rule
// @Tags         OverrideRule
// @Accept       json
// @Produce      json
// @Param        env       path  string          		  false  "environment"       default(dev)
// @Param        id        path  string          		  true   "rule id"
// @Success      200  {boolean} true
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/override/{id} [delete]
func DeleteOverride(c *gin.Context) {
	id := c.Param("id")
	err := overrideServiceImpl.DeleteOverride(id)
	if err != nil {
		logger.Errorf("Delete override rule with id %s failed, err msg is: %s", id, err.Error())
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, true)
}

// DisableOverride Disable the specified rule
// @Summary      Disable the specified rule
// @Description  Disable the specified rule
// @Tags         OverrideRule
// @Accept       json
// @Produce      json
// @Param        env       path  string          		  false  "environment"       default(dev)
// @Param        id        path  string          		  true   "rule id"
// @Success      200  {boolean} true
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/override/disable/{id} [put]
func DisableOverride(c *gin.Context) {
	id := c.Param("id")
	err := overrideServiceImpl.DisableOverride(id)
	if err != nil {
		logger.Errorf("Disable override rule with id %s failed, err msg is: %s", id, err.Error())
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, true)
}
