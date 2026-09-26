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

package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testManager() *Manager {
	return NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestManagerSelectsEventsAndTools(t *testing.T) {
	tests := []struct {
		name      string
		tools     []string
		toolName  string
		wantCalls int
	}{
		{name: "one tool matches", tools: []string{"query_metrics"}, toolName: "query_metrics", wantCalls: 1},
		{name: "one tool does not match", tools: []string{"query_metrics"}, toolName: "get_service", wantCalls: 0},
		{name: "one of many matches", tools: []string{"query_metrics", "get_service"}, toolName: "get_service", wantCalls: 1},
		{name: "wildcard matches", tools: []string{AllTools}, toolName: "anything", wantCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := testManager()
			calls := 0
			err := manager.Register(Registration{
				Events: []Event{EventToolCallStart},
				Tools:  tt.tools,
				Hook: func(ctx context.Context, state State) context.Context {
					calls++
					return ctx
				},
			})
			if err != nil {
				t.Fatalf("Register() error = %v", err)
			}

			manager.Emit(context.Background(), State{Event: EventStageStart, ToolName: tt.toolName})
			manager.Emit(context.Background(), State{Event: EventToolCallStart, ToolName: tt.toolName})
			if calls != tt.wantCalls {
				t.Fatalf("calls = %d, want %d", calls, tt.wantCalls)
			}
		})
	}
}

func TestManagerRejectsToolHookWithoutSelector(t *testing.T) {
	err := testManager().Register(Registration{
		Events: []Event{EventToolCallStart},
		Hook:   func(ctx context.Context, _ State) context.Context { return ctx },
	})
	if err == nil {
		t.Fatal("Register() accepted a tool hook without tool selectors")
	}
}

func TestManagerNeedsContentMatchesEventAndToolSelector(t *testing.T) {
	manager := testManager()
	if err := manager.Register(Registration{
		Events:         []Event{EventToolCallStart},
		Tools:          []string{"lookup"},
		CaptureContent: true,
		Hook:           func(ctx context.Context, _ State) context.Context { return ctx },
	}); err != nil {
		t.Fatal(err)
	}
	if !manager.NeedsContent(EventToolCallStart, "lookup") {
		t.Fatal("matching content registration was ignored")
	}
	if manager.NeedsContent(EventToolCallStart, "other") || manager.NeedsContent(EventModelCallStart, "") {
		t.Fatal("content capture escaped its registration selector")
	}
}

func TestManagerLoggingOnlyDoesNotNeedContent(t *testing.T) {
	manager := testManager()
	if err := manager.Register(Registration{
		Events: AllEvents(),
		Tools:  []string{AllTools},
		Hook:   func(ctx context.Context, _ State) context.Context { return ctx },
	}); err != nil {
		t.Fatal(err)
	}
	if manager.NeedsContent(EventModelCallStart, "") {
		t.Fatal("observational logging hook unexpectedly requested content")
	}
}

func TestManagerObservesRegistrationsAddedAfterEmptyEmit(t *testing.T) {
	manager := testManager()
	manager.Emit(context.Background(), State{Event: EventInteractionStart})

	calls := 0
	if err := manager.Register(Registration{
		Events: []Event{EventInteractionStart},
		Hook: func(ctx context.Context, _ State) context.Context {
			calls++
			return ctx
		},
	}); err != nil {
		t.Fatal(err)
	}
	manager.Emit(context.Background(), State{Event: EventInteractionStart})
	if calls != 1 {
		t.Fatalf("late registration calls = %d, want 1", calls)
	}
}

type contextKey struct{}

func TestManagerPropagatesDerivedContextAndRecoversPanic(t *testing.T) {
	manager := testManager()
	if err := manager.Register(Registration{
		Events: []Event{EventInteractionStart},
		Hook: func(context.Context, State) context.Context {
			panic("broken observer")
		},
	}); err != nil {
		t.Fatalf("register panic hook: %v", err)
	}
	if err := manager.Register(Registration{
		Events:         []Event{EventInteractionStart, EventInteractionEnd},
		DerivesContext: true,
		Hook: func(ctx context.Context, _ State) context.Context {
			return context.WithValue(ctx, contextKey{}, "derived")
		},
	}); err != nil {
		t.Fatalf("register context hook: %v", err)
	}

	ctx := manager.Emit(context.Background(), State{Event: EventInteractionStart})
	if got := ctx.Value(contextKey{}); got != "derived" {
		t.Fatalf("derived context value = %v, want derived", got)
	}
}

func TestManagerRejectsMultipleContextDerivers(t *testing.T) {
	manager := testManager()
	registration := Registration{
		Events:         []Event{EventInteractionStart, EventInteractionEnd},
		DerivesContext: true,
		Hook:           func(ctx context.Context, _ State) context.Context { return ctx },
	}
	if err := manager.Register(registration); err != nil {
		t.Fatalf("register first context deriver: %v", err)
	}
	if err := manager.Register(registration); err == nil {
		t.Fatal("Register() accepted a second context-deriving hook")
	}
}

func TestManagerRejectsUnpairedContextLifecycleEvents(t *testing.T) {
	pairs := []struct {
		name  string
		start Event
		end   Event
	}{
		{name: "interaction", start: EventInteractionStart, end: EventInteractionEnd},
		{name: "iteration", start: EventIterationStart, end: EventIterationEnd},
		{name: "stage", start: EventStageStart, end: EventStageEnd},
		{name: "model", start: EventModelCallStart, end: EventModelCallEnd},
		{name: "tool", start: EventToolCallStart, end: EventToolCallEnd},
	}
	for _, pair := range pairs {
		for _, event := range []Event{pair.start, pair.end} {
			t.Run(pair.name+"/"+string(event), func(t *testing.T) {
				registration := Registration{
					Events:         []Event{event},
					DerivesContext: true,
					Hook:           func(ctx context.Context, _ State) context.Context { return ctx },
				}
				if event.toolCall() {
					registration.Tools = []string{AllTools}
				}
				if err := testManager().Register(registration); err == nil {
					t.Fatalf("Register() accepted unpaired context event %q", event)
				}
			})
		}
	}
}

func TestManagerIgnoresUndeclaredDerivedContext(t *testing.T) {
	manager := testManager()
	if err := manager.Register(Registration{
		Events: []Event{EventInteractionStart},
		Hook: func(ctx context.Context, _ State) context.Context {
			return context.WithValue(ctx, contextKey{}, "unexpected")
		},
	}); err != nil {
		t.Fatal(err)
	}
	ctx := manager.Emit(context.Background(), State{Event: EventInteractionStart})
	if value := ctx.Value(contextKey{}); value != nil {
		t.Fatalf("undeclared derived context escaped hook: %v", value)
	}
}

func TestManagerConcurrentEmit(t *testing.T) {
	manager := testManager()
	var calls atomic.Int64
	if err := manager.Register(Registration{
		Events: []Event{EventStageStart},
		Hook: func(ctx context.Context, _ State) context.Context {
			calls.Add(1)
			return ctx
		},
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	const workers = 64
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			manager.Emit(context.Background(), State{Event: EventStageStart})
		}()
	}
	wg.Wait()
	if got := calls.Load(); got != workers {
		t.Fatalf("calls = %d, want %d", got, workers)
	}
}

func TestHookContentSnapshotCannotMutateSource(t *testing.T) {
	input := map[string]any{"query": "original"}
	manager := testManager()
	if err := manager.Register(Registration{
		Events:         []Event{EventModelCallStart},
		CaptureContent: true,
		Hook: func(ctx context.Context, state State) context.Context {
			var copy map[string]any
			if err := json.Unmarshal([]byte(state.Input), &copy); err != nil {
				t.Fatalf("unmarshal hook snapshot: %v", err)
			}
			copy["query"] = "modified"
			state.Input = SnapshotContent(copy)
			return ctx
		},
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	state := State{Event: EventModelCallStart}.WithInputContent(func() any { return input })
	manager.Emit(context.Background(), state)
	if got := input["query"]; got != "original" {
		t.Fatalf("hook mutated source input: %v", got)
	}
}

func TestManagerOnlyExposesDeferredContentToOptedInHook(t *testing.T) {
	manager := testManager()
	var observedWithoutOptIn string
	if err := manager.Register(Registration{
		Events: []Event{EventModelCallStart},
		Hook: func(ctx context.Context, state State) context.Context {
			observedWithoutOptIn = state.snapshotInputContent()
			return ctx
		},
	}); err != nil {
		t.Fatal(err)
	}
	var observedWithOptIn string
	if err := manager.Register(Registration{
		Events:         []Event{EventModelCallStart},
		CaptureContent: true,
		Hook: func(ctx context.Context, state State) context.Context {
			observedWithOptIn = state.Input
			return ctx
		},
	}); err != nil {
		t.Fatal(err)
	}

	state := State{Event: EventModelCallStart}.WithInputContent(func() any {
		return map[string]string{"message": "secret"}
	})
	manager.Emit(context.Background(), state)
	if observedWithoutOptIn != "" {
		t.Fatalf("non-capturing hook received content: %s", observedWithoutOptIn)
	}
	if observedWithOptIn == "" {
		t.Fatal("capturing hook did not receive content")
	}
}

func BenchmarkDisabledHookFastPath(b *testing.B) {
	ctx := context.Background()
	state := State{Event: EventInteractionStart}
	var manager *Manager
	b.ReportAllocs()
	for b.Loop() {
		if manager != nil {
			ctx = manager.Emit(ctx, state)
		}
	}
}

func BenchmarkEmptyManagerFastPath(b *testing.B) {
	ctx := context.Background()
	state := State{Event: EventInteractionStart}
	manager := testManager()
	b.ReportAllocs()
	for b.Loop() {
		ctx = manager.Emit(ctx, state)
	}
}

func TestManagerConcurrentRegisterEmitAndNeedsContent(t *testing.T) {
	manager := testManager()
	var baseline atomic.Int64
	if err := manager.Register(Registration{
		Events: []Event{EventToolCallStart}, Tools: []string{"lookup"}, CaptureContent: true,
		Hook: func(ctx context.Context, _ State) context.Context { baseline.Add(1); return ctx },
	}); err != nil {
		t.Fatal(err)
	}
	const workers, iterations = 8, 32
	var added [workers]atomic.Int64
	start := make(chan struct{})
	failures := make(chan error, workers*2)
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			if err := manager.Register(Registration{
				Events: []Event{EventToolCallStart}, Tools: []string{"lookup"}, CaptureContent: true,
				Hook: func(ctx context.Context, _ State) context.Context { added[i].Add(1); return ctx },
			}); err != nil {
				failures <- err
			}
		}()
		go func() {
			defer wg.Done()
			<-start
			for range iterations {
				if !manager.NeedsContent(EventToolCallStart, "lookup") || manager.NeedsContent(EventToolCallStart, "other") {
					failures <- fmt.Errorf("content selectors changed during concurrent registration")
					return
				}
				manager.Emit(context.Background(), State{Event: EventToolCallStart, ToolName: "lookup"})
			}
		}()
	}
	close(start)
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if got := baseline.Load(); got != workers*iterations {
		t.Fatalf("baseline calls = %d, want %d", got, workers*iterations)
	}
	var before [workers]int64
	for i := range workers {
		before[i] = added[i].Load()
	}
	manager.Emit(context.Background(), State{Event: EventToolCallStart, ToolName: "lookup"})
	for i := range workers {
		if got := added[i].Load(); got != before[i]+1 {
			t.Errorf("registration %d calls = %d, want %d", i, got, before[i]+1)
		}
	}
}

func TestManagerReentrantRegistrationUsesNextSnapshot(t *testing.T) {
	manager := testManager()
	var first, tail, late atomic.Int64
	registrationResult := make(chan error, 1)
	if err := manager.Register(Registration{
		Events: []Event{EventModelCallStart},
		Hook: func(ctx context.Context, _ State) context.Context {
			if first.Add(1) == 1 {
				err := manager.Register(Registration{
					Events: []Event{EventModelCallStart}, CaptureContent: true,
					Hook: func(ctx context.Context, _ State) context.Context { late.Add(1); return ctx },
				})
				if err == nil && !manager.NeedsContent(EventModelCallStart, "") {
					err = fmt.Errorf("callback cannot see the registration it just added")
				}
				registrationResult <- err
			}
			return ctx
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := manager.Register(Registration{
		Events: []Event{EventModelCallStart},
		Hook:   func(ctx context.Context, _ State) context.Context { tail.Add(1); return ctx },
	}); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		manager.Emit(context.Background(), State{Event: EventModelCallStart})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Emit deadlocked when a callback registered another hook")
	}
	if err := <-registrationResult; err != nil {
		t.Fatal(err)
	}
	if first.Load() != 1 || tail.Load() != 1 || late.Load() != 0 {
		t.Fatalf("first snapshot calls = %d/%d/%d, want 1/1/0", first.Load(), tail.Load(), late.Load())
	}
	manager.Emit(context.Background(), State{Event: EventModelCallStart})
	if first.Load() != 2 || tail.Load() != 2 || late.Load() != 1 {
		t.Fatalf("second snapshot calls = %d/%d/%d, want 2/2/1", first.Load(), tail.Load(), late.Load())
	}
}

func BenchmarkManagerEmitWithRegistrations(b *testing.B) {
	for _, count := range []int{1, 4, 16} {
		b.Run(fmt.Sprintf("hooks=%d", count), func(b *testing.B) {
			manager := testManager()
			for range count {
				if err := manager.Register(Registration{
					Events: []Event{EventInteractionStart},
					Hook:   func(ctx context.Context, _ State) context.Context { return ctx },
				}); err != nil {
					b.Fatal(err)
				}
			}
			ctx := context.Background()
			state := State{Event: EventInteractionStart}
			b.ReportAllocs()
			for b.Loop() {
				manager.Emit(ctx, state)
			}
		})
	}
}
