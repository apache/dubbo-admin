package utils

import (
	"context"

	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/pinecone"
	"github.com/ledongthuc/pdf"
	"github.com/tmc/langchaingo/textsplitter"

	cohere "github.com/cohere-ai/cohere-go/v2"
	cohereClient "github.com/cohere-ai/cohere-go/v2/client"
)

// pdfTextCleaner - 清洗从PDF中提取的文本数据
func TextCleaner(text string) string {
	// 1. 移除控制字符和不可打印字符（保留换行符、制表符和普通空格）
	cleaned := ""
	for _, r := range text {
		if r == '\n' || r == '\t' || r == ' ' || (r >= 32 && r < 127) || r > 127 {
			// 保留换行符、制表符、空格、可打印ASCII字符和非ASCII字符（如中文）
			cleaned += string(r)
		}
	}

	// 2. 移除多余的空白字符和换行符
	cleaned = strings.ReplaceAll(cleaned, "\n \n", "\n")
	cleaned = strings.ReplaceAll(cleaned, " \n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\n ", "\n")

	// 3. 将多个连续的换行符合并为单个换行符
	multipleNewlines := regexp.MustCompile(`\n{3,}`)
	cleaned = multipleNewlines.ReplaceAllString(cleaned, "\n\n")

	// 4. 移除单独的字符行（可能是PDF解析错误）
	lines := strings.Split(cleaned, "\n")
	var cleanedLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过空行
		if line == "" {
			continue
		}
		// 跳过只有1个字符的行（通常是PDF解析错误）
		if len(line) <= 1 {
			continue
		}
		// 跳过只包含特殊字符的行
		if regexp.MustCompile(`^[^\w\s]+$`).MatchString(line) {
			continue
		}
		cleanedLines = append(cleanedLines, line)
	}

	// 5. 重新组合文本
	result := strings.Join(cleanedLines, "\n")

	// 6. 清理常见的PDF解析问题
	// 移除单独的数字（可能是页码）
	result = regexp.MustCompile(`(?m)^\d+$`).ReplaceAllString(result, "")

	// 移除多余的空格
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")

	// 恢复合理的换行
	result = strings.ReplaceAll(result, " \n", "\n")
	result = strings.ReplaceAll(result, "\n ", "\n")

	// 7. 最后的清理
	result = strings.TrimSpace(result)

	return result
}

// Define Pinecone retriever with index name and embedder
func DefinePineconeRetriever(g *genkit.Genkit, indexName, embedderName string) (*pinecone.Docstore, ai.Retriever, error) {
	ctx := context.Background()
	embedder := genkit.LookupEmbedder(g, embedderName)
	if embedder == nil {
		return nil, nil, fmt.Errorf("failed to find embedder %s", embedderName)
	}

	// Define retriever with Google AI embedder
	store, retriever, err := pinecone.DefineRetriever(ctx, g,
		pinecone.Config{
			IndexID:  indexName,
			Embedder: embedder,
		},
		&ai.RetrieverOptions{
			Label:        indexName,
			ConfigSchema: core.InferSchemaMap(pinecone.PineconeRetrieverOptions{}),
		})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to define retriever: %w", err)
	}

	return store, retriever, nil
}

type PineconeResult struct {
	Content  string
	Metadata map[string]any
}

func SearchInPinecone(g *genkit.Genkit, indexName, namespace string, queries []string, topK int) (resp map[string][]*string, err error) {
	if ok := pinecone.IsDefinedRetriever(g, indexName); !ok {
		return nil, fmt.Errorf("retriever '%s' is not defined", indexName)
	}

	ctx := context.Background()
	retriever := pinecone.Retriever(g, indexName)
	option := &pinecone.PineconeRetrieverOptions{
		K:         topK,
		Namespace: namespace,
	}

	resp = make(map[string][]*string, len(queries))
	for _, query := range queries {
		response, err := retriever.Retrieve(ctx, &ai.RetrieverRequest{
			Query:   ai.DocumentFromText(query, nil),
			Options: option,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to search for query '%s': %v", query, err)
		}

		resp[query] = make([]*string, 0, len(response.Documents))
		for _, doc := range response.Documents {
			resp[query] = append(resp[query], &doc.Content[0].Text)
		}
	}

	return resp, nil
}

func Rerank(apiKey, model, query string, documents []*string, topN int) ([]*cohere.RerankResponseResultsItem, error) {
	client := cohereClient.NewClient(cohereClient.WithToken(apiKey))

	var rerankDocs []*cohere.RerankRequestDocumentsItem
	for _, doc := range documents {
		rerankDoc := &cohere.RerankRequestDocumentsItem{}
		rerankDoc.String = *doc
		rerankDocs = append(rerankDocs, rerankDoc)
	}

	rerankResponse, err := client.Rerank(
		context.Background(),
		&cohere.RerankRequest{
			Query:     query,
			Documents: rerankDocs,
			TopN:      &topN,
			Model:     &model,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call rerank API: %w", err)
	}

	return rerankResponse.Results, nil
}

// Helper function to extract plain text from a PDF.
func ReadPDF(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if f != nil {
		defer f.Close()
	}
	if err != nil {
		return "", err
	}

	reader, err := r.GetPlainText()
	if err != nil {
		return "", err
	}

	bytes, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func SplitPDFWithClean(pdfPath string, chunkSize, chunkOverlap int) ([]string, error) {
	pdfText, err := ReadPDF(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF: %w", err)
	}
	cleanedText := TextCleaner(pdfText)

	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(chunkSize),
		textsplitter.WithChunkOverlap(chunkOverlap),
	)
	chunks, err := splitter.SplitText(cleanedText)
	if err != nil {
		return nil, fmt.Errorf("failed to split text: %w", err)
	}

	return chunks, nil
}
