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

package console

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"

	ui "github.com/apache/dubbo-admin/app/dubbo-ui"
	"github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/config/console"
	configauth "github.com/apache/dubbo-admin/pkg/config/console/auth"
	consoleauth "github.com/apache/dubbo-admin/pkg/console/auth"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/router"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/mcp"
)

func init() {
	runtime.RegisterComponent(&consoleWebServer{})
}

type consoleWebServer struct {
	mcpPath   string
	mcpAPIKey string
	Engine    *gin.Engine
	cfg       *console.Config
	cs        consolectx.Context
}

func (c *consoleWebServer) RequiredDependencies() []runtime.ComponentType {
	return []runtime.ComponentType{
		runtime.ResourceManager, // Console needs Manager for resource operations
		// Note: No need to list ResourceStore explicitly as Manager already depends on it
	}
}

func (c *consoleWebServer) Type() runtime.ComponentType {
	return runtime.Console
}

func (c *consoleWebServer) Order() int {
	return math.MaxInt - 5
}

func (c *consoleWebServer) Init(ctx runtime.BuilderContext) error {
	c.cfg = ctx.Config().Console
	r := gin.New()
	// Admin UI
	r.StaticFS("/admin", http.FS(ui.FS()))
	r.NoRoute(func(c *gin.Context) {
		if c.Request.URL.Path == "/admin" || strings.HasPrefix(c.Request.URL.Path, "/admin/") {
			c.FileFromFS("/", http.FS(ui.FS())) // Serve the index.html for SPA
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		}
	})
	r.Handle(http.MethodGet, "/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "UP",
		})
	})
	store := cookie.NewStore([]byte(c.cfg.Auth.SessionSecret))
	store.Options(adminSessionOptions(c.cfg.Auth))
	r.Use(sessions.Sessions("session", store))
	r.Use(consoleauth.SessionMiddleware())
	r.Use(ginzap.Ginzap(logger.Logger(), time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(logger.Logger(), true))
	c.Engine = r
	gin.SetMode(string(c.cfg.GinMode))
	return nil
}

func adminSessionOptions(cfg *configauth.Config) sessions.Options {
	return sessions.Options{
		Path:     "/",
		MaxAge:   cfg.ExpirationTime,
		Secure:   cfg.SessionCookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func (c *consoleWebServer) Start(coreRt runtime.Runtime, stop <-chan struct{}) error {
	errChan := make(chan error)
	c.cs = consolectx.NewConsoleContext(coreRt)
	if err := router.InitRouter(c.Engine, c.cs); err != nil {
		return err
	}
	c.registerMCPEndpoints(coreRt, c.Engine)
	httpServer := c.startHttpServer(errChan)
	select {
	case <-stop:
		logger.Sugar().Info("stopping console")
		if httpServer != nil {
			return httpServer.Shutdown(context.Background())
		}
	case err := <-errChan:
		return err
	}
	return nil
}

func (c *consoleWebServer) registerMCPEndpoints(coreRt runtime.Runtime, engine *gin.Engine) {
	// 从runtime获取完整配置
	var cfg app.AdminConfig = coreRt.Config()

	// 检查MCP是否启用
	if cfg.MCP == nil || !cfg.MCP.Enabled {
		return
	}

	// 确定端点路径
	path := cfg.MCP.Path
	if path == "" {
		path = "/api/mcp"
	}

	// 存储MCP路径和API Key供auth中间件使用
	c.mcpPath = path
	c.mcpAPIKey = cfg.MCP.APIKey

	// 直接创建MCP服务器
	consoleCtx := consolectx.NewConsoleContext(coreRt)
	server := mcp.NewServer("dubbo-admin", "1.0.0")
	server.SetConsoleContext(consoleCtx)

	// 注册所有工具
	mcp.RegisterTools(server)

	// 注册路由
	engine.POST(path, func(ctx *gin.Context) {
		if c.mcpAPIKey != "" && ctx.GetHeader("Authorization") != "Bearer "+c.mcpAPIKey {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			return
		}
		server.HandleHTTP(ctx)
	})

	authStatus := "no-auth"
	if c.mcpAPIKey != "" {
		authStatus = "with-auth"
	}
	logger.Sugar().Infof("MCP endpoint registered at %s (%s)", path, authStatus)
}

func (c *consoleWebServer) startHttpServer(errChan chan error) *http.Server {
	server := &http.Server{
		Addr:    ":" + strconv.Itoa(c.cfg.Port),
		Handler: c.Engine,
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			switch {
			case errors.Is(err, http.ErrServerClosed):
				logger.Sugar().Info("shutting down bufman HTTP Server")
			default:
				logger.Sugar().Error(err, "could not start bufman HTTP Server")
				errChan <- err
			}
		}
	}()

	return server
}
