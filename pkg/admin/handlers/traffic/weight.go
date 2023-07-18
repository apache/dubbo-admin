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

package traffic

import (
	"net/http"

	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services/traffic"
	"github.com/gin-gonic/gin"
)

var weightSvc = &traffic.WeightService{}

// CreateWeight create rule
// @Summary      create rule
// @Description  create rule
// @Tags         TrafficWeight
// @Accept       json
// @Produce      json
// @Param        weight  body  model.Percentage      true   "rule"
// @Success      200  {bool}    true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/weight [post]
func CreateWeight(c *gin.Context) {
	doWeightUpdate(c, func(p *model.Percentage) error {
		return weightSvc.CreateOrUpdate(p)
	})
}

// UpdateWeight update rule
// @Summary      update rule
// @Description  update rule
// @Tags         TrafficWeight
// @Accept       json
// @Produce      json
// @Param        weight  body  model.Percentage      true   "rule"
// @Success      200  {bool}    true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/weight [put]
func UpdateWeight(c *gin.Context) {
	doWeightUpdate(c, func(p *model.Percentage) error {
		return weightSvc.CreateOrUpdate(p)
	})
}

// DeleteWeight delete rule
// @Summary      delete rule
// @Description  delete rule
// @Tags         TrafficWeight
// @Accept       json
// @Produce      json
// @Param        service  query  string  true   "service name"
// @Param        version  query  string  true   "service version"
// @Param        group    query  string  true   "service group"
// @Success      200  {bool}    true
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/weight [delete]
func DeleteWeight(c *gin.Context) {
	p := &model.Percentage{
		Service: c.Query("service"),
		Group:   c.Query("group"),
		Version: c.Query("version"),
	}

	err := weightSvc.Delete(p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, true)
}

// SearchWeight get rule list
// @Summary      get rule list
// @Description  get rule list
// @Tags         TrafficWeight
// @Accept       json
// @Produce      json
// @Param        service  query  string  true   "service name"
// @Param        version  query  string  true   "service version"
// @Param        group    query  string  true   "service group"
// @Success      200  {object}  []model.Weight
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/weight [get]
func SearchWeight(c *gin.Context) {
	p := &model.Percentage{
		Service: c.Query("service"),
		Group:   c.Query("group"),
		Version: c.Query("version"),
	}

	result, err := weightSvc.Search(p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, result)
}

func doWeightUpdate(c *gin.Context, handle func(p *model.Percentage) error) {
	var p *model.Percentage
	if err := c.ShouldBindJSON(&p); err != nil {
		logger.Errorf("Error parsing rule input when trying to create override rule, err msg is %s.", err.Error())
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}

	err := handle(p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, true)
}
