package agent

import (
	"context"
	"dubbo-admin-ai/internal/schema"
	"fmt"

	"github.com/firebase/genkit/go/core"
)

type Flow = *core.Flow[schema.Schema, schema.Schema, struct{}]

type Stage struct {
	flow   Flow
	input  schema.Schema
	output schema.Schema
}

func NewStage(flow Flow, inputSchema schema.Schema, outputSchema schema.Schema) *Stage {
	return &Stage{
		flow:   flow,
		input:  inputSchema,
		output: outputSchema,
	}
}

func (s *Stage) Execute() (err error) {
	// 执行flow
	s.output, err = s.flow.Run(context.Background(), s.input)
	if err != nil {
		return err
	}

	return nil
}

type Orchestrator struct {
	stages map[string]*Stage
}

func NewOrchestrator(stages ...*Stage) *Orchestrator {
	stagesMap := make(map[string]*Stage, len(stages))
	for s := range stages {
		stagesMap[stages[s].flow.Name()] = stages[s]
	}
	return &Orchestrator{stages: stagesMap}
}

func (p *Orchestrator) Run() error {
	for _, stage := range p.stages {
		if stage.input == nil {
			return fmt.Errorf("stage input is nil")
		}
		if err := stage.Execute(); err != nil {
			return err
		}
	}
	return nil
}

func (o *Orchestrator) RunStage(key string, input schema.Schema) (output schema.Schema, err error) {
	if _, ok := o.stages[key]; !ok {
		return nil, fmt.Errorf("stage %s not found", key)
	}

	o.stages[key].input = input

	if err = o.stages[key].Execute(); err != nil {
		return nil, err
	}

	return o.stages[key].output, nil
}
