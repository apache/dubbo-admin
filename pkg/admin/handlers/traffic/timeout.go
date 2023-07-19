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

var timeoutSvc = &traffic.TimeoutService{}

// CreateTimeout create a new timeout rule
// @Summary      Create a new timeout rule
// @Description  Create a new timeout rule
// @Tags         TrafficTimeout
// @Accept       json
// @Produce      json
// @Param        timeout  body  model.Timeout      true   "timeout rule"
// @Success      200  {boolean} true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/timeout [post]
func CreateTimeout(c *gin.Context) {
	doTimeoutUpdate(c, func(t *model.Timeout) error {
		return timeoutSvc.CreateOrUpdate(t)
	})
}

// UpdateTimeout update a new timeout rule
// @Summary      update a new timeout rule
// @Description  update a new timeout rule
// @Tags         TrafficTimeout
// @Accept       json
// @Produce      json
// @Param        timeout  body  model.Timeout      true   "timeout rule"
// @Success      200  {boolean} true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/timeout [put]
func UpdateTimeout(c *gin.Context) {
	doTimeoutUpdate(c, func(t *model.Timeout) error {
		return timeoutSvc.CreateOrUpdate(t)
	})
}

// DeleteTimeout delete a new timeout rule
// @Summary      delete a new timeout rule
// @Description  delete a new timeout rule
// @Tags         TrafficTimeout
// @Accept       json
// @Produce      json
// @Param        service  query  string  true   "service name"
// @Param        version  query  string  true   "service version"
// @Param        group    query  string  true   "service group"
// @Success      200  {boolean} true
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/timeout [delete]
func DeleteTimeout(c *gin.Context) {
	t := &model.Timeout{
		Service: c.Query("service"),
		Group:   c.Query("group"),
		Version: c.Query("version"),
	}

	err := timeoutSvc.Delete(t)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, true)
}

// SearchTimeout get timeout rule list
// @Summary      get timeout rule list
// @Description  get timeout rule list
// @Tags         TrafficTimeout
// @Accept       json
// @Produce      json
// @Param        service  query  string  true   "service name"
// @Param        version  query  string  true   "service version"
// @Param        group    query  string  true   "service group"
// @Success      200  {object}  []model.Timeout
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/traffic/timeout [get]
func SearchTimeout(c *gin.Context) {
	t := &model.Timeout{
		Service: c.Query("service"),
		Group:   c.Query("group"),
		Version: c.Query("version"),
	}

	result, err := timeoutSvc.Search(t)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, result)
}

func doTimeoutUpdate(c *gin.Context, handle func(t *model.Timeout) error) {
	var t *model.Timeout
	if err := c.ShouldBindJSON(&t); err != nil {
		logger.Errorf("Error parsing rule input when trying to create override rule, err msg is %s.", err.Error())
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}

	err := handle(t)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, true)
}
