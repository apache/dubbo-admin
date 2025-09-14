package test

import (
	"dubbo-admin-ai/agent/react"
	"dubbo-admin-ai/internal/manager"
	"dubbo-admin-ai/internal/memory"
	"dubbo-admin-ai/plugins/dashscope"
	"dubbo-admin-ai/schema"
	"fmt"
	"testing"

	"github.com/firebase/genkit/go/core"
)

func TestStreamingReActAgent(t *testing.T) {
	manager.Init(dashscope.Qwen3.Key(), manager.PrettyLogger())
	prompt := react.BuildThinkPrompt(manager.GetRegistry())
	chatHistoryCtx := memory.NewMemoryContext(memory.ChatHistoryKey)

	agentInput := react.ThinkIn{
		Content: "我的微服务 order-service 运行缓慢，请帮助我诊断原因",
	}

	stream := react.StreamThink(manager.GetRegistry(), prompt).Stream(chatHistoryCtx, agentInput)
	stream(func(val *core.StreamingFlowValue[schema.Schema, schema.StreamChunk], err error) bool {
		if err != nil {
			fmt.Printf("Stream error: %v\n", err)
			return false
		}
		if !val.Done {
			fmt.Print(val.Stream.Chunk.Text())
		} else if val.Output != nil {
			fmt.Printf("Stream final output: %+v\n", val.Output)
		}

		return true
	})
}
