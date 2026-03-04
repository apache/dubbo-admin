package engine

import (
	"fmt"
	"net/http"

	"dubbo-admin-ai/component/server/engine/session"
	"dubbo-admin-ai/component/server/engine/sse"
	rt "dubbo-admin-ai/runtime"

	"dubbo-admin-ai/component/agent"
	"dubbo-admin-ai/schema"

	"github.com/gin-gonic/gin"
)

// AgentHandler handles AI Agent requests
type AgentHandler struct {
	agent      agent.Agent
	sessionMgr *session.Manager
}

// NewAgentHandler creates an AI Agent handler
func NewAgentHandler(agent agent.Agent, sessionMgr *session.Manager) *AgentHandler {
	sessionMgr.CreateMockSession()
	return &AgentHandler{
		agent:      agent,
		sessionMgr: sessionMgr,
	}
}

// StreamChat handles streaming chat endpoint
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

	// Parse request
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse("Invalid request: "+err.Error()))
		return
	}

	sessionID = req.SessionID
	// Validate session exists and update activity time
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

	// Set response headers and error recovery
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
				rt.GetLogger().Error("Agent interaction error", "session_id", sessionID, "error", err)
				channels.Close()
				return
			}
		case feedback, ok = <-channels.UserRespChan:
			if !ok {
				channels.UserRespChan = nil
				continue
			}
			if feedback.IsFinal() {
				h.MessageDelta(sseHandler, feedback.Final())
			} else if feedback.IsDone() {
				if err := sseHandler.HandleContentBlockStop(feedback.Index()); err != nil {
					rt.GetLogger().Error("Failed to handle content block stop", "error", err)
				}
			} else {
				if err := sseHandler.HandleText(feedback.Text(), feedback.Index()); err != nil {
					rt.GetLogger().Error("Failed to handle text", "error", err)
				}
			}

		case <-c.Request.Context().Done():
			rt.GetLogger().Info("Client disconnected from stream")
			return

		default:
			if channels.Closed() {
				if err := sseHandler.FinishStream(); err != nil {
					rt.GetLogger().Error("Failed to finish stream", "error", err)
				}
				rt.GetLogger().Info("Stream processing completed", "session_id", sessionID)
				return
			}
		}
	}
}

// MessageDelta finishes stream and handles usage
func (h *AgentHandler) MessageDelta(sseHandler *sse.SSEHandler, output schema.Schema) {
	stopReason := "end_turn"
	if err := sseHandler.MessageDeltaWithUsage(stopReason, output); err != nil {
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

// DeleteSession deletes a session
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

	// Delete corresponding history
	if agentMemory := h.agent.GetMemory(); agentMemory != nil {
		agentMemory.Clear(sessionID)
		rt.GetLogger().Info("Session history cleared", "session_id", sessionID)
	}

	c.JSON(http.StatusOK, NewSuccessResponse(map[string]string{
		"message": "Session deleted successfully",
	}))
}
