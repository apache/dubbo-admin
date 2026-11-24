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
	"testing"

	"dubbo-admin-ai/security"

	"github.com/cloudwego/eino/components/indexer"
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
