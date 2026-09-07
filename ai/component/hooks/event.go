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
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"
)

// Event identifies a stable boundary in an agent interaction.
type Event string

const (
	EventInteractionStart   Event = "interaction.start"
	EventInteractionEnd     Event = "interaction.end"
	EventInteractionError   Event = "interaction.error"
	EventInteractionCancel  Event = "interaction.cancel"
	EventInteractionDegrade Event = "interaction.degrade"
	EventIterationStart     Event = "iteration.start"
	EventIterationEnd       Event = "iteration.end"
	EventStageStart         Event = "stage.start"
	EventStageEnd           Event = "stage.end"
	EventStageError         Event = "stage.error"
	EventModelCallStart     Event = "model_call.start"
	EventModelCallEnd       Event = "model_call.end"
	EventModelCallError     Event = "model_call.error"
	EventModelCallChunk     Event = "model_call.chunk"
	EventToolCallStart      Event = "tool_call.start"
	EventToolCallEnd        Event = "tool_call.end"
	EventToolCallError      Event = "tool_call.error"
)

const (
	FallbackReasonEmptyResponse = "empty_response"
	FallbackReasonTimeout       = "timeout"
	FallbackReasonParseError    = "parse_error"
)

var allEvents = []Event{
	EventInteractionStart,
	EventInteractionError,
	EventInteractionCancel,
	EventInteractionDegrade,
	EventInteractionEnd,
	EventIterationStart,
	EventIterationEnd,
	EventStageStart,
	EventStageError,
	EventStageEnd,
	EventModelCallStart,
	EventModelCallError,
	EventModelCallEnd,
	EventToolCallStart,
	EventToolCallError,
	EventToolCallEnd,
}

// AllEvents returns a copy of every supported event in lifecycle order.
func AllEvents() []Event {
	events := make([]Event, len(allEvents))
	copy(events, allEvents)
	return events
}

func (e Event) valid() bool {
	for _, candidate := range allEvents {
		if e == candidate {
			return true
		}
	}
	return false
}

func (e Event) toolCall() bool {
	return e == EventToolCallStart || e == EventToolCallEnd || e == EventToolCallError
}

// State is a read-only snapshot supplied to hooks. Go context propagation is
// carried separately through context.Context and must not be stored here.
type State struct {
	Event          Event
	InteractionID  string
	SessionID      string
	Iteration      int
	Stage          string
	Model          string
	ToolName       string
	ToolCallID     string
	Input          string
	Output         string
	Error          string
	ErrorType      string
	Degraded       bool
	FallbackUsed   bool
	FallbackReason string
	InputTokens    int
	OutputTokens   int
	TotalTokens    int
	StartedAt      time.Time
	EndedAt        time.Time
	inputContent   *lazyContentSnapshot
	outputContent  *lazyContentSnapshot
}

type lazyContentSnapshot struct {
	once     sync.Once
	provider func() any
	content  string
}

func newLazyContentSnapshot(provider func() any) *lazyContentSnapshot {
	if provider == nil {
		return nil
	}
	return &lazyContentSnapshot{provider: provider}
}

func (s *lazyContentSnapshot) snapshot() string {
	if s == nil {
		return ""
	}
	s.once.Do(func() {
		s.content = SnapshotContent(s.provider())
		s.provider = nil
	})
	return s.content
}

// WithInputContent attaches an immutable input snapshot that is materialized
// only if a matching content-capturing hook requests it.
func (s State) WithInputContent(provider func() any) State {
	s.inputContent = newLazyContentSnapshot(provider)
	return s
}

// WithOutputContent attaches an immutable output snapshot that is materialized
// only if a matching content-capturing hook requests it.
func (s State) WithOutputContent(provider func() any) State {
	s.outputContent = newLazyContentSnapshot(provider)
	return s
}

func (s State) snapshotInputContent() string {
	if s.Input != "" {
		return s.Input
	}
	return s.inputContent.snapshot()
}

func (s State) snapshotOutputContent() string {
	if s.Output != "" {
		return s.Output
	}
	return s.outputContent.snapshot()
}

// ErrorMessage snapshots an error without exposing a mutable implementation
// through the error interface.
func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// ErrorType snapshots the concrete error type for OpenTelemetry error.type.
func ErrorType(err error) string {
	if err == nil {
		return ""
	}
	for unwrapped := errors.Unwrap(err); unwrapped != nil; unwrapped = errors.Unwrap(err) {
		err = unwrapped
	}
	typ := reflect.TypeOf(err)
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ.PkgPath() + "." + typ.Name()
}

// SnapshotContent returns an immutable JSON representation of hook content.
// Hooks must never receive the original pointers, maps, or slices used by the
// agent because observational hooks are only allowed to derive a context.
func SnapshotContent(value any) string {
	if value == nil {
		return ""
	}
	content, err := json.Marshal(value)
	if err == nil {
		return string(content)
	}
	return strconv.Quote(fmt.Sprint(value))
}
