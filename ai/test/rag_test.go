package test

import (
	"context"
	"fmt"
	"testing"

	"dubbo-admin-ai/config"
	"dubbo-admin-ai/manager"
	"dubbo-admin-ai/plugins/dashscope"
	"dubbo-admin-ai/utils"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/pinecone"
)

func pdf2Docs(pdfPath string, chunkSize, chunkOverlap int) ([]*ai.Document, error) {
	chunks, err := utils.SplitPDFWithClean(pdfPath, chunkSize, chunkOverlap)
	if err != nil {
		return nil, fmt.Errorf("Failed to split PDF into chunks: %w", err)
	}

	metadata := map[string]any{
		"description": "A collection of classic cocktail recipes.",
	}

	docs := make([]*ai.Document, len(chunks))
	for i, chunk := range chunks {
		docs[i] = ai.DocumentFromText(chunk, metadata)
	}

	return docs, nil
}

func docs2Index(g *genkit.Genkit, docs []*ai.Document, indexName, namespace string) error {
	ctx := context.Background()
	embedder := genkit.LookupEmbedder(g, dashscope.Qwen3_embedding.Key())
	if embedder == nil {
		return fmt.Errorf("failed to find embedder %s", dashscope.Qwen3_embedding.Key())
	}
	docstore, _, err := pinecone.DefineRetriever(ctx, g,
		pinecone.Config{
			IndexID:  indexName,
			Embedder: embedder,
		},
		&ai.RetrieverOptions{
			Label:        "cocktail-retriever",
			ConfigSchema: core.InferSchemaMap(pinecone.PineconeRetrieverOptions{}),
		})

	if err != nil {
		return fmt.Errorf("failed to setup retriever: %w", err)
	}

	// 分批索引文档，每批最多10个
	batchSize := 10
	for i := 0; i < len(docs); i += batchSize {
		end := min(i+batchSize, len(docs))
		batch := docs[i:end]
		manager.GetLogger().Info("正在索引文档", "start", i+1, "end", end, "total", len(docs))
		if err := pinecone.Index(ctx, batch, docstore, namespace); err != nil {
			return fmt.Errorf("failed to index documents batch %d-%d: %w", i+1, end, err)
		}
		manager.GetLogger().Info("成功索引文档", "count", len(batch))
	}

	return nil
}

func TestChunks(t *testing.T) {
	pdfPath := config.PROJECT_ROOT + "/reference/Classic-Cocktails.pdf"
	chunks, err := utils.SplitPDFWithClean(pdfPath, 50, 10)
	if err != nil {
		t.Fatal("Failed to split PDF into chunks:", err)
	}
	manager.ProductionLogger().Info("成功分割 PDF", "length", len(chunks), "chunks", chunks)
}

// TestCreateIndex - 创建索引
func TestCreateIndex(t *testing.T) {
	namespace := "cocktails"
	pdfPath := config.PROJECT_ROOT + "/reference/Classic-Cocktails.pdf"
	docs, err := pdf2Docs(pdfPath, 100, 20)
	if err != nil {
		t.Fatal("Failed to convert PDF to documents:", err)
	}

	g := manager.Registry(config.DEFAULT_MODEL.Key(), config.PROJECT_ROOT+"/.env", manager.ProductionLogger())
	err = docs2Index(g, docs, config.PINECONE_INDEX_NAME, namespace)
	if err != nil {
		t.Fatal("Failed to create index:", err)
	}
}

// func TestReRank(t *testing.T) {
// 	queries := []string{
// 		"请给我玛格丽特鸡尾酒的配方",
// 		"Please give me the recipe of Margarita cocktail",
// 		"What are the ingredients of whiskey sour?",
// 		"How to make a vodka martini?",
// 		"What are tropical fruit cocktails?",
// 	}

// }

// TestSearch - 搜索测试（可以运行多次）
func TestSearch(t *testing.T) {
	g := manager.Registry(config.DEFAULT_MODEL.Key(), config.PROJECT_ROOT+"/.env", manager.ProductionLogger())
	indexName := config.PINECONE_INDEX_NAME
	queries := []string{
		"请给我玛格丽特鸡尾酒的配方",
		"Please give me the recipe of Margarita cocktail",
		"What are the ingredients of whiskey sour?",
		"How to make a vodka martini?",
		"What are tropical fruit cocktails?",
	}
	results, err := utils.RetrieveFromPinecone(g, dashscope.Qwen3_embedding.Key(), indexName, "cocktails", queries, 10, true, 5)
	if err != nil {
		t.Fatalf("search in pinecone failed: %v", err)
	}

	for query, docs := range results {
		manager.GetLogger().Info("搜索结果", "query", query, "result", docs)
	}

}

func TestRerank(t *testing.T) {
	g := manager.Registry(config.DEFAULT_MODEL.Key(), config.PROJECT_ROOT+"/.env", manager.ProductionLogger())
	indexName := config.PINECONE_INDEX_NAME
	queries := []string{
		"What are the ingredients of whiskey sour?",
	}
	results, err := utils.RetrieveFromPinecone(g, dashscope.Qwen3_embedding.Key(), indexName, "cocktails", queries, 10, true, 5)
	if err != nil {
		t.Fatalf("search in pinecone failed: %v", err)
	}
	manager.GetLogger().Info("重排序结果", "result", results)
}
