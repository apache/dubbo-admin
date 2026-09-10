package engine

import (
	"errors"
	"fmt"
	"net/http"

	"dubbo-admin-ai/component/server/engine/session"
	"dubbo-admin-ai/component/server/engine/sse"
	rt "dubbo-admin-ai/runtime"

	"dubbo-admin-ai/component/agent"
	"dubbo-admin-ai/schema"
	conversationstore "dubbo-admin-ai/store"

	"github.com/gin-gonic/gin"
)

// AgentHandler handles AI Agent requests
type AgentHandler struct {
	agent      agent.Agent
	sessionMgr *session.Manager
}

// NewAgentHandler creates an AI Agent handler
func NewAgentHandler(agent agent.Agent, sessionMgr *session.Manager) *AgentHandler {
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
	requestCtx := c.Request.Context()
	if _, err = h.sessionMgr.GetSession(requestCtx, sessionID); err != nil {
		h.writeSessionError(c, "Invalid session ID", err, http.StatusBadRequest)
		return
	}
	if err = h.sessionMgr.TouchSession(requestCtx, sessionID); err != nil {
		h.writeSessionError(c, "Invalid session ID", err, http.StatusBadRequest)
		return
	}

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

	channels = h.agent.Interact(requestCtx, &schema.UserInput{Content: req.Message}, sessionID)
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
			rt.GetLogger().Info("Handler received feedback",
				"session_id", sessionID,
				"text", feedback.Text(),
				"done", feedback.IsDone(),
				"final", feedback.IsFinal(),
				"final_nil", feedback.Final() == nil)
			if feedback.IsFinal() {
				rt.GetLogger().Info("MessageDelta called with output type", "type", fmt.Sprintf("%T", feedback.Final()))
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

		case <-requestCtx.Done():
			rt.GetLogger().Info("Client disconnected from stream")
			return
		case <-channels.Done():
			rt.GetLogger().Info("Channels closed, draining remaining messages", "session_id", sessionID)
			h.drainAndFinish(sseHandler, channels, sessionID)
			return
		}
	}
}

func (h *AgentHandler) drainAndFinish(sseHandler *sse.SSEHandler, channels *agent.Channels, sessionID string) {
	for {
		select {
		case feedback := <-channels.UserRespChan:
			h.writeFeedback(sseHandler, feedback)
		case err := <-channels.ErrorChan:
			if err != nil {
				sseHandler.HandleError("agent_error", fmt.Sprintf("agent error: %v", err))
			}
		default:
			if err := sseHandler.FinishStream(); err != nil {
				rt.GetLogger().Error("Failed to finish stream", "error", err)
			}
			rt.GetLogger().Info("Stream processing completed", "session_id", sessionID)
			return
		}
	}
}

func (h *AgentHandler) writeFeedback(sseHandler *sse.SSEHandler, feedback *schema.StreamFeedback) {
	if feedback == nil {
		return
	}
	if feedback.IsFinal() {
		h.MessageDelta(sseHandler, feedback.Final())
	} else if feedback.IsDone() {
		if err := sseHandler.HandleContentBlockStop(feedback.Index()); err != nil {
			rt.GetLogger().Error("Failed to handle content block stop", "error", err)
		}
	} else if err := sseHandler.HandleText(feedback.Text(), feedback.Index()); err != nil {
		rt.GetLogger().Error("Failed to handle text", "error", err)
	}
}

// MessageDelta finishes the stream and reports token usage. The observation's
// Summary/FinalAnswer text was already streamed live by the observe stage
// (react emitObservation), so this only emits the stop reason + usage — it must
// NOT re-stream the text, or the client would receive the answer twice.
func (h *AgentHandler) MessageDelta(sseHandler *sse.SSEHandler, output schema.Schema) {
	stopReason := "end_turn"

	if err := sseHandler.MessageDeltaWithUsage(stopReason, output); err != nil {
		sseHandler.HandleError("finish_stream_error", fmt.Sprintf("failed to finish stream: %v", err))
	}
}

func (h *AgentHandler) CreateSession(c *gin.Context) {
	sessionObj, err := h.sessionMgr.CreateSession(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse("Failed to create session"))
		return
	}
	c.JSON(http.StatusOK, NewSuccessResponse(toSessionInfo(sessionObj)))
}

func (h *AgentHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse("Session ID is required"))
		return
	}

	sessionObj, err := h.sessionMgr.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		h.writeSessionError(c, "Session not found", err, http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, NewSuccessResponse(toSessionInfo(sessionObj)))
}

func (h *AgentHandler) ListSessions(c *gin.Context) {
	sessionObjs, err := h.sessionMgr.ListSessions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse("Failed to list sessions"))
		return
	}
	sessions := make([]map[string]any, 0, len(sessionObjs))
	for _, sessionObj := range sessionObjs {
		sessions = append(sessions, toSessionInfo(sessionObj))
	}

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

	err := h.sessionMgr.DeleteSession(c.Request.Context(), sessionID)
	if err != nil {
		h.writeSessionError(c, "Session not found", err, http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, NewSuccessResponse(map[string]string{
		"message": "Session deleted successfully",
	}))
}

func (h *AgentHandler) writeSessionError(c *gin.Context, message string, err error, invalidStatus int) {
	status := http.StatusInternalServerError
	responseMessage := "Session storage unavailable"
	if errors.Is(err, conversationstore.ErrSessionNotFound) || errors.Is(err, conversationstore.ErrSessionExpired) {
		status = invalidStatus
		responseMessage = message + ": " + err.Error()
	}
	rt.GetLogger().Error("Session store request failed", "error", err)
	c.JSON(status, NewErrorResponse(responseMessage))
}

func toSessionInfo(sessionObj *session.Session) map[string]any {
	return map[string]any{
		"session_id": sessionObj.ID,
		"created_at": sessionObj.CreatedAt,
		"updated_at": sessionObj.UpdatedAt,
		"status":     sessionObj.Status,
	}
}
