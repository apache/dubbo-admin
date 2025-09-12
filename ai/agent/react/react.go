package react

import (
	"context"
	"dubbo-admin-ai/agent"
	"dubbo-admin-ai/config"
	"dubbo-admin-ai/internal/manager"
	"dubbo-admin-ai/internal/memory"
	"dubbo-admin-ai/internal/tools"
	"dubbo-admin-ai/schema"
	"fmt"
	"os"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type ThinkIn = schema.ThinkInput
type ThinkOut = schema.ThinkOutput
type ActIn = ThinkOut
type ActOut = schema.ToolOutputs

// ReActAgent 实现Agent接口
type ReActAgent struct {
	registry     *genkit.Genkit
	orchestrator agent.Orchestrator
}

func Create(registry *genkit.Genkit) (react *ReActAgent) {
	if registry == nil {
		panic(fmt.Errorf("cannot create reAct agent: registry is nil"))
	}

	thinkPrompt := buildThinkPrompt(registry)

	// Create orchestrator with executable stages
	thinkStage := agent.NewStage(think(registry, thinkPrompt), ThinkIn{}, ThinkOut{})
	actStage := agent.NewStage(act(registry), ActIn{}, ActOut{})
	orchestrator := agent.NewOrderOrchestrator(thinkStage, actStage)

	react = &ReActAgent{
		registry:     registry,
		orchestrator: orchestrator,
	}
	react.AgentFlow(registry)

	return react
}

func (ra *ReActAgent) Interact(ctx context.Context, input schema.Schema) (output schema.Schema, err error) {
	return ra.orchestrator.Run(ctx, input)
}

func (ra *ReActAgent) AgentFlow(registry *genkit.Genkit) agent.Flow {
	return ra.orchestrator.ToFlow(registry)
}

func buildThinkPrompt(registry *genkit.Genkit) ai.Prompt {
	// Load system prompt from filesystem
	data, err := os.ReadFile(config.PROMPT_DIR_PATH + "/agentSystem.prompt")
	if err != nil {
		panic(fmt.Errorf("failed to read agentSystem prompt: %w", err))
	}
	systemPromptText := string(data)

	// Build and Register ReAct think prompt
	mockToolManager := tools.NewMockToolManager(registry)
	toolRefs, err := mockToolManager.AllToolRefs()
	if err != nil {
		panic(fmt.Errorf("failed to get mock mock_tools: %v", err))
	}
	return genkit.DefinePrompt(registry, "agentThinking",
		ai.WithSystem(systemPromptText),
		ai.WithInputType(ThinkIn{}),
		ai.WithOutputType(ThinkOut{}),
		ai.WithPrompt(schema.UserThinkPromptTemplate),
		ai.WithTools(toolRefs...),
		ai.WithReturnToolRequests(true),
	)
}

func think(g *genkit.Genkit, prompt ai.Prompt) agent.Flow {
	return genkit.DefineFlow(g, agent.ThinkFlowName,
		func(ctx context.Context, in schema.Schema) (out schema.Schema, err error) {
			manager.GetLogger().Info("Thinking...", "input", in)
			defer func() {
				manager.GetLogger().Info("Think Done.", "output", out)
			}()

			history, ok := ctx.Value(memory.ChatHistoryKey).(*memory.History)
			if !ok {
				panic(fmt.Errorf("failed to get history from context"))
			}

			var resp *ai.ModelResponse
			if !history.IsEmpty() {
				resp, err = prompt.Execute(ctx,
					ai.WithInput(in),
					ai.WithMessages(history.AllHistory()...),
				)
			} else {
				resp, err = prompt.Execute(ctx,
					ai.WithInput(in),
				)
			}

			if err != nil {
				return nil, fmt.Errorf("failed to execute agentThink prompt: %w", err)
			}
			// 解析输出
			var response ThinkOut
			err = resp.Output(&response)

			if err != nil {
				return out, fmt.Errorf("failed to parse agentThink prompt response: %w", err)
			}

			history.AddHistory(resp.History()...)
			return response, nil
		})
}

// ----------------------------------------------------------------------------
// act: 执行者的核心逻辑
func act(g *genkit.Genkit) agent.Flow {
	return genkit.DefineFlow(g, agent.ActFlowName,
		func(ctx context.Context, in schema.Schema) (out schema.Schema, err error) {
			manager.GetLogger().Info("Acting...", "input", in)
			defer func() {
				manager.GetLogger().Info("Act Done.", "output", out)
			}()

			input, ok := in.(ActIn)
			if !ok {
				return nil, fmt.Errorf("input is not of type ActIn, got %T", in)
			}

			var actOuts ActOut

			// 执行工具调用
			history, ok := ctx.Value(memory.ChatHistoryKey).(*memory.History)
			if !ok {
				panic(fmt.Errorf("failed to get history from context"))
			}

			var parts []*ai.Part
			for _, req := range input.ToolRequests {
				output, err := req.Call(g, ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to call tool %s: %w", req.ToolName, err)
				}

				parts = append(parts,
					ai.NewToolResponsePart(&ai.ToolResponse{
						Name:   req.ToolName,
						Ref:    req.ToolName,
						Output: output.Map(),
					}))

				actOuts.Add(&output)
			}

			history.AddHistory(ai.NewMessage(ai.RoleTool, nil, parts...))

			return actOuts, nil
		})
}
