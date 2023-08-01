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

var accesslogSvc = &traffic.AccesslogService{}

// CreateAccesslog   create dds
// @Summary          create dds
// @Description      create dds
// @Tags         TrafficAccesslog
// @Accept       json
// @Produce      json
// @Param        accesslog  body  model.Accesslog    true   "dds"
// @Success      200  {bool}    true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/resource/accesslog [post]
func CreateAccesslog(c *gin.Context) {
	doAccesslogUpdate(c, func(a *model.Accesslog) error {
		return accesslogSvc.CreateOrUpdate(a)
	})
}

// UpdateAccesslog   create dds
// @Summary          create dds
// @Description      create dds
// @Tags         TrafficAccesslog
// @Accept       json
// @Produce      json
// @Param        accesslog  body  model.Accesslog      true   "dds"
// @Success      200  {bool}    true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/resource/accesslog [put]
func UpdateAccesslog(c *gin.Context) {
	doAccesslogUpdate(c, func(a *model.Accesslog) error {
		return accesslogSvc.CreateOrUpdate(a)
	})
}

// DeleteAccesslog   delete dds
// @Summary          delete dds
// @Description      delete dds
// @Tags         TrafficAccesslog
// @Accept       json
// @Produce      json
// @Param        application  query  string  true   "application name"
// @Success      200  {bool}    true
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/resource/accesslog [delete]
func DeleteAccesslog(c *gin.Context) {
	a := &model.Accesslog{
		Application: c.Query("application"),
	}

	err := accesslogSvc.Delete(a)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, true)
}

// SearchAccesslog   get dds list
// @Summary          get dds list
// @Description      get dds list
// @Tags         TrafficAccesslog
// @Accept       json
// @Produce      json
// @Param        application  query  string  true   "application name"
// @Success      200  {object}  []model.Accesslog
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/resource/accesslog [get]
func SearchAccesslog(c *gin.Context) {
	a := &model.Accesslog{
		Application: c.Query("application"),
	}

	result, err := accesslogSvc.Search(a)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, result)
}

func doAccesslogUpdate(c *gin.Context, handle func(a *model.Accesslog) error) {
	var a *model.Accesslog
	if err := c.ShouldBindJSON(&a); err != nil {
		logger.Errorf("Error parsing dds input when trying to create override dds, err msg is %s.", err.Error())
		c.JSON(http.StatusBadRequest, model.HTTPError{Error: err.Error()})
		return
	}

	err := handle(a)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, true)
}
