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
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/schema"
	conversationstore "dubbo-admin-ai/store"
	memorystore "dubbo-admin-ai/store/memory"
	"github.com/firebase/genkit/go/ai"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type observedTurnStore struct {
	conversationstore.MessageStore
	turnID uint64
}

func (s *observedTurnStore) BeginTurn(ctx context.Context, sessionID string) (uint64, error) {
	id, err := s.MessageStore.BeginTurn(ctx, sessionID)
	s.turnID = id
	return id, err
}

func TestStoreHooksFinalizeOrAbortBeforeInteractionEnd(t *testing.T) {
	for _, tc := range []struct {
		name     string
		prompt   ai.Prompt
		messages int
		special  hooks.Event
	}{
		{"success", &stubPrompt{resp: textResp("answer")}, 2, ""},
		{"model error", &stubPrompt{err: errors.New("model failed")}, 0, hooks.EventInteractionError},
		{"cancel", &stubPrompt{err: context.Canceled}, 0, hooks.EventInteractionCancel},
		{"panic", &panicPrompt{}, 0, hooks.EventInteractionError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			store := memorystore.NewMemoryStore(2)
			now := time.Now()
			const sessionID = "hooks-store"
			if err := store.Create(ctx, &conversationstore.Session{ID: sessionID, CreatedAt: now, UpdatedAt: now, Status: "active"}); err != nil {
				t.Fatal(err)
			}
			observed := &observedTurnStore{MessageStore: store}
			recorder := tracetest.NewSpanRecorder()
			provider := trace.NewTracerProvider(trace.WithSpanProcessor(recorder))
			t.Cleanup(func() { _ = provider.Shutdown(ctx) })
			manager := hooks.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
			if err := manager.Register(hooks.NewTracingRegistration(provider.Tracer("store-hooks"), hooks.CaptureNone)); err != nil {
				t.Fatal(err)
			}
			var events []hooks.Event
			var endTurnErr error
			if err := manager.Register(hooks.Registration{Events: hooks.AllEvents(), Tools: []string{hooks.AllTools}, Hook: func(ctx context.Context, state hooks.State) context.Context {
				events = append(events, state.Event)
				if state.Event == hooks.EventInteractionEnd {
					_, endTurnErr = store.IsTurnEmpty(context.Background(), sessionID, observed.turnID)
				}
				return ctx
			}}); err != nil {
				t.Fatal(err)
			}
			ra := &ReActAgent{messageStore: observed, hookManager: manager, model: "test/model", actPrompt: tc.prompt, answerPrompt: tc.prompt, maxIterations: 1, bufferSize: 8}
			channels := ra.Interact(ctx, &schema.UserInput{Content: "hello"}, sessionID)
			select {
			case <-channels.Done():
			case <-time.After(5 * time.Second):
				t.Fatal("interaction did not finish")
			}
			ra.Stop()
			if !errors.Is(endTurnErr, conversationstore.ErrTurnNotFound) {
				t.Fatalf("turn still active at interaction.end: %v", endTurnErr)
			}
			if len(ra.active) != 0 {
				t.Fatal("completed interaction remains active")
			}
			messages, err := store.AllMemory(ctx, sessionID)
			if err != nil || len(messages) != tc.messages {
				t.Fatalf("persisted messages=%d, want=%d, err=%v", len(messages), tc.messages, err)
			}
			if events[0] != hooks.EventInteractionStart || events[len(events)-1] != hooks.EventInteractionEnd {
				t.Fatalf("unbalanced lifecycle: %v", events)
			}
			if tc.special != "" && events[len(events)-2] != tc.special {
				t.Fatalf("terminal events: %v", events)
			}
			spans := recorder.Ended()
			if len(spans) != 4 {
				t.Fatalf("ended spans=%d, want interaction/iteration/stage/model", len(spans))
			}
			for _, span := range spans {
				if span.SpanContext().TraceID().String() != channels.TraceID() {
					t.Fatal("trace ID differs from the response")
				}
			}
		})
	}
}
