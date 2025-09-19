package server

import (
	"dubbo-admin-ai/agent/react"
	"dubbo-admin-ai/server/session"

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
	// 添加CORS中间件
	r.engine.Use(corsMiddleware())

	// API v1 组
	v1 := r.engine.Group("/api/v1/ai")
	{
		// 聊天相关
		v1.POST("/chat/stream", r.handler.StreamChat) // 流式聊天

		// 会话管理
		v1.POST("/sessions", r.handler.CreateSession)              // 创建会话
		v1.GET("/sessions", r.handler.ListSessions)                // 列出会话
		v1.GET("/sessions/:sessionId", r.handler.GetSession)       // 获取会话信息
		v1.DELETE("/sessions/:sessionId", r.handler.DeleteSession) // 删除会话
	}
}

// GetEngine 获取Gin引擎
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}

// corsMiddleware CORS中间件
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
