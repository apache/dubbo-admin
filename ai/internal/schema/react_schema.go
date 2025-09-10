package schema

import (
	"dubbo-admin-ai/internal/tools"
	"encoding/json"
	"fmt"
	"reflect"
)

type ReActInput struct {
	UserInput string `json:"userInput"`
}

func (ri ReActInput) Validate(T reflect.Type) error {
	if reflect.TypeOf(ri) != T {
		return fmt.Errorf("ReActInput: %v is not of type %v", ri, T)
	}
	return nil
}

// ReActInput
func (ri ReActInput) String() string {
	data, err := json.MarshalIndent(ri, "", "  ")
	if err != nil {
		return fmt.Sprintf("ReActInput{error: %v}", err)
	}
	return string(data)
}

type Status string

const (
	Continued Status = "CONTINUED"
	Finished  Status = "FINISHED"
	Pending   Status = "PENDING"
)

type ThinkAggregation struct {
	ToolInput   []tools.ToolInput `json:"tool_requests"`
	Status      Status            `json:"status" jsonschema:"enum=CONTINUED,enum=FINISHED,enum=PENDING"`
	Thought     string            `json:"thought"`
	FinalAnswer string            `json:"final_answer,omitempty" jsonschema:"required=false"`
}

func (ta ThinkAggregation) Validate(T reflect.Type) error {
	if reflect.TypeOf(ta) != T {
		return fmt.Errorf("ThinkAggregation: %v is not of type %v", ta, T)
	}
	return nil
}

func (ta ThinkAggregation) IsStop() bool {
	return ta.Status == Finished && ta.FinalAnswer != ""
}

func (ta ThinkAggregation) String() string {
	data, err := json.MarshalIndent(ta, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("ThinkAggregation{error: %v}", err))
	}
	return string(data)
}

type ToolRequest struct {
	Thought   string          `json:"thought"`
	ToolInput tools.ToolInput `json:"tool_input"`
}

func (tr ToolRequest) Validate(T reflect.Type) error {
	if reflect.TypeOf(tr) != T {
		return fmt.Errorf("ToolRequest: %v is not of type %v", tr, T)
	}
	return nil
}

func (tr ToolRequest) HasAction() bool {
	return tr.ToolInput.Parameter != nil && tr.ToolInput.ToolName != ""
}

func (tr ToolRequest) String() string {
	data, err := json.MarshalIndent(tr, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("ToolRequest{error: %v}", err))
	}
	return string(data)
}

type ToolOutputs struct {
	outputs []tools.ToolOutput
}

func (to ToolOutputs) Validate(T reflect.Type) error {
	if reflect.TypeOf(to) != T {
		return fmt.Errorf("ToolRequest: %v is not of type %v", to, T)
	}
	return nil
}

func (to *ToolOutputs) Add(output *tools.ToolOutput) {
	to.outputs = append(to.outputs, *output)
}

func (to ToolOutputs) String() string {
	var str string
	for _, output := range to.outputs {
		str += output.String() + "\n"
	}
	return fmt.Sprintf("ToolOutputs{outputs: %s}", str)
}
