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
