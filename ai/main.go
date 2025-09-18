package main

import (
	"dubbo-admin-ai/agent/react"
	"dubbo-admin-ai/internal/manager"
	"dubbo-admin-ai/plugins/dashscope"
	"fmt"
)

func main() {
	manager.Init(dashscope.Qwen3.Key(), manager.PrettyLogger())
	reActAgent := react.Create(manager.GetRegistry())

	agentInput := react.ThinkIn{
		Content: "我的微服务 order-service 运行缓慢，请帮助我诊断原因",
	}

	_, resp, err := reActAgent.Interact(agentInput)
	if err != nil {
		fmt.Printf("failed to run interaction: %v\n", err)
		return
	}

	fmt.Printf("Final Response: %v\n", resp)
}
