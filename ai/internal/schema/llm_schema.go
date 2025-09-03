package schema

import (
	"dubbo-admin-ai/internal/tools"
	"encoding/json"
	"fmt"
)

// 表示 LLM 的响应结构
type ReActIn = ReActInput
type ReActOut = string

type ThinkIn = ReActInput
type ThinkOut = ThinkAggregation

type ActIn = ThinkOut
type ActOut = []string

type ReActInput struct {
	UserInput string `json:"userInput"`
	// 移除SessionID和SessionMemory，这些由Session管理
}

type Status string

const (
	Continued Status = "CONTINUED"
	Finished  Status = "FINISHED"
	Pending   Status = "PENDING"
)

type ToolRequest struct {
	Thought  string         `json:"thought"`
	ToolDesc tools.ToolDesc `json:"tool_desc"`
}

type ThinkAggregation struct {
	ToolRequests []ToolRequest `json:"tool_requests"`
	Status       Status        `json:"status" jsonschema:"enum=CONTINUED,enum=FINISHED,enum=PENDING"`
	Thought      string        `json:"thought"`
	FinalAnswer  string        `json:"final_answer,omitempty" jsonschema:"required=false"`
}

// ReActInput
func (a ReActInput) String() string {
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return fmt.Sprintf("ReActInput{error: %v}", err)
	}
	return string(data)
}

// ToolRequest
// HasAction 检查是否包含行动指令
func (tr ToolRequest) HasAction() bool {
	return tr.ToolDesc.ToolInput != nil && tr.ToolDesc.ToolName != ""
}

// ValidateToolDesc 验证行动指令的有效性
func (tr ToolRequest) ValidateToolDesc() error {
	if !tr.HasAction() {
		return fmt.Errorf("no valid action found")
	}

	if tr.ToolDesc.ToolName == "" {
		return fmt.Errorf("tool_name cannot be empty")
	}

	// 验证支持的工具名称
	supportedTools, err := tools.AllMockToolNames()
	if err != nil {
		return fmt.Errorf("failed to get supported tools: %w", err)
	}

	if _, ok := supportedTools[tr.ToolDesc.ToolName]; !ok {
		return fmt.Errorf("unsupported tool: %s", tr.ToolDesc.ToolName)
	}

	return nil
}

// String 实现 fmt.Stringer 接口，返回格式化的 JSON
func (tr ToolRequest) String() string {
	data, err := json.MarshalIndent(tr, "", "  ")
	if err != nil {
		return fmt.Sprintf("ToolRequest{error: %v}", err)
	}
	return string(data)
}

// ThinkAggregation
func (ta ThinkAggregation) IsStop() bool {
	return ta.Status == Finished && ta.FinalAnswer != ""
}

func (ta ThinkAggregation) String() string {
	data, err := json.MarshalIndent(ta, "", "  ")
	if err != nil {
		return fmt.Sprintf("ThinkAggregation{error: %v}", err)
	}
	return string(data)
}
