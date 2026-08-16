package schema

import (
	"encoding/json"
	"fmt"
	"reflect"

	toolEngine "dubbo-admin-ai/component/tools/engine"

	"github.com/firebase/genkit/go/ai"
)

type UserInput struct {
	Content string `json:"content,omitempty"`
}

type ToolOutputs struct {
	Outputs   []toolEngine.ToolOutput `json:"tool_responses"`
	UsageInfo *ai.GenerationUsage     `json:"usage,omitempty" jsonschema_description:"DO NOT USE THIS FIELD, IT IS FOR INTERNAL USAGE ONLY"`
}

func (to *ToolOutputs) Add(output *toolEngine.ToolOutput) {
	to.Outputs = append(to.Outputs, *output)
}

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

// StreamFeedback carries one chunk of streaming output. Its content-block
// `index` is assigned by the owning Channels at send time (see Channels.Send),
// keeping it per-interaction and free of the cross-session interference and
// data races a shared global counter would cause.
type StreamFeedback struct {
	index int
	done  bool
	final *Observation
	text  string
}

func StreamEnd() *StreamFeedback {
	return &StreamFeedback{done: true, text: "", final: nil}
}

func StreamFinal(final *Observation) *StreamFeedback {
	return &StreamFeedback{done: true, text: "", final: final}
}

func (sf *StreamFeedback) Text() string {
	return sf.text
}

func (sf *StreamFeedback) Final() Schema {
	return sf.final
}

func NewStreamFeedback(text string) *StreamFeedback {
	return &StreamFeedback{
		done: false,
		text: text,
	}
}

func (sf *StreamFeedback) Index() int {
	return sf.index
}

// SetIndex assigns the content-block index. Called by Channels.Send so the
// counter lives with the single-consumer stream rather than a global.
func (sf *StreamFeedback) SetIndex(i int) {
	sf.index = i
}

func (sf *StreamFeedback) IsDone() bool {
	return sf.done && sf.text == ""
}

func (sf *StreamFeedback) IsFinal() bool {
	return sf.done && sf.text == "" && sf.final != nil
}
