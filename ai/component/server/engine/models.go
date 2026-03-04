package engine

import (
	"time"

	"github.com/google/uuid"
)

// Response defines unified API response format
type Response struct {
	Message   string `json:"message"`        // Response message
	Data      any    `json:"data,omitempty"` // Response data
	RequestID string `json:"request_id"`     // Request ID for tracking
	Timestamp int64  `json:"timestamp"`      // Response timestamp
}

// NewResponse creates a response
func NewResponse(message string, data any) *Response {
	return &Response{
		Message:   message,
		Data:      data,
		RequestID: generateRequestID(),
		Timestamp: time.Now().Unix(),
	}
}

// NewSuccessResponse creates a success response
func NewSuccessResponse(data any) *Response {
	return NewResponse("success", data)
}

// NewErrorResponse creates an error response
func NewErrorResponse(message string) *Response {
	return NewResponse(message, nil)
}

// ChatRequest defines streaming chat request
type ChatRequest struct {
	Message   string `json:"message" binding:"required"`   // User message
	SessionID string `json:"sessionID" binding:"required"` // Session ID
}

// generateRequestID generates a request ID
func generateRequestID() string {
	return "req_" + uuid.New().String()
}
