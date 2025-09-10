package react

import (
	"dubbo-admin-ai/internal/manager"
	"dubbo-admin-ai/plugins/dashscope"
	"encoding/json"
	"fmt"
	"testing"
)

var (
	reActAgent *ReActAgent
)

func init() {
	manager.Init(dashscope.Qwen3.Key(), nil)
	reAct, err := Create(manager.GetRegister())
	if err != nil {
		panic(err)
	}
	reActAgent = reAct
}

func TestThinking(t *testing.T) {
	agentInput := ThinkIn{
		UserInput: "我的微服务 order-service 运行缓慢，请帮助我诊断原因",
	}

	resp, err := reActAgent.orchestrator.RunStage(thinkFlowName, agentInput)
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
                "latency_type": "P95",
                "service": "order-service",
                "time_range": "last_15m"
            }
        },
        {
            "tool_name": "prometheus_query_service_traffic",
            "parameter": {
                "service": "order-service",
                "time_range": "last_15m"
            }
        },
        {
            "tool_name": "trace_dependency_view",
            "parameter": {
                "service": "order-service"
            }
        },
        {
            "tool_name": "dubbo_service_status",
            "parameter": {
                "service": "order-service"
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

	actOuts, err := reActAgent.orchestrator.RunStage(actFlowName, actIn)
	resp := actOuts
	if err != nil {
		t.Fatalf("failed to run act flow: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	fmt.Println(resp)

}
