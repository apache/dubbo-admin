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
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	consolecfg "github.com/apache/dubbo-admin/pkg/config/console"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/console/util"
)

type DashboardType string

const (
	MetricDashboard DashboardType = "metric"
	TraceDashboard  DashboardType = "trace"
)

type Dimension string

const (
	AppDimension      Dimension = constants.Application
	InstanceDimension Dimension = constants.Instance
	ServiceDimension  Dimension = constants.Service
)

func GetGrafanaDashboard(ctx consolectx.Context, dim Dimension, dashboardType DashboardType) gin.HandlerFunc {
	return func(c *gin.Context) {
		var grafanaDashboards *consolecfg.GrafanaDashboardConfig
		switch dashboardType {
		case MetricDashboard:
			grafanaDashboards = ctx.Config().Console.MetricDashboards
		case TraceDashboard:
			grafanaDashboards = ctx.Config().Console.TraceDashboards
		}
		if grafanaDashboards == nil {
			c.JSON(http.StatusOK, model.NewBizErrorResp(bizerror.New(bizerror.NotFoundError,
				fmt.Sprintf("please configure grafana dashboard for %s %s in config yaml", dim, dashboardType))))
		}

		var fullURL string
		var err error
		switch dim {
		case AppDimension:
			var req model.AppDashboardReq
			if err = c.ShouldBindQuery(&req); err != nil {
				break
			}
			fullURL, err = service.GetAppDashboard(grafanaDashboards.AppDashboardURL, &req)
		case InstanceDimension:
			var req model.InstanceDashboardReq
			if err = c.ShouldBindQuery(&req); err != nil {
				break
			}
			fullURL, err = service.GetInstanceDashboard(ctx, grafanaDashboards.InstanceDashboardURL, &req)
		case ServiceDimension:
			var req model.ServiceDashboardReq
			if err = c.ShouldBindQuery(&req); err != nil {
				break
			}
			fullURL, err = service.GetServiceDashboard(grafanaDashboards.ServiceDashboardURL, &req)
		}

		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(model.DashboardResp{
			FullURL: fullURL,
		}))
	}
}

func GetPrometheus(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ctx.Config().Observability.PrometheusBaseURL == nil {
			c.JSON(http.StatusOK, model.NewSuccessResp(nil))
			return
		}
		resp := ctx.Config().Observability.PrometheusBaseURL.String()
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
		return
	}
}

func GetMetricsList(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.MetricsReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusOK, model.NewErrorResp(err.Error()))
			return
		}
		resp, err := service.GetInstanceMetrics(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}
