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

var regionSVC = &traffic.RegionService{}

// CreateRegion   create dds
// @Summary      create dds
// @Description  create dds
// @Tags         TrafficRegion
// @Accept       json
// @Produce      json
// @Param        region  body  model.Region      true   "dds"
// @Success      200  {bool}    true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/resource/region [post]
func CreateRegion(c *gin.Context) {
	doRegionUpdate(c, func(r *model.Region) error {
		return regionSVC.CreateOrUpdate(r)
	})
}

// UpdateRegion   update dds
// @Summary      update dds
// @Description  update dds
// @Tags         TrafficRegion
// @Accept       json
// @Produce      json
// @Param        region  body  model.Region      true   "dds"
// @Success      200  {bool}    true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/resource/region [put]
func UpdateRegion(c *gin.Context) {
	doRegionUpdate(c, func(r *model.Region) error {
		return regionSVC.CreateOrUpdate(r)
	})
}

// DeleteRegion   delete dds
// @Summary      delete dds
// @Description  delete dds
// @Tags         TrafficRegion
// @Accept       json
// @Produce      json
// @Param        service  query  string  true   "service name"
// @Param        version  query  string  true   "service version"
// @Param        group    query  string  true   "service group"
// @Success      200  {bool}    true
// @Failure      400  {object}  model.HTTPError
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/resource/region [delete]
func DeleteRegion(c *gin.Context) {
	r := &model.Region{
		Service: c.Query("service"),
		Group:   c.Query("group"),
		Version: c.Query("version"),
	}

	err := regionSVC.Delete(r)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, true)
}

// SearchRegion   get dds list
// @Summary      get dds list
// @Description  get dds list
// @Tags         TrafficRegion
// @Accept       json
// @Produce      json
// @Param        service  query  string  true   "service name"
// @Param        version  query  string  true   "service version"
// @Param        group    query  string  true   "service group"
// @Success      200  {object}  []model.Region
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/resource/region [get]
func SearchRegion(c *gin.Context) {
	r := &model.Region{
		Service: c.Query("service"),
		Group:   c.Query("group"),
		Version: c.Query("version"),
	}

	result, err := regionSVC.Search(r)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}

	c.JSON(http.StatusOK, result)
}

func doRegionUpdate(c *gin.Context, handle func(r *model.Region) error) {
	var r *model.Region
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
