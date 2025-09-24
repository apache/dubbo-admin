package react

import (
	"context"
	"encoding/json"
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

func onStreaming2Nil(channels *agent.Channels, chunk schema.StreamChunk) error {
	if channels == nil {
		return fmt.Errorf("channels is nil")
	}
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

func Create(g *genkit.Genkit) (*ReActAgent, error) {
	var (
		thinkPrompt    ai.Prompt
		feedbackPrompt ai.Prompt
		toolPrompt     ai.Prompt
		observePrompt  ai.Prompt
		err            error
	)

	toolRefs, err := tools.NewMockToolManager(g).AllToolRefs()
	if err != nil {
		return nil, fmt.Errorf("failed to get mock mock_tools: %v", err)
	}
	// mcpToolManager, err := tools.NewMCPToolManager(g, config.MCP_HOST_NAME)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create MCP tool manager: %v", err)
	// }
	// mcpToolRefs := mcpToolManager.AllToolRefs()
	// toolRefs = append(toolRefs, mcpToolRefs...)

	// Build and Register ReAct think prompt
	if thinkPrompt, err = buildThinkPrompt(g, toolRefs...); err != nil {
		return nil, err
	}
	if feedbackPrompt, err = buildFeedBackPrompt(g); err != nil {
		return nil, err
	}
	if toolPrompt, err = buildToolSelectionPrompt(g, toolRefs...); err != nil {
		return nil, err
	}
	if observePrompt, err = buildObservePrompt(g); err != nil {
		return nil, err
	}

	channels := agent.NewChannels(config.STAGE_CHANNEL_BUFFER_SIZE)

	thinkStage := agent.NewStreamStage(
		think(g, thinkPrompt, feedbackPrompt),
		agent.InLoop,
		onStreaming2User,
		onOutput2Flow,
	)
	actStage := agent.NewStage(
		act(g, nil, toolPrompt),
		agent.InLoop,
	)
	observerStage := agent.NewStreamStage(
		observe(g, observePrompt, feedbackPrompt),
		agent.InLoop,
		onStreaming2User,
		onOutput2Flow,
	)

	orchestrator := agent.NewOrderOrchestrator(thinkStage, actStage, observerStage)

	return &ReActAgent{
		registry:     g,
		orchestrator: orchestrator,
		memoryCtx:    memory.NewMemoryContext(memory.ChatHistoryKey),
		channels:     channels,
	}, nil
}

func (ra *ReActAgent) Interact(input *schema.UserInput) *agent.Channels {
	ra.channels.Reset()
	go func() {
		err := ra.orchestrator.Run(ra.memoryCtx, schema.ThinkInput{UserInput: input}, ra.channels)
		if err != nil {
			ra.channels.ErrorChan <- err
		}
		ra.channels.Close()
	}()
	return ra.channels
}

func buildThinkPrompt(registry *genkit.Genkit, tools ...ai.ToolRef) (ai.Prompt, error) {
	// Load system prompt from filesystem
	data, err := os.ReadFile(config.PROMPT_DIR_PATH + "/agentThink.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to read agentThink prompt: %w", err)
	}
	systemPromptText := string(data)

	toolsJson, err := json.Marshal(tools)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tools: %w", err)
	}

	return genkit.DefinePrompt(registry, "agentThinking",
		ai.WithSystem(systemPromptText),
		ai.WithInputType(ThinkIn{}),
		ai.WithOutputType(ThinkOut{}),
		ai.WithPrompt(fmt.Sprintf("available tools: %s", string(toolsJson))),
	), nil
}

func buildToolSelectionPrompt(registry *genkit.Genkit, toolRefs ...ai.ToolRef) (ai.Prompt, error) {
	data, err := os.ReadFile(config.PROMPT_DIR_PATH + "/agentTool.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to read agentTool prompt: %w", err)
	}
	return genkit.DefinePrompt(registry, "agentTool",
		ai.WithSystem(string(data)),
		ai.WithInputType(ThinkOut{}),
		ai.WithTools(toolRefs...),
		ai.WithReturnToolRequests(true),
	), nil
}

func buildFeedBackPrompt(registry *genkit.Genkit) (ai.Prompt, error) {
	data, err := os.ReadFile(config.PROMPT_DIR_PATH + "/agentFeedback.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to read agentFeedback prompt: %w", err)
	}
	return genkit.DefinePrompt(registry, "agentFeedback",
		ai.WithSystem(string(data)),
		ai.WithInputType(ThinkIn{}),
	), nil
}

func buildObservePrompt(registry *genkit.Genkit) (ai.Prompt, error) {
	data, err := os.ReadFile(config.PROMPT_DIR_PATH + "/agentObserve.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to read agentObserve prompt: %w", err)
	}
	return genkit.DefinePrompt(registry, "observe",
		ai.WithSystem(string(data)),
		ai.WithOutputType(schema.Observation{}),
	), nil
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

func feedback(feedbackPrompt ai.Prompt, ctx context.Context, cb core.StreamCallback[schema.StreamChunk], messages ...*ai.Message) error {
	_, err := feedbackPrompt.Execute(
		ctx,
		ai.WithMessages(messages...),
		ai.WithStreaming(rawChunkHandler(cb)),
	)
	if err != nil {
		return fmt.Errorf("failed to execute agentFeedback prompt: %w", err)
	}
	schema.IncreaseIndex()
	return nil
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

			history, ok := ctx.Value(memory.ChatHistoryKey).(*memory.History)
			if !ok {
				return nil, fmt.Errorf("failed to get history from context")
			}

			var (
				opts     []ai.PromptExecuteOption
				inputMsg *ai.Message
			)
			if !history.IsEmpty() {
				opts = append(opts, ai.WithMessages(history.AllHistory()...))
			} else {
				inputJson, err := json.Marshal(in)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal input: %w", err)
				}
				inputMsg = ai.NewUserMessage(ai.NewJSONPart(string(inputJson)))
				opts = append(opts, ai.WithMessages(inputMsg))
			}

			// execute prompt
			resp, err := thinkPrompt.Execute(ctx, opts...)
			manager.GetLogger().Info("Think response:", "response", resp.Text())
			if err != nil {
				return nil, fmt.Errorf("failed to execute agentThink prompt: %w", err)
			}

			// Parse output
			var thinkOut ThinkOut
			err = resp.Output(&thinkOut)
			if err != nil {
				return nil, fmt.Errorf("failed to parse agentThink prompt response: %w", err)
			}

			// Don't add the original user input
			// if inputMsg != nil {
			// 	history.AddHistory(inputMsg)
			// }
			history.AddHistory(resp.Message)
			thinkOut.Usage = resp.Usage

			return thinkOut, nil
		})
}

func act(g *genkit.Genkit, mcpToolManager *tools.MCPToolManager, toolPrompt ai.Prompt) agent.NormalFlow {
	return genkit.DefineFlow(g, agent.ActFlowName,
		func(ctx context.Context, in schema.Schema) (out schema.Schema, err error) {
			manager.GetLogger().Info("Acting...", "input", in)
			defer func() {
				manager.GetLogger().Info("Act Done.", "output", out, "error", err)
			}()

			// Beacause the input is in the history, so don't need to use, just check the type
			if _, ok := in.(ActIn); !ok {
				return nil, fmt.Errorf("input is not of type ActIn, got %T", in)
			}

			history, ok := ctx.Value(memory.ChatHistoryKey).(*memory.History)
			if !ok {
				return nil, fmt.Errorf("failed to get history from context")
			}

			// Get tool requests form LLM
			if history.IsEmpty() {
				return nil, fmt.Errorf("history is empty")
			}
			toolReqs, err := toolPrompt.Execute(ctx,
				ai.WithMessages(history.AllHistory()...),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to execute tool selection prompt: %w", err)
			}
			if len(toolReqs.ToolRequests()) == 0 {
				return nil, fmt.Errorf("no tool requests returned")
			}
			manager.GetLogger().Info("tool requests:", "req", toolReqs.ToolRequests())

			// Call tool requests and collect outputs
			var parts []*ai.Part
			var actOuts ActOut
			for _, req := range toolReqs.ToolRequests() {
				output, err := tools.Call(g, mcpToolManager, req.Name, req.Input)
				if err != nil {
					return nil, fmt.Errorf("failed to call tool %s: %w", req.Name, err)
				}

				outputJson, err := json.Marshal(output)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal output: %w", err)
				}
				parts = append(parts, ai.NewJSONPart(string(outputJson)))
				actOuts.Add(&output)
			}

			// ai.RoleTool's messages will be ingored by ai.WithMessages
			history.AddHistory(ai.NewMessage(ai.RoleModel, nil, parts...))
			actOuts.Usage = toolReqs.Usage

			return actOuts, nil
		})
}

func observe(g *genkit.Genkit, observePrompt ai.Prompt, feedbackPrompt ai.Prompt) agent.StreamFlow {
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
				ai.WithMessages(history.AllHistory()...),
			)

			if err != nil {
				return nil, fmt.Errorf("failed to execute observe prompt: %w", err)
			}

			// Parse output
			var response schema.Observation
			err = resp.Output(&response)
			if err != nil {
				return nil, fmt.Errorf("failed to parse observe prompt response: %w", err)
			}

			history.AddHistory(resp.Message)
			feedback(feedbackPrompt, ctx, cb, history.AllHistory()...)
			response.Usage = resp.Usage

			return response, err
		})
}
