package main

import (
	"dubbo-admin-ai/agent/react"
	"dubbo-admin-ai/internal/manager"
	"dubbo-admin-ai/plugins/dashscope"
	"fmt"
)

func main() {
	reActAgent := react.Create(manager.Registry(dashscope.Qwen3.Key(), manager.PrettyLogger()))

	agentInput := react.ThinkIn{
		Content: "我的微服务 order-service 运行缓慢，请帮助我诊断原因",
	}

	channels, err := reActAgent.Interact(agentInput)
	if err != nil {
		fmt.Printf("failed to run interaction: %v\n", err)
		return
	}

	for {
		select {
		case chunk, ok := <-channels.StreamChunkChan:
			if !ok {
				channels.StreamChunkChan = nil
				continue
			}
			if chunk != nil {
				fmt.Print(chunk.Chunk.Text())
			}

		case userResp, ok := <-channels.UserRespChan:
			if !ok {
				channels.UserRespChan = nil
				continue
			}
			fmt.Printf("Final answer: %s\n", userResp)

		case flowOutput, ok := <-channels.FlowChan:
			if !ok {
				channels.FlowChan = nil
				continue
			}
			fmt.Printf("Flow output: %+v\n", flowOutput)

		default:
			if channels.StreamChunkChan == nil && channels.UserRespChan == nil && channels.FlowChan == nil {
				return
			}
		}
	}
}
