package engine

import (
	"dubbo-admin-ai/component/agent/react"
	"dubbo-admin-ai/component/server/engine/session"

	"github.com/gin-gonic/gin"
)

type Router struct {
	engine     *gin.Engine
	handler    *AgentHandler
	sessionMgr *session.Manager
}

func NewRouter(agent *react.ReActAgent) *Router {
	sessionMgr := session.NewManager()
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
