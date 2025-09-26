package main

import (
	"dubbo-admin-ai/config"
	"dubbo-admin-ai/manager"
	"dubbo-admin-ai/plugins/dashscope"
	"dubbo-admin-ai/utils"
)

func main() {
	mdDir := "./reference/k8s_docs/concepts"
	chunks, err := utils.ProcessMarkdownDirectory(mdDir)
	if err != nil {
		panic(err)
	}
	g := manager.Registry(dashscope.Qwen3.Key(), config.PROJECT_ROOT+"/.env", manager.ProductionLogger())
	err = utils.IndexInPinecone(g, "kube-docs", "concepts", dashscope.Qwen3_embedding.Key(), nil, chunks)
	if err != nil {
		panic(err)
	}
}
