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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cloudwego/eino-ext/components/document/loader/file"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/document/parser"
	"github.com/cloudwego/eino/schema"
)

type ExtType = string

const (
	DotPDF ExtType = ".pdf"
	DotMD  ExtType = ".md"
	DotTxt ExtType = ".txt"
)

// newLocalFileLoader creates a FileLoader with an ExtParser that supports PDF and Markdown.
func newLocalFileLoader(ctx context.Context) (*file.FileLoader, error) {
	// 1. Create Parsers
	pdfParser, err := newPDFParserWrapper(ctx)
	if err != nil {
		return nil, err
	}
	mdParser := newMarkdownParser()
	plainParser := parser.TextParser{}

	// 2. Create ExtParser
	extParser, err := parser.NewExtParser(ctx, &parser.ExtParserConfig{
		Parsers: map[string]parser.Parser{
			DotPDF: pdfParser,
			DotMD:  mdParser,
			DotTxt: plainParser,
		},
		FallbackParser: plainParser, // Fallback to text parser for other files
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ext parser: %w", err)
	}

	// 3. Create FileLoader
	loader, err := file.NewFileLoader(ctx, &file.FileLoaderConfig{
		Parser: extParser,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create file loader: %w", err)
	}

	return loader, nil
}

// LoadDirectory loads all supported files from a directory recursively.
func LoadDirectory(ctx context.Context, loader document.Loader, dirPath string, opts ...LoaderOption) ([]*schema.Document, error) {
	lo := defaultLoaderOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&lo)
		}
	}

	// Normalize extensions
	var targetExts []string
	if len(lo.TargetExtensions) > 0 {
		for _, e := range lo.TargetExtensions {
			e = strings.ToLower(e)
			if !strings.HasPrefix(e, ".") {
				e = "." + e
			}
			targetExts = append(targetExts, e)
		}
	}

	var allDocs []*schema.Document

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check extension if provided
		if len(targetExts) > 0 {
			ext := strings.ToLower(filepath.Ext(path))
			if !slices.Contains(targetExts, ext) {
				return nil
			}
		}

		// Load file
		docs, err := loader.Load(ctx, document.Source{URI: path})
		if err != nil {
			return fmt.Errorf("failed to load file %s: %w", path, err)
		}

		allDocs = append(allDocs, docs...)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return allDocs, nil
}

// ---- Loader options ----

type LoaderOptions struct {
	TargetExtensions []string
}

type LoaderOption func(*LoaderOptions)

func defaultLoaderOptions() LoaderOptions {
	return LoaderOptions{TargetExtensions: []string{".md", ".pdf", ".txt"}}
}

// WithLoaderTargetExtensions sets the file extensions to include when loading directories.
// Extensions are normalized to lowercase and ensured to start with '.'.
func WithLoaderTargetExtensions(exts ...string) LoaderOption {
	return func(o *LoaderOptions) {
		if len(exts) == 0 {
			o.TargetExtensions = nil
			return
		}
		norm := make([]string, 0, len(exts))
		for _, e := range exts {
			e = strings.ToLower(strings.TrimSpace(e))
			if e == "" {
				continue
			}
			if !strings.HasPrefix(e, ".") {
				e = "." + e
			}
			norm = append(norm, e)
		}
		o.TargetExtensions = norm
	}
}
