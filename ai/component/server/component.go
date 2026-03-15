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

package server

import (
	"context"
	"dubbo-admin-ai/component/agent/react"
	"dubbo-admin-ai/component/server/engine"
	"dubbo-admin-ai/runtime"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ServerComponent Server component implementation
type ServerComponent struct {
	rt           *runtime.Runtime
	srv          *http.Server
	instanceName string

	port         int
	host         string
	debug        bool
	corsOrigins  []string
	readTimeout  int
	writeTimeout int
}

func NewServerComponent(
	port int,
	host string,
	debug bool,
	corsOrigins []string,
	timeouts ...int,
) (runtime.Component, error) {
	readTimeout := 30
	writeTimeout := 30
	if len(timeouts) > 0 {
		readTimeout = timeouts[0]
	}
	if len(timeouts) > 1 {
		writeTimeout = timeouts[1]
	}
	return &ServerComponent{
		port:         port,
		host:         host,
		debug:        debug,
		corsOrigins:  corsOrigins,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}, nil
}

// Name returns the component name
func (s *ServerComponent) Name() string {
	if s.instanceName != "" {
		return s.instanceName
	}
	return "server"
}

func (s *ServerComponent) SetName(name string) {
	s.instanceName = name
}

func (s *ServerComponent) Validate() error {
	if s.host == "" {
		return fmt.Errorf("host is required")
	}
	if s.port <= 0 || s.port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if s.readTimeout <= 0 {
		return fmt.Errorf("read_timeout must be greater than 0")
	}
	if s.writeTimeout <= 0 {
		return fmt.Errorf("write_timeout must be greater than 0")
	}
	return nil
}

func (s *ServerComponent) Init(rt *runtime.Runtime) error {
	s.rt = rt
	rt.GetLogger().Info("Server component initialized",
		"port", s.port,
		"host", s.host)

	return nil
}

func (s *ServerComponent) Start() error {
	// Retrieve Agent component from Runtime
	agentComp, err := s.rt.GetComponent("agent")
	if err != nil {
		s.rt.GetLogger().Error("Failed to get agent component", "error", err)
		return fmt.Errorf("failed to get agent component: %w", err)
	}

	agentComponent, ok := agentComp.(*react.AgentComponent)
	if !ok {
		s.rt.GetLogger().Error("Agent component is not an AgentComponent")
		return fmt.Errorf("agent component is not an AgentComponent")
	}

	// Configure Gin mode
	if !s.debug {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Create router with AI interface
	router := engine.NewRouter(agentComponent.Agent)

	// Add health check endpoint
	router.GetEngine().GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// Create HTTP server
	s.srv = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.host, s.port),
		Handler: router.GetEngine(),
	}

	s.rt.GetLogger().Info("Server starting",
		"addr", s.srv.Addr,
		"debug", s.debug)

	go func() {
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.rt.GetLogger().Error("Server failed", "error", err)
		}
	}()

	return nil
}

func (s *ServerComponent) Stop() error {
	if s.srv != nil {
		s.rt.GetLogger().Info("Server shutting down")

		// Gracefully shutdown the server using Shutdown instead of Close
		// Shutdown will:
		// 1. Stop accepting new connections
		// 2. Wait for existing requests to complete (up to 10 seconds)
		// 3. Close all connections
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Force close if timeout occurs
		if err := s.srv.Shutdown(ctx); err != nil {
			s.rt.GetLogger().Error("Server shutdown timeout, forcing close", "error", err)
			return s.srv.Close()
		}

		s.rt.GetLogger().Info("Server shutdown gracefully completed")
		return nil
	}
	return nil
}
