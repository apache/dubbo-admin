package main

import (
	"dubbo-admin-ai/agent/react"
	"dubbo-admin-ai/manager"
	"dubbo-admin-ai/plugins/dashscope"
)

func main() {
	_, _ = react.Create(manager.Registry(dashscope.Qwen_max.Key(), manager.ProductionLogger()))
	select {}
}
