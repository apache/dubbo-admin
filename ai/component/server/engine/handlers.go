package engine

import (
	"context"
	"fmt"
	"net/http"

	"dubbo-admin-ai/component/server/engine/session"
	"dubbo-admin-ai/component/server/engine/sse"
	rt "dubbo-admin-ai/runtime"

	"dubbo-admin-ai/component/agent"
	"dubbo-admin-ai/schema"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
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
		pageContext  *schema.AIContextSnapshot
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
	if pageContext, err = req.ParseContext(); err != nil {
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
			// The interaction is detached from the request context, so nothing
			// else will stop it. Drain its bounded channels or the agent blocks
			// forever on its next send and never runs its end hooks.
			if channels != nil {
				go discardAgentOutput(channels)
			}
			sseHandler.HandleError("internal_error", fmt.Sprintf("internal error: %v", r))
		}
	}()

	// The interaction can outlive the SSE request. Preserve request-scoped
	// values (including an extracted trace context) while detaching cancellation
	// and deadlines from the client connection.
	extractedCtx := otel.GetTextMapPropagator().Extract(
		c.Request.Context(),
		propagation.HeaderCarrier(c.Request.Header),
	)
	interactionCtx := context.WithoutCancel(extractedCtx)
	channels = h.agent.Interact(interactionCtx, &schema.UserInput{Content: req.Message, Context: pageContext}, sessionID)
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
				sseHandler.HandleError("agent_error", fmt.Sprintf("agent error: %v", err))
				rt.GetLogger().Error("Agent interaction error", "session_id", sessionID, "error", err)
				go discardAgentOutput(channels)
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

		case <-c.Request.Context().Done():
			rt.GetLogger().Info("Client disconnected from stream")
			go discardAgentOutput(channels)
			return

		default:
			if channels.Closed() {
				// Drain remaining messages before finishing
				rt.GetLogger().Info("Channels closed, draining remaining messages", "session_id", sessionID)
			drainLoop:
				for {
					select {
					case feedback, ok = <-channels.UserRespChan:
						if !ok {
							channels.UserRespChan = nil
							break drainLoop
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
					case err, ok = <-channels.ErrorChan:
						if !ok {
							channels.ErrorChan = nil
							break drainLoop
						}
						if err != nil {
							sseHandler.HandleError("agent_error", fmt.Sprintf("agent error: %v", err))
						}
					default:
						break drainLoop
					}
				}
				if err := sseHandler.FinishStream(); err != nil {
					rt.GetLogger().Error("Failed to finish stream", "error", err)
				}
				rt.GetLogger().Info("Stream processing completed", "session_id", sessionID)
				return
			}
		}
	}
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

// MessageDelta finishes the stream and reports token usage. The observation's
// answer text was already streamed live by the ReAct loop, so this only
// emits the stop reason + usage — it must
// NOT re-stream the text, or the client would receive the answer twice.
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
