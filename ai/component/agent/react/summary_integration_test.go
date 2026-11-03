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
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/component/memory/summary"
	"dubbo-admin-ai/schema"
	"dubbo-admin-ai/security"
	"dubbo-admin-ai/store"
	"dubbo-admin-ai/testutils"
	"github.com/firebase/genkit/go/ai"
)

type summaryUsageRecorder struct {
	mu     sync.Mutex
	events []store.UsageEvent
}

func (r *summaryUsageRecorder) RecordUsage(_ context.Context, e store.UsageEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
	return nil
}
func (r *summaryUsageRecorder) RecordAudit(context.Context, store.AuditEvent) error { return nil }

func TestSummaryReachesProviderAndAccountsAuxiliaryCalls(t *testing.T) {
	provider := &providerFixture{summaries: true}
	_, backend, spec, g, _ := compilerAgentFixture(t, provider)
	ctx := testutils.ContextWithTestAccess(context.Background())
	access := testutils.TestAccess()
	p, err := backend.PrepareInteraction(ctx, access, "compiled", "seed", "seed", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.AddHistoryToTurn(ctx, access, "compiled", p.TurnID, ai.NewUserTextMessage("initial goal: identify outage"), ai.NewModelTextMessage("original answer")); err != nil {
		t.Fatal(err)
	}
	if err := backend.CompleteInteraction(ctx, access, "compiled", p.InteractionID, p.TurnID, "original answer"); err != nil {
		t.Fatal(err)
	}
	spec.ConversationSummary.Enabled = true
	spec.ConversationSummary.TriggerHeadroomTurns = 1
	spec.ContextWindowTurns = 2
	spec.MaxIterations = 1
	recorder := &summaryUsageRecorder{}
	manager := hooks.NewManager(nil)
	if err := manager.Register(hooks.NewUsageAuditRegistration(recorder, nil)); err != nil {
		t.Fatal(err)
	}
	ra, err := NewReActAgentWithStore(g, backend, spec, newToolTimeoutResolver(5, nil), manager, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ra.Stop)
	for i := 0; i < 5; i++ {
		runCompiledInteraction(t, ra, backend, fmt.Sprintf("summary-%d", i))
	}
	v, err := backend.GetSummary(ctx, access, "compiled")
	if err != nil || v.Covered == 0 || v.Progress != nil || len(v.Payload.Goals) != 1 {
		t.Fatalf("summary not persisted: %+v %v", v, err)
	}
	provider.mu.Lock()
	requests := append([]map[string]any(nil), provider.requests...)
	provider.mu.Unlock()
	summaryCalls := 0
	lastMain := ""
	for _, request := range requests {
		encoded, _ := json.Marshal(request)
		text := string(encoded)
		if strings.Contains(text, "Create a bounded rolling conversation summary.") {
			summaryCalls++
			if tools, ok := request["tools"].([]any); ok && len(tools) > 0 {
				t.Fatal("summary received tools")
			}
			if int(request["max_tokens"].(float64)) != spec.ConversationSummary.OutputTokens {
				t.Fatal("summary output reserve missing")
			}
		} else {
			lastMain = text
		}
		messages := request["messages"].([]any)
		systems := 0
		for _, m := range messages {
			if m.(map[string]any)["role"] == "system" {
				systems++
			}
		}
		if systems != 1 {
			t.Fatal("duplicate/hidden system binding")
		}
	}
	if summaryCalls == 0 || !strings.Contains(lastMain, "earliest objective preserved") || strings.Contains(lastMain, "initial goal: identify outage") {
		t.Fatalf("provider lost summary or duplicated canonical input: %s", lastMain)
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	aux := 0
	for _, e := range recorder.events {
		if e.Auxiliary {
			aux++
			if e.CallType != "summary" || e.InteractionID == "" || e.CallID == "" || e.EstimatedInputTokens == nil || e.AmountSource != store.UsageAmountProvider || e.CostStatus != store.UsageCostUnknown {
				t.Fatalf("invalid auxiliary accounting: %+v", e)
			}
		}
	}
	if aux != summaryCalls {
		t.Fatalf("auxiliary events=%d provider calls=%d", aux, summaryCalls)
	}
}

func TestSummaryDisabledAndMandatoryOverflowDoNotCallAuxiliaryModel(t *testing.T) {
	provider := &providerFixture{summaries: true}
	_, backend, spec, g, _ := compilerAgentFixture(t, provider)
	spec.ConversationSummary.Enabled = true
	spec.ContextTokens = 1100
	spec.SafetyMargin = 100
	spec.MaxTokens = 512
	ra, err := NewReActAgentWithStore(g, backend, spec, newToolTimeoutResolver(5, nil), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ra.Stop)
	ctx := testutils.ContextWithTestAccess(context.Background())
	p, err := backend.PrepareInteraction(ctx, testutils.TestAccess(), "compiled", "overflow", "overflow", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	_, state, err := ra.newInteraction(store.WithPreparedInteraction(ctx, p), &schema.UserInput{Content: strings.Repeat("large", 300)}, "compiled")
	if state != nil {
		state.cancelPersistence()
	}
	if err == nil {
		t.Fatal("mandatory overflow accepted")
	}
	provider.mu.Lock()
	defer provider.mu.Unlock()
	if len(provider.requests) != 0 {
		t.Fatal("model called on mandatory overflow")
	}
}

type disappearingSummaryStore struct {
	store.SummaryStore
	value *store.Summary
	reads int
}

func (s *disappearingSummaryStore) GetSummary(context.Context, security.AccessContext, string) (*store.Summary, error) {
	s.reads++
	if s.reads == 1 {
		return s.value, nil
	}
	return nil, store.ErrSummaryNotFound
}

func TestSummaryFinalRevalidationRebudgetsHistory(t *testing.T) {
	provider := &providerFixture{}
	ra, backend, _, _, _ := compilerAgentFixture(t, provider)
	ctx := testutils.ContextWithTestAccess(context.Background())
	a := testutils.TestAccess()
	turn, err := backend.BeginTurn(ctx, a, "compiled")
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.AddHistoryToTurn(ctx, a, "compiled", turn, ai.NewUserTextMessage("recent canonical question"), ai.NewModelTextMessage("recent canonical answer")); err != nil {
		t.Fatal(err)
	}
	if err := backend.NextTurnForTurn(ctx, a, "compiled", turn); err != nil {
		t.Fatal(err)
	}
	_, state, err := ra.newInteraction(ctx, &schema.UserInput{Content: "current question"}, "compiled")
	if err != nil {
		t.Fatal(err)
	}
	defer state.cancelPersistence()
	snap, err := backend.OpenHistorySnapshot(ctx, a, "compiled", state.turnID, 0)
	if err != nil {
		t.Fatal(err)
	}
	payload := store.SummaryPayload{Goals: []store.SummaryStatement{{Text: "summary should disappear", Sources: []string{"user:turn:compiled:1:record:0:offset:0"}}}}
	v := &store.Summary{TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, SessionID: "compiled", Version: 1, Covered: snap.Upper, SchemaVersion: store.SummarySchemaVersion, Payload: payload, Sources: store.SummarySources(payload)}
	state.summaryResult = summary.Result{Snapshot: snap, Summary: v}
	state.recovery = store.ToolRecovery{TerminalSequences: map[uint64]uint64{turn: snap.Upper}, Batches: []store.ToolBatch{{
		ID: "covered-recovery", TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, SessionID: "compiled", TurnID: turn, Sequence: 1, Version: 1, SchemaVersion: "v1",
		Calls: []store.ToolCall{{ID: "uncertain-call", Name: "lookup", Input: json.RawMessage(`{}`), Status: "pending"}},
	}}}
	ra.summaryStore = &disappearingSummaryStore{SummaryStore: backend, value: v}
	compiled, err := ra.compile(ctx, state, &interactionTrace{Session: "compiled", InteractionID: "interaction"}, false)
	if err != nil {
		t.Fatal(err)
	}
	text := stringMustJSON(compiled.Messages)
	if strings.Contains(text, "summary should disappear") || !strings.Contains(text, "recent canonical question") || !strings.Contains(text, "Recovered tool batch") || !strings.Contains(text, "automatically replay") || compiled.Report.InputTokens+compiled.OutputTokens+compiled.SafetyMargin > compiled.ContextTokens {
		t.Fatalf("invalid final fallback: %s", text)
	}
	if len(provider.requests) != 0 {
		t.Fatal("compilation dispatched a model")
	}
}

func stringMustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
