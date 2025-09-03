package test

import (
	"dubbo-admin-ai/internal/agent"
	"dubbo-admin-ai/plugins/dashscope"
)

func init() {
	if err := agent.InitAgent(dashscope.Qwen3); err != nil {
		panic(err)
	}
}
