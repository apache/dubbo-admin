package agent

import (
	"context"
	"errors"
	"fmt"

	"dubbo-admin-ai/config"
	"dubbo-admin-ai/memory"
	"dubbo-admin-ai/schema"

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
	IntentFlowName  string = "intent"
	ThinkFlowName   string = "think"
	ActFlowName     string = "act"
	ObserveFlowName string = "observe"
	ReActFlowName   string = "reAct"
)

type Agent interface {
	Interact(*schema.UserInput, string) *Channels
	GetMemory() *memory.History
}

type Channels struct {
	closed bool

	UserRespChan chan *schema.StreamFeedback
	FlowChan     chan schema.Schema
	ErrorChan    chan error
}

func NewChannels(bufferSize int) *Channels {
	return &Channels{
		closed:       false,
		UserRespChan: make(chan *schema.StreamFeedback, bufferSize),
		FlowChan:     make(chan schema.Schema, bufferSize),
		ErrorChan:    make(chan error, bufferSize),
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
	close(chans.UserRespChan)
	close(chans.FlowChan)
	close(chans.ErrorChan)
	chans = nil
}

type StageType int

const (
	BeforeLoop StageType = iota
	InLoop
	AfterLoop
)

type Stage struct {
	flow       any
	streamFunc StreamFunc
	Type       StageType
}

func NewStage(flow any, t StageType) *Stage {
	return &Stage{
		flow:       flow,
		streamFunc: nil,
		Type:       t,
	}
}

func NewStreamStage(flow any, t StageType, onStreaming func(*Channels, schema.StreamChunk) error, onDone func(*Channels, schema.Schema) error) (stage *Stage) {
	if onStreaming == nil || onDone == nil {
		panic("onStreaming, onDone and streamChan callbacks cannot be nil for streaming stage")
	}

	stage = NewStage(flow, t)
	stage.streamFunc = func(channels *Channels) StreamHandler {
		return func(val *core.StreamingFlowValue[schema.Schema, schema.StreamChunk], err error) bool {
			if err != nil {
				channels.ErrorChan <- err
				return false
			}
			if !val.Done {
				if err := onStreaming(channels, val.Stream); err != nil {
					channels.ErrorChan <- err
					return false
				}
			} else if val.Output != nil {
				if err := onDone(channels, val.Output); err != nil {
					channels.ErrorChan <- err
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
	Run(context.Context, schema.Schema, *Channels) error
	RunStage(context.Context, string, schema.Schema, *Channels) error
}

type OrderOrchestrator struct {
	stages     map[string]*Stage
	beforeLoop []string
	loop       []string
	afterLoop  []string
}

// The order of stages is the order in which they are executed
func NewOrderOrchestrator(stages ...*Stage) *OrderOrchestrator {
	stagesMap := make(map[string]*Stage, len(stages))
	loop := make([]string, 0, len(stages))
	beforeLoop := make([]string, 0, len(stages))
	afterLoop := make([]string, 0, len(stages))

	for _, stage := range stages {
		// Get the flow name through interface method
		var flowName string
		if nFlow, ok := stage.flow.(NormalFlow); ok {
			flowName = nFlow.Name()
		} else if sFlow, ok := stage.flow.(StreamFlow); ok {
			flowName = sFlow.Name()
		}
		stagesMap[flowName] = stage
		switch stage.Type {
		case BeforeLoop:
			beforeLoop = append(beforeLoop, flowName)
		case InLoop:
			loop = append(loop, flowName)
		case AfterLoop:
			afterLoop = append(afterLoop, flowName)
		}
	}

	return &OrderOrchestrator{
		stages:     stagesMap,
		loop:       loop,
		beforeLoop: beforeLoop,
		afterLoop:  afterLoop,
	}
}

func (orchestrator *OrderOrchestrator) Run(ctx context.Context, input schema.Schema, chans *Channels) (err error) {
	defer func() {
		if err != nil {
			chans.ErrorChan <- err
		}
	}()
	// Use user initial input for the first round
	if input == nil {
		return errors.New("userInput cannot be nil")
	}

	for _, key := range orchestrator.beforeLoop {
		curStage, ok := orchestrator.stages[key]
		if !ok {
			return fmt.Errorf("stage %s not found", key)
		}

		if err := curStage.Execute(ctx, chans, input); err != nil {
			return fmt.Errorf("failed to execute stage %s: %w", key, err)
		}
		output := <-chans.FlowChan

		input = output
	}

	// Iterate until reaching maximum iterations or status is Finished
	var finalOutput schema.Observation
Outer:
	for range config.MAX_REACT_ITERATIONS {
		for _, order := range orchestrator.loop {
			// Execute current stage
			curStage, ok := orchestrator.stages[order]
			if !ok {
				return fmt.Errorf("stage %s not found", order)
			}

			if err := curStage.Execute(ctx, chans, input); err != nil {
				return fmt.Errorf("failed to execute stage %s: %w", order, err)
			}
			output := <-chans.FlowChan

			// Check if LLM returned final answer
			if out, ok := output.(schema.Observation); ok {
				if !out.Heartbeat && out.FinalAnswer != "" {
					finalOutput = out
					break Outer
				}
			}
			// The output of current stage will be the input of the next stage
			if val, ok := output.(schema.Observation); ok {
				finalOutput = val
			}
			input = output
		}
	}
	chans.UserRespChan <- schema.StreamFinal(&finalOutput)

	for _, key := range orchestrator.afterLoop {
		curStage, ok := orchestrator.stages[key]
		if !ok {
			return fmt.Errorf("stage %s not found", key)
		}

		if err := curStage.Execute(ctx, chans, input); err != nil {
			return fmt.Errorf("failed to execute stage %s: %w", key, err)
		}
		output := <-chans.FlowChan

		input = output
	}

	return nil
}

func (orchestrator *OrderOrchestrator) RunStage(ctx context.Context, key string, input schema.Schema, chans *Channels) (err error) {
	defer func() {
		chans.Close()
		if err != nil {
			chans.ErrorChan <- err
		}
	}()
	stage, ok := orchestrator.stages[key]
	if !ok {
		return fmt.Errorf("stage %s not found", key)
	}
	if err := stage.Execute(ctx, chans, input); err != nil {
		return fmt.Errorf("failed to execute stage %s: %w", key, err)
	}
	return nil
}
