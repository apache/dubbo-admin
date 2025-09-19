package server

import (
	"time"

	"dubbo-admin-ai/schema"
)

// Response 统一API响应格式
type Response struct {
	Message   string `json:"message"`        // 响应消息
	Data      any    `json:"data,omitempty"` // 响应数据
	RequestID string `json:"request_id"`     // 请求ID，用于追踪
	Timestamp int64  `json:"timestamp"`      // 响应时间戳
}

// NewResponse 创建响应
func NewResponse(message string, data any) *Response {
	return &Response{
		Message:   message,
		Data:      data,
		RequestID: generateRequestID(),
		Timestamp: time.Now().Unix(),
	}
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data any) *Response {
	return NewResponse("success", data)
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(message string) *Response {
	return NewResponse(message, nil)
}

// ChatRequest 流式聊天请求
type ChatRequest struct {
	Message string `json:"message" binding:"required"` // 用户消息
}

// StreamEvent SSE流事件
type StreamEvent struct {
	Event string `json:"event"` // 事件类型: "message", "error", "done"
	Data  any    `json:"data"`  // 事件数据
}

// 转换函数
func ToThinkInput(req *ChatRequest) schema.ThinkInput {
	return schema.ThinkInput{
		Content: req.Message,
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return "req_" + time.Now().Format("20060102150405") + "_" + randomString(6)
}

// randomString 生成随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
