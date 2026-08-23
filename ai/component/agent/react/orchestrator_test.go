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
	"strings"
	"testing"

	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/schema"
)

func newState() *state {
	return &state{Input: &schema.UserInput{Content: "hi"}, Session: "s"}
}

func TestRunLoop_TerminatesWhenStepReportsDone(t *testing.T) {
	var think, act, observe int
	steps := []step{
		func(ctx context.Context, s *state) (bool, error) { think++; return false, nil },
		func(ctx context.Context, s *state) (bool, error) { act++; return false, nil },
		func(ctx context.Context, s *state) (bool, error) {
			observe++
			return observe >= 2, nil // done on the second round
		},
	}

	if err := runLoop(context.Background(), newState(), 10, steps...); err != nil {
		t.Fatalf("runLoop error: %v", err)
	}
	if think != 2 || act != 2 || observe != 2 {
		t.Fatalf("expected 2 rounds, got think=%d act=%d observe=%d", think, act, observe)
	}
}

func TestRunLoop_StopsAtMaxIterations(t *testing.T) {
	var calls int
	stepFn := func(ctx context.Context, s *state) (bool, error) { calls++; return false, nil }

	if err := runLoop(context.Background(), newState(), 3, stepFn); err != nil {
		t.Fatalf("runLoop error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected step to run maxIter=3 times, got %d", calls)
	}
}

func TestRunLoop_PropagatesStepError(t *testing.T) {
	sentinel := errors.New("boom")
	var second int
	steps := []step{
		func(ctx context.Context, s *state) (bool, error) { return false, sentinel },
		func(ctx context.Context, s *state) (bool, error) { second++; return false, nil },
	}

	err := runLoop(context.Background(), newState(), 5, steps...)
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	if second != 0 {
		t.Fatalf("later step should not run after an error, ran %d times", second)
	}
}

func TestRunLoop_RejectsNilInput(t *testing.T) {
	if err := runLoop(context.Background(), nil, 1, func(context.Context, *state) (bool, error) { return true, nil }); err == nil {
		t.Fatal("expected error for nil state")
	}
	if err := runLoop(context.Background(), &state{}, 1, func(context.Context, *state) (bool, error) { return true, nil }); err == nil {
		t.Fatal("expected error for nil input")
	}
}

func TestRunLoopEmitsIterationAndStageSequence(t *testing.T) {
	manager := hooks.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
	var events []hooks.Event
	if err := manager.Register(hooks.Registration{
		Events: hooks.AllEvents(),
		Tools:  []string{hooks.AllTools},
		Hook: func(ctx context.Context, state hooks.State) context.Context {
			events = append(events, state.Event)
			if state.InteractionID != "interaction-test" || state.SessionID != "session-test" {
				t.Fatalf("unexpected hook identity: %+v", state)
			}
			return ctx
		},
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	s := newState()
	s.Session = "session-test"
	s.InteractionID = "interaction-test"
	s.hookManager = manager
	stage := builtStage{name: "reasonAct", model: "test/model"}
	stepFn := withStageHooks(stage, func(context.Context, *state) (bool, error) {
		return true, nil
	})
	if err := runLoop(context.Background(), s, 2, stepFn); err != nil {
		t.Fatalf("runLoop() error = %v", err)
	}

	want := []hooks.Event{
		hooks.EventIterationStart,
		hooks.EventStageStart,
		hooks.EventStageEnd,
		hooks.EventIterationEnd,
	}
	if len(events) != len(want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Fatalf("events[%d] = %s, want %s (all: %v)", i, events[i], want[i], events)
		}
	}
}

func TestRunLoopPanicMarksStageAndIterationEnd(t *testing.T) {
	manager := hooks.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
	var endStates []hooks.State
	if err := manager.Register(hooks.Registration{
		Events: []hooks.Event{hooks.EventStageEnd, hooks.EventIterationEnd},
		Hook: func(ctx context.Context, state hooks.State) context.Context {
			endStates = append(endStates, state)
			return ctx
		},
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	s := newState()
	s.Session = "session"
	s.InteractionID = "interaction"
	s.hookManager = manager
	stage := builtStage{name: "reasonAct"}

	func() {
		defer func() { _ = recover() }()
		_ = runLoop(context.Background(), s, 1, withStageHooks(stage, func(context.Context, *state) (bool, error) {
			panic("step panic")
		}))
	}()

	if len(endStates) != 2 {
		t.Fatalf("end states = %+v, want stage and iteration end", endStates)
	}
	for _, state := range endStates {
		if state.Error == "" {
			t.Fatalf("%s did not record panic error", state.Event)
		}
	}
}

var benchmarkHookState hooks.State

func benchmarkContentProvider() any {
	return map[string]any{"query": strings.Repeat("value", 100)}
}

func BenchmarkHookContentDisabled(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchmarkHookState = withHookInput(nil, hooks.EventModelCallStart, "", hooks.State{}, benchmarkContentProvider)
	}
}

func BenchmarkHookContentLoggingOnly(b *testing.B) {
	manager := hooks.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := manager.Register(hooks.Registration{
		Events: hooks.AllEvents(),
		Tools:  []string{hooks.AllTools},
		Hook:   func(ctx context.Context, _ hooks.State) context.Context { return ctx },
	}); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		benchmarkHookState = withHookInput(manager, hooks.EventModelCallStart, "", hooks.State{}, benchmarkContentProvider)
	}
}
