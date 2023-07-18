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

var mockSvc = &traffic.MockService{}

// CreateMock   create rule
// @Summary      create rule
// @Description  create rule
// @Tags         TrafficMock
// @Accept       json
// @Produce      json
// @Param        mock  body  model.Mock      true   "rule"
// @Success      200  {bool}    true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/mock [post]
func CreateMock(c *gin.Context) {
	doMockUpdate(c, func(m *model.Mock) error {
		return mockSvc.CreateOrUpdate(m)
	})
}

// UpdateMock   update rule
// @Summary      update rule
// @Description  update rule
// @Tags         TrafficMock
// @Accept       json
// @Produce      json
// @Param        mock  body  model.Mock      true   "rule"
// @Success      200  {bool}    true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/mock [put]
func UpdateMock(c *gin.Context) {
	doMockUpdate(c, func(m *model.Mock) error {
		return mockSvc.CreateOrUpdate(m)
	})
}

// DeleteMock   delete rule
// @Summary      delete rule
// @Description  delete rule
// @Tags         TrafficMock
// @Accept       json
// @Produce      json
// @Param        service  query  string  true   "service name"
// @Param        version  query  string  true   "service version"
// @Param        group    query  string  true   "service group"
// @Success      200  {bool}    true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/mock [delete]
func DeleteMock(c *gin.Context) {
	m := &model.Mock{
		Service: c.Query("service"),
		Group:   c.Query("group"),
		Version: c.Query("version"),
	}

	err := mockSvc.Delete(m)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, true)
}

// SearchMock   get rule list
// @Summary      get rule list
// @Description  get rule list
// @Tags         TrafficMock
// @Accept       json
// @Produce      json
// @Param        service  query  string  true   "service name"
// @Param        version  query  string  true   "service version"
// @Param        group    query  string  true   "service group"
// @Success      200  {object}  []model.Mock
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/mock [get]
func SearchMock(c *gin.Context) {
	m := &model.Mock{
		Service: c.Query("service"),
		Group:   c.Query("group"),
		Version: c.Query("version"),
	}

	result, err := mockSvc.Search(m)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, result)
}

func doMockUpdate(c *gin.Context, handle func(m *model.Mock) error) {
	var m *model.Mock
	if err := c.ShouldBindJSON(&m); err != nil {
		logger.Errorf("Error parsing rule input when trying to create override rule, err msg is %s.", err.Error())
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}

	err := handle(m)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, true)
}
