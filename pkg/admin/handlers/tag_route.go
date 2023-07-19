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
	"strings"

	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/gin-gonic/gin"
)

var routeService services.RouteService = &services.RouteServiceImpl{}

// CreateRule create a new tag rule
// @Summary      Create a new tag rule
// @Description  Create a new tag rule
// @Tags         TagRule
// @Accept       json
// @Produce      json
// @Param        env       		path  string          		   false  "environment"       default(dev)
// @Param        tagRoute       body  model.TagRouteDto        true   "rule input"
// @Success      200  {boolean} true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/route/tag [post]
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

// UpdateRule update rule
// @Summary      Update rule
// @Description  Update rule
// @Tags         TagRule
// @Accept       json
// @Produce      json
// @Param        env       path  string          		  false  "environment"       default(dev)
// @Param        id        path  string          		  true   "rule id"
// @Param        tagRoute  body  model.TagRouteDto  true   "rule input"
// @Success      200  {boolean} true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/route/tag/{id} [post]
func UpdateRule(c *gin.Context) {
	var tagRouteDto model.TagRouteDto
	err := c.BindJSON(&tagRouteDto)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
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

// SearchRoutes search rule with key word
// @Summary      Search rule
// @Description  Search rule
// @Tags         TagRule
// @Accept       json
// @Produce      json
// @Param        env       			path     string     false  "environment"       default(dev)
// @Param        application        query    string     false  "application and service must not left empty at the same time"
// @Success      200  {object}  []model.TagRouteDto
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/route/tag [get]
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

// DetailRoute show the detail of one specified rule
// @Summary      Show the detail of one specified rule
// @Description  Show the detail of one specified rule
// @Tags         TagRule
// @Accept       json
// @Produce      json
// @Param        env       path  string          		  false  "environment"       default(dev)
// @Param        id        path  string          		  true   "rule id"
// @Success      200  {object}  model.TagRouteDto
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/route/tag/{id} [get]
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

// DeleteRoute delete the specified rule
// @Summary      Delete the specified rule
// @Description  Delete the specified rule
// @Tags         TagRule
// @Accept       json
// @Produce      json
// @Param        env       path  string          		  false  "environment"       default(dev)
// @Param        id        path  string          		  true   "rule id"
// @Success      200  {boolean} true
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/route/tag/{id} [delete]
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

// EnableRoute Enable the specified rule
// @Summary      Enable the specified rule
// @Description  Enable the specified rule
// @Tags         TagRule
// @Accept       json
// @Produce      json
// @Param        env       path  string          		  false  "environment"       default(dev)
// @Param        id        path  string          		  true   "rule id"
// @Success      200  {boolean} true
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/route/tag/enable/{id} [put]
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

// DisableRoute Disable the specified rule
// @Summary      Disable the specified rule
// @Description  Disable the specified rule
// @Tags         TagRule
// @Accept       json
// @Produce      json
// @Param        env       path  string          		  false  "environment"       default(dev)
// @Param        id        path  string          		  true   "rule id"
// @Success      200  {boolean} true
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/rules/route/tag/disable/{id} [put]
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
