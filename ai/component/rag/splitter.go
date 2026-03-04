package rag

import (
	"context"
	"dubbo-admin-ai/runtime"
	"fmt"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
	"github.com/cloudwego/eino/components/document"
)

// splitterComponent Splitter 组件包装器
type splitterComponent struct {
	splitterType       string
	recursiveChunkSize int
	recursiveOverlap   int
	markdownHeaders    map[string]string
	markdownTrim       bool
	splitter           document.Transformer
}

func NewSplitterComponent(
	splitterType string,
	recursiveChunkSize int,
	recursiveOverlap int,
	markdownHeaders map[string]string,
	markdownTrim bool,
) (runtime.Component, error) {
	if splitterType == "" {
		splitterType = "recursive"
	}
	return &splitterComponent{
		splitterType:       splitterType,
		recursiveChunkSize: recursiveChunkSize,
		recursiveOverlap:   recursiveOverlap,
		markdownHeaders:    markdownHeaders,
		markdownTrim:       markdownTrim,
	}, nil
}

func (c *splitterComponent) Name() string { return "splitter" }

func (c *splitterComponent) Validate() error { return nil }

func (c *splitterComponent) Init(rt *runtime.Runtime) error {
	splitter, err := newSplitterByType(
		context.Background(),
		c.splitterType,
		c.recursiveChunkSize,
		c.recursiveOverlap,
		c.markdownHeaders,
		c.markdownTrim,
	)
	if err != nil {
		return fmt.Errorf("failed to create splitter: %w", err)
	}
	c.splitter = splitter

	rt.GetLogger().Info("Splitter component initialized",
		"type", c.splitterType,
		"chunk_size", c.recursiveChunkSize,
		"overlap_size", c.recursiveOverlap,
	)
	return nil
}

func (c *splitterComponent) Start() error { return nil }

func (c *splitterComponent) Stop() error { return nil }

func (c *splitterComponent) get() document.Transformer {
	return c.splitter
}

func defaultMarkdownHeaders() map[string]string {
	return map[string]string{"#": "h1", "##": "h2", "###": "h3", "####": "h4"}
}

func newMarkdownHeaderSplitter(ctx context.Context, headers map[string]string, trim bool) (document.Transformer, error) {
	effectiveHeaders := headers
	if len(effectiveHeaders) == 0 {
		effectiveHeaders = defaultMarkdownHeaders()
	}
	return markdown.NewHeaderSplitter(ctx, &markdown.HeaderConfig{Headers: effectiveHeaders, TrimHeaders: trim})
}

func newRecursiveSplitter(ctx context.Context, chunkSize int, overlap int) (document.Transformer, error) {
	if chunkSize <= 0 {
		chunkSize = DefaultSplitterSpec().ChunkSize
	}
	if overlap < 0 {
		overlap = DefaultSplitterSpec().OverlapSize
	}
	return recursive.NewSplitter(ctx, &recursive.Config{ChunkSize: chunkSize, OverlapSize: overlap})
}

func newSplitterByType(
	ctx context.Context,
	splitterType string,
	recursiveChunkSize int,
	recursiveOverlap int,
	markdownHeaders map[string]string,
	markdownTrim bool,
) (document.Transformer, error) {
	switch splitterType {
	case "markdown_header":
		return newMarkdownHeaderSplitter(ctx, markdownHeaders, markdownTrim)
	case "", "recursive":
		return newRecursiveSplitter(ctx, recursiveChunkSize, recursiveOverlap)
	default:
		return nil, fmt.Errorf("unsupported splitter type: %s", splitterType)
	}
}
