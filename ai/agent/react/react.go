package react

import (
	"context"
	"fmt"
	"os"

	"dubbo-admin-ai/agent"
	"dubbo-admin-ai/config"
	"dubbo-admin-ai/manager"
	"dubbo-admin-ai/memory"
	"dubbo-admin-ai/schema"
	"dubbo-admin-ai/tools"

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

func onStreaming2User(channels *agent.Channels, chunk schema.StreamChunk) error {
	if channels == nil {
		return fmt.Errorf("channels is nil")
	}
	channels.UserRespChan <- schema.NewStreamFeedback(chunk.Chunk.Text())
	return nil
}

func onStreaming2StdOut(channels *agent.Channels, chunk schema.StreamChunk) error {
	if channels == nil {
		return fmt.Errorf("channels is nil")
	}
	fmt.Print(chunk.Chunk.Text())
	return nil
}

func onOutput2Flow(channels *agent.Channels, output schema.Schema) error {
	if channels == nil {
		return fmt.Errorf("channels is nil")
	}
	channels.FlowChan <- output
	return nil
}

func Create(g *genkit.Genkit) *ReActAgent {
	thinkPrompt := buildThinkPrompt(g)
	feedBackPrompt := buildFeedBackPrompt(g)
	observePrompt := buildObservePrompt(g)
	// intentParsePrompt := buildIntentParsePrompt(g)

	channels := agent.NewChannels(config.STAGE_CHANNEL_BUFFER_SIZE)

	// intentParseStage := agent.NewStreamStage(
	// 	intentParse(g, intentParsePrompt),
	// 	agent.BeforeLoop,
	// 	onStreaming2StdOut,
	// 	onOutput2Flow,
	// )
	thinkStage := agent.NewStreamStage(
		think(g, thinkPrompt, feedBackPrompt),
		agent.InLoop,
		onStreaming2User,
		onOutput2Flow,
	)
	actStage := agent.NewStage(
		act(g),
		agent.InLoop,
	)
	observerStage := agent.NewStreamStage(
		observe(g, observePrompt),
		agent.InLoop,
		onStreaming2StdOut,
		onOutput2Flow,
	)

	orchestrator := agent.NewOrderOrchestrator(thinkStage, actStage, observerStage)

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
	toolRefs, err := tools.NewMockToolManager(registry).AllToolRefs()
	if err != nil {
		panic(fmt.Errorf("failed to get mock mock_tools: %v", err))
	}
	return genkit.DefinePrompt(registry, "agentThinking",
		ai.WithSystem(systemPromptText),
		ai.WithInputType(ThinkIn{}),
		ai.WithOutputType(ThinkOut{}),
		ai.WithPrompt(schema.UserPromptTemplate),
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
		ai.WithSystem(string(data)),
		ai.WithInputType(schema.ThinkInput{}),
		ai.WithPrompt(schema.UserPromptTemplate),
	)
}

func buildObservePrompt(registry *genkit.Genkit) ai.Prompt {
	data, err := os.ReadFile(config.PROMPT_DIR_PATH + "/agentObserve.txt")
	if err != nil {
		panic(fmt.Errorf("failed to read agentObserve prompt: %w", err))
	}
	return genkit.DefinePrompt(registry, "observe",
		ai.WithSystem(string(data)),
		ai.WithInputType(ActOut{}),
		ai.WithOutputType(schema.ThinkInput{}),
		ai.WithPrompt(schema.UserPromptTemplate),
	)
}

func buildIntentParsePrompt(registry *genkit.Genkit) ai.Prompt {
	data, err := os.ReadFile(config.PROMPT_DIR_PATH + "/agentIntentParse.txt")
	if err != nil {
		panic(fmt.Errorf("failed to read agentIntentParse prompt: %w", err))
	}
	return genkit.DefinePrompt(registry, "intentParse",
		ai.WithSystem(string(data)),
		ai.WithInputType(schema.UserInput{}),
		ai.WithOutputType(schema.ThinkInput{}),
		ai.WithPrompt(schema.UserPromptTemplate),
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

func intentParse(g *genkit.Genkit,
	intentParsePrompt ai.Prompt,
) agent.StreamFlow {
	return genkit.DefineStreamingFlow(g, agent.IntentFlowName,
		func(ctx context.Context, in schema.Schema, cb core.StreamCallback[schema.StreamChunk]) (out schema.Schema, err error) {
			manager.GetLogger().Info("Intent Parsing...", "input", in)
			defer func() {
				if r := recover(); r != nil {
					manager.GetLogger().Error("Intent Parsing panicked", "error", r, "input", in)
					err = fmt.Errorf("intent parsing failed: %v", r)
					out = nil
					return
				}
				manager.GetLogger().Info("Intent Parsing Done.", "output", out, "error", err)
			}()

			history, ok := ctx.Value(memory.ChatHistoryKey).(*memory.History)
			if !ok {
				return nil, fmt.Errorf("failed to get history from context")
			}

			var resp *ai.ModelResponse
			resp, err = intentParsePrompt.Execute(ctx,
				ai.WithInput(in),
				ai.WithStreaming(rawChunkHandler(cb)),
			)
			if err != nil {
				manager.GetLogger().Error("Intent parse prompt execution failed", "error", err)
				return nil, err
			}

			var intent schema.ThinkInput
			if err := resp.Output(&intent); err != nil {
				manager.GetLogger().Error("Failed to parse intent response", "error", err, "response", resp)
				return nil, fmt.Errorf("failed to parse intent response: %w", err)
			}

			history.AddHistory(resp.History()...)
			return intent, nil
		})

}

// ai.WithStreaming() receives ai.ModelStreamCallback type callback function
// This callback function is called when the model generates each raw streaming chunk, used for raw chunk processing
// The passed cb is user-defined callback function for handling streaming data logic, such as printing
func think(
	g *genkit.Genkit,
	thinkPrompt ai.Prompt,
	feedBackPrompt ai.Prompt,
) agent.StreamFlow {
	return genkit.DefineStreamingFlow(g, agent.ThinkFlowName,
		func(ctx context.Context, in schema.Schema, cb core.StreamCallback[schema.StreamChunk]) (out schema.Schema, err error) {
			manager.GetLogger().Info("Thinking...", "input", in)
			defer func() {
				manager.GetLogger().Info("Think Done.", "output", out, "error", err)
			}()

			// Feedback
			_, err = feedBackPrompt.Execute(
				ctx,
				ai.WithInput(in),
				ai.WithStreaming(rawChunkHandler(cb)),
			)
			if err != nil {
				manager.GetLogger().Error("Failed to execute agentFeedback prompt", "error", err)
				return nil, fmt.Errorf("failed to execute agentFeedback prompt: %w", err)
			}

			schema.IncreaseIndex()
			history, ok := ctx.Value(memory.ChatHistoryKey).(*memory.History)
			if !ok {
				return nil, fmt.Errorf("failed to get history from context")
			}

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
				manager.GetLogger().Error("Failed to execute agentThink prompt", "error", err)
				return nil, fmt.Errorf("failed to execute agentThink prompt: %w", err)
			}

			// Parse output
			var thinkOut ThinkOut
			err = resp.Output(&thinkOut)
			if err != nil {
				manager.GetLogger().Error("Failed to parse agentThink prompt response", "error", err)
				return nil, fmt.Errorf("failed to parse agentThink prompt response: %w", err)
			}

			thinkOut.Usage = resp.Usage
			history.AddHistory(resp.History()...)

			return thinkOut, nil
		})
}

func act(g *genkit.Genkit) agent.NormalFlow {
	return genkit.DefineFlow(g, agent.ActFlowName,
		func(ctx context.Context, in schema.Schema) (out schema.Schema, err error) {
			manager.GetLogger().Info("Acting...", "input", in)
			defer func() {
				manager.GetLogger().Info("Act Done.", "output", out, "error", err)
			}()

			input, ok := in.(ActIn)
			if !ok {
				return nil, fmt.Errorf("input is not of type ActIn, got %T", in)
			}

			var actOuts ActOut

			// Execute tool calls
			history, ok := ctx.Value(memory.ChatHistoryKey).(*memory.History)
			if !ok {
				return nil, fmt.Errorf("failed to get history from context")
			}

			var parts []*ai.Part
			for _, req := range input.ToolRequests {

				output, err := req.Call(g, ctx)
				if err != nil {
					manager.GetLogger().Error("Failed to call tool", "tool", req.ToolName, "error", err)
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

func observe(g *genkit.Genkit, observePrompt ai.Prompt) agent.StreamFlow {
	return genkit.DefineStreamingFlow(g, agent.ObserveFlowName,
		func(ctx context.Context, in schema.Schema, cb core.StreamCallback[schema.StreamChunk]) (out schema.Schema, err error) {
			manager.GetLogger().Info("Observing...", "input", in)
			defer func() {
				manager.GetLogger().Info("Observe Done.", "output", out, "error", err)
			}()

			history, ok := ctx.Value(memory.ChatHistoryKey).(*memory.History)
			if !ok || history.IsEmpty() {
				return nil, fmt.Errorf("failed to get history from context or history is empty")
			}

			resp, err := observePrompt.Execute(ctx,
				ai.WithInput(in),
				ai.WithMessages(history.AllHistory()...),
				ai.WithStreaming(rawChunkHandler(cb)),
			)
			if err != nil {
				manager.GetLogger().Error("Failed to execute observe prompt", "error", err)
				return nil, fmt.Errorf("failed to execute observe prompt: %w", err)
			}

			// Parse output
			var response schema.ThinkInput
			err = resp.Output(&response)
			if err != nil {
				manager.GetLogger().Error("Failed to parse observe prompt response", "error", err)
				return nil, fmt.Errorf("failed to parse observe prompt response: %w", err)
			}
			history.AddHistory(resp.History()...)

			return response, err
		})
}
