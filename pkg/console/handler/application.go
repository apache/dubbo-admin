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

package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/gin-gonic/gin"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/console/util"
)

func GetApplicationDetail(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.ApplicationDetailReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		resp, err := service.GetApplicationDetail(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetApplicationTabInstanceInfo(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewApplicationTabInstanceInfoReq()
		if err := c.ShouldBindQuery(req); err != nil {
			util.HandleArgumentError(c, err)
			return
		}

		resp, err := service.GetAppInstanceInfo(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetApplicationServiceForm(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewApplicationServiceFormReq()
		if err := c.ShouldBindQuery(req); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		resp, err := service.GetAppServiceInfo(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func ApplicationSearch(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewApplicationSearchReq()
		if err := c.ShouldBindQuery(req); err != nil {
			util.HandleArgumentError(c, err)
			return
		}

		resp, err := service.SearchApplications(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetApplicationGraph(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewApplicationGraphReq()
		if err := c.ShouldBindQuery(req); err != nil {
			util.HandleArgumentError(c, err)
			return
		}

		resp, err := service.GraphApplications(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func ApplicationConfigAccessLogPut(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		appName := c.Query("appName")
		if strutil.IsBlank(appName) {
			util.HandleArgumentError(c, errors.New("appName is required"))
			return
		}
		mesh := c.Query("mesh")
		if strutil.IsBlank(mesh) {
			util.HandleArgumentError(c, errors.New("mesh is required"))
			return
		}
		operatorLogOpen, err := strconv.ParseBool(c.Query("operatorLog"))
		if err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		if err := service.UpInsertAppAccessLog(ctx, appName, operatorLogOpen, mesh); err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(true))
	}
}

func ApplicationConfigAccessLogGet(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		appName := c.Query("appName")
		if strutil.IsBlank(appName) {
			util.HandleArgumentError(c, errors.New("appName is required"))
			return
		}
		mesh := c.Query("mesh")
		if strutil.IsBlank(mesh) {
			util.HandleArgumentError(c, errors.New("mesh is required"))
			return
		}
		resp, err := service.GetAppAccessLog(ctx, appName, mesh)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func ApplicationConfigFlowWeightGET(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		appName := c.Query("appName")
		mesh := c.Query("mesh")
		if strutil.IsBlank(appName) {
			util.HandleArgumentError(c, errors.New("appName is required"))
			return
		}
		if strutil.IsBlank(mesh) {
			util.HandleArgumentError(c, errors.New("mesh is required"))
			return
		}
		resp, err := service.GetAppFlowWeight(ctx, appName, mesh)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func ApplicationConfigFlowWeightPUT(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		body := struct {
			FlowWeightSets []model.FlowWeightSet `json:"flowWeightSets"`
		}{}
		appName := c.Query("appName")
		mesh := c.Query("mesh")
		if strutil.IsBlank(appName) {
			util.HandleArgumentError(c, errors.New("appName is required"))
			return
		}
		if strutil.IsBlank(mesh) {
			util.HandleArgumentError(c, errors.New("mesh is required"))
			return
		}
		if err := c.Bind(&body); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		err := service.UpInsertAppFlowWeightConfig(ctx, appName, mesh, body.FlowWeightSets)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(true))
	}
}

func ApplicationConfigGrayGET(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		appName := c.Query("appName")
		mesh := c.Query("mesh")
		if strutil.IsBlank(appName) {
			util.HandleArgumentError(c, errors.New("appName is required"))
			return
		}
		if strutil.IsBlank(mesh) {
			util.HandleArgumentError(c, errors.New("mesh is required"))
			return
		}
		resp, err := service.GetGrayConfig(ctx, appName, mesh)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func ApplicationConfigGrayPUT(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		body := struct {
			GraySets []model.GraySet `json:"graySets"`
		}{}
		appName := c.Query("appName")
		mesh := c.Query("mesh")
		if strutil.IsBlank(appName) {
			util.HandleArgumentError(c, errors.New("appName is required"))
			return
		}
		if strutil.IsBlank(mesh) {
			util.HandleArgumentError(c, errors.New("mesh is required"))
			return
		}
		if err := c.Bind(&body); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		err := service.UpInsertAppGrayConfig(ctx, appName, mesh, body.GraySets)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(true))
	}
}
