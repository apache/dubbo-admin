package server

import (
	"time"

	"dubbo-admin-ai/schema"

	"github.com/google/uuid"
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

// 转换函数
func ToThinkInput(req *ChatRequest) schema.ThinkInput {
	return schema.ThinkInput{
		Content: req.Message,
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return "req_" + uuid.New().String()
}
