package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"dubbo-admin-ai/component/server/engine/session"
	"dubbo-admin-ai/component/server/engine/sse"
	rt "dubbo-admin-ai/runtime"

	"dubbo-admin-ai/component/agent"
	compcontext "dubbo-admin-ai/component/context"
	"dubbo-admin-ai/schema"
	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// AgentHandler handles AI Agent requests
type AgentHandler struct {
	agent            agent.Agent
	sessionMgr       *session.Manager
	interactionStore conversationstore.MessageStore
	interactionLease time.Duration
	newStreamWriter  func(*gin.Context) (*sse.StreamWriter, error)
}

const (
	clientInvalidRequest    = "Invalid request"
	clientInternalError     = "Internal server error"
	clientAgentError        = "Agent interaction failed"
	clientFinishStreamError = "Failed to finish stream"
)

// NewAgentHandler creates an AI Agent handler
func NewAgentHandler(agent agent.Agent, sessionMgr *session.Manager, stores ...conversationstore.MessageStore) *AgentHandler {
	handler := &AgentHandler{
		agent: agent, sessionMgr: sessionMgr,
		interactionLease: conversationstore.DefaultLifecyclePolicy().InteractionLease,
		newStreamWriter:  sse.NewStreamWriter,
	}
	if len(stores) > 0 {
		handler.interactionStore = stores[0]
	}
	return handler
}

func (h *AgentHandler) SetInteractionLease(lease time.Duration) {
	if lease > 0 {
		h.interactionLease = lease
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
		rt.GetLogger().Warn("Invalid stream request", "error_type", fmt.Sprintf("%T", err))
		c.JSON(http.StatusBadRequest, NewErrorResponse(clientInvalidRequest))
		return
	}

	sessionID = req.SessionID
	requestCtx := c.Request.Context()
	access, accessErr := security.AccessContextFromContext(requestCtx)
	if accessErr != nil {
		c.JSON(http.StatusUnauthorized, NewAPIErrorResponse("AUTHENTICATION_REQUIRED", "Authentication required"))
		return
	}
	if _, err = h.sessionMgr.GetSession(requestCtx, access, sessionID); err != nil {
		h.writeSessionError(c, "Invalid session ID", err, http.StatusBadRequest)
		return
	}
	var prepared *conversationstore.PreparedInteraction
	handlerOwnsInteraction := false
	if h.interactionStore != nil {
		idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if !validIdempotencyKey(idempotencyKey) {
			c.JSON(http.StatusBadRequest, NewAPIErrorResponse("IDEMPOTENCY_KEY_REQUIRED", "A valid Idempotency-Key is required"))
			return
		}
		prepared, err = h.interactionStore.PrepareInteraction(requestCtx, access, sessionID, idempotencyKey, hashChatRequest(req), h.interactionLease)
		if err != nil {
			h.writeInteractionError(c, err)
			return
		}
		c.Header("X-Interaction-ID", prepared.InteractionID)
		handlerOwnsInteraction = !prepared.Replayed
		defer func() {
			if !handlerOwnsInteraction {
				return
			}
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(requestCtx), 30*time.Second)
			defer cancel()
			if cleanupErr := h.interactionStore.FailInteraction(cleanupCtx, access, sessionID, prepared.InteractionID, prepared.TurnID); cleanupErr != nil && !errors.Is(cleanupErr, conversationstore.ErrTurnNotFound) {
				rt.GetLogger().Error("Failed to release unclaimed interaction", "session_id", sessionID, "interaction_id", prepared.InteractionID, "error", security.SafeError(cleanupErr))
			}
		}()
	}
	if err = h.sessionMgr.TouchSession(requestCtx, access, sessionID); err != nil {
		h.writeSessionError(c, "Invalid session ID", err, http.StatusBadRequest)
		return
	}

	if streamWriter, err = h.newStreamWriter(c); err != nil {
		rt.GetLogger().Error("Failed to create stream writer", "error_type", fmt.Sprintf("%T", err))
		c.JSON(http.StatusInternalServerError, NewErrorResponse(clientInternalError))
		return
	}
	sseHandler = sse.NewStreamHandler(streamWriter, sessionID)
	if prepared != nil && prepared.Replayed {
		if err := sseHandler.HandleText(prepared.Result, 0); err == nil {
			err = sseHandler.HandleContentBlockStop(0)
		}
		if err == nil {
			err = sseHandler.MessageDeltaWithUsage("end_turn", &schema.Observation{})
		}
		if err == nil {
			err = sseHandler.FinishStream()
		}
		if err != nil {
			rt.GetLogger().Error("Failed to replay interaction", "interaction_id", prepared.InteractionID, "error_type", fmt.Sprintf("%T", err))
		}
		return
	}

	// Set response headers and error recovery
	defer func() {
		if r := recover(); r != nil {
			if channels != nil {
				go discardAgentOutput(channels)
			}
			rt.GetLogger().Error("Recovered panic while streaming AI response", "panic_recovered", true)
			sseHandler.HandleError("internal_error", clientInternalError)
		}
	}()

	// Preserve inbound tracing values while allowing the interaction to finish
	// after a client disconnects. The agent cancels and drains it on shutdown.
	extractedCtx := otel.GetTextMapPropagator().Extract(requestCtx, propagation.HeaderCarrier(c.Request.Header))
	if prepared != nil {
		extractedCtx = conversationstore.WithPreparedInteraction(extractedCtx, prepared)
	}
	channels = h.agent.Interact(context.WithoutCancel(extractedCtx), &schema.UserInput{Content: req.Message}, sessionID)
	if !channels.Valid() {
		rt.GetLogger().Error("Agent returned invalid channels", "session_id", sessionID)
		sseHandler.HandleError("agent_error", clientAgentError)
		return
	}
	handlerOwnsInteraction = false
	if traceID := channels.TraceID(); traceID != "" {
		c.Header("X-Trace-ID", traceID)
	}
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
				code, message := contextError(err)
				sseHandler.HandleError(code, message)
				rt.GetLogger().Error("Agent interaction error", "session_id", sessionID, "error_type", fmt.Sprintf("%T", err))
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
				"done", feedback.IsDone(),
				"final", feedback.IsFinal(),
				"final_nil", feedback.Final() == nil)
			if feedback.IsFinal() {
				rt.GetLogger().Info("MessageDelta called with output type", "type", fmt.Sprintf("%T", feedback.Final()))
				h.MessageDelta(sseHandler, feedback.Final())
			} else if feedback.IsDone() {
				if err := sseHandler.HandleContentBlockStop(feedback.Index()); err != nil {
					rt.GetLogger().Error("Failed to handle content block stop", "error_type", fmt.Sprintf("%T", err))
				}
			} else {
				if err := sseHandler.HandleText(feedback.Text(), feedback.Index()); err != nil {
					rt.GetLogger().Error("Failed to handle text", "error_type", fmt.Sprintf("%T", err))
				}
			}

		case <-requestCtx.Done():
			rt.GetLogger().Info("Client disconnected from stream")
			go discardAgentOutput(channels)
			return
		case <-channels.Done():
			rt.GetLogger().Info("Channels closed, draining remaining messages", "session_id", sessionID)
			h.drainAndFinish(sseHandler, channels, sessionID)
			return
		}
	}
}

func validIdempotencyKey(key string) bool {
	if len(key) == 0 || len(key) > 128 {
		return false
	}
	for _, r := range key {
		if r < 0x21 || r > 0x7e {
			return false
		}
	}
	return true
}

func hashChatRequest(req ChatRequest) string {
	payload, _ := json.Marshal(struct {
		Version   int    `json:"version"`
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}{Version: 1, SessionID: req.SessionID, Message: req.Message})
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

func (h *AgentHandler) writeInteractionError(c *gin.Context, err error) {
	status, code, message := http.StatusInternalServerError, "INTERNAL_ERROR", clientInternalError
	switch {
	case errors.Is(err, conversationstore.ErrIdempotencyConflict):
		status, code, message = http.StatusConflict, "IDEMPOTENCY_CONFLICT", "Idempotency key conflicts with the original request"
	case errors.Is(err, conversationstore.ErrInteractionInProgress):
		status, code, message = http.StatusConflict, "INTERACTION_IN_PROGRESS", "The original interaction is still in progress"
	case errors.Is(err, conversationstore.ErrSessionBusy):
		status, code, message = http.StatusConflict, "SESSION_BUSY", "The session already has an active interaction"
	case errors.Is(err, conversationstore.ErrResourceExhausted), errors.Is(err, conversationstore.ErrTurnLimitReached):
		status, code, message = http.StatusTooManyRequests, "RESOURCE_EXHAUSTED", "Conversation storage quota exhausted"
	}
	rt.GetLogger().Error("Interaction preparation failed", "error", security.SafeError(err))
	c.JSON(status, NewAPIErrorResponse(code, message))
}

// discardAgentOutput keeps detached interactions from blocking on their
// bounded response channels after the SSE consumer disconnects.
func discardAgentOutput(channels *agent.Channels) {
	for {
		select {
		case <-channels.UserRespChan:
		case <-channels.ErrorChan:
		case <-channels.Done():
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
				rt.GetLogger().Error("Agent interaction error while draining", "session_id", sessionID, "error_type", fmt.Sprintf("%T", err))
				code, message := contextError(err)
				sseHandler.HandleError(code, message)
			}
		default:
			if err := sseHandler.FinishStream(); err != nil {
				rt.GetLogger().Error("Failed to finish stream", "error_type", fmt.Sprintf("%T", err))
				sseHandler.HandleError("finish_stream_error", clientFinishStreamError)
			}
			rt.GetLogger().Info("Stream processing completed", "session_id", sessionID)
			return
		}
	}
}

func contextError(err error) (string, string) {
	switch {
	case errors.Is(err, compcontext.ErrMandatoryBudget):
		return "CONTEXT_MANDATORY_BUDGET_EXCEEDED", "Required context exceeds the model input budget"
	case errors.Is(err, compcontext.ErrTemplateInput):
		return "PROMPT_REQUIRED_INPUT_MISSING", "A required server template value is unavailable"
	case errors.Is(err, compcontext.ErrTemplateVersion):
		return "PROMPT_VERSION_UNAVAILABLE", "The session template version is unavailable"
	default:
		return "agent_error", clientAgentError
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
			rt.GetLogger().Error("Failed to handle content block stop", "error_type", fmt.Sprintf("%T", err))
		}
	} else if err := sseHandler.HandleText(feedback.Text(), feedback.Index()); err != nil {
		rt.GetLogger().Error("Failed to handle text", "error_type", fmt.Sprintf("%T", err))
	}
}

// MessageDelta finishes the stream and reports token usage. The observation's
// Summary/FinalAnswer text was already streamed live by the observe stage
// (react emitObservation), so this only emits the stop reason + usage — it must
// NOT re-stream the text, or the client would receive the answer twice.
func (h *AgentHandler) MessageDelta(sseHandler *sse.SSEHandler, output schema.Schema) {
	stopReason := "end_turn"

	if err := sseHandler.MessageDeltaWithUsage(stopReason, output); err != nil {
		rt.GetLogger().Error("Failed to write message delta", "error_type", fmt.Sprintf("%T", err))
		sseHandler.HandleError("finish_stream_error", clientFinishStreamError)
	}
}

func (h *AgentHandler) CreateSession(c *gin.Context) {
	access, err := security.AccessContextFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewAPIErrorResponse("AUTHENTICATION_REQUIRED", "Authentication required"))
		return
	}
	var templateRefs []string
	if provider, ok := h.agent.(interface{ TemplateReference() string }); ok && provider.TemplateReference() != "" {
		templateRefs = append(templateRefs, provider.TemplateReference())
	}
	sessionObj, err := h.sessionMgr.CreateSession(c.Request.Context(), access, templateRefs...)
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

	access, err := security.AccessContextFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewAPIErrorResponse("AUTHENTICATION_REQUIRED", "Authentication required"))
		return
	}
	sessionObj, err := h.sessionMgr.GetSession(c.Request.Context(), access, sessionID)
	if err != nil {
		h.writeSessionError(c, "Session not found", err, http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, NewSuccessResponse(toSessionInfo(sessionObj)))
}

func (h *AgentHandler) ListSessions(c *gin.Context) {
	access, err := security.AccessContextFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewAPIErrorResponse("AUTHENTICATION_REQUIRED", "Authentication required"))
		return
	}
	sessionObjs, err := h.sessionMgr.ListSessions(c.Request.Context(), access)
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

	access, err := security.AccessContextFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewAPIErrorResponse("AUTHENTICATION_REQUIRED", "Authentication required"))
		return
	}
	err = h.sessionMgr.DeleteSession(c.Request.Context(), access, sessionID)
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
	if errors.Is(err, conversationstore.ErrSessionNotFound) || errors.Is(err, conversationstore.ErrSessionExpired) || errors.Is(err, security.ErrResourceNotAccessible) {
		status = invalidStatus
		responseMessage = message
	}
	rt.GetLogger().Error("Session store request failed", "error", security.SafeError(err))
	if status == invalidStatus {
		c.JSON(status, NewAPIErrorResponse("RESOURCE_NOT_ACCESSIBLE", responseMessage))
		return
	}
	c.JSON(status, NewErrorResponse(responseMessage))
}

func toSessionInfo(sessionObj *session.Session) map[string]any {
	return map[string]any{
		"session_id": sessionObj.ID,
		"tenant_id":  sessionObj.TenantID,
		"owner_id":   sessionObj.OwnerID,
		"version":    sessionObj.Version,
		"created_at": sessionObj.CreatedAt,
		"updated_at": sessionObj.UpdatedAt,
		"expires_at": sessionObj.ExpiresAt,
		"status":     sessionObj.Status,
	}
}
