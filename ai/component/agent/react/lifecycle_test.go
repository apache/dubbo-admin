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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"dubbo-admin-ai/component/agent"
	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/schema"

	"github.com/firebase/genkit/go/ai"
)

type cancelAwarePrompt struct {
	started chan struct{}
	once    sync.Once
}

func (p *cancelAwarePrompt) Name() string { return "cancel-aware" }
func (p *cancelAwarePrompt) Render(context.Context, any) (*ai.GenerateActionOptions, error) {
	return &ai.GenerateActionOptions{}, nil
}
func (p *cancelAwarePrompt) Execute(ctx context.Context, _ ...ai.PromptExecuteOption) (*ai.ModelResponse, error) {
	p.once.Do(func() { close(p.started) })
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestStopCancelsAndDrainsActiveInteraction(t *testing.T) {
	prompt := &cancelAwarePrompt{started: make(chan struct{})}
	manager := hooks.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
	var interactionEnded atomic.Bool
	if err := manager.Register(hooks.Registration{
		Events: []hooks.Event{hooks.EventInteractionEnd},
		Hook: func(ctx context.Context, state hooks.State) context.Context {
			interactionEnded.Store(true)
			return ctx
		},
	}); err != nil {
		t.Fatalf("register hook: %v", err)
	}

	ra := testAgent(nil, 1, nil)
	ra.hookManager = manager
	ra.maxIterations = 1
	ra.bufferSize = 16
	ra.actPrompt = prompt
	ra.answerPrompt = prompt
	ra.callTimeout = time.Minute
	channels := ra.Interact(context.Background(), &schema.UserInput{Content: "hello"}, "session")
	select {
	case <-prompt.started:
	case <-time.After(time.Second):
		t.Fatal("interaction did not start")
	}

	stopped := make(chan struct{})
	go func() {
		ra.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Stop() did not drain the cancelled interaction")
	}
	if !interactionEnded.Load() {
		t.Fatal("Stop() returned before interaction end hooks ran")
	}
	select {
	case <-channels.Done():
	default:
		t.Fatal("interaction channels were not closed")
	}

	rejected := ra.Interact(context.Background(), &schema.UserInput{Content: "late"}, "session")
	select {
	case err := <-rejected.ErrorChan:
		if !errors.Is(err, ErrAgentStopping) {
			t.Fatalf("rejected interaction error = %v", err)
		}
	default:
		t.Fatal("stopping agent accepted a new interaction")
	}
}

func TestInteractionPanicStillEmitsEnd(t *testing.T) {
	manager := hooks.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
	var endState hooks.State
	if err := manager.Register(hooks.Registration{
		Events: []hooks.Event{hooks.EventInteractionEnd},
		Hook: func(ctx context.Context, state hooks.State) context.Context {
			endState = state
			return ctx
		},
	}); err != nil {
		t.Fatalf("register hook: %v", err)
	}

	ra := testAgent(nil, 1, nil)
	ra.hookManager = manager
	ra.maxIterations = 1
	ra.bufferSize = 16
	ra.actPrompt = &panicPrompt{}
	ra.answerPrompt = &panicPrompt{}
	ra.callTimeout = time.Second
	channels := ra.Interact(context.Background(), &schema.UserInput{Content: "hello"}, "session")
	select {
	case <-channels.Done():
	case <-time.After(time.Second):
		t.Fatal("panicking interaction did not finish")
	}
	if endState.Event != hooks.EventInteractionEnd || endState.Error == "" {
		t.Fatalf("interaction end state = %+v, want panic error", endState)
	}
}

func TestInteractionWithoutToolsEmitsCompleteLifecycle(t *testing.T) {
	manager := hooks.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
	var (
		mu     sync.Mutex
		states []hooks.State
	)
	if err := manager.Register(hooks.Registration{
		Events: hooks.AllEvents(),
		Tools:  []string{hooks.AllTools},
		Hook: func(ctx context.Context, state hooks.State) context.Context {
			mu.Lock()
			states = append(states, state)
			mu.Unlock()
			return ctx
		},
	}); err != nil {
		t.Fatal(err)
	}
	ra := testAgent(nil, 1, nil)
	ra.hookManager = manager
	ra.maxIterations = 1
	ra.bufferSize = 16
	ra.actPrompt = &stubPrompt{resp: textResp("answer")}
	ra.answerPrompt = ra.actPrompt
	channels := ra.Interact(context.Background(), &schema.UserInput{Content: "hello"}, "session")
	select {
	case <-channels.Done():
	case <-time.After(time.Second):
		t.Fatal("interaction did not finish")
	}

	want := []hooks.Event{
		hooks.EventInteractionStart,
		hooks.EventIterationStart,
		hooks.EventStageStart, hooks.EventModelCallStart, hooks.EventModelCallEnd, hooks.EventStageEnd,
		hooks.EventIterationEnd,
		hooks.EventInteractionEnd,
	}
	mu.Lock()
	defer mu.Unlock()
	if len(states) != len(want) {
		t.Fatalf("events = %+v, want %v", states, want)
	}
	interactionID := states[0].InteractionID
	for i, event := range want {
		if states[i].Event != event {
			t.Fatalf("events[%d] = %s, want %s", i, states[i].Event, event)
		}
		if states[i].InteractionID != interactionID || states[i].SessionID != "session" {
			t.Fatalf("event identity leaked at %d: %+v", i, states[i])
		}
	}
}

func TestInteractionContextsAreIsolated(t *testing.T) {
	ra := &ReActAgent{}
	first, ok := ra.beginInteraction(context.Background(), "first")
	if !ok {
		t.Fatal("first interaction was rejected")
	}
	second, ok := ra.beginInteraction(context.Background(), "second")
	if !ok {
		t.Fatal("second interaction was rejected")
	}

	ra.finishInteraction("first")
	select {
	case <-first.Done():
	default:
		t.Fatal("finished interaction context was not cancelled")
	}
	select {
	case <-second.Done():
		t.Fatal("finishing one interaction cancelled another")
	default:
	}

	finished := make(chan struct{})
	go func() {
		<-second.Done()
		ra.finishInteraction("second")
		close(finished)
	}()
	ra.Stop()
	<-finished
	select {
	case <-second.Done():
	default:
		t.Fatal("agent stop did not cancel remaining interaction")
	}
}

func TestConcurrentInteractionsKeepHookIdentityIsolated(t *testing.T) {
	manager := hooks.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
	type interactionEvents struct {
		session string
		events  []hooks.Event
	}
	interactions := make(map[string]*interactionEvents)
	var mu sync.Mutex
	if err := manager.Register(hooks.Registration{
		Events: hooks.AllEvents(), Tools: []string{hooks.AllTools},
		Hook: func(ctx context.Context, state hooks.State) context.Context {
			mu.Lock()
			defer mu.Unlock()
			entry := interactions[state.InteractionID]
			if entry == nil {
				entry = &interactionEvents{session: state.SessionID}
				interactions[state.InteractionID] = entry
			}
			if entry.session != state.SessionID {
				t.Errorf("interaction %s mixed sessions %s and %s", state.InteractionID, entry.session, state.SessionID)
			}
			entry.events = append(entry.events, state.Event)
			return ctx
		},
	}); err != nil {
		t.Fatal(err)
	}
	ra := testAgent(nil, 1, nil)
	ra.hookManager = manager
	ra.maxIterations = 1
	ra.bufferSize = 16
	ra.actPrompt = &stubPrompt{resp: textResp("answer")}
	ra.answerPrompt = ra.actPrompt
	first := ra.Interact(context.Background(), &schema.UserInput{Content: "first"}, "session-1")
	second := ra.Interact(context.Background(), &schema.UserInput{Content: "second"}, "session-2")
	for i, channels := range []*agent.Channels{first, second} {
		select {
		case <-channels.Done():
		case <-time.After(time.Second):
			t.Fatalf("interaction %d did not finish", i+1)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(interactions) != 2 {
		t.Fatalf("interaction identities = %v, want 2", interactions)
	}
	for id, entry := range interactions {
		if len(entry.events) == 0 || entry.events[0] != hooks.EventInteractionStart || entry.events[len(entry.events)-1] != hooks.EventInteractionEnd {
			t.Fatalf("interaction %s lifecycle = %v", id, entry.events)
		}
	}
}

type stubPrompt struct {
	resp *ai.ModelResponse
	err  error
}

type panicPrompt struct{ stubPrompt }

func (p *panicPrompt) Execute(context.Context, ...ai.PromptExecuteOption) (*ai.ModelResponse, error) {
	panic("model panic")
}

func (s *stubPrompt) Name() string { return "stub" }
func (s *stubPrompt) Execute(ctx context.Context, opts ...ai.PromptExecuteOption) (*ai.ModelResponse, error) {
	return s.resp, s.err
}
func (s *stubPrompt) Render(ctx context.Context, input any) (*ai.GenerateActionOptions, error) {
	return &ai.GenerateActionOptions{}, nil
}
