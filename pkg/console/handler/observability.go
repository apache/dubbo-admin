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

	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/console/constants"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
)

type Dimension string

const (
	AppDimension      Dimension = constants.Application
	InstanceDimension Dimension = constants.Instance
	ServiceDimension  Dimension = constants.Service
)

func GetMetricDashBoard(ctx consolectx.Context, dim Dimension) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.DashboardReq
		var url string
		switch dim {
		case AppDimension:
			req = &model.AppDashboardReq{}
			url = ctx.Config().Console.MetricDashboards.Application.BaseURL
		case InstanceDimension:
			req = &model.InstanceDashboardReq{}
			url = ctx.Config().Console.MetricDashboards.Instance.BaseURL
		case ServiceDimension:
			req = &model.ServiceDashboardReq{}
			url = ctx.Config().Console.MetricDashboards.Service.BaseURL
		}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		resp := model.DashboardResp{
			BaseURL: url + req.GetKeyVariable(),
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetTraceDashBoard(ctx consolectx.Context, dim Dimension) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.DashboardReq
		var url string
		switch dim {
		case AppDimension:
			req = &model.AppDashboardReq{}
			url = ctx.Config().Console.TraceDashboards.Application.BaseURL + "?var-application="
		case InstanceDimension:
			req = &model.InstanceDashboardReq{}
			url = ctx.Config().Console.TraceDashboards.Instance.BaseURL + "?var-instance="
		case ServiceDimension:
			req = &model.ServiceDashboardReq{}
			url = ctx.Config().Console.TraceDashboards.Service.BaseURL + "?var-service="
		}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		resp := model.DashboardResp{
			BaseURL: url + req.GetKeyVariable(),
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetPrometheus(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := ctx.Config().Console.Prometheus
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetMetricsList(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.MetricsReq{}
		if err := c.ShouldBindQuery(req); err != nil {
		}
		resp, err := service.GetInstanceMetrics(ctx, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.NewErrorResp(err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}
