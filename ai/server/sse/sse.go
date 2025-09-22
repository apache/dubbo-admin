package sse

import (
	"context"
	"dubbo-admin-ai/manager"
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

// ErrorInfo 错误信息
type ErrorInfo struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// StreamWriter SSE流写入器
type StreamWriter struct {
	ctx     context.Context
	writer  gin.ResponseWriter
	flusher http.Flusher
}

// NewStreamWriter 创建SSE流写入器
func NewStreamWriter(c *gin.Context) (*StreamWriter, error) {
	// 设置SSE响应头
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

// WriteEvent 写入SSE事件的通用方法
func (sw *StreamWriter) WriteEvent(eventType SSEType, data any) error {
	// 检查上下文是否已取消
	select {
	case <-sw.ctx.Done():
		return sw.ctx.Err()
	default:
	}

	// 构建SSE数据
	eventData := map[string]any{
		"type": eventType,
	}

	// 根据事件类型添加相应数据
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

	// 将数据序列化为JSON
	jsonData, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	// 写入SSE格式数据
	_, err = fmt.Fprintf(sw.writer, "event: %s\ndata: %s\n\n", eventType, string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to write SSE data: %w", err)
	}

	// 立即刷新缓冲区
	sw.flusher.Flush()
	return nil
}

// WriteMessageStart 写入消息开始事件
func (sw *StreamWriter) WriteMessageStart(msg *Message) error {
	return sw.WriteEvent(MessageStart, msg)
}

// WriteContentBlockStart 写入内容块开始事件
func (sw *StreamWriter) WriteContentBlockStart(index int, contentBlock *ContentBlock) error {
	data := map[string]any{
		"index":         index,
		"content_block": contentBlock,
	}
	return sw.WriteEvent(ContentBlockStart, data)
}

// WriteContentBlockDelta 写入内容块增量事件
func (sw *StreamWriter) WriteContentBlockDelta(index int, delta *Delta) error {
	data := map[string]any{
		"index": index,
		"delta": delta,
	}
	return sw.WriteEvent(ContentBlockDelta, data)
}

// WriteContentBlockStop 写入内容块结束事件
func (sw *StreamWriter) WriteContentBlockStop(index int) error {
	return sw.WriteEvent(ContentBlockStop, index)
}

// WriteMessageDelta 写入消息增量事件
func (sw *StreamWriter) WriteMessageDelta(delta *Delta, usage *ai.GenerationUsage) error {
	data := map[string]any{
		"delta": delta,
	}
	if usage != nil {
		data["usage"] = usage
	}
	return sw.WriteEvent(MessageDelta, data)
}

// WriteMessageStop 写入消息结束事件
func (sw *StreamWriter) WriteMessageStop() error {
	return sw.WriteEvent(MessageStop, nil)
}

// WriteError 写入错误事件
func (sw *StreamWriter) WriteError(errorInfo *ErrorInfo) error {
	return sw.WriteEvent(ErrorEvent, errorInfo)
}

// WritePing 写入ping事件
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

// NewStreamHandler 创建流式处理器
func NewStreamHandler(writer *StreamWriter, sessionID string) *SSEHandler {
	// 生成消息ID
	messageID := fmt.Sprintf("msg_%s", uuid.New().String())

	return &SSEHandler{
		writer:         writer,
		sessionID:      sessionID,
		messageID:      messageID,
		ContentStarted: false,
	}
}

// HandleText 处理纯文本消息（如最终答案）
func (sh *SSEHandler) HandleText(text string, index int) error {
	// 如果还没有开始内容块，先发送消息开始和内容块开始事件
	if !sh.ContentStarted {
		// 发送消息开始事件
		msg := &Message{
			ID:      sh.messageID,
			Type:    "message",
			Role:    "assistant",
			Content: []ContentItem{},
		}
		if err := sh.writer.WriteMessageStart(msg); err != nil {
			manager.GetLogger().Error("Failed to write message start event", "error", err)
			return err
		}

		// 发送内容块开始事件
		contentBlock := &ContentBlock{
			Type: "text",
			Text: "",
		}
		if err := sh.writer.WriteContentBlockStart(index, contentBlock); err != nil {
			manager.GetLogger().Error("Failed to write content block start event", "error", err)
			return err
		}

		sh.ContentStarted = true
	}

	// 发送文本内容作为增量
	if text != "" {
		delta := &Delta{
			Type: TextDelta,
			Text: text,
		}
		if err := sh.writer.WriteContentBlockDelta(index, delta); err != nil {
			manager.GetLogger().Error("Failed to write content block delta", "error", err)
			return err
		}
	}

	return nil
}

// HandleStreamChunk 处理流式数据块
func (sh *SSEHandler) HandleStreamChunk(chunk schema.StreamChunk) error {
	// 处理第一个chunk时发送消息开始和内容块开始事件
	if !sh.ContentStarted {
		// 发送消息开始事件
		msg := &Message{
			ID:      sh.messageID,
			Type:    "message",
			Role:    "assistant",
			Content: []ContentItem{},
		}
		if err := sh.writer.WriteMessageStart(msg); err != nil {
			manager.GetLogger().Error("Failed to write message start event", "error", err)
			return err
		}

		// 发送内容块开始事件
		contentBlock := &ContentBlock{
			Type: "text",
			Text: "",
		}
		if err := sh.writer.WriteContentBlockStart(0, contentBlock); err != nil {
			manager.GetLogger().Error("Failed to write content block start event", "error", err)
			return err
		}

		sh.ContentStarted = true
	}

	// 根据Agent的流式输出格式处理
	if chunk.Chunk != nil {
		// 直接获取增量文本
		deltaText := chunk.Chunk.Text()
		if deltaText != "" {
			delta := &Delta{
				Type: TextDelta,
				Text: deltaText,
			}
			if err := sh.writer.WriteContentBlockDelta(0, delta); err != nil {
				manager.GetLogger().Error("Failed to write content block delta", "error", err)
				return err
			}
		}
	}

	return nil
}

// FinishStream 完成流式响应，发送结束事件
func (sh *SSEHandler) FinishStream(stopReason string, usage *ai.GenerationUsage, index int) error {
	// 发送内容块结束事件
	if err := sh.writer.WriteContentBlockStop(index); err != nil {
		manager.GetLogger().Error("Failed to write content block stop event", "error", err)
		return err
	}

	// 发送消息增量事件（包含停止原因和使用情况）
	delta := &Delta{
		StopReason: &stopReason,
	}
	if err := sh.writer.WriteMessageDelta(delta, usage); err != nil {
		manager.GetLogger().Error("Failed to write message delta event", "error", err)
		return err
	}

	// 发送消息结束事件
	if err := sh.writer.WriteMessageStop(); err != nil {
		manager.GetLogger().Error("Failed to write message stop event", "error", err)
		return err
	}

	return nil
}

// HandleError 处理错误情况，发送错误事件
func (sh *SSEHandler) HandleError(errorType, errorMessage string) {
	errorInfo := &ErrorInfo{
		Type:    errorType,
		Message: errorMessage,
	}
	sh.writer.WriteError(errorInfo)
	sh.FinishStream("error", nil, 0)
}
