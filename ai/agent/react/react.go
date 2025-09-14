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
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
)

type ThinkIn = schema.ThinkInput
type ThinkOut = schema.ThinkOutput
type ActIn = ThinkOut
type ActOut = schema.ToolOutputs

// ReActAgent implements Agent interface
type ReActAgent struct {
	registry     *genkit.Genkit
	orchestrator agent.Orchestrator
}

func Create(g *genkit.Genkit) *ReActAgent {
	prompt := BuildThinkPrompt(g)

	// thinkStage := agent.NewStage(think(g, prompt), schema.ThinkInput{}, schema.ThinkOutput{})
	actStage := agent.NewStage(act(g), schema.ThinkOutput{}, schema.ToolOutputs{})
	streamThinkStage := agent.NewStage(StreamThink(g, prompt), schema.ThinkInput{}, schema.ThinkOutput{})

	orchestrator := agent.NewOrderOrchestrator(streamThinkStage, actStage)

	return &ReActAgent{
		registry:     g,
		orchestrator: orchestrator,
	}
}

func (ra *ReActAgent) Interact(ctx context.Context, input schema.Schema) (output schema.Schema, err error) {
	return ra.orchestrator.Run(ctx, input)
}

func (ra *ReActAgent) StreamingInteract(ctx context.Context, input schema.Schema, streamCallback core.StreamCallback[schema.StreamChunk]) (output schema.Schema, err error) {
	return ra.orchestrator.Run(ctx, input)
}

func BuildThinkPrompt(registry *genkit.Genkit) ai.Prompt {
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

func streamFunc(cb core.StreamCallback[schema.StreamChunk]) ai.ModelStreamCallback {
	return func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
		if cb != nil {
			return cb(ctx, schema.StreamChunk{
				Chunk: chunk,
			})
		}
		return nil
	}
}

func StreamThink(g *genkit.Genkit, prompt ai.Prompt) agent.StreamFlow {
	return genkit.DefineStreamingFlow(g, agent.ThinkFlowName,
		func(ctx context.Context, in schema.Schema, cb core.StreamCallback[schema.StreamChunk]) (out schema.Schema, err error) {
			manager.GetLogger().Info("Thinking...", "input", in)
			defer func() {
				manager.GetLogger().Info("Think Done.", "output", out)
			}()

			history, ok := ctx.Value(memory.ChatHistoryKey).(*memory.History)
			if !ok {
				panic(fmt.Errorf("failed to get history from context"))
			}

			var resp *ai.ModelResponse
			// ai.WithStreaming() receives ai.ModelStreamCallback type callback function
			// This callback function is called when the model generates each raw streaming chunk, used for raw chunk processing

			// The passed cb is user-defined callback function for handling streaming data logic, such as printing
			if !history.IsEmpty() {
				resp, err = prompt.Execute(ctx,
					ai.WithInput(in),
					ai.WithMessages(history.AllHistory()...),
					ai.WithStreaming(streamFunc(cb)),
				)
			} else {
				resp, err = prompt.Execute(ctx,
					ai.WithInput(in),
					ai.WithStreaming(streamFunc(cb)),
				)
			}

			// Parse output
			var response ThinkOut
			err = resp.Output(&response)

			if err != nil {
				return out, fmt.Errorf("failed to parse agentThink prompt response: %w", err)
			}

			history.AddHistory(resp.History()...)
			return response, nil
		})
}

func think(g *genkit.Genkit, prompt ai.Prompt) agent.NormalFlow {
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

			// Parse output
			var response ThinkOut
			err = resp.Output(&response)

			if err != nil {
				return out, fmt.Errorf("failed to parse agentThink prompt response: %w", err)
			}

			history.AddHistory(resp.History()...)
			return response, nil
		})
}

// act: Core logic of the executor
func act(g *genkit.Genkit) agent.NormalFlow {
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

			// Execute tool calls
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
