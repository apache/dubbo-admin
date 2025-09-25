package server

import (
	"fmt"
	"net/http"

	"dubbo-admin-ai/manager"
	"dubbo-admin-ai/server/sse"

	"dubbo-admin-ai/agent"
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
	sessionMgr.CreateMockSession()
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

	// 解析请求
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse("Invalid request: "+err.Error()))
		return
	}

	sessionID = req.SessionID
	// 验证session存在并更新活动时间
	session, err = h.sessionMgr.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse("Invalid session ID: "+err.Error()))
		return
	}
	session.UpdateActivity()

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
	}()

	channels = h.agent.Interact(&schema.UserInput{Content: req.Message}, sessionID)
	var (
		feedback *schema.StreamFeedback
		ok       bool
	)
	for {
		select {
		case err, ok = <-channels.ErrorChan:
			if !ok {
				channels.ErrorChan = nil
				continue
			}
			if err != nil {
				sseHandler.HandleError("agent_error", fmt.Sprintf("agent error: %v", err))
				manager.GetLogger().Error("Agent interaction error", "session_id", sessionID, "error", err)
				channels.Close()
				return
			}
		case feedback, ok = <-channels.UserRespChan:
			if !ok {
				channels.UserRespChan = nil
				continue
			}
			if feedback.IsDone() {
				if err := sseHandler.HandleContentBlockStop(feedback.Index()); err != nil {
					manager.GetLogger().Error("Failed to handle content block stop", "error", err)
				}
			} else {
				if err := sseHandler.HandleText(feedback.Text, feedback.Index()); err != nil {
					manager.GetLogger().Error("Failed to handle text", "error", err)
				}
			}

		case <-c.Request.Context().Done():
			manager.GetLogger().Info("Client disconnected from stream")
			return

		default:
			if channels.Closed() {
				h.finishStreamWithUsage(sseHandler, feedback, feedback.Index())
				manager.GetLogger().Info("Stream processing completed", "session_id", sessionID)
				return
			}
		}
	}
}

// finishStreamWithUsage 完成流并处理使用情况
func (h *AgentHandler) finishStreamWithUsage(sseHandler *sse.SSEHandler, output any, index int) {
	stopReason := "end_turn"
	var usage *ai.GenerationUsage

	if thinkOutput, ok := output.(schema.ThinkOutput); ok && thinkOutput.Usage != nil {
		usage = thinkOutput.Usage
	}

	if err := sseHandler.FinishStream(stopReason, usage, index); err != nil {
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

	// 删除对应的 history
	if agentMemory := h.agent.GetMemory(); agentMemory != nil {
		agentMemory.Clear(sessionID)
		manager.GetLogger().Info("Session history cleared", "session_id", sessionID)
	}

	c.JSON(http.StatusOK, NewSuccessResponse(map[string]string{
		"message": "Session deleted successfully",
	}))
}
