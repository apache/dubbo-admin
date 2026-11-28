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
	"errors"
	"fmt"
	"strings"
	"testing"

	"dubbo-admin-ai/component/rag/indexers"
	"dubbo-admin-ai/component/rag/rerankers"
	"dubbo-admin-ai/security"

	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

type recordingIndexer struct {
	called bool
}

func (r *recordingIndexer) Store(context.Context, []*schema.Document, ...indexer.Option) ([]string, error) {
	r.called = true
	return nil, errors.New("index backend must not be reached")
}

func TestIndexRejectsCookieAndSingleQuotedYAMLCredentialsBeforeBackend(t *testing.T) {
	const secret = "rag-boundary-secret"
	for _, content := range []string{
		"Cookie: theme=dark; JSESSIONID=" + secret,
		`'password': '` + secret + `'`,
	} {
		backend := &recordingIndexer{}
		engine, err := NewRAG(&Components{Indexer: backend})
		if err != nil {
			t.Fatal(err)
		}
		_, err = engine.Index(context.Background(), "", []*schema.Document{{ID: "doc", Content: content}})
		if !errors.Is(err, security.ErrSensitiveDerivedData) {
			t.Fatalf("Index() error = %v for %q, want ErrSensitiveDerivedData", err, content)
		}
		if backend.called {
			t.Fatalf("index backend was reached for %q", content)
		}
	}
}

// preparedRecordingIndexer implements both entry points. RAG.Index must use
// StorePrepared for concrete indexers so documents are validated only once.
type preparedRecordingIndexer struct {
	storeCalls         int
	storePreparedCalls int
}

func (r *preparedRecordingIndexer) Store(context.Context, []*schema.Document, ...indexer.Option) ([]string, error) {
	r.storeCalls++
	return nil, errors.New("Store must not be used for a PreparedIndexer")
}

func (r *preparedRecordingIndexer) StorePrepared(context.Context, indexers.PreparedDocuments, ...indexer.Option) ([]string, error) {
	r.storePreparedCalls++
	return []string{"stored"}, nil
}

func TestIndexUsesPreparedIndexerWithoutRescanning(t *testing.T) {
	backend := &preparedRecordingIndexer{}
	engine, err := NewRAG(&Components{Indexer: backend})
	if err != nil {
		t.Fatal(err)
	}

	ids, err := engine.Index(context.Background(), "", []*schema.Document{{ID: "doc", Content: "safe document"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || backend.storePreparedCalls != 1 || backend.storeCalls != 0 {
		t.Fatalf("Index() used the wrong entry point: ids=%v store=%d prepared=%d", ids, backend.storeCalls, backend.storePreparedCalls)
	}

	backend.storePreparedCalls = 0
	if _, err := engine.Index(context.Background(), "", []*schema.Document{{ID: "doc", Content: "password=rag-secret"}}); !errors.Is(err, security.ErrSensitiveDerivedData) {
		t.Fatalf("Index() error = %v, want ErrSensitiveDerivedData", err)
	}
	if backend.storePreparedCalls != 0 {
		t.Fatal("sensitive document reached StorePrepared")
	}
}

type recordingRetriever struct {
	query string
	docs  []*schema.Document
}

func (r *recordingRetriever) Retrieve(_ context.Context, query string, _ ...retriever.Option) ([]*schema.Document, error) {
	r.query = query
	return r.docs, nil
}

type recordingReranker struct {
	query string
	docs  []*schema.Document
}

func (r *recordingReranker) Rerank(_ context.Context, query string, docs any, _ ...rerankers.Option) ([]*rerankers.Result, error) {
	r.query = query
	typed, _ := docs.([]*schema.Document)
	r.docs = typed
	results := make([]*rerankers.Result, len(typed))
	for i, doc := range typed {
		results[i] = &rerankers.Result{Content: doc.Content, RelevanceScore: 1, Index: i}
	}
	return results, nil
}

func TestProviderCallsReceiveSanitizedEphemeralInputs(t *testing.T) {
	const secret = "provider-boundary-secret"
	source := []*schema.Document{{
		ID:       "doc",
		Content:  "password=" + secret,
		MetaData: map[string]any{"api_key": secret, "source": "docs/guide.md"},
	}}
	retrieverRec := &recordingRetriever{docs: source}
	rerankerRec := &recordingReranker{}
	engine, err := NewRAG(&Components{Retriever: retrieverRec, Reranker: rerankerRec})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := engine.RetrieveV2(context.Background(), &RetrieveRequest{Query: "password=" + secret, TopK: 1}); err != nil {
		t.Fatal(err)
	}
	assertProviderInputsSanitized(t, secret, retrieverRec.query, rerankerRec.query, rerankerRec.docs)
	if source[0].Content != "password="+secret || source[0].MetaData["api_key"] != secret {
		t.Fatalf("canonical retriever document was mutated: %#v", source[0])
	}
}

func TestLegacyRetrieveSanitizesProviderInputs(t *testing.T) {
	const secret = "legacy-provider-secret"
	source := []*schema.Document{{
		ID:       "doc",
		Content:  "password=" + secret,
		MetaData: map[string]any{"api_key": secret},
	}}
	retrieverRec := &recordingRetriever{docs: source}
	rerankerRec := &recordingReranker{}
	engine, err := NewRAG(&Components{Retriever: retrieverRec, Reranker: rerankerRec})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := engine.Retrieve(context.Background(), "", []string{"password=" + secret}); err != nil {
		t.Fatal(err)
	}
	assertProviderInputsSanitized(t, secret, retrieverRec.query, rerankerRec.query, rerankerRec.docs)
	if source[0].Content != "password="+secret || source[0].MetaData["api_key"] != secret {
		t.Fatalf("canonical retriever document was mutated: %#v", source[0])
	}
}

func assertProviderInputsSanitized(t *testing.T, secret, retrieverQuery, rerankerQuery string, rerankerDocs []*schema.Document) {
	t.Helper()
	if strings.Contains(retrieverQuery, secret) {
		t.Fatalf("retriever received raw query: %q", retrieverQuery)
	}
	if strings.Contains(rerankerQuery, secret) {
		t.Fatalf("reranker received raw query: %q", rerankerQuery)
	}
	if len(rerankerDocs) != 1 {
		t.Fatalf("reranker input documents = %d, want 1", len(rerankerDocs))
	}
	if strings.Contains(rerankerDocs[0].Content, secret) {
		t.Fatalf("reranker received raw content: %q", rerankerDocs[0].Content)
	}
	if strings.Contains(fmt.Sprint(rerankerDocs[0].MetaData), secret) {
		t.Fatalf("reranker received raw metadata: %#v", rerankerDocs[0].MetaData)
	}
}
