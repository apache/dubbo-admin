package utils

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"dubbo-admin-ai/config"
	"dubbo-admin-ai/manager"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/pinecone"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
	"github.com/ledongthuc/pdf"
	"github.com/tmc/langchaingo/textsplitter"

	cohere "github.com/cohere-ai/cohere-go/v2"
	cohereClient "github.com/cohere-ai/cohere-go/v2/client"
)

type PineconeResult struct {
	Content        string
	RelevanceScore float64
}

func IndexInPinecone(g *genkit.Genkit, indexName string, namespace string, embedderName string, metadata map[string]any, chunks []string) error {
	docs := make([]*ai.Document, len(chunks))
	for i, chunk := range chunks {
		docs[i] = ai.DocumentFromText(chunk, metadata)
	}

	ctx := context.Background()
	embedder := genkit.LookupEmbedder(g, embedderName)
	if embedder == nil {
		return fmt.Errorf("failed to find embedder %s", embedderName)
	}
	docstore, _, err := pinecone.DefineRetriever(ctx, g,
		pinecone.Config{
			IndexID:  indexName,
			Embedder: embedder,
		},
		&ai.RetrieverOptions{
			Label:        indexName,
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

func RetrieveFromPinecone(g *genkit.Genkit, embedderName, indexName, namespace string, queries []string, topK int, rerank bool, topN int) (resp map[string][]*PineconeResult, err error) {
	ctx := context.Background()
	embedder := genkit.LookupEmbedder(g, embedderName)
	if embedder == nil {
		return nil, fmt.Errorf("failed to find embedder %s", embedderName)
	}

	// Define retriever with embedder
	var retriever ai.Retriever
	if !pinecone.IsDefinedRetriever(g, indexName) {
		_, retriever, err = pinecone.DefineRetriever(ctx, g,
			pinecone.Config{
				IndexID:  indexName,
				Embedder: embedder,
			},
			&ai.RetrieverOptions{
				Label:        indexName,
				ConfigSchema: core.InferSchemaMap(pinecone.PineconeRetrieverOptions{}),
			})
	} else {
		retriever = pinecone.Retriever(g, indexName)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to define retriever: %w", err)
	}

	// Search for each query
	option := &pinecone.PineconeRetrieverOptions{
		K:         topK,
		Namespace: namespace,
	}
	resp = make(map[string][]*PineconeResult, len(queries))
	for _, query := range queries {
		response, err := retriever.Retrieve(ctx, &ai.RetrieverRequest{
			Query:   ai.DocumentFromText(query, nil),
			Options: option,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to search for query '%s': %v", query, err)
		}

		resp[query] = make([]*PineconeResult, 0, len(response.Documents))
		for _, doc := range response.Documents {
			// 初始化 PineconeResult，暂时不设置 RelevanceScore（将在 rerank 后设置）
			result := &PineconeResult{
				Content:        doc.Content[0].Text,
				RelevanceScore: 0,
			}
			resp[query] = append(resp[query], result)
		}
	}

	results := make(map[string][]*PineconeResult, len(queries))
	if rerank && len(queries) > 0 {
		for query, docs := range resp {
			// 提取文档内容用于 rerank
			docTexts := make([]*string, len(docs))
			for i, doc := range docs {
				docTexts[i] = &doc.Content
			}

			rerankRes, err := Rerank(config.COHERE_API_KEY, config.RERANK_MODEL, query, docTexts, topN)
			if err != nil {
				return nil, err
			}

			// 根据 rerank 结果构建最终结果，包含 RelevanceScore
			for _, res := range rerankRes {
				originalDoc := docs[res.Index]
				resultDoc := &PineconeResult{
					Content:        originalDoc.Content,
					RelevanceScore: res.RelevanceScore,
				}
				results[query] = append(results[query], resultDoc)
			}
		}
	}

	return results, nil
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

// pdfTextCleaner - 清洗从PDF中提取的文本数据
func pdfTextCleaner(text string) string {
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

func SplitPDFWithClean(pdfPath string, chunkSize, chunkOverlap int) ([]string, error) {
	pdfText, err := ReadPDF(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF: %w", err)
	}
	cleanedText := pdfTextCleaner(pdfText)

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

// MarkdownCleaner 用于清洗 Markdown 文档为 RAG 友好的纯文本
type MarkdownCleaner struct {
	// 配置选项
	preserveCodeContent   bool // 是否保留代码块内容
	preserveListStructure bool // 是否保留列表结构
	preserveTableContent  bool // 是否保留表格内容
	maxLineLength         int  // 最大行长度

	// 内部状态
	result    strings.Builder
	inList    bool
	listDepth int
	inTable   bool
}

// NewMarkdownCleaner 创建新的清洗器实例
func NewMarkdownCleaner() *MarkdownCleaner {
	return &MarkdownCleaner{
		preserveCodeContent:   true,
		preserveListStructure: true,
		preserveTableContent:  true,
		maxLineLength:         500,
	}
}

// SetOptions 设置清洗器选项
func (c *MarkdownCleaner) SetOptions(preserveCode, preserveList, preserveTable bool, maxLineLen int) {
	c.preserveCodeContent = preserveCode
	c.preserveListStructure = preserveList
	c.preserveTableContent = preserveTable
	c.maxLineLength = maxLineLen
}

// Clean 清洗 Markdown 文本
func (c *MarkdownCleaner) Clean(markdown string) string {
	// 重置前置元数据和内部状态
	c.result.Reset()
	c.inList = false
	c.listDepth = 0
	c.inTable = false

	// 预处理：删除 frontmatter 和 Hugo shortcodes
	markdown = c.removeFrontmatter(markdown)
	// 移除 HugoShortcodes
	hugoRe := regexp.MustCompile(`{{<[^>]+>}}|{{%[^%]+%}}`)
	markdown = hugoRe.ReplaceAllString(markdown, "")

	// 创建解析器，包含 Frontmatter 扩展
	extensions := parser.CommonExtensions | parser.Mmark | parser.Footnotes
	p := parser.NewWithExtensions(extensions)

	// 解析 markdown 为 AST
	doc := p.Parse([]byte(markdown))

	// 遍历 AST 并提取内容
	c.walkAST(doc)

	// 后处理
	cleaned := c.postProcess(c.result.String())

	return cleaned
}

// walkAST 遍历 AST 节点
func (c *MarkdownCleaner) walkAST(node ast.Node) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.Document:
		c.processChildren(n)

	case *ast.Heading:
		c.result.WriteString("\r\n")
		c.processHeading(n)

	case *ast.Paragraph:
		c.processParagraph(n)
		c.result.WriteString("\r\n")

	case *ast.List:
		c.processList(n)
		c.result.WriteString("\n")

	case *ast.ListItem:
		c.processListItem(n)

	case *ast.CodeBlock:
		c.processCodeBlock(n)
		c.result.WriteString("\n")

	case *ast.Table:
		c.result.WriteString("\r\n")
		c.processTable(n)
		c.result.WriteString("\r\n")

	case *ast.TableRow:
		c.processTableRow(n)
		c.result.WriteString("\n")

	case *ast.TableCell:
		c.processTableCell(n)

	case *ast.Text:
		c.processText(n)

	case *ast.Emph, *ast.Strong:
		c.processChildren(n)

	case *ast.Link:
		c.processLink(n)

	case *ast.Image:
		c.processImage(n)

	case *ast.Code:
		c.processInlineCode(n)

	case *ast.Softbreak, *ast.Hardbreak:
		c.result.WriteString(" ")

	case *ast.BlockQuote:
		c.processBlockQuote(n)

	case *ast.HorizontalRule:
		c.result.WriteString("\n")

	default:
		c.result.WriteString("")
	}
}

// processChildren 处理子节点
func (c *MarkdownCleaner) processChildren(node ast.Node) {
	children := node.GetChildren()
	for _, child := range children {
		c.walkAST(child)
	}
}

// processHeading 处理标题
func (c *MarkdownCleaner) processHeading(h *ast.Heading) {
	c.result.WriteString("# ")
	c.processChildren(h)
	c.result.WriteString(": ")
}

// processParagraph 处理段落
func (c *MarkdownCleaner) processParagraph(p *ast.Paragraph) {
	c.processChildren(p)
}

// processList 处理列表
func (c *MarkdownCleaner) processList(l *ast.List) {
	if !c.preserveListStructure {
		c.processChildren(l)
		return
	}

	wasInList := c.inList
	c.inList = true
	c.listDepth++

	c.processChildren(l)

	c.listDepth--
	if c.listDepth == 0 {
		c.inList = wasInList
	}
}

// processListItem 处理列表项
func (c *MarkdownCleaner) processListItem(li *ast.ListItem) {
	if c.preserveListStructure {
		indent := strings.Repeat("  ", c.listDepth-1)
		c.result.WriteString(indent + "- ")
	}
	c.processChildren(li)
}

// processCodeBlock 处理代码块
func (c *MarkdownCleaner) processCodeBlock(cb *ast.CodeBlock) {
	if !c.preserveCodeContent {
		return
	}
	code := string(cb.Literal)
	if string(cb.Info) != "" {
		c.result.WriteString(string(cb.Info) + ":")
	}
	cleanCode := c.cleanCodeContent(code)
	c.result.WriteString(cleanCode)
}

// processTable 处理表格
func (c *MarkdownCleaner) processTable(t *ast.Table) {
	if !c.preserveTableContent {
		return
	}

	c.inTable = true
	c.processChildren(t)
	c.inTable = false
}

// processTableRow 处理表格行
func (c *MarkdownCleaner) processTableRow(tr *ast.TableRow) {
	if !c.preserveTableContent {
		return
	}

	var cellContents []string

	// 收集单元格内容
	children := tr.GetChildren()
	for _, cell := range children {
		if tableCell, ok := cell.(*ast.TableCell); ok {
			var cellBuilder strings.Builder
			tempResult := c.result
			c.result = cellBuilder
			c.processChildren(tableCell)
			c.result = tempResult

			content := strings.TrimSpace(cellBuilder.String())
			if content != "" {
				cellContents = append(cellContents, content)
			}
		}
	}

	// 输出行内容
	if len(cellContents) > 0 {
		c.result.WriteString(strings.Join(cellContents, " | "))
	}
}

// processTableCell 处理表格单元格
func (c *MarkdownCleaner) processTableCell(tc *ast.TableCell) {
	c.processChildren(tc)
}

// processText 处理纯文本
func (c *MarkdownCleaner) processText(t *ast.Text) {
	text := string(t.Literal)
	cleanText := c.cleanText(text)
	c.result.WriteString(cleanText)
}

// processLink 处理链接
func (c *MarkdownCleaner) processLink(l *ast.Link) {
	c.result.WriteString(" [")
	c.processChildren(l)
	c.result.WriteString("] ")
}

// processImage 处理图片
func (c *MarkdownCleaner) processImage(img *ast.Image) {
	c.processChildren(img)
}

// processInlineCode 处理行内代码
func (c *MarkdownCleaner) processInlineCode(code *ast.Code) {
	if c.preserveCodeContent {
		cleanCode := c.cleanText(string(code.Literal))
		c.result.WriteString(cleanCode)
	}
}

// processBlockQuote 处理引用块
func (c *MarkdownCleaner) processBlockQuote(bq *ast.BlockQuote) {
	c.processChildren(bq)
}

// cleanText 清洗文本内容
func (c *MarkdownCleaner) cleanText(text string) string {
	// 移除HTML标签
	htmlRe := regexp.MustCompile(`<[^>]*>`)
	text = htmlRe.ReplaceAllString(text, "")

	// 移除多余的空白字符
	spaceRe := regexp.MustCompile(`\s+`)
	text = spaceRe.ReplaceAllString(text, " ")

	// 移除控制字符
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, text)

	return strings.TrimSpace(text)
}

func (c *MarkdownCleaner) cleanCodeContent(code string) string {
	lines := strings.Split(code, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		cleanLines = append(cleanLines, line)
	}
	return strings.Join(cleanLines, " ")
}

// removeFrontmatter 删除 Markdown 文本开头的 frontmatter
func (c *MarkdownCleaner) removeFrontmatter(markdown string) string {
	// 检查是否以 --- 开头
	if !strings.HasPrefix(markdown, "---") {
		return markdown
	}

	lines := strings.Split(markdown, "\n")
	if len(lines) < 3 {
		return markdown
	}

	// 寻找第二个 ---
	endIndex := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIndex = i
			break
		}
	}

	// 如果找到结束标记，删除整个 frontmatter 块
	if endIndex > 0 {
		// 返回 frontmatter 之后的内容，跳过空行
		remainingLines := lines[endIndex+1:]
		for len(remainingLines) > 0 && strings.TrimSpace(remainingLines[0]) == "" {
			remainingLines = remainingLines[1:]
		}
		return strings.Join(remainingLines, "\n")
	}

	return markdown
}

// postProcess 后处理清洗后的文本
func (c *MarkdownCleaner) postProcess(text string) string {
	// 移除多余的空行
	multiNewlineRe := regexp.MustCompile(`\n{3,}`)
	text = multiNewlineRe.ReplaceAllString(text, "\n\n")

	// 移除首尾空白
	text = strings.TrimSpace(text)

	// 确保UTF-8编码有效
	if !utf8.ValidString(text) {
		text = strings.ToValidUTF8(text, "")
	}

	return text
}

// CleanMarkdownFile 从文件路径读取并清洗 Markdown 文件
func CleanMarkdownFile(mdPath string) (string, error) {
	// 检查文件是否存在
	if _, err := os.Stat(mdPath); os.IsNotExist(err) {
		return "", fmt.Errorf("文件不存在: %s", mdPath)
	}

	// 读取文件内容
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	// 检查文件是否为空
	if len(content) == 0 {
		return "", fmt.Errorf("文件为空: %s", mdPath)
	}

	// 创建清洗器并进行清洗
	cleaner := NewMarkdownCleaner()

	// 可以根据需要调整配置
	// cleaner.SetOptions(preserveCode, preserveList, preserveTable, extractFrontMatter, maxLineLength)

	// 清洗内容
	cleaned := cleaner.Clean(string(content))

	return cleaned, nil
}

// BatchCleanMarkdownFiles 批量处理多个 Markdown 文件
func BatchCleanMarkdownFiles(mdPaths []string) (map[string]string, error) {
	results := make(map[string]string)
	var errs []string

	for _, path := range mdPaths {
		cleaned, err := CleanMarkdownFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		results[path] = cleaned
	}

	if len(errs) > 0 {
		return results, fmt.Errorf("部分文件处理失败: %s", strings.Join(errs, "; "))
	}

	return results, nil
}

func MDSplitter() textsplitter.TextSplitter {
	return textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(1000),
		textsplitter.WithChunkOverlap(100),
		textsplitter.WithSeparators([]string{"\r\n\r\n", "\n\n", "\r\n", "\n", " ", ""}),
	)
}

// ProcessMarkdownDirectory 处理目录下的所有 Markdown 文件，清洗并分块
func ProcessMarkdownDirectory(dirPath string) ([]string, error) {
	// 获取目录下的所有 .md 文件
	mdFiles, err := getMDFiles(dirPath)
	if err != nil {
		return nil, fmt.Errorf("获取 MD 文件失败: %w", err)
	}

	if len(mdFiles) == 0 {
		return nil, fmt.Errorf("目录中未找到 .md 文件: %s", dirPath)
	}

	// 创建清洗器和分割器
	cleaner := NewMarkdownCleaner()
	splitter := MDSplitter()

	var allChunks []string

	// 处理每个 MD 文件
	for _, filePath := range mdFiles {
		chunks, err := processMarkdownFile(filePath, cleaner, splitter)
		if err != nil {
			// 记录错误但继续处理其他文件
			fmt.Printf("处理文件 %s 时出错: %v\n", filePath, err)
			continue
		}

		allChunks = append(allChunks, chunks...)
	}

	return allChunks, nil
}

// getMDFiles 递归获取目录下的所有 .md 文件
func getMDFiles(dirPath string) ([]string, error) {
	var mdFiles []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 检查是否为 .md 文件
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			mdFiles = append(mdFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return mdFiles, nil
}

// processMarkdownFile 处理单个 Markdown 文件，清洗并分块
func processMarkdownFile(filePath string, cleaner *MarkdownCleaner, splitter textsplitter.TextSplitter) ([]string, error) {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	// 检查文件是否为空
	if len(content) == 0 {
		return nil, fmt.Errorf("文件为空: %s", filePath)
	}

	// 清洗 Markdown 内容
	cleaned := cleaner.Clean(string(content))

	// 如果清洗后内容为空，跳过
	if strings.TrimSpace(cleaned) == "" {
		return nil, fmt.Errorf("清洗后内容为空: %s", filePath)
	}

	// 使用分割器进行分块
	chunks, err := splitter.SplitText(cleaned)
	if err != nil {
		return nil, fmt.Errorf("分块失败: %w", err)
	}
	return chunks, nil
}
