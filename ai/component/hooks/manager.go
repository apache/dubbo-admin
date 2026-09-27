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
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
)

const AllTools = "*"

// Hook observes an event. A returned context affects nested work only when its
// Registration declares DerivesContext; returning nil keeps the current
// context unchanged.
type Hook func(context.Context, State) context.Context

// Registration declares which events and tools a Hook observes.
type Registration struct {
	Events         []Event
	Tools          []string
	// CaptureContent opts into payload capture. When true, State.Input and
	// State.Output are materialized eagerly before the hook is called.
	// Built-in hooks may use the unexported lazyContent field to defer
	// serialization until a recording span or structured log is emitted;
	// external hooks always receive eagerly-snapshotted content.
	CaptureContent bool
	// DerivesContext grants this hook sole ownership of context propagation.
	// A Manager rejects a second registration with this capability and requires
	// every selected lifecycle Start event to include its matching End event.
	DerivesContext bool
	lazyContent    bool
	Hook           Hook
}

type compiledRegistration struct {
	events         map[Event]struct{}
	tools          map[string]struct{}
	allTools       bool
	captureContent bool
	derivesContext bool
	lazyContent    bool
	hook           Hook
}

var contextLifecyclePairs = [][2]Event{
	{EventInteractionStart, EventInteractionEnd},
	{EventIterationStart, EventIterationEnd},
	{EventStageStart, EventStageEnd},
	{EventModelCallStart, EventModelCallEnd},
	{EventToolCallStart, EventToolCallEnd},
}

// Manager dispatches immutable state snapshots in registration order.
type Manager struct {
	mu            sync.RWMutex
	logger        *slog.Logger
	registrations []compiledRegistration
	needsContent  atomic.Bool
	hasHooks      atomic.Bool
	hasDeriver    bool
}

func NewManager(logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{logger: logger}
}

func (m *Manager) Register(reg Registration) error {
	if reg.Hook == nil {
		return fmt.Errorf("hook is required")
	}
	if len(reg.Events) == 0 {
		return fmt.Errorf("at least one hook event is required")
	}

	compiled := compiledRegistration{
		events:         make(map[Event]struct{}, len(reg.Events)),
		tools:          make(map[string]struct{}, len(reg.Tools)),
		captureContent: reg.CaptureContent,
		derivesContext: reg.DerivesContext,
		lazyContent:    reg.lazyContent,
		hook:           reg.Hook,
	}
	needsToolSelector := false
	for _, event := range reg.Events {
		if !event.valid() {
			return fmt.Errorf("unsupported hook event %q", event)
		}
		compiled.events[event] = struct{}{}
		needsToolSelector = needsToolSelector || event.toolCall()
	}
	if needsToolSelector && len(reg.Tools) == 0 {
		return fmt.Errorf("tool call hooks require at least one tool name or %q", AllTools)
	}
	for _, tool := range reg.Tools {
		if tool == "" {
			return fmt.Errorf("tool name must not be empty")
		}
		if tool == AllTools {
			compiled.allTools = true
			continue
		}
		compiled.tools[tool] = struct{}{}
	}
	if compiled.derivesContext {
		for _, pair := range contextLifecyclePairs {
			_, hasStart := compiled.events[pair[0]]
			_, hasEnd := compiled.events[pair[1]]
			if hasStart != hasEnd {
				return fmt.Errorf("context-deriving hook must register %q and %q together", pair[0], pair[1])
			}
		}
	}

	m.mu.Lock()
	if compiled.derivesContext && m.hasDeriver {
		m.mu.Unlock()
		return fmt.Errorf("only one context-deriving hook may be registered")
	}
	m.registrations = append(m.registrations, compiled)
	m.hasHooks.Store(true)
	if compiled.derivesContext {
		m.hasDeriver = true
	}
	if compiled.captureContent {
		m.needsContent.Store(true)
	}
	m.mu.Unlock()
	return nil
}

// NeedsContent reports whether a matching hook requested payload capture.
// The common logging-only path returns without locking or allocating.
func (m *Manager) NeedsContent(event Event, toolName string) bool {
	if m == nil || !m.needsContent.Load() {
		return false
	}
	state := State{Event: event, ToolName: toolName}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, reg := range m.registrations {
		if reg.captureContent && reg.matches(state) {
			return true
		}
	}
	return false
}

// HasHooks reports whether the manager has any registered observers.
func (m *Manager) HasHooks() bool {
	return m != nil && m.hasHooks.Load()
}

// Emit dispatches one event and returns the context derived by matching hooks.
func (m *Manager) Emit(ctx context.Context, state State) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if m == nil || !m.hasHooks.Load() {
		return ctx
	}

	m.mu.RLock()
	registrations := make([]compiledRegistration, len(m.registrations))
	copy(registrations, m.registrations)
	m.mu.RUnlock()

	for _, reg := range registrations {
		if !reg.matches(state) {
			continue
		}
		hookState := state
		if !reg.captureContent {
			hookState.Input = ""
			hookState.Output = ""
			hookState.inputContent = nil
			hookState.outputContent = nil
		} else if !reg.lazyContent {
			hookState.Input = state.snapshotInputContent()
			hookState.Output = state.snapshotOutputContent()
		}
		derived := m.invoke(ctx, hookState, reg.hook)
		if reg.derivesContext {
			ctx = derived
		}
	}
	return ctx
}

func (r compiledRegistration) matches(state State) bool {
	if _, ok := r.events[state.Event]; !ok {
		return false
	}
	if !state.Event.toolCall() {
		return true
	}
	if r.allTools {
		return true
	}
	_, ok := r.tools[state.ToolName]
	return ok
}

func (m *Manager) invoke(ctx context.Context, state State, hook Hook) (next context.Context) {
	next = ctx
	defer func() {
		if recovered := recover(); recovered != nil {
			m.logger.Error("Agent hook panicked", "event", state.Event, "error", recovered)
		}
	}()
	if derived := hook(ctx, state); derived != nil {
		next = derived
	}
	return next
}
