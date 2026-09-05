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
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

func InitRouter(r *gin.Engine, ctx consolectx.Context) {
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
		instance.GET("/event", handler.GetInstanceEvents(ctx))
		instance.GET("/metric-dashboard", handler.GetGrafanaDashboard(ctx, handler.InstanceDimension, handler.MetricDashboard))
		instance.GET("/trace-dashboard", handler.GetGrafanaDashboard(ctx, handler.InstanceDimension, handler.TraceDashboard))
		instance.GET("/metrics-list", handler.GetMetricsList(ctx))
	}

	{
		application := router.Group("/application")
		application.GET("/detail", handler.GetApplicationDetail(ctx))
		application.GET("/instance/info", handler.GetApplicationTabInstanceInfo(ctx))
		application.GET("/service/form", handler.GetApplicationServiceForm(ctx))
		application.GET("/search", handler.ApplicationSearch(ctx))
		application.GET("/graph", handler.GetApplicationGraph(ctx))
		{
			applicationConfig := application.Group("/config")
			applicationConfig.PUT("/operatorLog", handler.ApplicationConfigAccessLogPut(ctx))
			applicationConfig.GET("/operatorLog", handler.ApplicationConfigAccessLogGet(ctx))

			applicationConfig.GET("/flowWeight", handler.ApplicationConfigFlowWeightGET(ctx))
			applicationConfig.PUT("/flowWeight", handler.ApplicationConfigFlowWeightPUT(ctx))

			applicationConfig.GET("/gray", handler.ApplicationConfigGrayGET(ctx))
			applicationConfig.PUT("/gray", handler.ApplicationConfigGrayPUT(ctx))
		}
		application.GET("/event", handler.GetApplicationEvents(ctx))
		application.GET("/metric-dashboard", handler.GetGrafanaDashboard(ctx, handler.AppDimension, handler.MetricDashboard))
		application.GET("/trace-dashboard", handler.GetGrafanaDashboard(ctx, handler.AppDimension, handler.TraceDashboard))
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
		service.GET("/metric-dashboard", handler.GetGrafanaDashboard(ctx, handler.ServiceDimension, handler.MetricDashboard))
		service.GET("/trace-dashboard", handler.GetGrafanaDashboard(ctx, handler.ServiceDimension, handler.TraceDashboard))
	}

	{
		service := router.Group("/service")
		service.POST("/generic/invoke", handler.ServiceGenericInvoke(ctx))
		service.GET("/method/detail", handler.GetServiceMethodDetail(ctx))
		service.GET("/distribution", handler.GetServiceTabDistribution(ctx))
		service.GET("/provider-instances", handler.GetServiceProviderInstances(ctx))
		service.GET("/methods", handler.GetServiceMethodNames(ctx))
		service.GET("/search", handler.SearchServices(ctx))
		service.GET("/graph", handler.GetServiceGraph(ctx))
		service.GET("/detail", handler.GetServiceDetail(ctx))
		service.GET("/event", handler.GetServiceEvents(ctx))
		service.GET("/interfaces", handler.GetServiceInterfaces(ctx))
	}

	{
		configuration := router.Group("/configurator")
		configuration.GET("/search", handler.ConfiguratorSearch(ctx))
		configuration.GET("/:ruleName/versions", handler.ListRuleVersions(ctx, meshresource.DynamicConfigKind))
		configuration.GET("/:ruleName/versions/:versionNo", handler.GetRuleVersion(ctx, meshresource.DynamicConfigKind))
		configuration.GET("/:ruleName/versions/:versionNo/diff", handler.DiffRuleVersion(ctx, meshresource.DynamicConfigKind))
		configuration.POST("/:ruleName/versions/:versionNo/rollback", handler.RollbackRuleVersion(ctx, meshresource.DynamicConfigKind))
		configuration.GET("/:ruleName", handler.GetConfiguratorWithRuleName(ctx))
		configuration.PUT("/:ruleName", handler.PutConfiguratorWithRuleName(ctx))
		configuration.POST("/:ruleName", handler.PostConfiguratorWithRuleName(ctx))
		configuration.DELETE("/:ruleName", handler.DeleteConfiguratorWithRuleName(ctx))
	}

	{
		conditionRule := router.Group("/condition-rule")
		conditionRule.GET("/search", handler.ConditionRuleSearch(ctx))
		conditionRule.GET("/:ruleName/versions", handler.ListRuleVersions(ctx, meshresource.ConditionRouteKind))
		conditionRule.GET("/:ruleName/versions/:versionNo", handler.GetRuleVersion(ctx, meshresource.ConditionRouteKind))
		conditionRule.GET("/:ruleName/versions/:versionNo/diff", handler.DiffRuleVersion(ctx, meshresource.ConditionRouteKind))
		conditionRule.POST("/:ruleName/versions/:versionNo/rollback", handler.RollbackRuleVersion(ctx, meshresource.ConditionRouteKind))
		conditionRule.GET("/:ruleName", handler.GetConditionRuleWithRuleName(ctx))
		conditionRule.PUT("/:ruleName", handler.PutConditionRuleWithRuleName(ctx))
		conditionRule.POST("/:ruleName", handler.PostConditionRuleWithRuleName(ctx))
		conditionRule.DELETE("/:ruleName", handler.DeleteConditionRuleWithRuleName(ctx))
	}

	{
		tagRule := router.Group("/tag-rule")
		tagRule.GET("/search", handler.TagRuleSearch(ctx))
		tagRule.GET("/:ruleName/versions", handler.ListRuleVersions(ctx, meshresource.TagRouteKind))
		tagRule.GET("/:ruleName/versions/:versionNo", handler.GetRuleVersion(ctx, meshresource.TagRouteKind))
		tagRule.GET("/:ruleName/versions/:versionNo/diff", handler.DiffRuleVersion(ctx, meshresource.TagRouteKind))
		tagRule.POST("/:ruleName/versions/:versionNo/rollback", handler.RollbackRuleVersion(ctx, meshresource.TagRouteKind))
		tagRule.GET("/:ruleName", handler.GetTagRuleWithRuleName(ctx))
		tagRule.PUT("/:ruleName", handler.PutTagRuleWithRuleName(ctx))
		tagRule.POST("/:ruleName", handler.PostTagRuleWithRuleName(ctx))
		tagRule.DELETE("/:ruleName", handler.DeleteTagRuleWithRuleName(ctx))
	}

	{
		affinityRule := router.Group("/affinity-rule")
		affinityRule.GET("/search", handler.AffinityRuleSearch(ctx))
		affinityRule.GET("/:ruleName/versions", handler.ListRuleVersions(ctx, meshresource.AffinityRouteKind))
		affinityRule.GET("/:ruleName/versions/:versionNo", handler.GetRuleVersion(ctx, meshresource.AffinityRouteKind))
		affinityRule.GET("/:ruleName/versions/:versionNo/diff", handler.DiffRuleVersion(ctx, meshresource.AffinityRouteKind))
		affinityRule.POST("/:ruleName/versions/:versionNo/rollback", handler.RollbackRuleVersion(ctx, meshresource.AffinityRouteKind))
		affinityRule.GET("/:ruleName", handler.GetAffinityRuleWithRuleName(ctx))
		affinityRule.PUT("/:ruleName", handler.PutAffinityRuleWithRuleName(ctx))
		affinityRule.POST("/:ruleName", handler.PostAffinityRuleWithRuleName(ctx))
		affinityRule.DELETE("/:ruleName", handler.DeleteAffinityRuleWithRuleName(ctx))
	}

	{
		scriptRule := router.Group("/script-rule")
		scriptRule.GET("/search", handler.ScriptRuleSearch(ctx))
		scriptRule.GET("/:ruleName/versions", handler.ListRuleVersions(ctx, meshresource.ScriptRouteKind))
		scriptRule.GET("/:ruleName/versions/:versionNo", handler.GetRuleVersion(ctx, meshresource.ScriptRouteKind))
		scriptRule.GET("/:ruleName/versions/:versionNo/diff", handler.DiffRuleVersion(ctx, meshresource.ScriptRouteKind))
		scriptRule.POST("/:ruleName/versions/:versionNo/rollback", handler.RollbackRuleVersion(ctx, meshresource.ScriptRouteKind))
		scriptRule.GET("/:ruleName", handler.GetScriptRuleWithRuleName(ctx))
		scriptRule.PUT("/:ruleName", handler.PutScriptRuleWithRuleName(ctx))
		scriptRule.POST("/:ruleName", handler.PostScriptRuleWithRuleName(ctx))
		scriptRule.DELETE("/:ruleName", handler.DeleteScriptRuleWithRuleName(ctx))
	}

	router.GET("/prometheus", handler.GetPrometheus(ctx))
	router.GET("/search", handler.BannerGlobalSearch(ctx))
	router.GET("/overview", handler.ClusterOverview(ctx))
	router.GET("/metadata", handler.AdminMetadata(ctx))
	router.GET("/meshes", handler.ListMeshes(ctx))
}
