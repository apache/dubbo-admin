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

package router

import (
	"github.com/gin-gonic/gin"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/handler"
)

func InitRouter(r *gin.Engine, ctx consolectx.Context) {
	grafanaRouter := r.Group("/grafana")
	{
		grafanaRouter.Any("/*any", handler.Grafana(ctx))
	}

	router := r.Group("/api/v1")
	{
		prometheus := router.Group("/promQL")
		prometheus.GET("/query", handler.PromQL(ctx))
	}

	{
		auth := router.Group("/auth")
		auth.POST("/login", handler.Login(ctx))
		auth.POST("/logout", handler.Logout(ctx))
	}

	{
		instance := router.Group("/instance")
		instance.GET("/search", handler.SearchInstances(ctx))
		instance.GET("/detail", handler.GetInstanceDetail(ctx))
		{
			instanceConfig := instance.Group("/config")
			instanceConfig.GET("/trafficDisable", handler.InstanceConfigTrafficDisableGET(ctx))
			instanceConfig.PUT("/trafficDisable", handler.InstanceConfigTrafficDisablePUT(ctx))

			instanceConfig.GET("/operatorLog", handler.InstanceConfigOperatorLogGET(ctx))
			instanceConfig.PUT("/operatorLog", handler.InstanceConfigOperatorLogPUT(ctx))
		}
		instance.GET("/metric-dashboard", handler.GetMetricDashBoard(ctx, handler.InstanceDimension))
		instance.GET("/trace-dashboard", handler.GetTraceDashBoard(ctx, handler.InstanceDimension))
		instance.GET("/metrics-list", handler.GetMetricsList(ctx))
	}

	{
		application := router.Group("/application")
		application.GET("/detail", handler.GetApplicationDetail(ctx))
		application.GET("/instance/info", handler.GetApplicationTabInstanceInfo(ctx))
		application.GET("/service/form", handler.GetApplicationServiceForm(ctx))
		application.GET("/search", handler.ApplicationSearch(ctx))
		{
			applicationConfig := application.Group("/config")
			applicationConfig.PUT("/operatorLog", handler.ApplicationConfigOperatorLogPut(ctx))
			applicationConfig.GET("/operatorLog", handler.ApplicationConfigOperatorLogGet(ctx))

			applicationConfig.GET("/flowWeight", handler.ApplicationConfigFlowWeightGET(ctx))
			applicationConfig.PUT("/flowWeight", handler.ApplicationConfigFlowWeightPUT(ctx))

			applicationConfig.GET("/gray", handler.ApplicationConfigGrayGET(ctx))
			applicationConfig.PUT("/gray", handler.ApplicationConfigGrayPUT(ctx))
		}
		application.GET("/metric-dashboard", handler.GetMetricDashBoard(ctx, handler.AppDimension))
		application.GET("/trace-dashboard", handler.GetTraceDashBoard(ctx, handler.AppDimension))
	}

	{
		service := router.Group("/service")
		{
			serviceConfig := service.Group("/config")
			serviceConfig.GET("/timeout", handler.ServiceConfigTimeoutGET(ctx))
			serviceConfig.PUT("/timeout", handler.ServiceConfigTimeoutPUT(ctx))

			serviceConfig.GET("/regionPriority", handler.ServiceConfigRegionPriorityGET(ctx))
			serviceConfig.PUT("/regionPriority", handler.ServiceConfigRegionPriorityPUT(ctx))

			serviceConfig.GET("/retry", handler.ServiceConfigRetryGET(ctx))
			serviceConfig.PUT("/retry", handler.ServiceConfigRetryPUT(ctx))

			serviceConfig.GET("/argumentRoute", handler.ServiceConfigArgumentRouteGET(ctx))
			serviceConfig.PUT("/argumentRoute", handler.ServiceConfigArgumentRoutePUT(ctx))
		}
		service.GET("/metric-dashboard", handler.GetMetricDashBoard(ctx, handler.ServiceDimension))
		service.GET("/trace-dashboard", handler.GetTraceDashBoard(ctx, handler.ServiceDimension))
	}

	{
		service := router.Group("/service")
		service.GET("/distribution", handler.GetServiceTabDistribution(ctx))
		service.GET("/search", handler.SearchServices(ctx))
		service.GET("/detail", handler.GetServiceDetail(ctx))
		service.GET("/interfaces", handler.GetServiceInterfaces(ctx))
	}

	{
		configuration := router.Group("/configurator")
		configuration.GET("/search", handler.ConfiguratorSearch(ctx))
		configuration.GET("/:ruleName", handler.GetConfiguratorWithRuleName(ctx))
		configuration.PUT("/:ruleName", handler.PutConfiguratorWithRuleName(ctx))
		configuration.POST("/:ruleName", handler.PostConfiguratorWithRuleName(ctx))
		configuration.DELETE("/:ruleName", handler.DeleteConfiguratorWithRuleName(ctx))
	}

	{
		conditionRule := router.Group("/condition-rule")
		conditionRule.GET("/search", handler.ConditionRuleSearch(ctx))
		conditionRule.GET("/:ruleName", handler.GetConditionRuleWithRuleName(ctx))
		conditionRule.PUT("/:ruleName", handler.PutConditionRuleWithRuleName(ctx))
		conditionRule.POST("/:ruleName", handler.PostConditionRuleWithRuleName(ctx))
		conditionRule.DELETE("/:ruleName", handler.DeleteConditionRuleWithRuleName(ctx))
	}

	{
		tagRule := router.Group("/tag-rule")
		tagRule.GET("/search", handler.TagRuleSearch(ctx))
		tagRule.GET("/:ruleName", handler.GetTagRuleWithRuleName(ctx))
		tagRule.PUT("/:ruleName", handler.PutTagRuleWithRuleName(ctx))
		tagRule.POST("/:ruleName", handler.PostTagRuleWithRuleName(ctx))
		tagRule.DELETE("/:ruleName", handler.DeleteTagRuleWithRuleName(ctx))
	}

	router.GET("/prometheus", handler.GetPrometheus(ctx))
	router.GET("/search", handler.BannerGlobalSearch(ctx))
	router.GET("/overview", handler.ClusterOverview(ctx))
	router.GET("/metadata", handler.AdminMetadata(ctx))
}
