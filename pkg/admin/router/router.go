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
	"github.com/apache/dubbo-admin/pkg/admin/handlers"
	"github.com/apache/dubbo-admin/pkg/admin/handlers/tag_routes"
	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	router := gin.Default()

	router.GET("/api/dev/services", handlers.AllServices)
	router.GET("/api/dev/service", handlers.SearchService)
	router.GET("api/dev/applications", handlers.AllApplications)
	router.GET("api/dev/consumers", handlers.AllConsumers)
	router.GET("api/dev/service/:service", handlers.ServiceDetail)
	router.GET("/api/dev/version", handlers.Version)
	router.GET("/api/dev/metrics/flow", handlers.FlowMetrics)
	router.GET("/api/dev/metrics/cluster", handlers.ClusterMetrics)

	override := router.Group("/api/:env/rules/override")
	{
		override.POST("/create", handlers.CreateOverride)
		override.GET("/", handlers.SearchOverride)
		override.DELETE("/:id", handlers.DeleteOverride)
		override.GET("/:id", handlers.DetailOverride)
		override.PUT("/enable/:id", handlers.EnableOverride)
		override.PUT("/disable/:id", handlers.DisableOverride)
		override.PUT("/:id", handlers.UpdateOverride)
	}

	router.POST("/api/dev/rules/route/tag", tag_routes.CreateRule)
	router.GET("/api/dev/rules/route/tag", tag_routes.SearchRoutes)

	return router
}
