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
	"context"
	"net/http"
	"strconv"

	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/apache/dubbo-admin/app/dubbo-ui"
	"github.com/apache/dubbo-admin/pkg/admin/handlers"
	"github.com/apache/dubbo-admin/pkg/admin/handlers/traffic"
	"github.com/gin-gonic/gin"
)

type Router struct {
	Engine *gin.Engine
}

// TODO maybe tls?
func (r *Router) Start(stop <-chan struct{}) error {
	errChan := make(chan error)

	var httpServer *http.Server
	httpServer = r.startHttpServer(errChan)
	select {
	case <-stop:
		logger.Sugar().Info("stopping admin")
		if httpServer != nil {
			return httpServer.Shutdown(context.Background())
		}
	case err := <-errChan:
		return err
	}
	return nil
}

func (r *Router) startHttpServer(errChan chan error) *http.Server {
	server := &http.Server{
		Addr:    ":" + strconv.Itoa(config.AdminPort),
		Handler: r.Engine,
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			switch err {
			case http.ErrServerClosed:
				logger.Sugar().Info("shutting down HTTP Server")
			default:
				logger.Sugar().Error(err, "could not start an HTTP Server")
				errChan <- err
			}
		}
	}()
	return server
}

func (r *Router) NeedLeaderElection() bool {
	return false
}

func InitRouter() *Router {
	router := gin.Default()

	server := router.Group("/api/:env")
	{
		server.GET("/services", handlers.AllServices)
		server.GET("/service", handlers.SearchService)
		server.GET("/applications", handlers.AllApplications)
		server.GET("/consumers", handlers.AllConsumers)
		server.GET("/service/:service", handlers.ServiceDetail)
	}

	router.GET("/api/:env/version", handlers.Version)
	router.GET("/api/:env/metrics/flow", handlers.FlowMetrics)
	router.GET("/api/:env/metrics/cluster", handlers.ClusterMetrics)
	router.GET("/api/:env/metrics/metadata", handlers.Metadata)
	router.GET("/api/:env/metrics/prometheus", handlers.PromDiscovery)

	override := router.Group("/api/:env/rules/override")
	{
		override.POST("/", handlers.CreateOverride)
		override.GET("/", handlers.SearchOverride)
		override.DELETE("/:id", handlers.DeleteOverride)
		override.GET("/:id", handlers.DetailOverride)
		override.PUT("/enable/:id", handlers.EnableOverride)
		override.PUT("/disable/:id", handlers.DisableOverride)
		override.PUT("/:id", handlers.UpdateOverride)
	}

	tagRoute := router.Group("/api/:env/rules/route/tag")
	{
		tagRoute.POST("/", handlers.CreateRule)
		tagRoute.PUT("/:id", handlers.UpdateRule)
		tagRoute.GET("/", handlers.SearchRoutes)
		tagRoute.GET("/:id", handlers.DetailRoute)
		tagRoute.DELETE("/:id", handlers.DeleteRoute)
		tagRoute.PUT("/enable/:id", handlers.EnableRoute)
		tagRoute.PUT("/disable/:id", handlers.DisableRoute)
	}

	conditionRoute := router.Group("/api/:env/rules/route/condition")
	{
		conditionRoute.POST("/", handlers.CreateConditionRule)
		conditionRoute.PUT("/:id", handlers.UpdateConditionRule)
		conditionRoute.GET("/", handlers.SearchConditionRoutes)
		conditionRoute.GET("/:id", handlers.DetailConditionRoute)
		conditionRoute.DELETE("/:id", handlers.DeleteConditionRoute)
		conditionRoute.PUT("/enable/:id", handlers.EnableConditionRoute)
		conditionRoute.PUT("/disable/:id", handlers.DisableConditionRoute)
	}

	mockRoute := router.Group("/api/:env/mock/rule")
	{
		mockRoute.POST("/", handlers.CreateOrUpdateMockRule)
		mockRoute.DELETE("/", handlers.DeleteMockRuleById)
		mockRoute.GET("/list", handlers.ListMockRulesByPage)
	}

	trafficTimeout := router.Group("/api/:env/traffic/timeout")
	{
		trafficTimeout.POST("/", traffic.CreateTimeout)
		trafficTimeout.PUT("/", traffic.UpdateTimeout)
		trafficTimeout.DELETE("/", traffic.DeleteTimeout)
		trafficTimeout.GET("/", traffic.SearchTimeout)
	}

	trafficRetry := router.Group("/api/:env/traffic/retry")
	{
		trafficRetry.POST("/", traffic.CreateRetry)
		trafficRetry.PUT("/", traffic.UpdateRetry)
		trafficRetry.DELETE("/", traffic.DeleteRetry)
		trafficRetry.GET("/", traffic.SearchRetry)
	}

	trafficAccesslog := router.Group("/api/:env/traffic/accesslog")
	{
		trafficAccesslog.POST("/", traffic.CreateAccesslog)
		trafficAccesslog.PUT("/", traffic.UpdateAccesslog)
		trafficAccesslog.DELETE("/", traffic.DeleteAccesslog)
		trafficAccesslog.GET("/", traffic.SearchAccesslog)
	}

	trafficMock := router.Group("/api/:env/traffic/mock")
	{
		trafficMock.POST("/", traffic.CreateMock)
		trafficMock.PUT("/", traffic.UpdateMock)
		trafficMock.DELETE("/", traffic.DeleteMock)
		trafficMock.GET("/", traffic.SearchMock)
	}

	trafficWeight := router.Group("/api/:env/traffic/weight")
	{
		trafficWeight.POST("/", traffic.CreateWeight)
		trafficWeight.PUT("/", traffic.UpdateWeight)
		trafficWeight.DELETE("/", traffic.DeleteWeight)
		trafficWeight.GET("/", traffic.SearchWeight)
	}

	trafficArgument := router.Group("/api/:env/traffic/argument")
	{
		trafficArgument.POST("/", traffic.CreateArgument)
		trafficArgument.PUT("/", traffic.UpdateArgument)
		trafficArgument.DELETE("/", traffic.DeleteArgument)
		trafficArgument.GET("/", traffic.SearchArgument)
	}

	trafficGray := router.Group("/api/:env/traffic/gray")
	{
		trafficGray.POST("/", traffic.CreateGray)
		trafficGray.PUT("/", traffic.UpdateGray)
		trafficGray.DELETE("/", traffic.DeleteGray)
		trafficGray.GET("/", traffic.SearchGray)
	}

	trafficRegion := router.Group("/api/:env/traffic/region")
	{
		trafficRegion.POST("/", traffic.CreateRegion)
		trafficRegion.PUT("/", traffic.UpdateRegion)
		trafficRegion.DELETE("/", traffic.DeleteRegion)
		trafficRegion.GET("/", traffic.SearchRegion)
	}

	// ServiceTest
	router.POST("/api/:env/test", handlers.Test)
	router.GET("/api/:env/test/method", handlers.MethodDetail)

	// Admin UI
	router.StaticFS("/admin", http.FS(ui.FS()))

	return &Router{
		Engine: router,
	}
}
