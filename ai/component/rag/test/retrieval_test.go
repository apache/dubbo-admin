/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 (the "License"); you may not use this file except in compliance with
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
	"os"
	"path/filepath"
	"testing"

	compRag "dubbo-admin-ai/component/rag"
	"dubbo-admin-ai/component/rag/loaders"
	"github.com/cloudwego/eino/schema"
)

// TestRetrievalWithScore 测试完整的索引+检索流程，并打印带分数的结果
func TestRetrievalWithScore(t *testing.T) {
	ctx := context.Background()

	// 1. 创建测试文档
	testDir := t.TempDir()
	_ = createTestDocuments(t, testDir)

	// 2. 初始化 Genkit (用于 embedder)
	// 注意：这需要配置环境变量，如 DASHSCOPE_API_KEY
	t.Skip("需要 API Key，手动测试时移除此行")

	// 4. 加载文档
	loader, err := loaders.NewLocalFileLoader(ctx)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	loadedDocs, err := loaders.LoadDirectory(ctx, loader, testDir)
	if err != nil {
		t.Fatalf("Failed to load directory: %v", err)
	}

	t.Logf("=== Loaded %d documents ===", len(loadedDocs))
	for i, doc := range loadedDocs {
		t.Logf("Doc %d: %d chars, metadata: %+v", i, len(doc.Content), doc.MetaData)
	}

	// 5. 分块
	chunkSize := 500
	overLapSize := 50

	chunks := manualSplit(loadedDocs, chunkSize, overLapSize)
	t.Logf("=== Split into %d chunks ===", len(chunks))

	// 6. 打印每个 chunk 的信息
	for i, chunk := range chunks {
		t.Logf("Chunk %d:", i)
		t.Logf("  Content: %q", truncateString(chunk.Content, 100))
		t.Logf("  Metadata: source=%q, title=%q",
			loaders.GetMetadataString(chunk.MetaData, loaders.MetaSource, ""),
			loaders.GetMetadataString(chunk.MetaData, loaders.MetaTitle, ""))
		if page, ok := chunk.MetaData["page"]; ok {
			t.Logf("  Metadata: page=%v", page)
		}
	}
}

// createTestDocuments 创建测试文档
func createTestDocuments(t *testing.T, dir string) []string {
	docs := []string{
		"introduction.md",
		"getting_started.md",
		"configuration.md",
	}

	content := map[string]string{
		"introduction.md": "# Introduction\n\nDubbo is a high-performance, lightweight Java RPC framework. It provides three core capabilities: remote interface invocation, graceful fault tolerance, and service discovery.\n\n## Key Features\n\n- High performance\n- Lightweight\n- Easy to use\n",
		"getting_started.md": "# Getting Started\n\n## Installation\n\nAdd Dubbo dependency to your project:\n\nUse Maven or Gradle to add the dependency.\n\n## Quick Start\n\n1. Define service interface\n2. Implement service provider\n3. Configure service reference\n",
		"configuration.md": "# Configuration\n\n## Provider Configuration\n\nConfigure the protocol, port, and registry address.\n\n## Consumer Configuration\n\nConfigure reference to the provider service.\n\n## Registry\n\nDubbo supports multiple registry types: Zookeeper, Nacos, etc.\n",
	}

	for _, filename := range docs {
		path := filepath.Join(dir, filename)
		if err := os.WriteFile(path, []byte(content[filename]), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	return docs
}

// manualSplit 手动分块（用于测试）
func manualSplit(docs []*schema.Document, chunkSize, overlapSize int) []*schema.Document {
	var chunks []*schema.Document
	chunkIndex := 0

	for _, doc := range docs {
		content := doc.Content
		source := loaders.GetMetadataString(doc.MetaData, loaders.MetaSource, "")
		title := loaders.GetMetadataString(doc.MetaData, loaders.MetaTitle, "")

		for start := 0; start < len(content); {
			end := start + chunkSize
			if end > len(content) {
				end = len(content)
			}

			chunk := &schema.Document{
				Content: content[start:end],
				MetaData: map[string]any{
					"source":      source,
					"title":       title,
					"chunk_index": chunkIndex,
					"chunk_start": start,
					"chunk_size":   end - start,
				},
			}
			chunks = append(chunks, chunk)
			chunkIndex++

			// Overlap
			if end >= len(content) {
				break
			}
			start = end - overlapSize
		}
	}

	return chunks
}

// TestMockRetrievalWithScore 模拟检索并打印带分数的结果
func TestMockRetrievalWithScore(t *testing.T) {
	// 模拟的文档数据
	mockChunks := []*schema.Document{
		{
			Content: "Dubbo is a high-performance, lightweight Java RPC framework.",
			MetaData: map[string]any{
				"source":      "/docs/introduction.md",
				"title":       "Introduction",
				"chunk_index": 0,
			},
		},
		{
			Content: "It provides three core capabilities: remote interface invocation.",
			MetaData: map[string]any{
				"source":      "/docs/introduction.md",
				"title":       "Introduction",
				"chunk_index": 1,
			},
		},
		{
			Content: "Add Dubbo dependency to your project using Maven or Gradle.",
			MetaData: map[string]any{
				"source":      "/docs/getting_started.md",
				"title":       "Getting Started",
				"chunk_index": 0,
			},
		},
	}

	// 模拟检索分数（余弦相似度）
	mockScores := []float64{0.92, 0.85, 0.78}

	t.Log("=== Mock Retrieval Results (with Score) ===")

	for i, chunk := range mockChunks {
		score := mockScores[i]
		source := loaders.GetMetadataString(chunk.MetaData, loaders.MetaSource, "")
		title := loaders.GetMetadataString(chunk.MetaData, loaders.MetaTitle, "")
		chunkIndex := loaders.GetMetadataInt(chunk.MetaData, loaders.MetaChunkIndex, 0)

		t.Logf("Result %d:", i)
		t.Logf("  Score: %.4f", score)
		t.Logf("  Content: %q", truncateString(chunk.Content, 60))
		t.Logf("  Source: %s", source)
		t.Logf("  Title: %s", title)
		t.Logf("  ChunkIndex: %d", chunkIndex)
		t.Log("  ---")
	}

	// 打印 RetrieveResult 格式
	t.Log("\n=== RetrieveResult Format ===")
	for i, chunk := range mockChunks {
		result := &compRag.RetrieveResult{
			Content: chunk.Content,
			Score:   mockScores[i],
			Source:  loaders.GetMetadataString(chunk.MetaData, loaders.MetaSource, ""),
			Title:   loaders.GetMetadataString(chunk.MetaData, loaders.MetaTitle, ""),
			Metadata: chunk.MetaData,
		}
		t.Logf("Result[%d]: Score=%.4f, Source=%q, Title=%q",
			i, result.Score, result.Source, result.Title)
	}
}
