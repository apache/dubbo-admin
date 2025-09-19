package agent

import (
	"context"
	"dubbo-admin-ai/config"
	"dubbo-admin-ai/internal/manager"
	"dubbo-admin-ai/schema"
	"fmt"

	"github.com/firebase/genkit/go/core"
)

type NoStream = struct{}
type StreamType interface {
	NoStream | schema.StreamChunk
}

type Flow = *core.Flow[schema.Schema, schema.Schema, any]
type NormalFlow = *core.Flow[schema.Schema, schema.Schema, NoStream]
type StreamFlow = *core.Flow[schema.Schema, schema.Schema, schema.StreamChunk]

type StreamHandler = func(*core.StreamingFlowValue[schema.Schema, schema.StreamChunk], error) bool

const (
	ThinkFlowName       string = "think"
	StreamThinkFlowName string = "stream_think"
	ActFlowName         string = "act"
	ReActFlowName       string = "reAct"
)

type Agent interface {
	Interact(schema.Schema) (chan *schema.StreamChunk, chan schema.Schema, error)
}

// TODO: Use input/Output channel to support async execution
type Stage struct {
	flow          any
	streamHandler StreamHandler

	Input      schema.Schema
	Output     schema.Schema
	StreamChan chan *schema.StreamChunk
	OutputChan chan schema.Schema
}

func NewStage(flow any, inputSchema schema.Schema, outputSchema schema.Schema) *Stage {
	return &Stage{
		flow:          flow,
		streamHandler: nil,
		Input:         inputSchema,
		Output:        outputSchema,
		// InputChan:     make(chan schema.Schema, config.STAGE_CHANNEL_BUFFER_SIZE),
		// OutputChan:    make(chan schema.Schema, config.STAGE_CHANNEL_BUFFER_SIZE),
	}
}

func NewStreamStage(flow any, inputSchema schema.Schema, outputSchema schema.Schema, onStreaming func(*Stage, schema.StreamChunk) error, onDone func(*Stage, schema.Schema) error) (stage *Stage) {
	if onStreaming == nil || onDone == nil {
		panic("onStreaming, onDone and streamChan callbacks cannot be nil for streaming stage")
	}

	stage = NewStage(flow, inputSchema, outputSchema)
	stage.streamHandler = func(val *core.StreamingFlowValue[schema.Schema, schema.StreamChunk], err error) bool {
		if err != nil {
			manager.GetLogger().Error("error", "err", err)
			return false
		}

		if !val.Done {
			if err := onStreaming(stage, val.Stream); err != nil {
				manager.GetLogger().Error("Error when streaming", "err", err)
				return false
			}
		} else if val.Output != nil {
			if err := onDone(stage, val.Output); err != nil {
				manager.GetLogger().Error("Error when finalizing", "err", err)
				return false
			}
		}

		return true
	}

	return stage
}

func (s *Stage) Execute(ctx context.Context, streamChan chan *schema.StreamChunk, outputChan chan schema.Schema) (err error) {
	if value, ok := s.flow.(NormalFlow); ok {
		s.Output, err = value.Run(ctx, s.Input)
	} else if value, ok := s.flow.(StreamFlow); ok {
		if streamChan == nil || outputChan == nil || s.streamHandler == nil {
			return fmt.Errorf("stream handler and channels cannot be nil for stream flow")
		}
		s.StreamChan = streamChan
		s.OutputChan = outputChan
		value.Stream(ctx, s.Input)(s.streamHandler)
	}

	return err
}

type Orchestrator interface {
	Run(context.Context, schema.Schema, chan *schema.StreamChunk, chan schema.Schema)
	RunStage(context.Context, string, schema.Schema, chan *schema.StreamChunk, chan schema.Schema) (schema.Schema, error)
}

type OrderOrchestrator struct {
	stages map[string]*Stage
	order  []string
}

func NewOrderOrchestrator(stages ...*Stage) *OrderOrchestrator {
	stagesMap := make(map[string]*Stage, len(stages))
	for _, stage := range stages {
		// Get the flow name through interface method
		var flowName string
		if nFlow, ok := stage.flow.(NormalFlow); ok {
			flowName = nFlow.Name()
		} else if sFlow, ok := stage.flow.(StreamFlow); ok {
			flowName = sFlow.Name()
		}
		stagesMap[flowName] = stage
	}

	order := make([]string, 2)
	order[0] = StreamThinkFlowName
	order[1] = ActFlowName

	return &OrderOrchestrator{
		stages: stagesMap,
		order:  order,
	}
}

func (o *OrderOrchestrator) Run(ctx context.Context, userInput schema.Schema, streamChan chan *schema.StreamChunk, outputChan chan schema.Schema) {
	defer func() {
		close(streamChan)
		close(outputChan)
	}()

	// Use user initial input for the first round
	if userInput == nil {
		panic("userInput cannot be nil")
	}
	var input schema.Schema = userInput
	// Iterate until reaching maximum iterations or status is Finished
Outer:
	for range config.MAX_REACT_ITERATIONS {
		for _, order := range o.order {
			// Execute current stage
			curStage, ok := o.stages[order]
			if !ok {
				panic(fmt.Errorf("stage %s not found", order))
			}

			curStage.Input = input
			if curStage.Input == nil {
				panic(fmt.Errorf("stage %s input is nil", order))
			}

			if err := curStage.Execute(ctx, streamChan, outputChan); err != nil {
				panic(fmt.Errorf("failed to execute stage %s: %w", order, err))
			}

			// Use current stage Output as next stage input
			if curStage.Output == nil {
				panic(fmt.Errorf("stage %s output is nil", order))
			}
			input = curStage.Output

			// Check if LLM returned final answer
			switch curStage.Output.(type) {
			case schema.ThinkOutput:
				out := curStage.Output.(schema.ThinkOutput)
				if out.Status == schema.Finished && out.FinalAnswer != "" {
					break Outer
				}
			}
		}
	}
}

func (o *OrderOrchestrator) RunStage(ctx context.Context, key string, input schema.Schema, streamChan chan *schema.StreamChunk, outputChan chan schema.Schema) (output schema.Schema, err error) {
	stage, ok := o.stages[key]
	if !ok {
		return nil, fmt.Errorf("stage %s not found", key)
	}

	stage.Input = input
	if err = stage.Execute(ctx, streamChan, outputChan); err != nil {
		return nil, err
	}
	return stage.Output, nil
}
