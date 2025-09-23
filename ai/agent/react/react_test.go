package react

import (
	"encoding/json"
	"fmt"
	"testing"

	"dubbo-admin-ai/agent"
	"dubbo-admin-ai/config"
	"dubbo-admin-ai/manager"
	"dubbo-admin-ai/memory"
	"dubbo-admin-ai/plugins/dashscope"
	"dubbo-admin-ai/schema"
	"dubbo-admin-ai/tools"
)

var (
	reActAgent     *ReActAgent
	chatHistoryCtx = memory.NewMemoryContext(memory.ChatHistoryKey)
)

func init() {
	reActAgent = Create(manager.Registry(dashscope.Qwen3.Key(), manager.DevLogger()))
}

func TestThinking(t *testing.T) {
	agentInput := schema.UserInput{
		Content: "我的微服务 order-service 运行缓慢，请帮助我诊断原因",
	}

	channels := agent.NewChannels(config.STAGE_CHANNEL_BUFFER_SIZE)

	defer func() {
		channels.Close()
		channels.Close()
	}()

	reActAgent.orchestrator.RunStage(chatHistoryCtx, agent.ThinkFlowName, agentInput, channels)

	// fmt.Println(resp)
}

func TestThinkWithToolReq(t *testing.T) {
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
	channels := agent.NewChannels(config.STAGE_CHANNEL_BUFFER_SIZE)

	defer func() {
		channels.Close()
		channels.Close()
	}()
	reActAgent.orchestrator.RunStage(chatHistoryCtx, agent.ThinkFlowName, input, channels)

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

	reActAgent.orchestrator.RunStage(chatHistoryCtx, agent.ActFlowName, actIn, nil)
	// resp := actOuts
	// if err != nil {
	// 	t.Fatalf("failed to run act flow: %v", err)
	// }
	// if resp == nil {
	// 	t.Fatal("expected non-nil response")
	// }

}

// func TestIntent(t *testing.T) {
// 	userInput := schema.UserInput{
// 		Content: "我的微服务 order-service 运行缓慢，请帮助我诊断原因",
// 	}

// 	channels := agent.NewChannels(config.STAGE_CHANNEL_BUFFER_SIZE)
// 	go reActAgent.orchestrator.RunStage(chatHistoryCtx, agent.IntentFlowName, userInput, channels)

// 	for !channels.Closed() {
// 		select {
// 		case stream := <-channels.UserRespChan:
// 			fmt.Print(stream.Text)

// 		case flowData := <-channels.FlowChan:
// 			fmt.Println(flowData)
// 		}
// 	}
// }

func TestAgent(t *testing.T) {
	agentInput := schema.UserInput{
		Content: "我的微服务 order-service 运行缓慢，请帮助我诊断原因",
	}

	channels := reActAgent.Interact(agentInput)
	for !channels.Closed() {
		select {
		case chunk, ok := <-channels.UserRespChan:
			if !ok {
				channels.UserRespChan = nil
				continue
			}
			if chunk != nil {
				fmt.Print(chunk.Text)
			}
		default:
		}
	}

}
