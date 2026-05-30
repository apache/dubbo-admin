package react

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"

	"dubbo-admin-ai/component/agent"
	"dubbo-admin-ai/component/memory"
	toolEngine "dubbo-admin-ai/component/tools/engine"
	"dubbo-admin-ai/runtime"
	"dubbo-admin-ai/schema"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	"github.com/openai/openai-go"
)

type ThinkIn = schema.ThinkInput
type ThinkOut = schema.ThinkOutput
type ActIn = ThinkOut
type ActOut = schema.ToolOutputs

type ReActAgent struct {
	registry     *genkit.Genkit
	memoryCtx    context.Context
	orchestrator agent.Orchestrator
	channels     *agent.Channels

	defaultModel   string // Default model in "provider/model" format (e.g., "dashscope/qwen-max")
	promptBasePath string
	maxIterations  int
}

func onStreaming2User(channels *agent.Channels, chunk schema.StreamChunk) error {
	if channels == nil {
		return fmt.Errorf("channels is nil")
	}
	channels.UserRespChan <- schema.NewStreamFeedback(chunk.Chunk.Text())
	return nil
}

func onOutput2Flow(channels *agent.Channels, output schema.Schema) error {
	if channels == nil {
		return fmt.Errorf("channels is nil")
	}
	if observation, ok := output.(schema.Observation); ok {
		if observation.Summary != "" {
			channels.UserRespChan <- schema.NewStreamFeedback(observation.Summary + "\n")
		}
		if observation.FinalAnswer != "" {
			channels.UserRespChan <- schema.NewStreamFeedback(observation.FinalAnswer + "\n")
		}
	}
	channels.FlowChan <- output
	channels.UserRespChan <- schema.StreamEnd()
	return nil
}

// stageTypeInfo defines metadata for each stage type
type stageTypeInfo struct {
	inType      any
	outType     any
	needsTools  bool
	isStreaming bool
}

var stageTypeRegistry = map[string]stageTypeInfo{
	"think":   {inType: ThinkIn{}, outType: ThinkOut{}, needsTools: true, isStreaming: false},
	"act":     {inType: ThinkOut{}, outType: nil, needsTools: true, isStreaming: false},
	"observe": {inType: nil, outType: schema.Observation{}, needsTools: false, isStreaming: true},
}

func NewReactAgent(g *genkit.Genkit, promptBasePath string, defaultModel string, maxIterations int, stagesCfg []StageInfo, toolRefs []ai.ToolRef) (*ReActAgent, error) {
	memoryCtx := memory.NewMemoryContext(memory.ChatHistoryKey)
	channels := agent.NewChannels(len(stagesCfg))
	stages, err := buildStagesFromConfig(g, stagesCfg, promptBasePath, defaultModel, toolRefs)
	if err != nil {
		return nil, err
	}

	return &ReActAgent{
		registry:       g,
		orchestrator:   agent.NewOrderOrchestrator(maxIterations, stages...),
		memoryCtx:      memoryCtx,
		channels:       channels,
		defaultModel:   defaultModel,
		promptBasePath: promptBasePath,
		maxIterations:  maxIterations,
	}, nil
}

func buildStagesFromConfig(g *genkit.Genkit, stagesCfg []StageInfo, promptBasePath string, defaultModel string, toolRefs []ai.ToolRef) ([]*agent.Stage, error) {
	var stages []*agent.Stage

	for _, stageCfg := range stagesCfg {
		// 1. Get type information
		typeInfo, ok := stageTypeRegistry[stageCfg.FlowType]
		if !ok {
			return nil, fmt.Errorf("unknown flow type: %s", stageCfg.FlowType)
		}

		// 2. Read and build prompt
		promptPath := path.Join(promptBasePath, stageCfg.PromptFile)
		systemPrompt, err := os.ReadFile(promptPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read prompt file %s: %w", promptPath, err)
		}

		// Prepare tools
		var tools []ai.ToolRef
		if typeInfo.needsTools && stageCfg.EnableTools {
			tools = toolRefs
		}

		// Prepare available tools names for think stage.
		extraPrompt := stageCfg.ExtraPrompt
		if stageCfg.FlowType == "think" && extraPrompt == "" {
			toolNames := make([]string, 0, len(toolRefs))
			for _, toolRef := range toolRefs {
				toolNames = append(toolNames, toolRef.Name())
			}
			toolsJson, err := json.Marshal(toolNames)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tool names: %w", err)
			}
			extraPrompt = fmt.Sprintf("available tools: %s", string(toolsJson))
			runtime.GetLogger().Debug("Tool details", "extraPrompt", extraPrompt)
		}

		// When building prompt, use default model if not specified in configuration
		model := stageCfg.Model
		if model == "" {
			model = defaultModel
		}

		prompt, err := buildPrompt(g, typeInfo.inType, typeInfo.outType, stageCfg.Name,
			string(systemPrompt), stageCfg.Temperature, model, extraPrompt, tools...)
		if err != nil {
			return nil, fmt.Errorf("failed to build prompt for stage %s: %w", stageCfg.Name, err)
		}

		// 3. Create stage
		var stage *agent.Stage
		switch stageCfg.FlowType {
		case "think":
			stage = agent.NewStage(ThinkFlow(g, prompt), agent.InLoop)
		case "act":
			stage = agent.NewStage(ActFlow(g, prompt), agent.InLoop)
		case "observe":
			stage = agent.NewStreamStage(observe(g, prompt),
				agent.InLoop, onStreaming2User, onOutput2Flow)
		}

		if stage != nil {
			stages = append(stages, stage)
		}
	}

	return stages, nil
}

func (ra *ReActAgent) Interact(input *schema.UserInput, sessionID string) *agent.Channels {
	ra.channels.Reset()
	go func() {
		var (
			err       error
			inputJson []byte
			in        schema.ThinkInput
		)
		in.UserInput = input
		in.SessionID = sessionID

		// Add user input to history
		ra.memoryCtx = context.WithValue(ra.memoryCtx, memory.SessionIDKey, sessionID)
		history, ok := ra.memoryCtx.Value(memory.ChatHistoryKey).(*memory.HistoryMemory)
		if !ok {
			err = fmt.Errorf("failed to get history from context")
			ra.channels.ErrorChan <- err
		}

		inputJson, err = json.Marshal(in)
		if err != nil {
			ra.channels.ErrorChan <- err
		}
		inputMsg := ai.NewUserMessage(ai.NewJSONPart(string(inputJson)))
		history.AddHistory(sessionID, inputMsg)

		err = ra.orchestrator.Run(ra.memoryCtx, in, ra.channels)
		if err != nil {
			ra.channels.ErrorChan <- err
		}
		ra.channels.Close()
		history.NextTurn(sessionID)
	}()
	return ra.channels
}

func (ra *ReActAgent) GetMemory() *memory.HistoryMemory {
	h, err := memory.GetHistoryMemory(ra.memoryCtx, memory.ChatHistoryKey)
	if err != nil {
		return nil
	}
	return h
}

func buildPrompt(registry *genkit.Genkit, inType, outType any, tag, prompt string, temp float64, model string, extraPrompt string, tools ...ai.ToolRef) (ai.Prompt, error) {
	opts := []ai.PromptOption{
		ai.WithSystem(prompt),
		ai.WithConfig(&openai.ChatCompletionNewParams{
			Temperature: openai.Float(temp),
		}),
		ai.WithModelName(model),
	}
	if inType != nil {
		opts = append(opts, ai.WithInputType(inType))
	}
	if outType != nil {
		opts = append(opts, ai.WithOutputType(outType))
	}
	if extraPrompt != "" {
		opts = append(opts, ai.WithPrompt(extraPrompt))
	}
	if tools != nil {
		opts = append(opts, ai.WithTools(tools...), ai.WithReturnToolRequests(true))
	}

	return genkit.DefinePrompt(registry, tag, opts...), nil
}

// ai.WithStreaming() receives ai.ModelStreamCallback type callback function
// This callback function is called when the model generates each raw streaming chunk, used for raw chunk processing
// The passed cb is user-defined callback function for handling streaming data logic, such as printing
func ThinkFlow(
	g *genkit.Genkit,
	thinkPrompt ai.Prompt,
) agent.NormalFlow {
	return genkit.DefineFlow(g, agent.ThinkFlowName,
		func(ctx context.Context, in schema.Schema) (out schema.Schema, err error) {
			runtime.GetLogger().Info("Thinking...", "input", in)
			defer func() {
				runtime.GetLogger().Info("Think Done.", "output", out, "error", err)
			}()

			history, ok := ctx.Value(memory.ChatHistoryKey).(*memory.HistoryMemory)
			if !ok {
				return nil, fmt.Errorf("failed to get history from context")
			}
			sessionID, ok := ctx.Value(memory.SessionIDKey).(string)
			if !ok || sessionID == "" {
				return nil, fmt.Errorf("session id not found in context")
			}
			if history.IsEmpty(sessionID) {
				return nil, fmt.Errorf("history is empty")
			}

			// Execute the thinking prompt with window memory context
			resp, err := thinkPrompt.Execute(ctx, ai.WithMessages(history.WindowMemory(sessionID)...))
			if err != nil {
				return nil, fmt.Errorf("failed to execute agentThink prompt: %w", err)
			}
			if resp == nil {
				return nil, fmt.Errorf("failed to execute agentThink prompt: empty response")
			}
			runtime.GetLogger().Info("Think response:", "response", resp.Text())

			// Parse output
			var thinkOut ThinkOut
			thinkOut.UsageInfo = &ai.GenerationUsage{}
			err = resp.Output(&thinkOut)
			if err != nil {
				return nil, fmt.Errorf("failed to parse agentThink prompt response: %w", err)
			}

			history.AddHistory(sessionID, resp.Message)
			schema.AccumulateUsage(thinkOut.UsageInfo, resp.Usage, in.Usage())

			return thinkOut, nil
		})
}

func ActFlow(g *genkit.Genkit, actPrompt ai.Prompt) agent.NormalFlow {
	return genkit.DefineFlow(g, agent.ActFlowName,
		func(ctx context.Context, in schema.Schema) (out schema.Schema, err error) {
			runtime.GetLogger().Info("Acting...", "input", in)
			defer func() {
				runtime.GetLogger().Info("Act Done.", "output", out, "error", err)
			}()

			// Try to get input from orchestrator or parse from history
			var input ActIn
			var hasInput bool
			if inTyped, ok := in.(ActIn); ok {
				input = inTyped
				hasInput = true
			}
			// If no valid input from orchestrator, we'll skip the general inquiry check
			// and let the LLM decide based on history
			if !hasInput || (input.Intent == schema.GeneralInquiry || len(input.SuggestedTools) == 0) {
				// Only skip if we actually have input and it indicates general inquiry
				if hasInput && (input.Intent == schema.GeneralInquiry || len(input.SuggestedTools) == 0) {
					actOuts := ActOut{
						Outputs: []toolEngine.ToolOutput{
							{
								ToolName: "no_need_tool_call",
								Summary:  "no need tool call",
								Result: map[string]any{
									"reason": "general inquiry or no suggested tools",
								},
							},
						},
						UsageInfo: &ai.GenerationUsage{},
					}
					schema.AccumulateUsage(actOuts.UsageInfo, in.Usage())
					return actOuts, nil
				}
				// If no valid input, continue to let LLM decide from history
			}

			history, ok := ctx.Value(memory.ChatHistoryKey).(*memory.HistoryMemory)
			if !ok {
				return nil, fmt.Errorf("failed to get history from context")
			}
			sessionID, ok := ctx.Value(memory.SessionIDKey).(string)
			if !ok || sessionID == "" {
				return nil, fmt.Errorf("session id not found in context")
			}

			// Get tool requests from LLM
			if history.IsEmpty(sessionID) {
				return nil, fmt.Errorf("history is empty")
			}
			toolReqs, err := actPrompt.Execute(ctx,
				ai.WithMessages(history.WindowMemory(sessionID)...),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to execute tool selection prompt: %w", err)
			}

			// If the model returns no tool requests while suggested_tools is non-empty, surface the real cause.
			if hasInput && len(toolReqs.ToolRequests()) == 0 && input.SuggestedTools != nil && len(input.SuggestedTools) > 0 {
				return nil, fmt.Errorf("model returned no tool calls for suggested_tools=%v", input.SuggestedTools)
			}
			runtime.GetLogger().Info("tool requests:", "req", toolReqs.ToolRequests())

			// Call tool requests and collect outputs
			var parts []*ai.Part
			var actOuts ActOut
			actOuts.UsageInfo = &ai.GenerationUsage{}
			for _, req := range toolReqs.ToolRequests() {
				output, err := toolEngine.Call(g, req.Name, req.Input)
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
			runtime.GetLogger().Info("act out:", "out", actOuts)
			// ai.RoleTool's messages will be ingored by ai.WithMessages
			history.AddHistory(sessionID, ai.NewMessage(ai.RoleModel, nil, parts...))
			schema.AccumulateUsage(actOuts.UsageInfo, toolReqs.Usage, in.Usage())

			return actOuts, nil
		})
}

func observe(g *genkit.Genkit, observePrompt ai.Prompt) agent.StreamFlow {
	return genkit.DefineStreamingFlow(g, agent.ObserveFlowName,
		func(ctx context.Context, in schema.Schema, _ core.StreamCallback[schema.StreamChunk]) (out schema.Schema, err error) {
			runtime.GetLogger().Info("Observing...", "input", in)
			defer func() {
				runtime.GetLogger().Info("Observe Done.", "output", out, "error", err)
			}()

			history, ok := ctx.Value(memory.ChatHistoryKey).(*memory.HistoryMemory)
			if !ok {
				return nil, fmt.Errorf("failed to get history from context")
			}
			sessionID, ok := ctx.Value(memory.SessionIDKey).(string)
			if !ok || sessionID == "" {
				return nil, fmt.Errorf("session id not found in context")
			}

			if history.IsEmpty(sessionID) {
				return nil, fmt.Errorf("history is empty")
			}

			resp, err := observePrompt.Execute(ctx,
				ai.WithMessages(history.WindowMemory(sessionID)...),
			)

			if err != nil {
				return nil, fmt.Errorf("failed to execute observe prompt: %w", err)
			}

			// Parse output
			var observation schema.Observation
			observation.UsageInfo = &ai.GenerationUsage{}
			err = resp.Output(&observation)
			if err != nil {
				return nil, fmt.Errorf("failed to parse observe prompt response: %w", err)
			}
			runtime.GetLogger().Info("Observe out:", "out", observation)

			history.AddHistory(sessionID, resp.Message)
			schema.AccumulateUsage(observation.UsageInfo, resp.Usage, in.Usage())

			return observation, err
		})
}
