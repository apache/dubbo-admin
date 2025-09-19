package handlers

import (
	"fmt"
	"net/http"
	"time"

	"dubbo-admin-ai/agent"
	"dubbo-admin-ai/server/session"

	"dubbo-admin-ai/internal/manager"
	"dubbo-admin-ai/schema"

	"github.com/gin-gonic/gin"
)

// 简化的数据类型
type Response struct {
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"request_id"`
	Timestamp int64  `json:"timestamp"`
}

type ChatRequest struct {
	Message string `json:"message" binding:"required"`
}

func NewResponse(message string, data any) *Response {
	return &Response{
		Message:   message,
		Data:      data,
		RequestID: generateRequestID(),
		Timestamp: time.Now().Unix(),
	}
}

func NewSuccessResponse(data any) *Response {
	return NewResponse("success", data)
}

func NewErrorResponse(message string) *Response {
	return NewResponse(message, nil)
}

func generateRequestID() string {
	return "req_" + time.Now().Format("20060102150405") + "_" + randomString(6)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

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
	sessionMgr := h.sessionMgr
	agent := h.agent

	// 解析请求
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse("Invalid request: "+err.Error()))
		return
	}

	// 从请求头获取session_id
	sessionID := c.GetHeader("X-Session-ID")
	var sessionObj *session.Session
	var err error

	if sessionID != "" {
		// 获取现有会话
		sessionObj, err = sessionMgr.GetSession(sessionID)
		if err != nil {
			// 如果session不存在或过期，创建新session
			sessionObj = sessionMgr.CreateSession()
			sessionID = sessionObj.ID
		}
	} else {
		// 创建新会话
		sessionObj = sessionMgr.CreateSession()
		sessionID = sessionObj.ID
	}

	// 创建SSE流写入器
	streamWriter, err := NewStreamWriter(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse("Failed to create stream writer: "+err.Error()))
		return
	}

	// 构建AI输入
	thinkInput, err := sessionMgr.BuildThinkInput(sessionID, req.Message)
	if err != nil {
		streamWriter.WriteError(fmt.Errorf("failed to build input: %w", err))
		return
	}

	// 发送开始事件
	if err := streamWriter.WriteEvent("start", map[string]string{
		"session_id": sessionID,
		"status":     "processing",
	}); err != nil {
		manager.GetLogger().Error("Failed to write start event", "error", err)
		return
	}

	// 调用AI Agent进行流式交互
	streamChan, outputChan, err := agent.Interact(thinkInput)
	if err != nil {
		streamWriter.WriteError(fmt.Errorf("failed to interact with agent: %w", err))
		return
	}

	// 创建流式处理器
	streamHandler := NewStreamHandler(streamWriter, sessionID)

	defer func() {
		if r := recover(); r != nil {
			manager.GetLogger().Error("Stream handler panic", "error", r)
			streamWriter.WriteError(fmt.Errorf("internal error: %v", r))
		}
		// 在响应完成后设置session_id到响应头
		c.Header("X-Session-ID", sessionID)
	}()

	for {
		select {
		case streamChunk, ok := <-streamChan:
			if !ok {
				// 流通道关闭，设置为 nil 避免重复读取
				streamChan = nil
				continue
			}
			if streamChunk != nil {
				// 处理流式数据块
				if err := streamHandler.HandleStreamChunk(*streamChunk); err != nil {
					manager.GetLogger().Error("Failed to handle stream chunk", "error", err)
				}
			}

		case output, ok := <-outputChan:
			if !ok {
				outputChan = nil
				continue
			}
			if output != nil {
				var msg any
				var isFinished bool

				if thinkOutput, ok := output.(schema.ThinkOutput); ok {
					if thinkOutput.Status == schema.Finished && thinkOutput.FinalAnswer != "" {
						msg = thinkOutput
						isFinished = true
					} else {
						msg = thinkOutput.Thought
					}
				}

				if err := streamWriter.WriteMessage(sessionID, msg, isFinished); err != nil {
					manager.GetLogger().Error("Failed to write message", "error", err)
				}
			}

		case <-c.Request.Context().Done():
			manager.GetLogger().Info("Client disconnected from stream")
			return

		default:
			if streamChan == nil && outputChan == nil {
				streamWriter.WriteDone()
				manager.GetLogger().Info("Stream processing completed", "session_id", sessionID)
				// 在流式响应完成后设置session_id到响应头
				c.Header("X-Session-ID", sessionID)
				return
			}
		}
	}
}

// CreateSession 创建会话
func (h *AgentHandler) CreateSession(c *gin.Context) {
	sessionObj := h.sessionMgr.CreateSession()
	sessionInfo := sessionObj.ToSessionInfo()
	c.JSON(http.StatusOK, NewSuccessResponse(sessionInfo))
}

// GetSession 获取会话信息
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

// ListSessions 列出所有会话
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

// Health 健康检查
func (h *AgentHandler) Health(c *gin.Context) {
	// 检查各个组件的健康状态
	details := make(map[string]string)
	isHealthy := true

	// 检查AI Agent
	if h.agent == nil {
		details["agent"] = "unavailable"
		isHealthy = false
	} else {
		details["agent"] = "running"
	}

	// 检查会话管理器
	if h.sessionMgr == nil {
		details["session_manager"] = "unavailable"
		isHealthy = false
	} else {
		details["session_manager"] = "running"
	}

	// 检查logger
	if manager.GetLogger() == nil {
		details["logger"] = "unavailable"
		isHealthy = false
	} else {
		details["logger"] = "running"
	}

	status := "healthy"
	statusCode := http.StatusOK
	if !isHealthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	response := map[string]any{
		"status":    status,
		"timestamp": time.Now().Unix(),
		"details":   details,
	}

	c.JSON(statusCode, NewSuccessResponse(response))
}

// Stop 停止正在进行的AI推理
func (h *AgentHandler) Stop(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse("Session ID is required"))
		return
	}

	// 目前暂时返回成功，实际的停止逻辑需要Agent支持
	// TODO: 实现Agent中断机制
	c.JSON(http.StatusOK, NewSuccessResponse(map[string]string{
		"message": "Stop request received",
		"session": sessionID,
	}))
}
