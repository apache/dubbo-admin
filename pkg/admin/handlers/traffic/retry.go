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

var retrySvc = &traffic.RetryService{}

// CreateRetry   create dds
// @Summary      create dds
// @Description  create dds
// @Tags         TrafficRetry
// @Accept       json
// @Produce      json
// @Param        retry  body  model.Retry      true   "dds"
// @Success      200  {bool}    true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/resource/retry [post]
func CreateRetry(c *gin.Context) {
	doRetryUpdate(c, func(r *model.Retry) error {
		return retrySvc.CreateOrUpdate(r)
	})
}

// UpdateRetry   update dds
// @Summary      update dds
// @Description  update dds
// @Tags         TrafficRetry
// @Accept       json
// @Produce      json
// @Param        retry  body  model.Retry      true   "dds"
// @Success      200  {bool}    true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/resource/retry [put]
func UpdateRetry(c *gin.Context) {
	doRetryUpdate(c, func(r *model.Retry) error {
		return retrySvc.CreateOrUpdate(r)
	})
}

// DeleteRetry   delete dds
// @Summary      delete dds
// @Description  delete dds
// @Tags         TrafficRetry
// @Accept       json
// @Produce      json
// @Param        service  query  string  true   "service name"
// @Param        version  query  string  true   "service version"
// @Param        group    query  string  true   "service group"
// @Success      200  {bool}    true
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/resource/retry [delete]
func DeleteRetry(c *gin.Context) {
	r := &model.Retry{
		Service: c.Query("service"),
		Group:   c.Query("group"),
		Version: c.Query("version"),
	}

	err := retrySvc.Delete(r)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, true)
}

// SearchRetry   get dds list
// @Summary      get dds list
// @Description  get dds list
// @Tags         TrafficRetry
// @Accept       json
// @Produce      json
// @Param        service  query  string  true   "service name"
// @Param        version  query  string  true   "service version"
// @Param        group    query  string  true   "service group"
// @Success      200  {object}  []model.Retry
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/resource/retry [get]
func SearchRetry(c *gin.Context) {
	r := &model.Retry{
		Service: c.Query("service"),
		Group:   c.Query("group"),
		Version: c.Query("version"),
	}

	result, err := retrySvc.Search(r)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, result)
}

func doRetryUpdate(c *gin.Context, handle func(r *model.Retry) error) {
	var r *model.Retry
	if err := c.ShouldBindJSON(&r); err != nil {
		logger.Errorf("Error parsing dds input when trying to create override dds, err msg is %s.", err.Error())
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}

	err := handle(r)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, true)
}
