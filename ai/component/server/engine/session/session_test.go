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

package session

import (
	"context"
	"errors"
	"testing"

	memorystore "dubbo-admin-ai/store/memory"
	"dubbo-admin-ai/testutils"

	"github.com/firebase/genkit/go/ai"
)

func TestManagerUsesSharedStore(t *testing.T) {
	ctx := context.Background()
	access := testutils.TestAccess()
	backingStore := memorystore.NewMemoryStore()
	manager := NewManager(backingStore, backingStore)
	defer manager.Close()

	created, err := manager.CreateSession(ctx, access)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if err := backingStore.AddHistory(ctx, access, created.ID, ai.NewUserMessage(ai.NewTextPart("hello"))); err != nil {
		t.Fatalf("AddHistory() error = %v", err)
	}

	if err := manager.TouchSession(ctx, access, created.ID); err != nil {
		t.Fatalf("TouchSession() error = %v", err)
	}
	if _, err := manager.GetSession(ctx, access, created.ID); err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}

	if err := manager.DeleteSession(ctx, access, created.ID); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}
	if _, err := backingStore.Get(ctx, access, created.ID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("backing Store Get() error = %v, want ErrSessionNotFound", err)
	}
	if _, err := backingStore.AllMemory(ctx, access, created.ID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("backing Store AllMemory() error = %v, want ErrSessionNotFound", err)
	}
}
