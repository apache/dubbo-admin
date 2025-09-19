package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"dubbo-admin-ai/internal/manager"
	"dubbo-admin-ai/schema"

	"github.com/gin-gonic/gin"
)

type ChatResponse struct {
	SessionID string `json:"session_id"`
	Message   any    `json:"message"`
	Finished  bool   `json:"finished"`
}

// StreamEventType 流式事件类型
type StreamEventType string

const (
	MessageStart      StreamEventType = "message_start"
	ContentBlockStart StreamEventType = "content_block_start"
	ContentBlockDelta StreamEventType = "content_block_delta"
	ContentBlockStop  StreamEventType = "content_block_stop"
	MessageDeltaEvent StreamEventType = "message_delta"
	MessageStop       StreamEventType = "message_stop"
)

// StreamMessageStartEvent 消息开始事件
type StreamMessageStartEvent struct {
	Type    StreamEventType `json:"type"`
	Message MessageInfo     `json:"message"`
}

// MessageInfo 消息信息
type MessageInfo struct {
	ID        string `json:"id"`
	Model     string `json:"model,omitempty"`
	Role      string `json:"role"`
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
}

// StreamContentBlockStartEvent 内容块开始事件
type StreamContentBlockStartEvent struct {
	Type         StreamEventType `json:"type"`
	Index        int             `json:"index"`
	ContentBlock ContentBlock    `json:"content_block"`
}

// MessageType 消息类型枚举
type MessageType string

const (
	MessageTypeText MessageType = "text"
)

// ContentBlock 内容块
type ContentBlock struct {
	Type MessageType `json:"type"`
}

// StreamContentBlockDeltaEvent 内容块增量事件
type StreamContentBlockDeltaEvent struct {
	Type  StreamEventType `json:"type"`
	Index int             `json:"index"`
	Delta Delta           `json:"delta"`
}

// Delta 文本增量
type Delta struct {
	Content string `json:"content"`
}

// StreamContentBlockStopEvent 内容块结束事件
type StreamContentBlockStopEvent struct {
	Type  StreamEventType `json:"type"`
	Index int             `json:"index"`
}

// StreamMessageDeltaEvent 消息增量事件
type StreamMessageDeltaEvent struct {
	Type  StreamEventType `json:"type"`
	Delta MessageDelta    `json:"delta"`
}

// MessageDelta 消息增量
type MessageDelta struct {
	StopReason string `json:"stop_reason,omitempty"`
	Usage      *Usage `json:"usage,omitempty"`
}

// Usage 使用情况统计
type Usage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
}

// StreamMessageStopEvent 消息结束事件
type StreamMessageStopEvent struct {
	Type StreamEventType `json:"type"`
}

type StandardResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"request_id"`
	Timestamp int64  `json:"timestamp"`
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

// WriteEvent 写入SSE事件
func (sw *StreamWriter) WriteEvent(event string, data any) error {
	select {
	case <-sw.ctx.Done():
		return sw.ctx.Err()
	default:
	}

	// 序列化数据
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// 写入SSE格式
	if event != "" {
		if _, err := fmt.Fprintf(sw.writer, "event: %s\n", event); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(sw.writer, "data: %s\n\n", jsonData); err != nil {
		return err
	}

	sw.flusher.Flush()
	return nil
}

// WriteMessage 写入聊天消息事件
func (sw *StreamWriter) WriteMessage(sessionID string, message any, finished bool) error {
	response := ChatResponse{
		SessionID: sessionID,
		Message:   message,
		Finished:  finished,
	}
	return sw.WriteEvent("message", response)
}

// WriteStreamEvent 写入标准化流式事件
func (sw *StreamWriter) WriteStreamEvent(eventType StreamEventType, data any) error {
	return sw.WriteEvent(string(eventType), data)
}

// WriteMessageStart 写入消息开始事件
func (sw *StreamWriter) WriteMessageStart(sessionID, messageID string) error {
	event := StreamMessageStartEvent{
		Type: MessageStart,
		Message: MessageInfo{
			ID:        messageID,
			Role:      "assistant",
			Type:      "message",
			SessionID: sessionID,
		},
	}
	return sw.WriteStreamEvent(MessageStart, event)
}

// WriteContentBlockStart 写入内容块开始事件
func (sw *StreamWriter) WriteContentBlockStart(index int) error {
	event := StreamContentBlockStartEvent{
		Type:  ContentBlockStart,
		Index: index,
		ContentBlock: ContentBlock{
			Type: MessageTypeText,
		},
	}
	return sw.WriteStreamEvent(ContentBlockStart, event)
}

// WriteContentBlockDelta 写入内容块增量事件
func (sw *StreamWriter) WriteContentBlockDelta(index int, text string) error {
	event := StreamContentBlockDeltaEvent{
		Type:  ContentBlockDelta,
		Index: index,
		Delta: Delta{
			Content: text,
		},
	}
	return sw.WriteStreamEvent(ContentBlockDelta, event)
}

// WriteContentBlockStop 写入内容块结束事件
func (sw *StreamWriter) WriteContentBlockStop(index int) error {
	event := StreamContentBlockStopEvent{
		Type:  ContentBlockStop,
		Index: index,
	}
	return sw.WriteStreamEvent(ContentBlockStop, event)
}

// WriteMessageDelta 写入消息增量事件
func (sw *StreamWriter) WriteMessageDelta(stopReason string, usage *Usage) error {
	event := StreamMessageDeltaEvent{
		Type: MessageDeltaEvent,
		Delta: MessageDelta{
			StopReason: stopReason,
			Usage:      usage,
		},
	}
	return sw.WriteStreamEvent(MessageDeltaEvent, event)
}

// WriteMessageStop 写入消息结束事件
func (sw *StreamWriter) WriteMessageStop() error {
	event := StreamMessageStopEvent{
		Type: MessageStop,
	}
	return sw.WriteStreamEvent(MessageStop, event)
}

// WriteError 写入错误事件
func (sw *StreamWriter) WriteError(err error) error {
	errorData := map[string]string{
		"error": err.Error(),
	}
	return sw.WriteEvent("error", errorData)
}

// WriteDone 写入完成事件
func (sw *StreamWriter) WriteDone() error {
	return sw.WriteEvent("done", map[string]string{"status": "completed"})
}

// WriteDoneWithData 写入包含数据的完成事件
func (sw *StreamWriter) WriteDoneWithData(data any) error {
	return sw.WriteEvent("done", data)
}

// StreamHandler 处理Agent流式响应的回调函数
type StreamHandler struct {
	writer         *StreamWriter
	sessionID      string
	messageID      string
	contentStarted bool
	response       *ChatResponse // 保持兼容性
}

// NewStreamHandler 创建流式处理器
func NewStreamHandler(writer *StreamWriter, sessionID string) *StreamHandler {
	// 生成消息ID
	messageID := fmt.Sprintf("msg_%s_%d", sessionID, time.Now().UnixNano())

	return &StreamHandler{
		writer:         writer,
		sessionID:      sessionID,
		messageID:      messageID,
		contentStarted: false,
		response: &ChatResponse{
			SessionID: sessionID,
			Message:   "",
			Finished:  false,
		},
	}
}

// HandleStreamChunk 处理流式数据块
func (sh *StreamHandler) HandleStreamChunk(chunk schema.StreamChunk) error {
	// 处理第一个chunk时发送消息开始和内容块开始事件
	if !sh.contentStarted {
		// 发送消息开始事件
		if err := sh.writer.WriteMessageStart(sh.sessionID, sh.messageID); err != nil {
			manager.GetLogger().Error("Failed to write message start event", "error", err)
			return err
		}

		// 发送内容块开始事件
		if err := sh.writer.WriteContentBlockStart(0); err != nil {
			manager.GetLogger().Error("Failed to write content block start event", "error", err)
			return err
		}

		sh.contentStarted = true
	}

	// 根据Agent的流式输出格式处理
	if chunk.Chunk != nil {
		// 直接获取增量文本
		deltaText := chunk.Chunk.Text()
		if deltaText != "" {
			if err := sh.writer.WriteContentBlockDelta(0, deltaText); err != nil {
				manager.GetLogger().Error("Failed to write content block delta", "error", err)
				return err
			}
		}
	}

	return nil
}
