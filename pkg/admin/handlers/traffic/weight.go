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
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/services/traffic"
	"github.com/apache/dubbo-admin/pkg/logger"
	"github.com/gin-gonic/gin"
	"net/http"
)

var weightSvc = &traffic.WeightService{}

// CreateWeight create rule
// @Summary      create rule
// @Description  create rule
// @Tags         TrafficWeight
// @Accept       json
// @Produce      json
// @Param        weight  body  model.Weight      true   "rule"
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
// @Param        weight  body  model.Weight      true   "rule"
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
// @Param        weight  body  model.Weight      true   "rule"
// @Success      200  {bool}    true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/weight [delete]
func DeleteWeight(c *gin.Context) {
	doWeightUpdate(c, func(p *model.Percentage) error {
		return weightSvc.Delete(p)
	})
}

// SearchWeight get rule list
// @Summary      get rule list
// @Description  get rule list
// @Tags         TrafficWeight
// @Accept       json
// @Produce      json
// @Param        weight  body  model.Weight      true   "rule"
// @Success      200  {object}  []model.Weight
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/weight [get]
func SearchWeight(c *gin.Context) {
	var p *model.Percentage
	if err := c.ShouldBindJSON(&p); err != nil {
		logger.Errorf("Error parsing rule input when trying to create override rule, err msg is %s.", err.Error())
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
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
