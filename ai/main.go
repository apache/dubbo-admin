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

	streamChan, outputChan, err := reActAgent.Interact(agentInput)
	if err != nil {
		fmt.Printf("failed to run interaction: %v\n", err)
		return
	}

	for {
		select {
		case chunk, ok := <-streamChan:
			if !ok {
				streamChan = nil
				continue
			}
			fmt.Print(chunk.Chunk.Text())

		case finalOutput, ok := <-outputChan:
			if !ok {
				outputChan = nil
				continue
			}
			fmt.Printf("Final output: %+v\n", finalOutput)
		default:
			if streamChan == nil && outputChan == nil {
				return
			}
		}
	}
}
