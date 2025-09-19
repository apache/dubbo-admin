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
	channels     *agent.Channels
}

func Create(g *genkit.Genkit) *ReActAgent {
	thinkPrompt := buildThinkPrompt(g)
	feedBackPrompt := buildFeedBackPrompt(g)
	summarizePrompt := buildSummarizePrompt(g)

	channels := agent.NewChannels(config.STAGE_CHANNEL_BUFFER_SIZE)

	thinkStage := agent.NewStreamStage(
		think(g, thinkPrompt, feedBackPrompt, summarizePrompt),
		func(channels *agent.Channels, chunk schema.StreamChunk) error {
			if channels == nil {
				panic(fmt.Errorf("channels is nil"))
			}
			channels.UserRespChan <- chunk.Chunk.Text()
			return nil
		},
		func(channels *agent.Channels, output schema.Schema) error {
			if channels == nil {
				panic(fmt.Errorf("channels is nil"))
			}
			channels.FlowChan <- output
			return nil
		},
	)
	actStage := agent.NewStage(act(g))
	orchestrator := agent.NewOrderOrchestrator(thinkStage, actStage)

	return &ReActAgent{
		registry:     g,
		orchestrator: orchestrator,
		memoryCtx:    memory.NewMemoryContext(memory.ChatHistoryKey),
		channels:     channels,
	}
}

func (ra *ReActAgent) Interact(input schema.Schema) *agent.Channels {
	ra.channels.Reset()
	go func() {
		ra.orchestrator.Run(ra.memoryCtx, input, ra.channels)
		ra.channels.Close()
	}()
	return ra.channels
}

func buildThinkPrompt(registry *genkit.Genkit) ai.Prompt {
	// Load system prompt from filesystem
	data, err := os.ReadFile(config.PROMPT_DIR_PATH + "/agentSystem.txt")
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

func buildFeedBackPrompt(registry *genkit.Genkit) ai.Prompt {
	data, err := os.ReadFile(config.PROMPT_DIR_PATH + "/agentFeedback.txt")
	if err != nil {
		panic(fmt.Errorf("failed to read agentFeedback prompt: %w", err))
	}
	return genkit.DefinePrompt(registry, "agentFeedback",
		ai.WithInputType(ThinkOut{}),
		ai.WithPrompt(string(data)),
	)
}

func buildSummarizePrompt(registry *genkit.Genkit) ai.Prompt {
	data, err := os.ReadFile(config.PROMPT_DIR_PATH + "/agentSummarize.txt")
	if err != nil {
		panic(fmt.Errorf("failed to read agentSummarize prompt: %w", err))
	}
	return genkit.DefinePrompt(registry, "summarize",
		ai.WithInputType(ThinkIn{}),
		ai.WithPrompt(string(data)),
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

func think(
	g *genkit.Genkit,
	thinkPrompt ai.Prompt,
	feedBackPrompt ai.Prompt,
	summarizePrompt ai.Prompt,
) agent.StreamFlow {
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

			// Summarize
			summary, err := summarizePrompt.Execute(ctx,
				ai.WithInput(in),
				// ai.WithStreaming() receives ai.ModelStreamCallback type callback function
				// This callback function is called when the model generates each raw streaming chunk, used for raw chunk processing
				// The passed cb is user-defined callback function for handling streaming data logic, such as printing
				ai.WithStreaming(rawChunkHandler(cb)),
			)
			history.AddHistory(summary.History()...)

			var resp *ai.ModelResponse
			if !history.IsEmpty() {
				resp, err = thinkPrompt.Execute(ctx,
					ai.WithInput(in),
					ai.WithMessages(history.AllHistory()...),
				)
			} else {
				resp, err = thinkPrompt.Execute(ctx,
					ai.WithInput(in),
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

			// Feedback
			_, err = feedBackPrompt.Execute(
				ctx,
				ai.WithInput(response),
				ai.WithStreaming(rawChunkHandler(cb)),
			)

			if err != nil {
				panic(fmt.Errorf("failed to execute agentFeedback prompt: %w", err))
			}

			response.Usage = resp.Usage
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
