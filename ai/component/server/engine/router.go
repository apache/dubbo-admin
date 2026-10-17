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

package engine

import (
	"fmt"
	"net/http"
	"slices"

	"dubbo-admin-ai/component/agent"
	"dubbo-admin-ai/component/server/engine/session"
	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"

	"github.com/gin-gonic/gin"
)

type Router struct {
	engine        *gin.Engine
	handler       *AgentHandler
	sessionMgr    *session.Manager
	authenticator security.Authenticator
}

func NewRouter(agent agent.Agent, store conversationstore.Store, maintenance conversationstore.MaintenanceStore, authenticator security.Authenticator) (*Router, error) {
	return NewRouterWithCORS(agent, store, maintenance, authenticator, nil)
}

func NewRouterWithCORS(agent agent.Agent, store conversationstore.Store, maintenance conversationstore.MaintenanceStore, authenticator security.Authenticator, corsOrigins []string) (*Router, error) {
	return NewRouterWithLifecycle(agent, store, maintenance, authenticator, corsOrigins, conversationstore.DefaultLifecyclePolicy())
}

func NewRouterWithLifecycle(agent agent.Agent, store conversationstore.Store, maintenance conversationstore.MaintenanceStore, authenticator security.Authenticator, corsOrigins []string, lifecycle conversationstore.LifecyclePolicy) (*Router, error) {
	if authenticator == nil {
		return nil, fmt.Errorf("authenticator is required")
	}
	if store == nil {
		return nil, fmt.Errorf("conversation store is required")
	}
	if maintenance == nil {
		return nil, fmt.Errorf("maintenance store is required")
	}
	// The Memory component owns the cleanup worker. The HTTP manager only owns
	// request-facing session operations.
	sessionMgr := session.NewManagerWithLifecycle(store, nil, security.OwnershipPolicy{}, lifecycle)
	handler := NewAgentHandler(agent, sessionMgr, store)
	handler.SetInteractionLease(lifecycle.InteractionLease)

	router := &Router{
		engine:        gin.Default(),
		handler:       handler,
		sessionMgr:    sessionMgr,
		authenticator: authenticator,
	}

	router.setupRoutes(corsOrigins)
	return router, nil
}

func (r *Router) setupRoutes(corsOrigins []string) {
	// Add CORS middleware
	r.engine.Use(corsMiddleware(corsOrigins))

	// API v1 group
	v1 := r.engine.Group("/api/v1/ai")
	v1.Use(authenticationMiddleware(r.authenticator))
	{
		// Chat endpoints
		v1.POST("/chat/stream", r.handler.StreamChat) // Streaming chat

		// Session management
		v1.POST("/sessions", r.handler.CreateSession)              // Create session
		v1.GET("/sessions", r.handler.ListSessions)                // List sessions
		v1.GET("/sessions/:sessionId", r.handler.GetSession)       // Get session info
		v1.DELETE("/sessions/:sessionId", r.handler.DeleteSession) // Delete session
	}
}

func authenticationMiddleware(authenticator security.Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		access, err := authenticator.Authenticate(c.Request)
		if err != nil || access.Validate() != nil || access.IsSystem() {
			c.AbortWithStatusJSON(http.StatusUnauthorized, NewAPIErrorResponse("AUTHENTICATION_REQUIRED", "Authentication required"))
			return
		}
		c.Request = c.Request.WithContext(security.WithAccessContext(c.Request.Context(), access))
		c.Next()
	}
}

// GetEngine returns the Gin engine
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}

// Close stops the session manager cleanup loop without closing the shared
// Store, whose lifetime is owned by the Memory component.
func (r *Router) Close() error {
	if r == nil || r.sessionMgr == nil {
		return nil
	}
	return r.sessionMgr.Close()
}

// corsMiddleware provides CORS middleware
func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowedOrigins = append([]string(nil), allowedOrigins...)
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if slices.Contains(allowedOrigins, "*") {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" && slices.Contains(allowedOrigins, origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		if c.Writer.Header().Get("Access-Control-Allow-Origin") != "" {
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Idempotency-Key, traceparent, tracestate, baggage")
			c.Header("Access-Control-Expose-Headers", "X-Trace-ID, X-Interaction-ID")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
