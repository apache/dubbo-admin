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

type ThinkInput struct {
	UserInput     *UserInput          `json:"user_input,omitempty"`
	SessionID     string              `json:"session_id"`
	ToolResponses []tools.ToolOutput  `json:"tool_responses,omitempty"`
	Observation   *Observation        `json:"observation,omitempty"`
	UsageInfo     *ai.GenerationUsage `json:"usage,omitempty" jsonschema_description:"DO NOT USE THIS FIELD, IT IS FOR INTERNAL USAGE ONLY"`
}

func (i ThinkInput) Validate(T reflect.Type) error {
	if reflect.TypeOf(i) != T {
		return fmt.Errorf("ThinkInput: %v is not of type %v", i, T)
	}
	return nil
}

func (i ThinkInput) Usage() *ai.GenerationUsage {
	return i.UsageInfo
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
	UsageInfo      *ai.GenerationUsage `json:"usage,omitempty" jsonschema_description:"DO NOT USE THIS FIELD, IT IS FOR INTERNAL USAGE ONLY"`
}

func (ta ThinkOutput) Validate(T reflect.Type) error {
	if reflect.TypeOf(ta) != T {
		return fmt.Errorf("ThinkOutput: %v is not of type %v", ta, T)
	}
	return nil
}

func (ta ThinkOutput) Usage() *ai.GenerationUsage {
	return ta.UsageInfo
}

func (ta ThinkOutput) String() string {
	data, err := json.MarshalIndent(ta, "", "  ")
	if err != nil {
		return fmt.Sprintf("ThinkOutput{error: %v}", err)
	}
	return string(data)
}

type ToolOutputs struct {
	Outputs   []tools.ToolOutput  `json:"tool_responses"`
	Thought   string              `json:"thought,omitempty"`
	UsageInfo *ai.GenerationUsage `json:"usage,omitempty" jsonschema_description:"DO NOT USE THIS FIELD, IT IS FOR INTERNAL USAGE ONLY"`
}

func (to ToolOutputs) Validate(T reflect.Type) error {
	if reflect.TypeOf(to) != T {
		return fmt.Errorf("ToolRequest: %v is not of type %v", to, T)
	}
	return nil
}

func (to ToolOutputs) Usage() *ai.GenerationUsage {
	return to.UsageInfo
}

func (to *ToolOutputs) Add(output *tools.ToolOutput) {
	to.Outputs = append(to.Outputs, *output)
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
	UsageInfo   *ai.GenerationUsage `json:"usage,omitempty" jsonschema_description:"DO NOT USE THIS FIELD, IT IS FOR INTERNAL USAGE ONLY"`
}

func (o Observation) Validate(t reflect.Type) error {
	if reflect.TypeOf(o) != t {
		return fmt.Errorf("Observation: %v is not of type %v", o, t)
	}
	return nil
}

func (o Observation) Usage() *ai.GenerationUsage {
	return o.UsageInfo
}

func (o Observation) String() string {
	data, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return fmt.Sprintf("Observation{error: %v}", err)
	}
	return string(data)
}

func AccumulateUsage(dist *ai.GenerationUsage, src ...*ai.GenerationUsage) {
	if dist == nil {
		return
	}
	for _, s := range src {
		if s == nil {
			continue
		}
		if dist.Custom == nil {
			dist.Custom = make(map[string]float64)
		}
		for k, v := range s.Custom {
			dist.Custom[k] += v
		}
		dist.CachedContentTokens += s.CachedContentTokens
		dist.InputAudioFiles += s.InputAudioFiles
		dist.InputCharacters += s.InputCharacters
		dist.InputImages += s.InputImages
		dist.InputTokens += s.InputTokens
		dist.InputVideos += s.InputVideos
		dist.OutputAudioFiles += s.OutputAudioFiles
		dist.OutputCharacters += s.OutputCharacters
		dist.OutputImages += s.OutputImages
		dist.OutputTokens += s.OutputTokens
		dist.OutputVideos += s.OutputVideos
		dist.ThoughtsTokens += s.ThoughtsTokens
		dist.TotalTokens += s.TotalTokens
	}
}

var index = 0

type StreamFeedback struct {
	index int
	done  bool
	Text  string
}

func StreamEnd() *StreamFeedback {
	defer func() { index++ }()
	return &StreamFeedback{index: index, done: true, Text: ""}
}

func NewStreamFeedback(text string) *StreamFeedback {
	return &StreamFeedback{
		index: index,
		done:  false,
		Text:  text,
	}
}

func (sf *StreamFeedback) Index() int {
	return sf.index
}

func (sf *StreamFeedback) IsDone() bool {
	return sf.done
}
