package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"dubbo-admin-ai/schema"

	"github.com/firebase/genkit/go/ai"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SSEType string

const (
	MessageStart      SSEType = "message_start"
	ContentBlockStart SSEType = "content_block_start"
	ContentBlockDelta SSEType = "content_block_delta"
	ContentBlockStop  SSEType = "content_block_stop"
	MessageDelta      SSEType = "message_delta"
	MessageStop       SSEType = "message_stop"
	ErrorEvent        SSEType = "error"
	Ping              SSEType = "ping"
)

type SSE struct {
	Type         SSEType             `json:"type"`
	Message      *Message            `json:"message,omitempty"`
	Index        *int                `json:"index,omitempty"`
	ContentBlock *ContentBlock       `json:"content_block,omitempty"`
	Delta        *Delta              `json:"delta,omitempty"`
	Usage        *ai.GenerationUsage `json:"usage,omitempty"`
	Error        *ErrorInfo          `json:"error,omitempty"`
}

type Message struct {
	ID           string              `json:"id"`
	Type         string              `json:"type"`
	Role         string              `json:"role"`
	Content      []ContentItem       `json:"content"`
	Model        string              `json:"model,omitempty"`
	StopReason   *string             `json:"stop_reason,omitempty"`
	StopSequence *string             `json:"stop_sequence,omitempty"`
	Usage        *ai.GenerationUsage `json:"usage,omitempty"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type DeltaType string

const (
	TextDelta      DeltaType = "text_delta"
	InputJsonDelta DeltaType = "input_json_delta"
	ThinkingDelta  DeltaType = "thinking_delta"
)

type Delta struct {
	Type         DeltaType `json:"type,omitempty"`
	Text         string    `json:"text,omitempty"`
	StopReason   *string   `json:"stop_reason,omitempty"`
	StopSequence *string   `json:"stop_sequence,omitempty"`
}

// ErrorInfo defines error information
type ErrorInfo struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// StreamWriter handles SSE stream writing
type StreamWriter struct {
	ctx     context.Context
	writer  gin.ResponseWriter
	flusher http.Flusher
}

// NewStreamWriter creates an SSE stream writer
func NewStreamWriter(c *gin.Context) (*StreamWriter, error) {
	// Set SSE response headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Content-Type")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming unsupported")
	}

	return &StreamWriter{
		ctx:     c.Request.Context(),
		writer:  c.Writer,
		flusher: flusher,
	}, nil
}

// WriteEvent writes SSE event (generic method)
func (sw *StreamWriter) WriteEvent(eventType SSEType, data any) error {
	// Check if context is cancelled
	select {
	case <-sw.ctx.Done():
		return sw.ctx.Err()
	default:
	}

	// Build SSE data
	eventData := map[string]any{
		"type": eventType,
	}

	// Add corresponding data based on event type
	if data != nil {
		switch eventType {
		case MessageStart:
			if msg, ok := data.(*Message); ok {
				eventData["message"] = msg
			}
		case ContentBlockStart:
			if blockData, ok := data.(map[string]any); ok {
				if index, exists := blockData["index"]; exists {
					eventData["index"] = index
				}
				if contentBlock, exists := blockData["content_block"]; exists {
					eventData["content_block"] = contentBlock
				}
			}
		case ContentBlockDelta:
			if deltaData, ok := data.(map[string]any); ok {
				if index, exists := deltaData["index"]; exists {
					eventData["index"] = index
				}
				if delta, exists := deltaData["delta"]; exists {
					eventData["delta"] = delta
				}
			}
		case ContentBlockStop:
			if index, ok := data.(int); ok {
				eventData["index"] = index
			}
		case MessageDelta:
			if deltaData, ok := data.(map[string]any); ok {
				if delta, exists := deltaData["delta"]; exists {
					eventData["delta"] = delta
				}
				if usage, exists := deltaData["usage"]; exists {
					eventData["usage"] = usage
				}

			}
		case ErrorEvent:
			if errorInfo, ok := data.(*ErrorInfo); ok {
				eventData["error"] = errorInfo
			}
		}
	}

	// Serialize data to JSON
	jsonData, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	// Write SSE formatted data
	_, err = fmt.Fprintf(sw.writer, "event: %s\ndata: %s\n\n", eventType, string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to write SSE data: %w", err)
	}

	// Flush buffer immediately
	sw.flusher.Flush()
	return nil
}

// WriteMessageStart writes message start event
func (sw *StreamWriter) WriteMessageStart(msg *Message) error {
	return sw.WriteEvent(MessageStart, msg)
}

// WriteContentBlockStart writes content block start event
func (sw *StreamWriter) WriteContentBlockStart(index int, contentBlock *ContentBlock) error {
	data := map[string]any{
		"index":         index,
		"content_block": contentBlock,
	}
	return sw.WriteEvent(ContentBlockStart, data)
}

// WriteContentBlockDelta writes content block delta event
func (sw *StreamWriter) WriteContentBlockDelta(index int, delta *Delta) error {
	data := map[string]any{
		"index": index,
		"delta": delta,
	}
	return sw.WriteEvent(ContentBlockDelta, data)
}

// WriteContentBlockStop writes content block stop event
func (sw *StreamWriter) WriteContentBlockStop(index int) error {
	return sw.WriteEvent(ContentBlockStop, index)
}

// WriteMessageDelta writes message delta event
func (sw *StreamWriter) WriteMessageDelta(delta *Delta, usage *ai.GenerationUsage) error {
	dataMap := map[string]any{
		"delta": delta,
	}
	dataMap["usage"] = usage
	return sw.WriteEvent(MessageDelta, dataMap)
}

// WriteMessageStop writes message stop event
func (sw *StreamWriter) WriteMessageStop() error {
	return sw.WriteEvent(MessageStop, nil)
}

// WriteError writes error event
func (sw *StreamWriter) WriteError(errorInfo *ErrorInfo) error {
	return sw.WriteEvent(ErrorEvent, errorInfo)
}

// WritePing writes ping event
// func (sw *StreamWriter) WritePing() error {
// 	_, err := fmt.Fprintf(sw.writer, "event: ping\ndata: {\"type\": \"ping\"}\n\n")
// 	if err != nil {
// 		return fmt.Errorf("failed to write ping event: %w", err)
// 	}
// 	sw.flusher.Flush()
// 	return nil
// }

type SSEHandler struct {
	writer         *StreamWriter
	sessionID      string
	messageID      string
	ContentStarted bool
}

// NewStreamHandler creates a stream handler
func NewStreamHandler(writer *StreamWriter, sessionID string) *SSEHandler {
	// Generate message ID
	messageID := fmt.Sprintf("msg_%s", uuid.New().String())

	return &SSEHandler{
		writer:         writer,
		sessionID:      sessionID,
		messageID:      messageID,
		ContentStarted: false,
	}
}

// HandleText handles plain text messages (e.g., final answers)
func (sh *SSEHandler) HandleText(text string, index int) error {
	// Send message start and content block start events if content hasn't started
	if !sh.ContentStarted {
		// Send message start event
		msg := &Message{
			ID:      sh.messageID,
			Type:    "message",
			Role:    "assistant",
			Content: []ContentItem{},
		}
		if err := sh.writer.WriteMessageStart(msg); err != nil {
			return err
		}

		// Send content block start event
		contentBlock := &ContentBlock{
			Type: "text",
			Text: "",
		}
		if err := sh.writer.WriteContentBlockStart(index, contentBlock); err != nil {
			return err
		}

		sh.ContentStarted = true
	}

	// Send text content as delta
	if text != "" {
		delta := &Delta{
			Type: TextDelta,
			Text: text,
		}
		if err := sh.writer.WriteContentBlockDelta(index, delta); err != nil {
			return err
		}
	}

	return nil
}

// HandleStreamChunk handles streaming data chunks
func (sh *SSEHandler) HandleStreamChunk(chunk schema.StreamChunk) error {
	// Send message start and content block start events when processing first chunk
	if !sh.ContentStarted {
		// Send message start event
		msg := &Message{
			ID:      sh.messageID,
			Type:    "message",
			Role:    "assistant",
			Content: []ContentItem{},
		}
		if err := sh.writer.WriteMessageStart(msg); err != nil {
			return err
		}

		// Send content block start event
		contentBlock := &ContentBlock{
			Type: "text",
			Text: "",
		}
		if err := sh.writer.WriteContentBlockStart(0, contentBlock); err != nil {
			return err
		}

		sh.ContentStarted = true
	}

	// Process based on Agent's streaming output format
	if chunk.Chunk != nil {
		// Get delta text directly
		deltaText := chunk.Chunk.Text()
		if deltaText != "" {
			delta := &Delta{
				Type: TextDelta,
				Text: deltaText,
			}
			if err := sh.writer.WriteContentBlockDelta(0, delta); err != nil {
				return err
			}
		}
	}

	return nil
}

// HandleContentBlockStop handles content block stop event
func (sh *SSEHandler) HandleContentBlockStop(index int) error {
	// Only send content_block_stop if content has started
	if !sh.ContentStarted {
		return nil
	}
	return sh.writer.WriteContentBlockStop(index)
}

func (sh *SSEHandler) MessageDeltaWithUsage(stopReason string, output schema.Schema) error {
	delta := &Delta{
		StopReason: &stopReason,
	}
	if err := sh.writer.WriteMessageDelta(delta, output.Usage()); err != nil {
		return err
	}
	return nil
}

// FinishStream completes streaming response and sends stop event
func (sh *SSEHandler) FinishStream() error {
	if err := sh.writer.WriteMessageStop(); err != nil {
		return err
	}

	return nil
}

// HandleError handles error conditions and sends error event
func (sh *SSEHandler) HandleError(errorType, errorMessage string) {
	errorInfo := &ErrorInfo{
		Type:    errorType,
		Message: errorMessage,
	}
	sh.writer.WriteError(errorInfo)
	sh.FinishStream()
}
