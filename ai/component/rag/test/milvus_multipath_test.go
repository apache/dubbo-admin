/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ragtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	compRag "dubbo-admin-ai/component/rag"
	"dubbo-admin-ai/component/rag/indexers"
	"dubbo-admin-ai/component/rag/mergers"
	"dubbo-admin-ai/component/rag/retrievers"
	"github.com/cloudwego/eino/schema"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/joho/godotenv"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/openai/openai-go/option"
)

const (
	// Test collection name
	testCollection = "dubbo_rag_test"
	// DashScope embedder name
	embedderName = "dashscope/text-embedding-v4"
	// DashScope embedding dimension
	embedderDim = 1024
)

// ANSI colors
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[90m"
)

// init loads .env file
func init() {
	dir, _ := os.Getwd()
	for {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			_ = godotenv.Load(envPath)
			return
		}
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			return
		}
		dir = parentDir
	}
}

// Printer provides colored output
type Printer struct {
	t *testing.T
}

func NewPrinter(t *testing.T) *Printer {
	return &Printer{t: t}
}

func (p *Printer) Title(s string) {
	p.t.Log("\n" + Cyan + strings.Repeat("═", 60) + Reset)
	p.t.Log(Cyan + "  " + s + Reset)
	p.t.Log(Cyan + strings.Repeat("═", 60) + Reset)
}

func (p *Printer) Section(s string) {
	p.t.Log("\n" + Yellow + "─── " + s + " ───" + Reset)
}

func (p *Printer) Success(s string) {
	p.t.Log(Green + "✓ " + s + Reset)
}

func (p *Printer) Info(s string) {
	p.t.Log("  " + s)
}

func (p *Printer) Error(s string) {
	p.t.Log(Red + "✗ " + s + Reset)
}

func (p *Printer) Warn(s string) {
	p.t.Log(Yellow + "⚠ " + s + Reset)
}

func (p *Printer) KeyValue(key, value string) {
	p.t.Logf("    %s: %s%s%s", key, Purple, value, Reset)
}

// setupGenkitWithEmbedder initializes genkit with DashScope embedder
func setupGenkitWithEmbedder(t *testing.T) (*genkit.Genkit, context.Context) {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		// Try alternative env var names
		apiKey = os.Getenv("EMBEDDING_API_KEY")
	}
	if apiKey == "" {
		t.Skip("DASHSCOPE_API_KEY/EMBEDDING_API_KEY not configured")
	}

	baseURL := os.Getenv("EMBEDDING_BASE_URL")
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	// Remove /embeddings suffix if present
	if len(baseURL) > 12 && baseURL[len(baseURL)-11:] == "/embeddings" {
		baseURL = baseURL[:len(baseURL)-11]
	}

	ctx := context.Background()

	dashscopePlugin := &compat_oai.OpenAICompatible{
		Provider: "dashscope",
		Opts: []option.RequestOption{
			option.WithAPIKey(apiKey),
			option.WithBaseURL(baseURL),
		},
	}

	g := genkit.Init(ctx, genkit.WithPlugins(dashscopePlugin))

	embedder := dashscopePlugin.DefineEmbedder("dashscope", "text-embedding-v4", &ai.EmbedderOptions{
		Label:      "text-embedding-v4",
		Supports:   &ai.EmbedderSupports{Input: []string{"text"}},
		Dimensions: embedderDim,
	})

	genkit.RegisterAction(g, embedder)

	if genkit.LookupEmbedder(g, embedderName) == nil {
		t.Fatalf("Failed to lookup embedder: %s", embedderName)
	}

	return g, ctx
}

// getMilvusClient creates Milvus client
func getMilvusClient(t *testing.T, ctx context.Context) *milvusclient.Client {
	host := os.Getenv("MILVUS_HOST")
	token := os.Getenv("MILVUS_TOKEN")

	if host == "" || token == "" {
		t.Skip("MILVUS_HOST or MILVUS_TOKEN not set")
	}

	cli, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: host,
		APIKey:  token,
	})
	if err != nil {
		t.Fatalf("Failed to create Milvus client: %v", err)
	}

	return cli
}

// getTestDocuments returns test documents
func getTestDocuments() []*schema.Document {
	return []*schema.Document{
		{ID: "doc1", Content: "Apache Dubbo 是一个高性能的 Java RPC 框架，提供了服务自动发现、负载均衡、流量控制等功能。它支持多种协议包括 Dubbo、REST、gRPC 等，广泛用于微服务架构中。"},
		{ID: "doc2", Content: "Milvus 是一个开源的向量数据库，专为海量向量数据的相似性搜索而设计。它支持多种索引类型，包括 HNSW、IVF 等，并提供高性能的向量检索能力。"},
		{ID: "doc3", Content: "BM25 是一种用于信息检索的排序算法，广泛应用于搜索引擎中。它通过词频和文档长度归一化来计算文档相关性，比传统的词频统计效果更好。"},
		{ID: "doc4", Content: "Dubbo Admin 是 Dubbo 的管理控制台，提供了服务治理、监控、动态配置等功能，帮助开发者更好地管理和维护 Dubbo 服务。"},
		{ID: "doc5", Content: "向量搜索是将查询向量与数据库中的向量进行相似度计算，找出最相似的结果。密集向量通常由神经网络模型如 BERT、text-embedding-ada-002 等生成。"},
		{ID: "doc6", Content: "稀疏向量搜索使用 BM25 等算法，基于关键词匹配进行检索。它不需要预先计算向量，可以直接使用原始文本进行搜索。"},
		{ID: "doc7", Content: "混合搜索结合了密集向量搜索和稀疏向量搜索的优势，可以同时捕获语义相似性和关键词匹配，提高检索准确率。"},
		{ID: "doc8", Content: "Zilliz Cloud 是基于 Milvus 的全托管向量数据库服务，提供了简单易用的 API 和自动扩展能力，开发者无需运维基础设施。"},
		{ID: "doc9", Content: "Go 语言（Golang）由 Google 开发，是一种静态类型、编译型语言，具有简洁的语法和强大的并发支持，非常适合构建高性能的分布式系统。"},
		{ID: "doc10", Content: "Kubernetes 是一个开源的容器编排平台，用于自动化部署、扩展和管理容器化应用。它提供了服务发现、负载均衡、滚动更新等功能。"},
	}
}

// getTestQueries returns test queries
func getTestQueries() []string {
	return []string{
		"RPC 框架的功能有哪些",
		"向量数据库如何工作",
		"BM25 算法的原理",
		"Go 语言的特点",
		"容器编排平台",
	}
}

// printResultsTable prints results comparison table
func printResultsTable(p *Printer, query string, denseResults, bm25Results, hybridResults []*schema.Document) {
	p.Section("查询: " + query)

	p.t.Log("\n  ┌─────────────────────────┬──────────┬────────────────────────────────────────┐")
	p.t.Log("  │ 搜索类型                │ 结果数   │ 首条结果预览                           │")
	p.t.Log("  ├─────────────────────────┼──────────┼────────────────────────────────────────┤")

	// Dense row
	densePreview := ""
	if len(denseResults) > 0 {
		densePreview = truncateString(denseResults[0].Content, 38)
	}
	p.t.Log(fmt.Sprintf("  │ %-23s │ %-8d │ %-38s │",
		"Dense 向量", len(denseResults), densePreview))

	// BM25 row
	bm25Preview := ""
	if len(bm25Results) > 0 {
		bm25Preview = truncateString(bm25Results[0].Content, 38)
	}
	p.t.Log(fmt.Sprintf("  │ %-23s │ %-8d │ %-38s │",
		"BM25 稀疏", len(bm25Results), bm25Preview))

	// Hybrid row
	hybridPreview := ""
	if len(hybridResults) > 0 {
		hybridPreview = truncateString(hybridResults[0].Content, 38)
	}
	p.t.Log(fmt.Sprintf("  │ %-23s │ %-8d │ %-38s │",
		"Hybrid 混合", len(hybridResults), hybridPreview))

	p.t.Log("  └─────────────────────────┴──────────┴────────────────────────────────────────┘")
}

// TestMilvusMultiPathRetrieval tests Milvus multi-path retrieval
func TestMilvusMultiPathRetrieval(t *testing.T) {
	p := NewPrinter(t)

	p.Title("Milvus 多路召回测试")

	// Get environment variables
	host := os.Getenv("MILVUS_HOST")
	token := os.Getenv("MILVUS_TOKEN")

	if host == "" || token == "" {
		t.Skip("MILVUS_HOST or MILVUS_TOKEN not set")
	}

	p.KeyValue("Milvus 地址", host)
	p.KeyValue("Collection", testCollection)
	p.KeyValue("Embedder", embedderName)
	p.KeyValue("向量维度", fmt.Sprintf("%d", embedderDim))

	// Initialize
	g, ctx := setupGenkitWithEmbedder(t)
	p.Success("DashScope embedder 初始化成功")

	cli := getMilvusClient(t, ctx)
	defer cli.Close(ctx)

	// Check if collection exists
	hasCollection, _ := cli.HasCollection(ctx, milvusclient.NewHasCollectionOption(testCollection))

	// Create or use existing collection
	p.Section("准备 Collection")
	idxerCfg := &indexers.MilvusConfig{
		Address:     host,
		Token:       token,
		Collection:  testCollection,
		Dimension:   embedderDim,
		Embedder:    embedderName,
		EnableBM25:  true,
		BM25K1:      1.2,
		BM25B:       0.75,
		DenseField:  "vector",
		TextField:   "text",
		SparseField: "sparse",
	}

	idxer, err := indexers.NewMilvusIndexer(g, idxerCfg)
	if err != nil {
		t.Fatalf("创建 indexer 失败: %v", err)
	}
	defer idxer.Close()

	if hasCollection {
		p.Info("使用已存在的 Collection")
	} else {
		p.Success("Collection 创建成功")
	}
	p.KeyValue("Dense 向量字段", "vector (dim=1024)")
	p.KeyValue("BM25 全文搜索", "text → sparse")

	time.Sleep(2 * time.Second)

	// Insert test data only if collection doesn't exist
	p.Section("准备测试数据")
	docs := getTestDocuments()

	if !hasCollection {
		p.Info("插入测试数据...")
		ids, err := idxer.Store(ctx, docs)
		if err != nil {
			t.Fatalf("插入文档失败: %v", err)
		}
		p.Success(fmt.Sprintf("插入了 %d 个文档", len(ids)))
		p.KeyValue("文档 IDs", fmt.Sprintf("%v", ids))
		p.Info("等待索引构建...")
		time.Sleep(10 * time.Second)
	} else {
		p.Info(fmt.Sprintf("Collection 已存在，跳过插入（包含 %d 个测试文档）", len(docs)))
		time.Sleep(2 * time.Second)
	}

	// Create retrievers
	p.Section("创建 Retriever")

	denseCfg := &retrievers.MilvusConfig{
		Address:    host,
		Token:      token,
		Collection: testCollection,
		TextField:  "text",
		Embedder:   embedderName,
		SearchType: "dense",
		DenseField: "vector",
		DenseTopK:  3,
	}

	bm25Cfg := &retrievers.MilvusConfig{
		Address:     host,
		Token:       token,
		Collection:  testCollection,
		TextField:   "text",
		Embedder:    embedderName,
		SearchType:  "sparse",
		SparseField: "sparse",
		EnableBM25:  true,
		SparseTopK:  3,
	}

	hybridCfg := &retrievers.MilvusConfig{
		Address:      host,
		Token:        token,
		Collection:   testCollection,
		TextField:    "text",
		Embedder:     embedderName,
		SearchType:   "hybrid",
		DenseField:   "vector",
		SparseField:  "sparse",
		EnableBM25:   true,
		DenseTopK:    3,
		SparseTopK:   3,
		DenseWeight:  0.7,
		SparseWeight: 0.3,
	}

	denseRtv, err := retrievers.NewMilvusRetriever(g, denseCfg)
	if err != nil {
		t.Fatalf("创建 Dense retriever 失败: %v", err)
	}
	defer denseRtv.Close()
	p.Success("Dense Retriever 创建成功")

	bm25Rtv, err := retrievers.NewMilvusRetriever(g, bm25Cfg)
	if err != nil {
		t.Fatalf("创建 BM25 retriever 失败: %v", err)
	}
	defer bm25Rtv.Close()
	p.Success("BM25 Retriever 创建成功")

	hybridRtv, err := retrievers.NewMilvusRetriever(g, hybridCfg)
	if err != nil {
		t.Fatalf("创建 Hybrid retriever 失败: %v", err)
	}
	defer hybridRtv.Close()
	p.Success("Hybrid Retriever 创建成功")

	// Execute multi-path retrieval tests
	queries := getTestQueries()

	p.Title("多路召回对比测试")

	totalDense, totalBM25, totalHybrid := 0, 0, 0

	for _, query := range queries {
		denseResults, _ := denseRtv.Retrieve(ctx, query)
		bm25Results, _ := bm25Rtv.Retrieve(ctx, query)
		hybridResults, _ := hybridRtv.Retrieve(ctx, query)

		totalDense += len(denseResults)
		totalBM25 += len(bm25Results)
		totalHybrid += len(hybridResults)

		printResultsTable(p, query, denseResults, bm25Results, hybridResults)
	}

	// Print statistics
	p.Title("测试统计摘要")

	p.Section("召回统计")
	p.t.Log("\n  ┌─────────────────────────┬──────────┬──────────────┐")
	p.t.Logf("  │ 搜索类型                │ 总召回数 │ 平均召回数   │")
	p.t.Logf("  ├─────────────────────────┼──────────┼──────────────┤")
	p.t.Logf("  │ %-23s │ %-8d │ %-12.1f │",
		"Dense 向量", totalDense, float64(totalDense)/float64(len(queries)))
	p.t.Logf("  │ %-23s │ %-8d │ %-12.1f │",
		"BM25 稀疏", totalBM25, float64(totalBM25)/float64(len(queries)))
	p.t.Logf("  │ %-23s │ %-8d │ %-12.1f │",
		"Hybrid 混合", totalHybrid, float64(totalHybrid)/float64(len(queries)))
	p.t.Logf("  └─────────────────────────┴──────────┴──────────────┘")

	p.Section("召回特点分析")
	p.t.Log("  • Dense 向量搜索: 捕获语义相似性，关键词不匹配也能找到相关内容")
	p.t.Log("  • BM25 稀疏搜索: 基于关键词精确匹配，适合查询特定术语")
	p.t.Log("  • Hybrid 混合搜索: 结合两者优势")

	p.Success("所有测试完成！")
}

// TestMilvusMultiPathWithMerge tests multi-path retrieval with RRF merge
func TestMilvusMultiPathWithMerge(t *testing.T) {
	p := NewPrinter(t)

	p.Title("Milvus 多路召回 + RRF 合并测试")

	host := os.Getenv("MILVUS_HOST")
	token := os.Getenv("MILVUS_TOKEN")

	if host == "" || token == "" {
		t.Skip("MILVUS_HOST or MILVUS_TOKEN not set")
	}

	p.KeyValue("Milvus 地址", host)
	p.KeyValue("Collection", testCollection)

	// Initialize
	g, ctx := setupGenkitWithEmbedder(t)
	p.Success("DashScope embedder 初始化成功")

	// Create retrievers
	p.Section("创建多路 Retriever")

	denseCfg := &retrievers.MilvusConfig{
		Address:    host,
		Token:      token,
		Collection: testCollection,
		TextField:  "text",
		Embedder:   embedderName,
		SearchType: "dense",
		DenseField: "vector",
		DenseTopK:  10,
	}

	sparseCfg := &retrievers.MilvusConfig{
		Address:     host,
		Token:       token,
		Collection:  testCollection,
		TextField:   "text",
		Embedder:    embedderName,
		SearchType:  "sparse",
		SparseField: "sparse",
		EnableBM25:  true,
		SparseTopK:  10,
	}

	denseRtv, err := retrievers.NewMilvusRetriever(g, denseCfg)
	if err != nil {
		t.Fatalf("创建 Dense retriever 失败: %v", err)
	}
	defer denseRtv.Close()
	p.Success("Dense Retriever 创建成功")

	sparseRtv, err := retrievers.NewMilvusRetriever(g, sparseCfg)
	if err != nil {
		p.Warn(fmt.Sprintf("创建 Sparse retriever 失败: %v (将仅使用 Dense)", err))
		sparseRtv = nil
	} else {
		defer sparseRtv.Close()
		p.Success("Sparse Retriever 创建成功")
	}

	// Create RAG with multi-path retrieval and merge
	p.Section("配置多路召回 + RRF 合并")

	retrievalPaths := []*compRag.RetrievalPath{
		{
			Label:     "dense",
			Retriever: denseRtv,
			TopK:      10,
			Weight:    0.7,
		},
	}

	if sparseRtv != nil {
		retrievalPaths = append(retrievalPaths, &compRag.RetrievalPath{
			Label:     "sparse",
			Retriever: sparseRtv,
			TopK:      10,
			Weight:    0.3,
		})
		p.Info("配置: Dense (权重 0.7) + Sparse (权重 0.3)")
	} else {
		p.Info("配置: Dense (权重 1.0)")
	}

	rag := &compRag.RAG{
		RetrievalPaths: retrievalPaths,
		Merger: mergers.NewMergeLayer(&mergers.MergeConfig{
			Strategy:        "rrf",
			NormalizeMethod: "minmax",
			EnableDedup:     true,
			TopK:            10,
			RRFK:            60,
		}),
	}

	p.Success("RRF 合并层配置完成")
	p.KeyValue("合并策略", "RRF (Reciprocal Rank Fusion)")
	p.KeyValue("归一化方法", "MinMax")
	p.KeyValue("去重", "启用")

	// Test queries
	queries := getTestQueries()

	p.Title("多路召回 + 合并测试")

	for _, query := range queries {
		p.Section("查询: " + query)

		// Test retrievers directly
		denseDocs, denseErr := denseRtv.Retrieve(ctx, query)
		if denseErr != nil {
			p.Warn(fmt.Sprintf("Dense Retriever 失败: %v", denseErr))
		} else {
			p.Info(fmt.Sprintf("Dense Retriever: %d 个结果", len(denseDocs)))
		}

		if sparseRtv != nil {
			sparseDocs, sparseErr := sparseRtv.Retrieve(ctx, query)
			if sparseErr != nil {
				p.Warn(fmt.Sprintf("Sparse Retriever 失败: %v", sparseErr))
			} else {
				p.Info(fmt.Sprintf("Sparse Retriever: %d 个结果", len(sparseDocs)))
			}
		}

		// Test merge directly
		testPaths := []*mergers.MultiPathResult{
			{Label: "test", Results: denseDocs, Weight: 1.0},
		}
		mergedDocs, mergeErr := rag.Merger.Merge(ctx, testPaths)
		if mergeErr != nil {
			p.Warn(fmt.Sprintf("Merge 失败: %v", mergeErr))
		} else {
			p.Info(fmt.Sprintf("Merge 后: %d 个结果", len(mergedDocs)))
		}

		req := &compRag.RetrieveRequest{
			Query: query,
			TopK:  10,
		}

		resp, err := rag.RetrieveV2(ctx, req)
		if err != nil {
			p.Error(fmt.Sprintf("RetrieveV2 失败: %v", err))
			continue
		}

		p.Info(fmt.Sprintf("RetrieveV2 返回 %d 个结果", len(resp.Results)))

		for i, result := range resp.Results {
			p.t.Logf("  [%d] Score: %.4f | Content: %s",
				i+1, result.Score, truncateString(result.Content, 60))
			if result.Source != "" {
				p.t.Logf("      Source: %s", result.Source)
			}
		}

		if resp.RetrievalMeta != nil {
			if total, ok := resp.RetrievalMeta["total_results"].(int); ok {
				p.Info(fmt.Sprintf("合并前总结果数: %d", total))
			}
		}
	}

	// Compare single vs multi path
	p.Section("单路 vs 多路对比")

	singleRAG := &compRag.RAG{
		RetrievalPaths: []*compRag.RetrievalPath{
			{
				Label:     "dense",
				Retriever: denseRtv,
				TopK:      10,
				Weight:    1.0,
			},
		},
		Merger: rag.Merger,
	}

	testQuery := "微服务架构的服务发现"
	singleReq := &compRag.RetrieveRequest{
		Query: testQuery,
		TopK:  10,
	}
	multiReq := &compRag.RetrieveRequest{
		Query: testQuery,
		TopK:  10,
	}

	singleResp, _ := singleRAG.RetrieveV2(ctx, singleReq)
	multiResp, _ := rag.RetrieveV2(ctx, multiReq)

	p.KeyValue("查询", testQuery)
	p.KeyValue("单路召回数", fmt.Sprintf("%d", len(singleResp.Results)))
	p.KeyValue("多路召回数", fmt.Sprintf("%d", len(multiResp.Results)))

	p.Success("测试完成！")
}
