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

package loaders

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino-ext/components/document/parser/pdf"
	"github.com/cloudwego/eino/components/document/parser"
	"github.com/cloudwego/eino/schema"
)

// --- Options ---

type ParserConfig struct {
	Preprocessors []PreprocessorFunc
}

type ParserOption func(*ParserConfig)

func WithPreprocessors(funcs ...PreprocessorFunc) ParserOption {
	return func(c *ParserConfig) {
		c.Preprocessors = append(c.Preprocessors, funcs...)
	}
}

// --- Parsers ---

// MarkdownParser uses MarkdownCleaner and optional preprocessors
type MarkdownParser struct {
	preprocessor *Preprocessor
}

// NewMarkdownParser creates a new MarkdownParser.
func NewMarkdownParser(opts ...ParserOption) *MarkdownParser {
	config := &ParserConfig{
		Preprocessors: []PreprocessorFunc{NewMarkdownCleaner().Clean},
	}
	for _, opt := range opts {
		opt(config)
	}

	return &MarkdownParser{
		preprocessor: NewPreprocessor(config.Preprocessors...),
	}
}

func (p *MarkdownParser) Parse(ctx context.Context, reader io.Reader, opts ...parser.Option) ([]*schema.Document, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	cleaned := p.preprocessor.Process(string(content))
	if strings.TrimSpace(cleaned) == "" {
		return nil, fmt.Errorf("empty content after cleaning")
	}

	// Apply common options (like extra meta)
	commonOpts := parser.GetCommonOptions(nil, opts...)

	return []*schema.Document{
		{
			Content:  cleaned,
			MetaData: commonOpts.ExtraMeta,
		},
	}, nil
}

// PDFParserWrapper wraps eino-ext PDF parser and adds preprocessing
type PDFParserWrapper struct {
	internalParser *pdf.PDFParser
	preprocessor   *Preprocessor
}

// NewPDFParserWrapper creates a new PDFParserWrapper.
func NewPDFParserWrapper(ctx context.Context, opts ...ParserOption) (*PDFParserWrapper, error) {
	p, err := pdf.NewPDFParser(ctx, nil)
	if err != nil {
		return nil, err
	}

	config := &ParserConfig{
		Preprocessors: []PreprocessorFunc{PDFTextCleaner},
	}
	for _, opt := range opts {
		opt(config)
	}

	return &PDFParserWrapper{
		internalParser: p,
		preprocessor:   NewPreprocessor(config.Preprocessors...),
	}, nil
}

func (p *PDFParserWrapper) Parse(ctx context.Context, reader io.Reader, opts ...parser.Option) ([]*schema.Document, error) {
	docs, err := p.internalParser.Parse(ctx, reader, opts...)
	if err != nil {
		return nil, err
	}

	// Apply preprocessing
	for _, doc := range docs {
		doc.Content = p.preprocessor.Process(doc.Content)
	}

	return docs, nil
}
