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
	memoryCtx    context.Context
	orchestrator agent.Orchestrator
}

func Create(g *genkit.Genkit) *ReActAgent {
	prompt := buildThinkPrompt(g)
	thinkStage := agent.NewStreamStage(
		streamThink(g, prompt),
		schema.ThinkInput{},
		schema.ThinkOutput{},
		func(stage *agent.Stage, chunk schema.StreamChunk) error {
			if stage.StreamChan == nil {
				return fmt.Errorf("streamChan is nil")
			}
			stage.StreamChan <- &chunk
			return nil
		},
		func(stage *agent.Stage, output schema.Schema) error {
			if stage.OutputChan == nil {
				return fmt.Errorf("outputChan is nil")
			}
			stage.Output = output
			stage.OutputChan <- output
			return nil
		},
	)
	actStage := agent.NewStage(act(g), schema.ThinkOutput{}, schema.ToolOutputs{})

	orchestrator := agent.NewOrderOrchestrator(thinkStage, actStage)

	return &ReActAgent{
		registry:     g,
		orchestrator: orchestrator,
		memoryCtx:    memory.NewMemoryContext(memory.ChatHistoryKey),
	}
}

func (ra *ReActAgent) Interact(input schema.Schema) (chan *schema.StreamChunk, chan schema.Schema, error) {
	streamChan := make(chan *schema.StreamChunk, config.STAGE_CHANNEL_BUFFER_SIZE)
	outputChan := make(chan schema.Schema, config.STAGE_CHANNEL_BUFFER_SIZE)
	go func() {
		ra.orchestrator.Run(ra.memoryCtx, input, streamChan, outputChan)
	}()
	return streamChan, outputChan, nil
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

func rawChunkHandler(cb core.StreamCallback[schema.StreamChunk]) ai.ModelStreamCallback {
	return func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
		if cb != nil {
			return cb(ctx, schema.StreamChunk{
				Stage: "think",
				Chunk: chunk,
			})
		}
		return nil
	}
}

func streamThink(g *genkit.Genkit, prompt ai.Prompt) agent.StreamFlow {
	return genkit.DefineStreamingFlow(g, agent.StreamThinkFlowName,
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
					ai.WithStreaming(rawChunkHandler(cb)),
				)
			} else {
				resp, err = prompt.Execute(ctx,
					ai.WithInput(in),
					ai.WithStreaming(rawChunkHandler(cb)),
				)
			}
			if err != nil {
				panic(fmt.Errorf("failed to execute agentThink prompt: %w", err))
			}

			// Parse output
			var response ThinkOut
			err = resp.Output(&response)
			if err != nil {
				panic(fmt.Errorf("failed to parse agentThink prompt response: %w", err))
			}
			response.Usage = resp.Usage
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
				return out, fmt.Errorf("failed to execute agentThink prompt: %w", err)
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
				panic(fmt.Errorf("input is not of type ActIn, got %T", in))
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
					panic(fmt.Errorf("failed to call tool %s: %w", req.ToolName, err))
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
