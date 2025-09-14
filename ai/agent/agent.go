package agent

import (
	"context"
	"dubbo-admin-ai/config"
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

type StreamFunc func(ctx context.Context, status schema.StreamChunk) error

const (
	ThinkFlowName string = "think"
	ActFlowName   string = "act"
	ReActFlowName string = "reAct"
)

type Agent interface {
	Interact(context.Context, schema.Schema) (schema.Schema, error)
}

type Stage struct {
	flow   any
	input  schema.Schema
	output schema.Schema
}

func NewStage(flow any, inputSchema schema.Schema, outputSchema schema.Schema) *Stage {
	return &Stage{
		flow:   flow,
		input:  inputSchema,
		output: outputSchema,
	}
}

func (s *Stage) Execute(ctx context.Context) (err error) {
	if value, ok := s.flow.(NormalFlow); ok {
		s.output, err = value.Run(ctx, s.input)
	} else if value, ok := s.flow.(StreamFlow); ok {
		streamOutFunc := value.Stream(ctx, s.input)
		streamOutFunc(func(val *core.StreamingFlowValue[schema.Schema, schema.StreamChunk], err error) bool {
			if err != nil {
				fmt.Printf("Stream error: %v\n", err)
				return false
			}
			if !val.Done {
				fmt.Print(val.Stream.Chunk.Text())
			} else if val.Output != nil {
				s.output = val.Output
			}

			return true
		})
	}

	return err
}

type Orchestrator interface {
	Run(context.Context, schema.Schema) (schema.Schema, error)
	RunStage(context.Context, string, schema.Schema) (schema.Schema, error)
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
	order[0] = ThinkFlowName
	order[1] = ActFlowName

	return &OrderOrchestrator{
		stages: stagesMap,
		order:  order,
	}
}

func (o *OrderOrchestrator) Run(ctx context.Context, userInput schema.Schema) (result schema.Schema, err error) {
	// Use user initial input for the first round
	var input schema.Schema = userInput

	// Iterate until reaching maximum iterations or status is Finished
Outer:
	for range config.MAX_REACT_ITERATIONS {
		for _, order := range o.order {
			// Execute current stage
			curStage := o.stages[order]
			curStage.input = input
			if err = curStage.Execute(ctx); err != nil {
				return nil, err
			}

			// Use current stage output as next stage input
			input = curStage.output

			// Check if LLM returned final answer
			switch curStage.output.(type) {
			case schema.ThinkOutput:
				out := curStage.output.(schema.ThinkOutput)
				if out.Status == schema.Finished && out.FinalAnswer != "" {
					// TODO: Process final answer
					fmt.Printf("Final Answer: %s\n", out.FinalAnswer)
					result = curStage.output
					break Outer

				}
				result = curStage.output
			}
		}
	}

	return result, nil
}

func (o *OrderOrchestrator) RunStage(ctx context.Context, key string, input schema.Schema) (output schema.Schema, err error) {
	stage, ok := o.stages[key]
	if !ok {
		return nil, fmt.Errorf("stage %s not found", key)
	}

	stage.input = input
	if err = stage.Execute(ctx); err != nil {
		return nil, err
	}
	return stage.output, nil
}
