package test

import (
	"context"
	"dubbo-admin-ai/internal/agent"
	"dubbo-admin-ai/internal/schema"
	"encoding/json"
	"fmt"
	"testing"
)

func TestThinking(t *testing.T) {
	ctx := context.Background()
	agentInput := schema.ReActInput{
		UserInput: "我的微服务 order-service 运行缓慢，请帮助我诊断原因",
	}

	resp, err := agent.ThinkingFlow.Run(ctx, agentInput)
	if err != nil {
		t.Fatalf("failed to run thinking flow: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	fmt.Print(resp)
}

func TestAct(t *testing.T) {
	ctx := context.Background()

	actInJson := `{
  "tool_requests": [
    {
      "thought": "查询order-service的P95和P99延迟指标，以确认性能问题的严重程度。",
      "tool_desc": {
        "tool_name": "prometheus_query_service_latency",
        "tool_input": {
          "quantile": 0.95,
          "serviceName": "order-service",
          "timeRangeMinutes": 10
        }
      }
    },
    {
      "thought": "查询order-service的QPS和错误率，评估服务负载和健康状态。",
      "tool_desc": {
        "tool_name": "prometheus_query_service_traffic",
        "tool_input": {
          "serviceName": "order-service",
          "timeRangeMinutes": 10
        }
      }
    },
    {
      "thought": "分析order-service的链路追踪数据，找出延迟最高的下游调用。",
      "tool_desc": {
        "tool_name": "trace_latency_analysis",
        "tool_input": {
          "serviceName": "order-service",
          "timeRangeMinutes": 10
        }
      }
    },
    {
      "thought": "检查order-service的Dubbo提供者和消费者列表及其状态，确认服务依赖是否正常。",
      "tool_desc": {
        "tool_name": "dubbo_service_status",
        "tool_input": {
          "serviceName": "order-service"
        }
      }
    }
  ],
  "status": "CONTINUED",
  "thought": "初步分析表明order-service的性能问题可能有多个潜在原因，包括高延迟的下游调用、资源不足、JVM问题或数据库连接池问题等。首先需要收集系统性能数据，以便进一步诊断。"
}`
	actIn := &schema.ThinkAggregation{}
	if err := json.Unmarshal([]byte(actInJson), actIn); err != nil {
		t.Fatalf("failed to unmarshal actInJson: %v", err)
	}

	resp, err := agent.ActFlow.Run(ctx, actIn)
	if err != nil {
		t.Fatalf("failed to run act flow: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	for _, r := range resp {
		fmt.Println(r)
	}
}

func TestReAct(t *testing.T) {
	ctx := context.Background()
	agentInput := schema.ReActInput{
		UserInput: "我的微服务 order-service 运行缓慢，请帮助我诊断原因",
	}

	reActResp, err := agent.ReActFlow.Run(ctx, agentInput)
	if err != nil {
		t.Fatalf("failed to run reAct flow: %v", err)
	}
	fmt.Println(reActResp)
}
