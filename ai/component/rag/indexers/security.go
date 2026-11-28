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

package indexers

import (
	"context"
	"fmt"

	"dubbo-admin-ai/security"

	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
)

var derivedDataPolicy = security.NewKnowledgeIndexPolicy()

// PreparedDocuments holds documents that passed PrepareDocuments. The field is
// unexported and PrepareDocuments is the only constructor, so a caller cannot
// build an unvalidated value and reach StorePrepared by accident.
type PreparedDocuments struct {
	docs []*schema.Document
}

// Documents returns a deep copy of the validated documents for callers that
// need the underlying []*schema.Document (for example a generic indexer that
// only implements Store). Copying the documents and their metadata prevents a
// caller from mutating prepared content after validation and then reaching
// StorePrepared with unvalidated data.
func (p PreparedDocuments) Documents() []*schema.Document {
	docs := make([]*schema.Document, 0, len(p.docs))
	for _, doc := range p.docs {
		if doc == nil {
			continue
		}
		clone := *doc
		clone.MetaData = cloneDocumentMetadata(doc.MetaData)
		docs = append(docs, &clone)
	}
	return docs
}

func cloneDocumentMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}
	clone := make(map[string]any, len(metadata))
	for key, value := range metadata {
		switch typed := value.(type) {
		case []string:
			clone[key] = append([]string(nil), typed...)
		case []any:
			clone[key] = append([]any(nil), typed...)
		default:
			clone[key] = value
		}
	}
	return clone
}

// PreparedIndexer is implemented by indexers that separate document validation
// from the backend write. RAG.Index calls StorePrepared directly after it has
// validated the documents at the public boundary, so each document is scanned
// exactly once. Store remains the safe entry point for direct callers.
type PreparedIndexer interface {
	StorePrepared(ctx context.Context, docs PreparedDocuments, opts ...indexer.Option) ([]string, error)
}

// PrepareDocuments validates canonical documents before they reach an index
// backend and wraps the validated copies so StorePrepared cannot be called
// with unchecked input.
func PrepareDocuments(docs []*schema.Document) (PreparedDocuments, error) {
	prepared := make([]*schema.Document, 0, len(docs))
	for index, doc := range docs {
		if doc == nil {
			return PreparedDocuments{}, fmt.Errorf("index document is nil")
		}
		idFindings, idErr := derivedDataPolicy.ValidateIdentifier(doc.ID)
		if idErr != nil {
			return PreparedDocuments{}, fmt.Errorf("index document at position %d rejected by rule %s: %w", index, ruleID(idFindings), idErr)
		}
		metadata, findings, err := derivedDataPolicy.Prepare(doc.Content, doc.MetaData)
		if err != nil {
			ruleID := "unknown"
			if len(findings) > 0 {
				ruleID = findings[0].RuleID
			}
			return PreparedDocuments{}, fmt.Errorf("index document at position %d rejected by rule %s: %w", index, ruleID, err)
		}
		prepared = append(prepared, &schema.Document{ID: doc.ID, Content: doc.Content, MetaData: metadata})
	}
	return PreparedDocuments{docs: prepared}, nil
}

func ruleID(findings []security.Finding) string {
	if len(findings) == 0 {
		return "unknown"
	}
	return findings[0].RuleID
}
