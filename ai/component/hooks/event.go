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

// contentKind classifies a captured payload.
//
// The hooks package applies the reasoning policy to a captured payload. That
// policy understands the generative AI message format only, so a payload of an
// unknown kind is left untouched: a tool argument that happens to use the
// field names of a message is business data, and the trace has to report it
// faithfully.
type contentKind int

const (
	// contentKindOpaque is a payload whose meaning the hooks package does not
	// know: the arguments of a tool call, the result of a tool call, or any
	// other business data.
	contentKindOpaque contentKind = iota
	// contentKindGenAIMessages is a generative AI messages document: the
	// serialized form of the messages the semantic conventions describe as
	// gen_ai.input.messages and gen_ai.output.messages.
	contentKindGenAIMessages
)

// contentKindForEvent reports what the captured content of an event holds.
//
// The events that carry a message document are the ones the semantic
// conventions map to gen_ai.input.messages and gen_ai.output.messages. The
// tool events carry the arguments and the result of a tool call, which are
// business data: the reasoning policy does not understand them, and rewriting
// them would make the trace report a call that never happened.
func contentKindForEvent(event Event) contentKind {
	switch event {
	case EventInteractionEnd, EventModelCallStart, EventModelCallEnd:
		return contentKindGenAIMessages
	default:
		return contentKindOpaque
	}
}

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
	// reasoningPolicy filters the reasoning parts of the captured content.
	// The Manager sets it from the registration that captured the content.
	reasoningPolicy string
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

// rawInputContent returns the captured input without applying any policy.
func (s State) rawInputContent() string {
	content := s.Input
	if content == "" {
		content = s.inputContent.snapshot()
	}
	return content
}

// rawOutputContent returns the captured output without applying any policy.
func (s State) rawOutputContent() string {
	content := s.Output
	if content == "" {
		content = s.outputContent.snapshot()
	}
	return content
}

func (s State) snapshotInputContent() string {
	return s.applyReasoningPolicy(s.rawInputContent(), s.reasoningPolicy)
}

func (s State) snapshotOutputContent() string {
	return s.applyReasoningPolicy(s.rawOutputContent(), s.reasoningPolicy)
}

// applyReasoningPolicy removes or truncates the reasoning parts of the
// captured content according to the given policy.
//
// The policy only understands the generative AI message format, so it is
// applied to a message document and to nothing else. A tool argument that
// happens to carry the field names of a message is business data: rewriting it
// would make the trace report a call that never happened.
func (s State) applyReasoningPolicy(content string, policy string) string {
	if policy == "" || content == "" {
		return content
	}
	if contentKindForEvent(s.Event) != contentKindGenAIMessages {
		return content
	}
	return ApplyReasoningPolicy(content, policy)
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
