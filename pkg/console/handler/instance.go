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
	"net/http"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/console/util"
)

func GetInstanceDetail(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.InstanceDetailReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := service.GetInstanceDetail(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}

		if resp == nil {
			util.HandleNotFoundError(c, req.InstanceName)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func SearchInstances(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewSearchInstanceReq()
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewBizErrorResp(
				bizerror.New(bizerror.InvalidArgument, "appName is empty")))
			return
		}

		instances, err := service.SearchInstances(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(instances))
	}
}

func InstanceConfigTrafficDisableGET(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := struct {
			TrafficDisable bool `json:"trafficDisable"`
		}{false}
		applicationName := c.Query("appName")
		mesh := c.Query("mesh")
		if strutil.IsBlank(applicationName) {
			c.JSON(http.StatusBadRequest, model.NewBizErrorResp(
				bizerror.New(bizerror.InvalidArgument, "appName is empty")))
			return
		}
		instanceIP := strings.TrimSpace(c.Query("instanceIP"))
		if strutil.IsBlank(instanceIP) {
			c.JSON(http.StatusBadRequest, model.NewBizErrorResp(
				bizerror.New(bizerror.InvalidArgument, "instanceIP is empty")))
			return
		}
		trafficStatus, err := service.GetInstanceTrafficStatus(ctx, mesh, applicationName, instanceIP)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		resp.TrafficDisable = trafficStatus
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func InstanceConfigTrafficDisablePUT(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		appName := strings.TrimSpace(c.Query("appName"))
		mesh := c.Query("mesh")
		if strutil.IsBlank(appName) {
			c.JSON(http.StatusBadRequest, model.NewBizErrorResp(
				bizerror.New(bizerror.InvalidArgument, "appName is empty")))
			return
		}
		instanceIP := strings.TrimSpace(c.Query("instanceIP"))
		if strutil.IsBlank(instanceIP) {
			c.JSON(http.StatusBadRequest, model.NewBizErrorResp(
				bizerror.New(bizerror.InvalidArgument, "instanceIP is empty")))
			return
		}
		disableTraffic, err := strconv.ParseBool(c.Query(`trafficDisable`))
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewBizErrorResp(
				bizerror.Wrap(err, bizerror.InvalidArgument, "parse trafficDisable failed")))
			return
		}
		err = service.UpdateInstanceTrafficStatus(ctx, mesh, appName, instanceIP, disableTraffic)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}

func InstanceConfigOperatorLogGET(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := struct {
			OperatorLog bool `json:"operatorLog"`
		}{false}
		applicationName := c.Query(`appName`)
		if strutil.IsBlank(applicationName) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("application name is empty"))
			return
		}
		instanceIP := c.Query(`instanceIP`)
		if strutil.IsBlank(instanceIP) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("instanceIP is empty"))
			return
		}
		mesh := c.Query(`mesh`)
		if strutil.IsBlank(mesh) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("mesh is empty"))
			return
		}
		accessLogOpenStatus, err := service.GetInstanceAccessLogOpenStatus(ctx, mesh, applicationName, instanceIP)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		resp.OperatorLog = accessLogOpenStatus
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func InstanceConfigOperatorLogPUT(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		applicationName := c.Query(`appName`)
		if strutil.IsBlank(applicationName) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("application name is empty"))
			return
		}
		instanceIP := c.Query(`instanceIP`)
		if strutil.IsBlank(instanceIP) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("instanceIP is empty"))
			return
		}
		openAccessLog, err := strconv.ParseBool(c.Query(`operatorLog`))
		if err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		mesh := c.Query("mesh")
		if strutil.IsBlank(mesh) {
			c.JSON(http.StatusBadRequest, model.NewErrorResp("mesh is empty"))
			return
		}
		if err = service.UpdateInstanceAccessLogOpenStatus(ctx, mesh, applicationName, instanceIP, openAccessLog); err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}
