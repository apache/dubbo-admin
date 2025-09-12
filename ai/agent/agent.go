package agent

import (
	"context"
	"dubbo-admin-ai/config"
	"dubbo-admin-ai/schema"
	"fmt"

	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
)

type Flow = *core.Flow[schema.Schema, schema.Schema, struct{}]

const (
	ThinkFlowName string = "think"
	ActFlowName   string = "act"
	ReActFlowName string = "reAct"
)

type Agent interface {
	Interact(schema.Schema) (schema.Schema, error)
	AgentFlow() Flow
}

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

func (s *Stage) Execute(ctx context.Context) (err error) {
	// 执行flow
	s.output, err = s.flow.Run(ctx, s.input)
	if err != nil {
		return err
	}

	return nil
}

type Orchestrator interface {
	ToFlow(*genkit.Genkit) Flow
	Run(context.Context, schema.Schema) (schema.Schema, error)
	RunStage(context.Context, string, schema.Schema) (schema.Schema, error)
}

type OrderOrchestrator struct {
	flow   Flow
	stages map[string]*Stage
	order  []string
}

func NewOrderOrchestrator(stages ...*Stage) *OrderOrchestrator {
	stagesMap := make(map[string]*Stage, len(stages))
	for s := range stages {
		stagesMap[stages[s].flow.Name()] = stages[s]
	}

	order := make([]string, 2)
	order[0] = ThinkFlowName
	order[1] = ActFlowName

	return &OrderOrchestrator{
		stages: stagesMap,
		order:  order,
	}
}

func (o *OrderOrchestrator) ToFlow(g *genkit.Genkit) Flow {
	if o.flow == nil {
		o.flow = genkit.DefineFlow(g, ReActFlowName, o.Run)
	}
	return o.flow
}

func (o *OrderOrchestrator) Run(ctx context.Context, userInput schema.Schema) (result schema.Schema, err error) {
	// 第一轮次时使用用户初始输入
	var input schema.Schema = userInput

	// 迭代执行，直到达到最大次数或者状态为 Finished
	for range config.MAX_REACT_ITERATIONS {
		for _, order := range o.order {
			// 执行当前阶段
			curStage := o.stages[order]
			curStage.input = input
			if err = curStage.Execute(ctx); err != nil {
				return nil, err
			}

			// 将当前阶段的输出作为下一个阶段的输入
			input = curStage.output

			//检查 LLM 是否返回了最终答案
			switch curStage.output.(type) {
			case schema.ThinkOutput:
				out := curStage.output.(schema.ThinkOutput)
				if out.Status == schema.Finished && out.FinalAnswer != "" {
					// TODO: 处理最终答案
					fmt.Printf("Final Answer: %s\n", out.FinalAnswer)
					result = curStage.output
					goto End
				}
				result = curStage.output
			}

		}
	}

End:
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
