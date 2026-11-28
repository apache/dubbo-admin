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
	"errors"
	"strings"
	"testing"

	"dubbo-admin-ai/security"

	"github.com/cloudwego/eino/schema"
)

func TestConcreteIndexerRejectsSensitiveDocumentBeforeBackendAccess(t *testing.T) {
	indexer := NewLocalIndexer(nil, "missing", "default", 1)
	_, err := indexer.Store(context.Background(), []*schema.Document{{
		ID: "secret", Content: "Authorization: Bearer index-secret-value",
	}})
	if !errors.Is(err, security.ErrSensitiveDerivedData) {
		t.Fatalf("Store() error = %v, want ErrSensitiveDerivedData", err)
	}
}

func TestConcreteIndexerRejectsCookieListsAndSingleQuotedYAMLCredentials(t *testing.T) {
	for _, content := range []string{
		"Cookie: theme=dark; JSESSIONID=index-cookie-secret",
		`'password': 'index-yaml-secret'`,
	} {
		indexer := NewLocalIndexer(nil, "missing", "default", 1)
		_, err := indexer.Store(context.Background(), []*schema.Document{{
			ID: "doc", Content: content,
		}})
		if !errors.Is(err, security.ErrSensitiveDerivedData) {
			t.Fatalf("Store() error = %v for %q, want ErrSensitiveDerivedData", err, content)
		}
	}
}

func TestConcreteIndexerRejectsSensitiveDocumentIDWithoutEchoingIt(t *testing.T) {
	secretID := "token=do-not-echo"
	indexer := NewLocalIndexer(nil, "missing", "default", 1)
	_, err := indexer.Store(context.Background(), []*schema.Document{{
		ID: secretID, Content: "safe document",
	}})
	if !errors.Is(err, security.ErrSensitiveDerivedData) {
		t.Fatalf("Store() error = %v, want ErrSensitiveDerivedData", err)
	}
	if strings.Contains(err.Error(), secretID) {
		t.Fatalf("Store() echoed sensitive document ID: %v", err)
	}
}

func TestPreparedDocumentsCannotBeMutatedAfterValidation(t *testing.T) {
	original := []*schema.Document{{
		ID:      "doc",
		Content: "safe document",
		MetaData: map[string]any{
			"header_path": []string{"Guide"},
			"source":      "docs/guide.md",
		},
	}}
	prepared, err := PrepareDocuments(original)
	if err != nil {
		t.Fatal(err)
	}

	// A caller that receives the exported documents must not be able to
	// rewrite validated content and then reach StorePrepared.
	exported := prepared.Documents()
	exported[0].Content = "password=mutated-after-validation"
	exported[0].MetaData["header_path"].([]string)[0] = "password=mutated-after-validation"

	// The same applies to mutating the original input after validation.
	original[0].Content = "password=mutated-after-validation"
	original[0].MetaData["header_path"].([]string)[0] = "password=mutated-after-validation"

	again := prepared.Documents()
	if again[0].Content != "safe document" || again[0].MetaData["header_path"].([]string)[0] != "Guide" {
		t.Fatalf("prepared documents were mutated after validation: %#v", again[0])
	}
	if prepared.docs[0].Content != "safe document" || prepared.docs[0].MetaData["header_path"].([]string)[0] != "Guide" {
		t.Fatalf("internal prepared documents were mutated: %#v", prepared.docs[0])
	}
}
