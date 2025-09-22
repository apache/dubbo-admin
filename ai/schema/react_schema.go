package schema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"dubbo-admin-ai/tools"

	"github.com/firebase/genkit/go/ai"
)

// StreamChunk represents streaming status information for ReAct Agent
type StreamChunk struct {
	Stage string                 `json:"stage"` // "think" | "act"
	Index int                    `json:"index"`
	Chunk *ai.ModelResponseChunk `json:"chunk"`
}

var (
	UserPromptTemplate = `input: 
{{#if content}} content: {{content}} {{/if}} 
{{#if tool_responses}} tool_responses: {{tool_responses}} {{/if}}
{{#if thought}} thought: {{thought}} {{/if}}
`
)

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

type TimeContext string

const (
	Immediate         TimeContext = "IMMEDIATE"
	Recent            TimeContext = "RECENT"
	Continuous        TimeContext = "CONTINUOUS"
	SpecificTimeframe TimeContext = "SPECIFIC_TIMEFRAME"
	UnspecifiedTime   TimeContext = "UNSPECIFIED_TIME"
)

type SeverityLevel string

const (
	Critical      SeverityLevel = "CRITICAL"
	High          SeverityLevel = "HIGH"
	Medium        SeverityLevel = "MEDIUM"
	Low           SeverityLevel = "LOW"
	Informational SeverityLevel = "INFORMATIONAL"
)

type InvestigationPriority string

const (
	PriorityHigh   InvestigationPriority = "HIGH"
	PriorityMedium InvestigationPriority = "MEDIUM"
	PriorityLow    InvestigationPriority = "LOW"
)

type Intent struct {
	PrimaryIntent         PrimaryIntent         `json:"primary_intent"`
	TargetServices        []string              `json:"target_services"`
	MetricsOfInterest     []string              `json:"metrics_of_interest"`
	TimeContext           TimeContext           `json:"time_context"`
	SeverityLevel         SeverityLevel         `json:"severity_level"`
	Keywords              []string              `json:"keywords"`
	ConfidenceScore       float64               `json:"confidence_score"`
	InvestigationPriority InvestigationPriority `json:"investigation_priority"`
	SuggestedTools        []string              `json:"suggested_tools"`
	Reasoning             string                `json:"reasoning"`
}

func (i Intent) Validate(t reflect.Type) error {
	if reflect.TypeOf(i) != t {
		return fmt.Errorf("Intent: %v is not of type %v", i, t)
	}
	return nil
}

type Observation struct {
	Content string `json:"content"`
}

func (u Observation) Validate(t reflect.Type) error {
	if reflect.TypeOf(u) != t {
		return fmt.Errorf("Observation: %v is not of type %v", u, t)
	}
	return nil
}

type ThinkInput struct {
	Content       any                `json:"content,omitempty"`
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

type ThinkOutput struct {
	ToolRequests []tools.Tool        `json:"tool_requests,omitempty"`
	Thought      string              `json:"thought"`
	Status       Status              `json:"status,omitempty" jsonschema:"enum=CONTINUED,enum=FINISHED"`
	FinalAnswer  string              `json:"final_answer,omitempty" jsonschema:"required=false"`
	Usage        *ai.GenerationUsage `json:"usage,omitempty" jsonschema_description:"DO NOT SET THIS FIELD"`
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
