package utils_test

import (
	"context"
	"dubbo-admin-ai/config"
	"dubbo-admin-ai/manager"
	"dubbo-admin-ai/plugins/dashscope"
	"dubbo-admin-ai/tools"
	"dubbo-admin-ai/utils"
	"fmt"
	"testing"

	"github.com/tmc/langchaingo/textsplitter"
)

func TestMdCleaner(t *testing.T) {
	mdPath := "/Users/liwener/programming/ospp/dubbo-admin/ai/reference/k8s_docs/concepts/overview/kubernetes-api.md"
	cleaned, err := utils.CleanMarkdownFile(mdPath)
	if err != nil {
		t.Fatalf("err: %v\n", err)
	}
	fmt.Printf("清洗后的内容:\n%s\n", cleaned)
}

func TestMdChunks(t *testing.T) {
	mdPath := "/Users/liwener/programming/ospp/dubbo-admin/ai/reference/k8s_docs/concepts/overview/kubernetes-api.md"
	cleaned, err := utils.CleanMarkdownFile(mdPath)
	if err != nil {
		t.Fatalf("err: %v\n", err)
	}
	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(1000),
		textsplitter.WithChunkOverlap(100),
		textsplitter.WithSeparators([]string{"\r\n\r\n", "\n\n", "\r\n", "\n", " ", ""}),
	)

	chunks, err := splitter.SplitText(cleaned)
	if err != nil {
		t.Fatalf("err: %v\n", err)
	}
	for i, chunk := range chunks {
		fmt.Printf("第 %d 个chunk:\n%s\n\n", i+1, chunk)
	}
}

func TestMdChunksInDir(t *testing.T) {
	mdDir := "/Users/liwener/programming/ospp/dubbo-admin/ai/reference/k8s_docs/concepts/overview"

	chunks, err := utils.ProcessMarkdownDirectory(mdDir)
	if err != nil {
		t.Fatalf("err: %v\n", err)
	}
	g := manager.Registry(dashscope.Qwen3.Key(), config.PROJECT_ROOT+"/.env", manager.DevLogger())
	err = utils.IndexInPinecone(g, "kubernetes", "concepts", dashscope.Qwen3_embedding.Key(), nil, chunks)
	if err != nil {
		t.Fatalf("err: %v\n", err)
	}
}

func TestMdRetrive(t *testing.T) {
	g := manager.Registry(dashscope.Qwen3.Key(), config.PROJECT_ROOT+"/.env", manager.ProductionLogger())
	query := []string{
		"什么是 Pod？",
		"什么是 Deployment？",
	}
	results, err := utils.RetrieveFromPinecone(g, dashscope.Qwen3_embedding.Key(), "kubernetes", "concepts", query, 10, true, 3)

	if err != nil {
		t.Fatalf("err: %v\n", err)
	}

	for q, docs := range results {
		fmt.Printf("查询: %s\n", q)
		for i, doc := range docs {
			fmt.Printf("结果 %d: %v\n\n", i+1, doc)
		}
	}
}

func TestRAGTool(t *testing.T) {
	g := manager.Registry(dashscope.Qwen3.Key(), config.PROJECT_ROOT+"/.env", manager.ProductionLogger())
	ragTool := tools.RetrieveBasicConceptFromK8SDoc(g, dashscope.Qwen3_embedding.Key(), "kube-docs", 10, 3)
	toolOutput, err := ragTool.RunRaw(context.Background(), tools.K8SRAGQueryInput{Querys: []string{"什么是 Deployment？"}})
	if err != nil {
		t.Fatalf("err: %v\n", err)
	}
	fmt.Printf("RAG Tool Output: \n%v+\n\n", toolOutput)
}
