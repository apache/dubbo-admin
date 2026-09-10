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
	"dubbo-admin-ai/component/agent"
	"dubbo-admin-ai/component/server/engine/session"
	conversationstore "dubbo-admin-ai/store"

	"github.com/gin-gonic/gin"
)

type Router struct {
	engine     *gin.Engine
	handler    *AgentHandler
	sessionMgr *session.Manager
}

func NewRouter(agent agent.Agent, sessionStore conversationstore.SessionStore) *Router {
	sessionMgr := session.NewManager(sessionStore)
	handler := NewAgentHandler(agent, sessionMgr)

	router := &Router{
		engine:     gin.Default(),
		handler:    handler,
		sessionMgr: sessionMgr,
	}

	router.setupRoutes()
	return router
}

func (r *Router) setupRoutes() {
	// Add CORS middleware
	r.engine.Use(corsMiddleware())

	// API v1 group
	v1 := r.engine.Group("/api/v1/ai")
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
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
