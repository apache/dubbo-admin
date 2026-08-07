package toolstest

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	toolEngine "dubbo-admin-ai/component/tools/engine"
	"dubbo-admin-ai/runtime"

	"github.com/firebase/genkit/go/genkit"
)

func TestCall_MockTool_PrometheusServiceLatency(t *testing.T) {
	rt := runtime.NewRuntime()
	rt.SetGenkitRegistry(genkit.Init(context.Background()))
	toolEngine.NewMockToolManager(rt)

	out, err := toolEngine.Call(context.Background(), rt.GetGenkitRegistry(), "prometheus_query_service_latency", toolEngine.PrometheusServiceLatencyInput{
		ServiceName:      "order-service",
		TimeRangeMinutes: 10,
		Quantile:         0.99,
	})
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	if out.ToolName != "prometheus_query_service_latency" {
		t.Fatalf("unexpected tool name: %s", out.ToolName)
	}
	if !strings.Contains(out.Summary, "order-service") {
		t.Fatalf("unexpected summary: %s", out.Summary)
	}

	var result toolEngine.PrometheusServiceLatencyOutput
	b, err := json.Marshal(out.Result)
	if err != nil {
		t.Fatalf("marshal result error = %v", err)
	}
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("unmarshal result error = %v", err)
	}
	if result.ValueMillis != 3500 {
		t.Fatalf("unexpected value_millis: %d", result.ValueMillis)
	}
	if result.Quantile != 0.99 {
		t.Fatalf("unexpected quantile: %f", result.Quantile)
	}
}
