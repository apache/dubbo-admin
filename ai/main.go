package main

import (
	"dubbo-admin-ai/agent/react"
	"dubbo-admin-ai/internal/manager"
	"dubbo-admin-ai/internal/memory"
	"dubbo-admin-ai/plugins/dashscope"
	"fmt"
)

func main() {
	manager.Init(dashscope.Qwen3.Key(), manager.ReleaseLogger())
	reActAgent := react.Create(manager.GetRegistry())
	chatHistoryCtx := memory.NewMemoryContext(memory.ChatHistoryKey)

	agentInput := react.ThinkIn{
		Content: "我的微服务 order-service 运行缓慢，请帮助我诊断原因",
	}

	resp, err := reActAgent.AgentFlow(manager.GetRegistry()).Run(chatHistoryCtx, agentInput)
	if err != nil {
		fmt.Printf("failed to run thinking flow: %v", err)
	}

	fmt.Println(resp)
}
