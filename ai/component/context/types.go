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

// Package context compiles bounded, authorized model requests without depending
// on a model SDK. Sources provide data; server-owned policy assigns privileges.
package context

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"dubbo-admin-ai/security"
	"dubbo-admin-ai/store"
)

var (
	ErrMandatoryBudget = errors.New("CONTEXT_MANDATORY_BUDGET_EXCEEDED")
	ErrTemplateInput   = errors.New("PROMPT_REQUIRED_INPUT_MISSING")
	ErrTemplateVersion = errors.New("PROMPT_VERSION_UNAVAILABLE")
)

type ContextSource interface {
	Descriptor() SourceDescriptor
	Collect(context.Context, CollectRequest) (CollectResult, error)
}

type SourceDescriptor struct {
	ID, Version        string
	RequiredPermission []string
}
type CollectRequest struct {
	SummaryCalls             int
	SummaryBacklog           bool
	Summary                  *store.Summary
	HistorySnapshot          store.HistorySnapshot
	SummaryReason            string
	Access                   security.AccessContext
	SessionID, InteractionID string
	TurnID                   uint64
	Turns                    []store.TurnContext
	Recovery                 store.ToolRecovery
	MaxItems                 int
	// MaxCollectBytes bounds raw Source content before rendering. MaxBytes is
	// the independent rendered message/tool-schema budget. Metadata is count-bounded.
	MaxCollectBytes int
	MaxBytes        int
	NotBefore       *time.Time
	NotAfter        time.Time
}
type CollectResult struct{ Items []ContextItem }

// Event.Kind expresses canonical provenance, never a caller-selected system role.
type Event struct {
	Kind  string
	Text  string
	Batch *store.ToolBatch
}
type ContextItem struct {
	SummarySequence              uint64
	HistoryPrevious              uint64
	HistorySequence              uint64
	CoversThrough                uint64
	Scope                        string
	Trust                        string
	Authority                    string
	Sensitivity                  string
	UnavailableReason            string
	ID, SourceID, SourceVersion  string
	TenantID, OwnerID, SessionID string
	CanonicalID, GroupID         string
	Version                      uint64
	DerivedFrom                  []string
	ObservedAt                   time.Time
	ExpiresAt                    *time.Time
	Events                       []Event
}

// SourcePolicy is assigned at registration by the trusted composition root.
type SourcePolicy struct {
	// SummaryDedup permits terminal-position deduplication without treating
	// recovery details as complete Turns in history omission accounting.
	SummaryDedup    bool
	History         bool
	HistoryCoverage bool
	Scope           string
	// Classifications are assigned by the server, never promoted by a Source.
	Trust           string
	Authority       string
	Sensitivity     string
	Required        bool
	OptionalFailure bool
	// RecentSuffix prevents gaps only within this Source's ordered history.
	RecentSuffix bool
	Order        int
	Kinds        []string
}

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type Message struct {
	Role      string
	Text      string
	Requests  []store.ToolCall
	Responses []store.ToolCall
}

type CompileRequest struct {
	CollectRequest
	Model, Phase, TemplateRef                 string
	Variables                                 map[string]string
	Tools                                     []Tool // already filtered by system/template/caller policy
	ContextTokens, OutputTokens, SafetyMargin int
	Temperature, TopP                         float64
}

type CompilationDecision struct {
	CanonicalID, GroupID                                string
	ItemVersion                                         uint64
	Scope, Trust, Authority, Sensitivity                string
	Redacted                                            bool
	ItemID, SourceID, SourceVersion, Status, ReasonCode string
	EstimatedTokens                                     int
}
type CompilationReport struct {
	Summary                                                                      *SummaryReport
	HistoryOmissions                                                             []HistoryOmission
	ContextTokens                                                                int
	InteractionID, CompilerVersion, PolicyVersion, TemplateRef, EstimatorVersion string
	InputTokens, ReservedOutput, SafetyMargin                                    int
	Decisions                                                                    []CompilationDecision
	Degraded                                                                     bool
	DegradationReasons                                                           []string
}

// HistoryOmission identifies a missing interval by opaque terminal cursors:
// (After, Through]. It makes no assumption that sequence numbers are contiguous.
type HistoryOmission struct {
	After, Through uint64
	ReasonCode     string
}

type SummaryReport struct {
	SnapshotUpper, Target, AvailableCovered, SelectedCovered uint64
	Calls                                                    int
	Backlog, FragmentPending                                 bool
}
type CompiledRequest struct {
	ContextTokens                            int
	Model, Phase, TemplateRef, PolicyVersion string
	Messages                                 []Message
	Tools                                    []Tool
	OutputTokens, SafetyMargin               int
	Temperature, TopP                        float64
	Report                                   CompilationReport
}

func JSON(value any) json.RawMessage { data, _ := json.Marshal(value); return data }
