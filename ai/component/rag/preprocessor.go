/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
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

package rag

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gomarkdown/markdown/ast"
	mdparser "github.com/gomarkdown/markdown/parser"
)

// PreprocessorFunc defines a function that processes text.
type PreprocessorFunc func(string) string

// Preprocessor manages a chain of preprocessing functions.
type Preprocessor struct {
	funcs []PreprocessorFunc
}

// newPreprocessor creates a new Preprocessor with the given functions.
func newPreprocessor(funcs ...PreprocessorFunc) *Preprocessor {
	return &Preprocessor{
		funcs: funcs,
	}
}

// Process applies all preprocessing functions in order.
func (p *Preprocessor) Process(text string) string {
	for _, f := range p.funcs {
		text = f(text)
	}
	return text
}

// Common Preprocessors

// PDFTextCleaner cleans text extracted from PDFs.
func PDFTextCleaner(text string) string {
	// 1. Remove control characters and non-printable characters (keep newlines, tabs, and spaces)
	cleaned := ""
	for _, r := range text {
		if r == '\n' || r == '\t' || r == ' ' || (r >= 32 && r < 127) || r > 127 {
			cleaned += string(r)
		}
	}

	// 2. Remove extra whitespace and newlines
	cleaned = strings.ReplaceAll(cleaned, "\n \n", "\n")
	cleaned = strings.ReplaceAll(cleaned, " \n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\n ", "\n")

	// 3. Merge multiple newlines
	multipleNewlines := regexp.MustCompile(`\n{3,}`)
	cleaned = multipleNewlines.ReplaceAllString(cleaned, "\n\n")

	// 4. Remove single character lines (likely PDF parsing errors)
	lines := strings.Split(cleaned, "\n")
	var cleanedLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) <= 1 {
			continue
		}
		if regexp.MustCompile(`^[^\w\s]+$`).MatchString(line) {
			continue
		}
		cleanedLines = append(cleanedLines, line)
	}

	// 5. Reassemble text
	result := strings.Join(cleanedLines, "\n")

	// 6. Clean common PDF parsing artifacts
	result = regexp.MustCompile(`(?m)^\d+$`).ReplaceAllString(result, "")
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
	result = strings.ReplaceAll(result, " \n", "\n")
	result = strings.ReplaceAll(result, "\n ", "\n")

	// 7. Final trim
	return strings.TrimSpace(result)
}

// MarkdownCleaner cleans Markdown content for RAG.
type MarkdownCleaner struct {
	preserveCodeContent   bool
	preserveListStructure bool
	preserveTableContent  bool
	maxLineLength         int
	result                strings.Builder
	inList                bool
	listDepth             int
	inTable               bool
}

// newMarkdownCleaner creates a new MarkdownCleaner.
func newMarkdownCleaner() *MarkdownCleaner {
	return &MarkdownCleaner{
		preserveCodeContent:   true,
		preserveListStructure: true,
		preserveTableContent:  true,
		maxLineLength:         500,
	}
}

// Clean cleans the markdown text.
func (c *MarkdownCleaner) Clean(markdown string) string {
	c.result.Reset()
	c.inList = false
	c.listDepth = 0
	c.inTable = false

	markdown = c.removeFrontmatter(markdown)
	hugoRe := regexp.MustCompile(`{{<[^>]+>}}|{{%[^%]+%}}`)
	markdown = hugoRe.ReplaceAllString(markdown, "")

	extensions := mdparser.CommonExtensions | mdparser.Mmark | mdparser.Footnotes
	p := mdparser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(markdown))

	c.walkAST(doc)

	return c.postProcess(c.result.String())
}

// Helper methods for MarkdownCleaner (private)

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

func (c *MarkdownCleaner) processChildren(node ast.Node) {
	children := node.GetChildren()
	for _, child := range children {
		c.walkAST(child)
	}
}

func (c *MarkdownCleaner) processHeading(h *ast.Heading) {
	c.result.WriteString(strings.Repeat("#", h.Level) + " ")
	c.processChildren(h)
	c.result.WriteString("\n")
}

func (c *MarkdownCleaner) processParagraph(p *ast.Paragraph) {
	c.processChildren(p)
}

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

func (c *MarkdownCleaner) processListItem(li *ast.ListItem) {
	if c.preserveListStructure {
		indent := strings.Repeat("  ", c.listDepth-1)
		c.result.WriteString(indent + "- ")
	}
	c.processChildren(li)
}

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

func (c *MarkdownCleaner) processTable(t *ast.Table) {
	if !c.preserveTableContent {
		return
	}
	c.inTable = true
	c.processChildren(t)
	c.inTable = false
}

func (c *MarkdownCleaner) processTableRow(tr *ast.TableRow) {
	if !c.preserveTableContent {
		return
	}
	var cellContents []string
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
	if len(cellContents) > 0 {
		c.result.WriteString(strings.Join(cellContents, " | "))
	}
}

func (c *MarkdownCleaner) processTableCell(tc *ast.TableCell) {
	c.processChildren(tc)
}

func (c *MarkdownCleaner) processText(t *ast.Text) {
	text := string(t.Literal)
	cleanText := c.cleanText(text)
	c.result.WriteString(cleanText)
}

func (c *MarkdownCleaner) processLink(l *ast.Link) {
	c.result.WriteString(" [")
	c.processChildren(l)
	c.result.WriteString("] ")
}

func (c *MarkdownCleaner) processImage(img *ast.Image) {
	c.processChildren(img)
}

func (c *MarkdownCleaner) processInlineCode(code *ast.Code) {
	if c.preserveCodeContent {
		cleanCode := c.cleanText(string(code.Literal))
		c.result.WriteString(cleanCode)
	}
}

func (c *MarkdownCleaner) processBlockQuote(bq *ast.BlockQuote) {
	c.processChildren(bq)
}

func (c *MarkdownCleaner) cleanText(text string) string {
	htmlRe := regexp.MustCompile(`<[^>]*>`)
	text = htmlRe.ReplaceAllString(text, "")
	spaceRe := regexp.MustCompile(`\s+`)
	text = spaceRe.ReplaceAllString(text, " ")
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

func (c *MarkdownCleaner) removeFrontmatter(markdown string) string {
	if !strings.HasPrefix(markdown, "---") {
		return markdown
	}
	lines := strings.Split(markdown, "\n")
	if len(lines) < 3 {
		return markdown
	}
	endIndex := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIndex = i
			break
		}
	}
	if endIndex > 0 {
		remainingLines := lines[endIndex+1:]
		for len(remainingLines) > 0 && strings.TrimSpace(remainingLines[0]) == "" {
			remainingLines = remainingLines[1:]
		}
		return strings.Join(remainingLines, "\n")
	}
	return markdown
}

func (c *MarkdownCleaner) postProcess(text string) string {
	multiNewlineRe := regexp.MustCompile(`\n{3,}`)
	text = multiNewlineRe.ReplaceAllString(text, "\n\n")
	text = strings.TrimSpace(text)
	if !utf8.ValidString(text) {
		text = strings.ToValidUTF8(text, "")
	}
	return text
}
