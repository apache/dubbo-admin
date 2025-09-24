package schema

import (
	"encoding/json"
	"fmt"
	"reflect"

	"dubbo-admin-ai/tools"

	"github.com/firebase/genkit/go/ai"
)

// StreamChunk represents streaming status information for ReAct Agent
type StreamChunk struct {
	Stage string                 `json:"stage"` // "think" | "act"
	Index int                    `json:"index"`
	Chunk *ai.ModelResponseChunk `json:"chunk"`
}

type UserInput struct {
	Content string `json:"content,omitempty"`
}

func (u UserInput) Validate(t reflect.Type) error {
	if reflect.TypeOf(u) != t {
		return fmt.Errorf("UserInput: %v is not of type %v", u, t)
	}
	return nil
}

type PrimaryIntent string

const (
	PerformanceInvestigation PrimaryIntent = "PERFORMANCE_INVESTIGATION"
	ErrorDiagnosis           PrimaryIntent = "ERROR_DIAGNOSIS"
	HealthCheck              PrimaryIntent = "HEALTH_CHECK"
	ResourceMonitoring       PrimaryIntent = "RESOURCE_MONITORING"
	TrafficAnalysis          PrimaryIntent = "TRAFFIC_ANALYSIS"
	ServiceDependency        PrimaryIntent = "SERVICE_DEPENDENCY"
	AlertingInvestigation    PrimaryIntent = "ALERTING_INVESTIGATION"
	GeneralInquiry           PrimaryIntent = "GENERAL_INQUIRY"
)

type Observation struct {
	Summary     string              `json:"summary"`
	Heartbeat   bool                `json:"heartbeat"`
	FinalAnswer string              `json:"final_answer,omitempty" jsonschema:"required=false"`
	Focus       string              `json:"focus,omitempty"`
	Evidence    string              `json:"evidence,omitempty"`
	Usage       *ai.GenerationUsage `json:"usage,omitempty" jsonschema_description:"DO NOT SET THIS FIELD"`
}

func (o Observation) Validate(t reflect.Type) error {
	if reflect.TypeOf(o) != t {
		return fmt.Errorf("Observation: %v is not of type %v", o, t)
	}
	return nil
}

func (o Observation) String() string {
	data, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return fmt.Sprintf("Observation{error: %v}", err)
	}
	return string(data)
}

type ThinkInput struct {
	UserInput     *UserInput         `json:"user_input,omitempty"`
	ToolResponses []tools.ToolOutput `json:"tool_responses,omitempty"`
	Observation   *Observation       `json:"observation,omitempty"`
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

type ThinkOutput struct {
	Thought        string              `json:"thought"`
	Intent         PrimaryIntent       `json:"intent,omitempty"`
	TargetServices []string            `json:"target_services,omitempty"`
	SuggestedTools []string            `json:"suggested_tools,omitempty"`
	Usage          *ai.GenerationUsage `json:"usage,omitempty" jsonschema_description:"DO NOT SET THIS FIELD"`
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
		return fmt.Sprintf("ThinkOutput{error: %v}", err)
	}
	return string(data)
}

type ToolOutputs struct {
	Outputs []tools.ToolOutput  `json:"tool_responses"`
	Usage   *ai.GenerationUsage `json:"usage,omitempty" jsonschema_description:"DO NOT SET THIS FIELD"`
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

var index = 0

func ResetIndex() {
	index = 0
}

func IncreaseIndex() {
	index++
}

type StreamFeedback struct {
	index *int
	Text  string
}

func NewStreamFeedback(text string) *StreamFeedback {
	return &StreamFeedback{
		index: &index,
		Text:  text,
	}
}

func (sf *StreamFeedback) Index() int {
	return *sf.index
}
