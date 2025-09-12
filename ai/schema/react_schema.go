package schema

import (
	"dubbo-admin-ai/internal/tools"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

type Status string

const (
	Continued Status = "CONTINUED"
	Finished  Status = "FINISHED"
	// Pending   Status = "PENDING"
)

const UserThinkPromptTemplate = `input: 
{{#if content}} content: {{content}} {{/if}} 
{{#if tool_responses}} tool_responses: {{tool_responses}} {{/if}}
{{#if thought}} thought: {{thought}} {{/if}}
`

type ThinkInput struct {
	Content       string             `json:"content,omitempty"`
	ToolResponses []tools.ToolOutput `json:"tool_responses,omitempty"`
}

func (i ThinkInput) Validate(T reflect.Type) error {
	if reflect.TypeOf(i) != T {
		return fmt.Errorf("ThinkInput: %v is not of type %v", i, T)
	}
	return nil
}

func (i ThinkInput) String() string {
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return fmt.Sprintf("ThinkInput{error: %v}", err)
	}
	return string(data)
}

type UserInput struct {
	Content string `json:"content"`
}

func (i UserInput) Validate(T reflect.Type) error {
	if reflect.TypeOf(i) != T {
		return fmt.Errorf("UserInput: %v is not of type %v", i, T)
	}
	return nil
}

// UserInput
func (i UserInput) String() string {
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return fmt.Sprintf("UserInput{error: %v}", err)
	}
	return string(data)
}

type ThinkOutput struct {
	ToolRequests []tools.ToolInput `json:"tool_requests,omitempty"`
	Thought      string            `json:"thought"`
	Status       Status            `json:"status,omitempty" jsonschema:"enum=CONTINUED,enum=FINISHED"`
	FinalAnswer  string            `json:"final_answer,omitempty" jsonschema:"required=false"`
}

func (ta ThinkOutput) Validate(T reflect.Type) error {
	if reflect.TypeOf(ta) != T {
		return fmt.Errorf("ThinkOutput: %v is not of type %v", ta, T)
	}
	return nil
}

func (ta ThinkOutput) String() string {
	data, err := json.MarshalIndent(ta, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("ThinkOutput{error: %v}", err))
	}
	return string(data)
}

type ToolOutputs struct {
	Outputs []tools.ToolOutput `json:"tool_responses"`
}

func (to ToolOutputs) Validate(T reflect.Type) error {
	if reflect.TypeOf(to) != T {
		return fmt.Errorf("ToolRequest: %v is not of type %v", to, T)
	}
	return nil
}

func (to *ToolOutputs) Add(output *tools.ToolOutput) {
	to.Outputs = append(to.Outputs, *output)
}

func (to ToolOutputs) String() string {
	if len(to.Outputs) == 0 {
		return "ToolOutputs[\n  <no outputs>\n]"
	}

	result := "ToolOutputs[\n"
	for i, output := range to.Outputs {
		// Indent each ToolOutput
		outputStr := output.String()
		lines := strings.Split(outputStr, "\n")
		for j, line := range lines {
			if j == len(lines)-1 && line == "" {
				continue // Skip empty last line
			}
			result += "  " + line + "\n"
		}

		// Add separator between outputs except for the last one
		if i < len(to.Outputs)-1 {
			result += "\n"
		}
	}
	result += "]"
	return result
}
