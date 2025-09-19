package server

import (
	"dubbo-admin-ai/server/sse"
	"fmt"
	"net/http"

	"dubbo-admin-ai/agent"
	"dubbo-admin-ai/internal/manager"
	"dubbo-admin-ai/schema"
	"dubbo-admin-ai/server/session"

	"github.com/firebase/genkit/go/ai"
	"github.com/gin-gonic/gin"
)

// AgentHandler AI Agent处理器
type AgentHandler struct {
	agent      agent.Agent
	sessionMgr *session.Manager
}

// NewAgentHandler 创建AI Agent处理器
func NewAgentHandler(agent agent.Agent, sessionMgr *session.Manager) *AgentHandler {
	return &AgentHandler{
		agent:      agent,
		sessionMgr: sessionMgr,
	}
}

// StreamChat 流式聊天接口
func (h *AgentHandler) StreamChat(c *gin.Context) {
	var (
		req          ChatRequest
		sessionID    string
		session      *session.Session
		sseHandler   *sse.SSEHandler
		streamWriter *sse.StreamWriter
		channels     *agent.Channels
		err          error
	)

	// 验证session存在并更新活动时间
	if sessionID = h.getOrCreateSession(c); sessionID != "" {
		session, err = h.sessionMgr.GetSession(sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, NewErrorResponse("invalid_session "+err.Error()))
			return
		}
		session.UpdateActivity()
	}

	if streamWriter, err = sse.NewStreamWriter(c); err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse("Failed to create stream writer: "+err.Error()))
		return
	}
	sseHandler = sse.NewStreamHandler(streamWriter, sessionID)

	// 设置响应头和错误恢复
	defer func() {
		if r := recover(); r != nil {
			sseHandler.HandleError("internal_error", fmt.Sprintf("internal error: %v", r))
		}
		c.Header("X-Session-ID", sessionID)
	}()

	// 与 Agent 交互
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse("Invalid request: "+err.Error()))
		return
	}

	channels = h.agent.Interact(schema.ThinkInput{Content: req.Message})
	for {
		select {
		// case streamChunk, ok := <-channels.StreamChunkChan:
		// 	if !ok {
		// 		channels.StreamChunkChan = nil
		// 		continue
		// 	}

		// 	if streamChunk != nil {
		// 		if err := sseHandler.HandleStreamChunk(*streamChunk); err != nil {
		// 			manager.GetLogger().Error("Failed to handle stream chunk", "error", err)
		// 		}
		// 	}

		case resp, ok := <-channels.UserRespChan:
			if !ok {
				channels.UserRespChan = nil
				continue
			}
			if err := sseHandler.HandleText(resp); err != nil {
				manager.GetLogger().Error("Failed to handle final answer", "error", err)
			}

		case <-c.Request.Context().Done():
			manager.GetLogger().Info("Client disconnected from stream")
			return

		default:
			if channels.Closed() {
				streamWriter.WriteMessageStop()
				manager.GetLogger().Info("Stream processing completed", "session_id", sessionID)
				c.Header("X-Session-ID", sessionID)
				return
			}
		}
	}
}

// getOrCreateSession 获取或创建会话
func (h *AgentHandler) getOrCreateSession(c *gin.Context) string {
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID != "" {
		if sessionObj, err := h.sessionMgr.GetSession(sessionID); err == nil {
			return sessionObj.ID
		}
	}

	sessionObj := h.sessionMgr.CreateSession()
	return sessionObj.ID
}

// finishStreamWithUsage 完成流并处理使用情况
func (h *AgentHandler) finishStreamWithUsage(sseHandler *sse.SSEHandler, output any) {
	stopReason := "end_turn"
	var usage *ai.GenerationUsage

	if thinkOutput, ok := output.(schema.ThinkOutput); ok && thinkOutput.Usage != nil {
		usage = thinkOutput.Usage
	}

	if err := sseHandler.FinishStream(stopReason, usage); err != nil {
		sseHandler.HandleError("finish_stream_error", fmt.Sprintf("failed to finish stream: %v", err))
	}
}

func (h *AgentHandler) CreateSession(c *gin.Context) {
	sessionObj := h.sessionMgr.CreateSession()
	sessionInfo := sessionObj.ToSessionInfo()
	c.JSON(http.StatusOK, NewSuccessResponse(sessionInfo))
}

func (h *AgentHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse("Session ID is required"))
		return
	}

	sessionObj, err := h.sessionMgr.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, NewErrorResponse("Session not found: "+err.Error()))
		return
	}

	sessionInfo := sessionObj.ToSessionInfo()
	c.JSON(http.StatusOK, NewSuccessResponse(sessionInfo))
}

func (h *AgentHandler) ListSessions(c *gin.Context) {
	sessions := h.sessionMgr.ListSessions()

	response := map[string]any{
		"sessions": sessions,
		"total":    len(sessions),
	}

	c.JSON(http.StatusOK, NewSuccessResponse(response))
}

// DeleteSession 删除会话
func (h *AgentHandler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse("Session ID is required"))
		return
	}

	err := h.sessionMgr.DeleteSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, NewErrorResponse("Session not found: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, NewSuccessResponse(map[string]string{
		"message": "Session deleted successfully",
	}))
}
