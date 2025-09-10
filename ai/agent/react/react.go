package react

import (
	"context"
	"dubbo-admin-ai/agent"
	"dubbo-admin-ai/config"
	"dubbo-admin-ai/internal/schema"
	"dubbo-admin-ai/internal/tools"
	"fmt"
	"os"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

const (
	thinkFlowName string = "think"
	actFlowName   string = "act"
)

type ThinkIn = schema.ReActInput
type ThinkOut = schema.ThinkAggregation
type ActIn = ThinkOut
type ActOut = schema.ToolOutputs

// ReActAgent 实现Agent接口
type ReActAgent struct {
	registry     *genkit.Genkit
	orchestrator *agent.Orchestrator
}

func Create(registry *genkit.Genkit) (*ReActAgent, error) {
	if registry == nil {
		return nil, fmt.Errorf("cannot create reAct agent: registry is nil")
	}

	thinkPrompt, err := buildThinkPrompt(registry)
	if err != nil {
		return nil, fmt.Errorf("failed to build think prompt: %w", err)
	}

	// Create orchestrator with executable stages
	thinkStage := agent.NewStage(think(registry, thinkPrompt), ThinkIn{}, ThinkOut{})
	actStage := agent.NewStage(act(registry), ActIn{}, ActOut{})

	return &ReActAgent{
		registry:     registry,
		orchestrator: agent.NewOrchestrator(thinkStage, actStage),
	}, nil
}

func buildThinkPrompt(registry *genkit.Genkit) (*ai.Prompt, error) {
	// Load system prompt from filesystem
	data, err := os.ReadFile(config.PROMPT_DIR_PATH + "/agentSystem.prompt")
	if err != nil {
		return nil, fmt.Errorf("failed to read agentSystem prompt: %w", err)
	}
	systemPromptText := string(data)

	// Build and Register ReAct think prompt
	mockToolManager := tools.NewMockToolManager(registry)
	toolRefs, err := mockToolManager.AllToolRefs()
	if err != nil {
		return nil, fmt.Errorf("failed to get mock mock_tools: %v", err)
	}
	return genkit.DefinePrompt(registry, "agentThinking",
		ai.WithSystem(systemPromptText),
		ai.WithInputType(ThinkIn{}),
		ai.WithOutputType(ThinkOut{}),
		ai.WithPrompt("{{userInput}}"),
		ai.WithTools(toolRefs...),
	)
}

func think(g *genkit.Genkit, prompt *ai.Prompt) agent.Flow {
	return genkit.DefineFlow(g, thinkFlowName,
		func(ctx context.Context, in schema.Schema) (out schema.Schema, err error) {
			input, ok := in.(ThinkIn)
			if !ok {
				return nil, fmt.Errorf("input is not of type ThinkIn, got %T", in)
			}

			resp, err := prompt.Execute(ctx,
				ai.WithInput(input),
			)

			if err != nil {
				return nil, fmt.Errorf("failed to execute agentThink prompt: %w", err)
			}

			// 解析输出
			var response ThinkOut
			err = resp.Output(&response)
			if err != nil {
				return nil, fmt.Errorf("failed to parse agentThink prompt response: %w", err)
			}

			return response, nil
		})
}

// ----------------------------------------------------------------------------
// act: 执行者的核心逻辑
// TODO: Genkit 的 Tool 设计不能保证任意 LLM 一定能执行工具调用，是否考虑设计成所有模型都能执行？
// 一种思路是让 LLM 直接生成工具调用的结构化描述，然后由适配器执行。
// 管线、流水线的思想能否用在这里？
func act(g *genkit.Genkit) agent.Flow {
	return genkit.DefineFlow(g, actFlowName,
		func(ctx context.Context, in schema.Schema) (out schema.Schema, err error) {
			input, ok := in.(ActIn)
			if !ok {
				return nil, fmt.Errorf("input is not of type ActIn, got %T", in)
			}

			var actOuts ActOut

			// 执行工具调用
			for _, resp := range input.ToolInput {
				output, err := resp.Call(g, ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to call tool %s: %w", resp.ToolName, err)
				}
				actOuts.Add(&output)
			}

			return actOuts, nil
		})
}
