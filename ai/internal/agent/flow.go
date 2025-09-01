package agent

import (
	"context"
	"dubbo-admin-ai/config"
	"dubbo-admin-ai/internal/schema"
	"dubbo-admin-ai/internal/tools"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/firebase/genkit/go/core/logger"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
)

// 公开的 Flow 变量，以便在编排器中调用
var (
	ReActFlow    *core.Flow[schema.ReActIn, schema.ReActOut, struct{}]
	ThinkingFlow *core.Flow[schema.ThinkIn, *schema.ThinkOut, struct{}]
	ActFlow      *core.Flow[*schema.ActIn, schema.ActOut, struct{}]

	ThinkPrompt *ai.Prompt
)

var g *genkit.Genkit

func InitFlows(registry *genkit.Genkit) error {
	if registry == nil {
		return fmt.Errorf("registry is nil")
	}
	g = registry

	data, err := os.ReadFile(config.PROMPT_DIR_PATH + "/agentSystem.prompt")
	if err != nil {
		return fmt.Errorf("failed to read agentSystem prompt: %w", err)
	}
	systemPromptText := string(data)

	mockTools, err := tools.AllMockToolRef()
	if err != nil {
		log.Fatalf("failed to get mock mock_tools: %v", err)
	}
	ThinkPrompt, err = genkit.DefinePrompt(g, "agentThinking",
		ai.WithSystem(systemPromptText),
		ai.WithInputType(schema.ThinkIn{}),
		ai.WithOutputType(schema.ThinkOut{}),
		ai.WithPrompt("{{userInput}}"),
		ai.WithTools(mockTools...),
		ai.WithToolChoice(ai.ToolChoiceNone),
	)

	if err != nil {
		return fmt.Errorf("failed to define agentThink prompt: %w", err)
	}

	ReActFlow = genkit.DefineFlow(g, "reAct", reAct)
	ThinkingFlow = genkit.DefineFlow(g, "thinking", thinking)
	ActFlow = genkit.DefineFlow(g, "act", act)

	return nil
}

// Flow 的核心函数实现 `fn` (不对外导出)
// ----------------------------------------------------------------------------
// 1. agentOrchestrator: 总指挥/编排器的核心逻辑
// ----------------------------------------------------------------------------
func reAct(ctx context.Context, reActInput schema.ReActIn) (reActOut schema.ReActOut, err error) {
	//TODO: 输入数据意图解析

	thinkingInput := reActInput
	// 主协调循环 (Reconciliation Loop)
	for range 5 { // 设置最大循环次数

		// a. 调用 thinkingFlow
		thinkingResp, err := ThinkingFlow.Run(ctx, thinkingInput)
		if err != nil {
			logger.FromContext(ctx).Error("failed to run thinking flow", "error", err)
			return "", err
		}
		if thinkingResp == nil {
			logger.FromContext(ctx).Error("expected non-nil response")
			return "", errors.New("expected non-nil response")
		}

		for _, r := range thinkingResp.ToolRequests {
			logger.FromContext(ctx).Info(r.String())
		}

		reActOut = thinkingResp.String()

		// b. 检查是否有最终答案
		if thinkingResp.IsStop() {
			break
		}

		// c. 调用 actFlow
		actOutArray, err := ActFlow.Run(ctx, thinkingResp)
		if err != nil {
			logger.FromContext(ctx).Error("failed to run act flow", "error", err)
			return "", err
		}

		// TODO: 这里将来会集成SessionMemory，暂时简化处理
		_ = actOutArray // 占位符，避免未使用变量错误
	}

	return reActOut, nil
}

// ----------------------------------------------------------------------------
// 2. thinking: 大脑/思考者的核心逻辑
// ----------------------------------------------------------------------------
func thinking(ctx context.Context, input schema.ThinkIn) (*schema.ThinkOut, error) {
	resp, err := ThinkPrompt.Execute(ctx,
		ai.WithInput(input),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to execute agentThink prompt: %w", err)
	}

	// 解析输出
	var response *schema.ThinkOut = nil
	err = resp.Output(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse agentThink prompt response: %w", err)
	}

	return response, nil
}

// ----------------------------------------------------------------------------
// 3. act: 执行者的核心逻辑
// TODO: Genkit 的 Tool 设计不能保证任意 LLM 一定能执行工具调用，是否考虑设计成所有模型都能执行？
// 一种思路是让 LLM 直接生成工具调用的结构化描述，然后由适配器执行。
// 管线、流水线的思想能否用在这里？
func act(ctx context.Context, actIn *schema.ActIn) (schema.ActOut, error) {
	var actOutArray schema.ActOut
	// 执行工具调用
	for _, resp := range actIn.ToolRequests {
		if err := resp.ValidateToolDesc(); err != nil {
			return nil, fmt.Errorf("invalid tool description in action response: %w", err)
		}

		output, err := resp.ToolDesc.Call(g, ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to call tool %s: %w", resp.ToolDesc.ToolName, err)
		}
		actOutArray = append(actOutArray, output)
	}

	return actOutArray, nil
}
