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
	"embed"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// seedFS embeds the bundled knowledge documents so the vector store always has
// baseline content to retrieve, even on a fresh deployment with no indexing run.
//
//go:embed seeds/*.md
var seedFS embed.FS

// SeedDocuments returns the bundled seed knowledge documents (Dubbo / Dubbo Admin
// fundamentals) as eino documents ready to be split and indexed.
func SeedDocuments() ([]*schema.Document, error) {
	entries, err := seedFS.ReadDir("seeds")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded seeds: %w", err)
	}

	docs := make([]*schema.Document, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		content, err := seedFS.ReadFile("seeds/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded seed %s: %w", entry.Name(), err)
		}
		text := strings.TrimSpace(string(content))
		if text == "" {
			continue
		}
		docs = append(docs, &schema.Document{
			ID:      "seed:" + entry.Name(),
			Content: text,
			MetaData: map[string]any{
				"source": "seed/" + entry.Name(),
				"title":  seedTitle(text, entry.Name()),
			},
		})
	}
	return docs, nil
}

// seedTitle extracts the first markdown H1 heading as the title, falling back to
// the file name when no heading is present.
func seedTitle(text, fallback string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return strings.TrimSuffix(fallback, ".md")
}

// Seed splits and indexes the bundled seed documents into the default index.
//
// It is best-effort and idempotent: localvec deduplicates by content hash, so
// re-running does not create duplicates. Callers (e.g. startup) may log the
// returned error without treating it as fatal, since indexing requires the
// embedder (and its API key) to be reachable.
func (r *RAG) Seed(ctx context.Context) (int, error) {
	if r == nil {
		return 0, fmt.Errorf("rag is nil")
	}
	if r.Indexer == nil {
		return 0, fmt.Errorf("indexer is nil")
	}

	docs, err := SeedDocuments()
	if err != nil {
		return 0, err
	}
	if len(docs) == 0 {
		return 0, nil
	}

	chunks, err := r.Split(ctx, docs)
	if err != nil {
		return 0, fmt.Errorf("failed to split seed documents: %w", err)
	}

	// The embedding backend (dashscope text-embedding-v4) caps each request at 10
	// inputs, so index in batches of at most seedEmbedBatch. Empty namespace ->
	// indexer uses its configured default target, the same target the runtime
	// retriever reads from.
	total := 0
	for start := 0; start < len(chunks); start += seedEmbedBatch {
		end := start + seedEmbedBatch
		if end > len(chunks) {
			end = len(chunks)
		}
		ids, err := r.Index(ctx, "", chunks[start:end])
		if err != nil {
			return total, fmt.Errorf("failed to index seed documents batch %d-%d: %w", start, end, err)
		}
		total += len(ids)
	}
	return total, nil
}

// seedEmbedBatch is the maximum number of chunks embedded per indexing request,
// bounded by the embedding backend's per-request input limit.
const seedEmbedBatch = 10
