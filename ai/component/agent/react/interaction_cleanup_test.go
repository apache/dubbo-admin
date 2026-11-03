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

package react

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"dubbo-admin-ai/component/agent"
	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/schema"
	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"
	memorystore "dubbo-admin-ai/store/memory"
	"dubbo-admin-ai/testutils"
)

func TestStoppingAgentReleasesPreparedInteraction(t *testing.T) {
	store := memorystore.NewMemoryStore()
	access := testutils.TestAccess()
	now := time.Now()
	session := &conversationstore.Session{ID: "stopping-agent-session", TenantID: access.Subject.TenantID, OwnerID: access.Subject.UserID, Version: 1, CreatedAt: now, UpdatedAt: now, Status: conversationstore.SessionStatusActive}
	if err := store.Create(context.Background(), access, session); err != nil {
		t.Fatal(err)
	}
	prepared, err := store.PrepareInteraction(context.Background(), access, session.ID, "request", "hash", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	parent := security.WithAccessContext(context.Background(), access)
	parent = conversationstore.WithPreparedInteraction(parent, prepared)
	ra := &ReActAgent{messageStore: store, stopping: true}
	channels := ra.Interact(parent, &schema.UserInput{Content: "hello"}, session.ID)
	if err := <-channels.ErrorChan; err != ErrAgentStopping {
		t.Fatalf("Interact() error = %v, want ErrAgentStopping", err)
	}
	if _, err := store.PrepareInteraction(context.Background(), access, session.ID, "request", "hash", time.Minute); err != nil {
		t.Fatalf("PrepareInteraction() after rejection error = %v", err)
	}
}

func TestRepeatedConcurrentInteractionsReleaseActiveEntries(t *testing.T) {
	ra := &ReActAgent{}
	const cycles, workers = 16, 32
	for cycle := range cycles {
		var ready, finished sync.WaitGroup
		ready.Add(workers)
		finished.Add(workers)
		release := make(chan struct{})
		contexts := make([]context.Context, workers)
		for i := range workers {
			go func() {
				defer finished.Done()
				id := fmt.Sprintf("cycle-%d-interaction-%d", cycle, i)
				ctx, ok := ra.beginInteraction(context.Background(), id)
				contexts[i] = ctx
				ready.Done()
				<-release
				if ok {
					ra.finishInteraction(id)
				}
			}()
		}
		ready.Wait()
		ra.lifecycleMu.Lock()
		active := len(ra.active)
		ra.lifecycleMu.Unlock()
		close(release)
		finished.Wait()
		ra.activeWG.Wait()
		if active != workers {
			t.Fatalf("cycle %d active entries = %d, want %d", cycle, active, workers)
		}
		assertInteractionsReleased(t, ra, contexts)
	}
}

func TestNilInputInteractionsReleaseActiveEntries(t *testing.T) {
	manager := hooks.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
	const cycles, workers = 4, 32
	started := make(chan context.Context, workers)
	if err := manager.Register(hooks.Registration{
		Events: []hooks.Event{hooks.EventInteractionStart},
		Hook: func(ctx context.Context, _ hooks.State) context.Context {
			started <- ctx
			return ctx
		},
	}); err != nil {
		t.Fatal(err)
	}
	// Nil input exits before memory, prompts, or model setup is required.
	ra := &ReActAgent{hookManager: manager}
	for cycle := range cycles {
		channels := make([]*agent.Channels, workers)
		contexts := make([]context.Context, workers)
		for i := range workers {
			channels[i] = ra.Interact(context.Background(), nil, fmt.Sprintf("cycle-%d-session-%d", cycle, i))
			contexts[i] = <-started
		}
		for i, channel := range channels {
			<-channel.Done()
			err := <-channel.ErrorChan
			if err == nil || err.Error() != "nil input" {
				t.Fatalf("cycle %d interaction %d error = %v, want nil input", cycle, i, err)
			}
		}
		// Channels close before the deferred finishInteraction; wait for cleanup too.
		ra.activeWG.Wait()
		assertInteractionsReleased(t, ra, contexts)
	}
}

func assertInteractionsReleased(t *testing.T, ra *ReActAgent, contexts []context.Context) {
	t.Helper()
	ra.lifecycleMu.Lock()
	active := len(ra.active)
	ra.lifecycleMu.Unlock()
	if active != 0 {
		t.Fatalf("completed interactions retained %d active entries", active)
	}
	for i, ctx := range contexts {
		if ctx == nil || ctx.Err() != context.Canceled {
			t.Fatalf("interaction %d context was not canceled after completion", i)
		}
	}
}
