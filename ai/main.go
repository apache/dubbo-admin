package main

import (
	"dubbo-admin-ai/agent/react"
	"dubbo-admin-ai/internal/manager"
	"dubbo-admin-ai/plugins/dashscope"
	"dubbo-admin-ai/schema"
	"fmt"
)

func main() {
	reActAgent := react.Create(manager.Registry(dashscope.Qwen3.Key(), manager.PrettyLogger()))

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
