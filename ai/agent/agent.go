package agent

import (
	"context"
	"dubbo-admin-ai/config"
	"dubbo-admin-ai/internal/manager"
	"dubbo-admin-ai/internal/tools"
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
type StreamFunc = func(*Channels) StreamHandler

const (
	ThinkFlowName       string = "think"
	StreamThinkFlowName string = "stream_think"
	ActFlowName         string = "act"
	ReActFlowName       string = "reAct"
)

type Agent interface {
	Interact(schema.Schema) *Channels
}

type Channels struct {
	closed bool

	ReasoningChan   chan string
	UserRespChan    chan string
	ToolRespChan    chan *tools.ToolOutput
	FlowChan        chan schema.Schema
	StreamChunkChan chan *schema.StreamChunk
}

func NewChannels(bufferSize int) *Channels {
	return &Channels{
		closed:          false,
		ReasoningChan:   make(chan string, bufferSize),
		ToolRespChan:    make(chan *tools.ToolOutput, bufferSize),
		UserRespChan:    make(chan string, bufferSize),
		StreamChunkChan: make(chan *schema.StreamChunk, bufferSize),
		FlowChan:        make(chan schema.Schema, bufferSize),
	}
}

func (chans *Channels) Reset() {
	chans.closed = false
}

func (chans *Channels) Close() {
	chans.closed = true
}

func (chans *Channels) Closed() bool {
	return chans.closed
}

func (chans *Channels) Destroy() {
	close(chans.ReasoningChan)
	close(chans.ToolRespChan)
	close(chans.UserRespChan)
	close(chans.StreamChunkChan)
	close(chans.FlowChan)
	chans = nil
}

type Stage struct {
	flow       any
	streamFunc StreamFunc
}

func NewStage(flow any) *Stage {
	return &Stage{
		flow:       flow,
		streamFunc: nil,
	}
}

func NewStreamStage(flow any, onStreaming func(*Channels, schema.StreamChunk) error, onDone func(*Channels, schema.Schema) error) (stage *Stage) {
	if onStreaming == nil || onDone == nil {
		panic("onStreaming, onDone and streamChan callbacks cannot be nil for streaming stage")
	}

	stage = NewStage(flow)
	stage.streamFunc = func(channels *Channels) StreamHandler {
		return func(val *core.StreamingFlowValue[schema.Schema, schema.StreamChunk], err error) bool {
			if err != nil {
				manager.GetLogger().Error("error", "err", err)
				return false
			}

			if !val.Done {
				if err := onStreaming(channels, val.Stream); err != nil {
					manager.GetLogger().Error("Error when streaming", "err", err)
					return false
				}
			} else if val.Output != nil {
				if err := onDone(channels, val.Output); err != nil {
					manager.GetLogger().Error("Error when finalizing", "err", err)
					return false
				}
			}

			return true
		}
	}

	return stage
}

// Execute will receive an input and produce an output
func (s *Stage) Execute(ctx context.Context, chans *Channels, input schema.Schema) error {
	if value, ok := s.flow.(NormalFlow); ok {
		output, err := value.Run(ctx, input)
		if err != nil {
			return fmt.Errorf("error when running normal flow: %w", err)
		}
		chans.FlowChan <- output
	} else if value, ok := s.flow.(StreamFlow); ok {
		if chans == nil || s.streamFunc == nil {
			return fmt.Errorf("stream handler and channels cannot be nil for stream flow")
		}
		// the stream function will produce the output
		value.Stream(ctx, input)(s.streamFunc(chans))
	}

	return nil
}

type Orchestrator interface {
	Run(context.Context, schema.Schema, *Channels)
	RunStage(context.Context, string, schema.Schema, *Channels) (schema.Schema, error)
}

type OrderOrchestrator struct {
	stages map[string]*Stage
	order  []string
}

// The order of stages is the order in which they are executed
func NewOrderOrchestrator(stages ...*Stage) *OrderOrchestrator {
	stagesMap := make(map[string]*Stage, len(stages))
	order := make([]string, len(stages))
	for i, stage := range stages {
		// Get the flow name through interface method
		var flowName string
		if nFlow, ok := stage.flow.(NormalFlow); ok {
			flowName = nFlow.Name()
		} else if sFlow, ok := stage.flow.(StreamFlow); ok {
			flowName = sFlow.Name()
		}
		stagesMap[flowName] = stage
		order[i] = flowName
	}

	return &OrderOrchestrator{
		stages: stagesMap,
		order:  order,
	}
}

func (o *OrderOrchestrator) Run(ctx context.Context, input schema.Schema, chans *Channels) {
	// Use user initial input for the first round
	if input == nil {
		panic("userInput cannot be nil")
	}
	// Iterate until reaching maximum iterations or status is Finished
	for range config.MAX_REACT_ITERATIONS {
		for _, order := range o.order {
			// Execute current stage
			curStage, ok := o.stages[order]
			if !ok {
				panic(fmt.Errorf("stage %s not found", order))
			}

			if err := curStage.Execute(ctx, chans, input); err != nil {
				panic(fmt.Errorf("failed to execute stage %s: %w", order, err))
			}
			output := <-chans.FlowChan

			// Check if LLM returned final answer
			if out, ok := output.(schema.ThinkOutput); ok {
				if out.Status == schema.Finished && out.FinalAnswer != "" {
					chans.UserRespChan <- out.FinalAnswer
					return
				}
			}
			// The output of current stage will be the input of the next stage
			input = output
		}
	}
}

func (o *OrderOrchestrator) RunStage(ctx context.Context, key string, input schema.Schema, chans *Channels) (output schema.Schema, err error) {
	stage, ok := o.stages[key]
	if !ok {
		return nil, fmt.Errorf("stage %s not found", key)
	}
	if err = stage.Execute(ctx, chans, input); err != nil {
		return nil, err
	}
	return <-chans.FlowChan, nil
}
