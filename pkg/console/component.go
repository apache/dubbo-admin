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
	"github.com/apache/dubbo-admin/pkg/config/console"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/router"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

func init() {
	runtime.RegisterComponent(&consoleWebServer{})
}

type consoleWebServer struct {
	Engine *gin.Engine
	cfg    *console.Config
	cs     consolectx.Context
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
	r := gin.Default()
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
	c.cfg = ctx.Config().Console
	return nil
}

func (c *consoleWebServer) Start(coreRt runtime.Runtime, stop <-chan struct{}) error {
	errChan := make(chan error)
	c.cs = consolectx.NewConsoleContext(coreRt)
	router.InitRouter(c.Engine, c.cs)
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

func (c *consoleWebServer) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// skip login api
		requestPath := c.Request.URL.Path
		if strings.HasSuffix(requestPath, "/login") {
			c.Next()
			return
		}
		session := sessions.Default(c)
		user := session.Get("user")
		if user == nil {
			authErr := bizerror.New(bizerror.Unauthorized, "no access, please login")
			c.JSON(http.StatusUnauthorized, model.NewBizErrorResp(authErr))
			c.Abort()
			return
		}
		c.Next()
	}
}
