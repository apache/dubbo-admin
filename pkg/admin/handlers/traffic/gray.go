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

var graySVC = &traffic.GrayService{}

// CreateGray   create rule
// @Summary      create rule
// @Description  create rule
// @Tags         TrafficGray
// @Accept       json
// @Produce      json
// @Param        gray  body  model.Gray      true   "rule"
// @Success      200  {bool}    true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/gray [post]
func CreateGray(c *gin.Context) {
	doGrayUpdate(c, func(g *model.Gray) error {
		return graySVC.CreateOrUpdate(g)
	})
}

// UpdateGray   update rule
// @Summary      update rule
// @Description  update rule
// @Tags         TrafficGray
// @Accept       json
// @Produce      json
// @Param        gray  body  model.Gray      true   "rule"
// @Success      200  {bool}    true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/gray [put]
func UpdateGray(c *gin.Context) {
	doGrayUpdate(c, func(g *model.Gray) error {
		return graySVC.CreateOrUpdate(g)
	})
}

// DeleteGray   delete rule
// @Summary      delete rule
// @Description  delete rule
// @Tags         TrafficGray
// @Accept       json
// @Produce      json
// @Param        application  query  string  true   "application name"
// @Success      200  {bool}    true
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/gray [delete]
func DeleteGray(c *gin.Context) {
	g := &model.Gray{
		Application: c.Query("application"),
	}

	err := graySVC.Delete(g)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, true)
}

// SearchGray   get rule list
// @Summary      get rule list
// @Description  get rule list
// @Tags         TrafficGray
// @Accept       json
// @Produce      json
// @Param        application  query  string  true   "application name"
// @Success      200  {object}  []model.Gray
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/gray [get]
func SearchGray(c *gin.Context) {
	g := &model.Gray{
		Application: c.Query("application"),
	}

	result, err := graySVC.Search(g)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, result)
}

func doGrayUpdate(c *gin.Context, handle func(g *model.Gray) error) {
	var g *model.Gray
	if err := c.ShouldBindJSON(&g); err != nil {
		logger.Errorf("Error parsing rule input when trying to create override rule, err msg is %s.", err.Error())
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}

	err := handle(g)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, true)
}
