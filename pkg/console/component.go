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
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/config/console"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/router"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	mcpcore "github.com/apache/dubbo-admin/pkg/mcp/core"
	mcphttp "github.com/apache/dubbo-admin/pkg/mcp/transport/http"
	mcp_tools "github.com/apache/dubbo-admin/pkg/mcp/tools"
)

func init() {
	runtime.RegisterComponent(&consoleWebServer{})
}

type consoleWebServer struct {
	Engine   *gin.Engine
	cfg      *console.Config
	cs       consolectx.Context
	mcpPath  string // MCP端点路径，用于auth中间件跳过认证
	mcpAPIKey string // MCP API密钥，用于认证
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

	// 提前读取 MCP 配置，供 auth 中间件使用
	cfg := ctx.Config()
	if cfg.MCP != nil {
		if cfg.MCP.Path != "" {
			c.mcpPath = cfg.MCP.Path
		}
		c.mcpAPIKey = cfg.MCP.APIKey
	}

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
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("session", store))
	r.Use(c.authMiddleware())
	r.Use(ginzap.Ginzap(logger.Logger(), time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(logger.Logger(), true))
	c.Engine = r
	gin.SetMode(string(c.cfg.GinMode))
	return nil
}

func (c *consoleWebServer) Start(coreRt runtime.Runtime, stop <-chan struct{}) error {
	// If console config is nil, skip starting (e.g., MCP mode)
	if c.cfg == nil {
		logger.Sugar().Info("Console config is nil, skipping console start")
		// Wait for stop signal since we need to keep the component "running"
		<-stop
		return nil
	}
	errChan := make(chan error)
	c.cs = consolectx.NewConsoleContext(coreRt)
	router.InitRouter(c.Engine, c.cs)

	// 注册MCP端点（如果启用）
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
	server := mcpcore.NewServer("dubbo-admin", "1.0.0")
	server.SetConsoleContext(consoleCtx)

	// 注册所有工具
	reg := server.GetRegistry()
	reg.RegisterRegistrar(&mcp_tools.MetricsRegistrar{})
	reg.RegisterRegistrar(&mcp_tools.ResourceSearchRegistrar{})
	reg.RegisterRegistrar(&mcp_tools.ServiceRegistrar{})
	reg.RegisterRegistrar(&mcp_tools.DetailRegistrar{})
	reg.RegisterAll()

	// 创建HTTP处理器
	handler := mcphttp.NewHandler(server)

	// 注册路由
	engine.POST(path, func(ctx *gin.Context) {
		handler.ServeHTTP(ctx.Writer, ctx.Request)
	})

	authStatus := "no-auth"
	if c.mcpAPIKey != "" {
		authStatus = "with-auth"
	}
	logger.Sugar().Infof("MCP endpoint registered at %s with %d tools (%s)", path, len(reg.List()), authStatus)
}

func (c *consoleWebServer) authMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestPath := ctx.Request.URL.Path

		// skip login api
		if strings.HasSuffix(requestPath, "/login") {
			ctx.Next()
			return
		}

		// check MCP endpoint authentication
		isMCPRequest := requestPath == "/api/mcp" || (c.mcpPath != "" && requestPath == c.mcpPath)
		if isMCPRequest {
			// 如果配置了 API Key，验证 Bearer Token
			if c.mcpAPIKey != "" {
				authHeader := ctx.GetHeader("Authorization")
				if authHeader == "" {
					ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization header"})
					ctx.Abort()
					return
				}
				// 检查 Bearer Token 格式
				if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
					ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format. Use: Bearer <token>"})
					ctx.Abort()
					return
				}
				token := authHeader[7:]
				if token != c.mcpAPIKey {
					ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
					ctx.Abort()
					return
				}
			}
			// API Key 验证通过或未配置 API Key，继续处理
			ctx.Next()
			return
		}

		// 其他 API 需要会话认证
		session := sessions.Default(ctx)
		user := session.Get("user")
		if user == nil {
			authErr := bizerror.New(bizerror.Unauthorized, "no access, please login")
			ctx.JSON(http.StatusUnauthorized, model.NewBizErrorResp(authErr))
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
