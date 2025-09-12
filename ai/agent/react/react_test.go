package react

import (
	"dubbo-admin-ai/agent"
	"dubbo-admin-ai/internal/manager"
	"dubbo-admin-ai/internal/memory"
	"dubbo-admin-ai/internal/tools"
	"dubbo-admin-ai/plugins/dashscope"
	"encoding/json"
	"fmt"
	"testing"
)

var (
	reActAgent     *ReActAgent
	chatHistoryCtx = memory.NewMemoryContext(memory.ChatHistoryKey)
)

func init() {
	manager.Init(dashscope.Qwen3.Key(), manager.PrettyLogger())
	reActAgent = Create(manager.GetRegistry())
}

func TestThinking(t *testing.T) {
	agentInput := ThinkIn{
		Content: "我的微服务 order-service 运行缓慢，请帮助我诊断原因",
	}

	resp, err := reActAgent.orchestrator.RunStage(chatHistoryCtx, agent.ThinkFlowName, agentInput)
	if err != nil {
		t.Fatalf("failed to run thinking flow: %v", err)
	}

	fmt.Println(resp)
}

func TestThink2(t *testing.T) {
	input := ActOut{
		Outputs: []tools.ToolOutput{
			{
				ToolName: "prometheus_query_service_latency",
				Summary:  "服务 order-service 在过去10分钟内的 P95 延迟为 3500ms",
				Result: map[string]any{
					"quantile":     0.95,
					"value_millis": 3500,
				},
			},
			{
				ToolName: "prometheus_query_service_traffic",
				Summary:  "服务 order-service 的 QPS 为 250.0, 错误率为 5.2%",
				Result: map[string]any{
					"error_rate_percentage": 5.2,
					"request_rate_qps":      250,
				},
			},
		},
	}

	resp, err := reActAgent.orchestrator.RunStage(chatHistoryCtx, agent.ThinkFlowName, input)
	if err != nil {
		t.Fatalf("failed to run thinking flow: %v", err)
	}

	fmt.Println(resp)
}

func TestAct(t *testing.T) {
	actInJson := `{
    "tool_requests": [
        {
            "tool_name": "prometheus_query_service_latency",
            "parameter": {
                "service_name": "order-service",
                "time_range_minutes": 15,
                "quantile": 0.95
            }
        },
        {
            "tool_name": "prometheus_query_service_traffic",
            "parameter": {
                "service_name": "order-service",
                "time_range_minutes": 15
            }
        },
        {
            "tool_name": "trace_dependency_view",
            "parameter": {
                "service_name": "order-service"
            }
        },
        {
            "tool_name": "dubbo_service_status",
            "parameter": {
                "service_name": "order-service"
            }
        }
    ],
    "status": "CONTINUED",
    "thought": "开始对 order-service 运行缓慢的问题进行系统性诊断。首先需要了解当前服务的整体性能表现，包括延迟、流量和错误率等关键指标。同时，获取服务的实例状态和拓扑依赖关系，以判断是否存在明显的异常。由于多个数据源可独立查询，为提高效率，将并行执行多个工具调用：查询延迟指标、服务流量、服务依赖关系和服务实例状态。"
}`
	actIn := ActIn{}
	if err := json.Unmarshal([]byte(actInJson), &actIn); err != nil {
		t.Fatalf("failed to unmarshal actInJson: %v", err)
	}

	actOuts, err := reActAgent.orchestrator.RunStage(chatHistoryCtx, agent.ActFlowName, actIn)
	resp := actOuts
	if err != nil {
		t.Fatalf("failed to run act flow: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

}

func TestAgent(t *testing.T) {
	agentInput := ThinkIn{
		Content: "我的微服务 order-service 运行缓慢，请帮助我诊断原因",
	}

	_, err := reActAgent.Interact(chatHistoryCtx, agentInput)
	if err != nil {
		t.Fatalf("failed to run thinking flow: %v", err)
	}
}

func TestAgentFlow(t *testing.T) {
	agentInput := ThinkIn{
		Content: "我的微服务 order-service 运行缓慢，请帮助我诊断原因",
	}
	_, err := reActAgent.AgentFlow(manager.GetRegistry()).Run(chatHistoryCtx, agentInput)
	if err != nil {
		t.Fatalf("failed to run agent flow: %v", err)
	}
}
