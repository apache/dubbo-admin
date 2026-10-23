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

package storetest

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"dubbo-admin-ai/component/memory/summary"
	"dubbo-admin-ai/security"
	"dubbo-admin-ai/store"
	"github.com/firebase/genkit/go/ai"
)

func RunSummaryContractTests(t *testing.T, newStore func() store.BackendStore) {
	t.Helper()
	runSummaryPackedTurnContractTests(t, newStore)
	for _, scenario := range []string{"advance", "fragments", "tools", "CAS", "deleted", "delete race", "released", "ownership", "invalid", "failure recovery", "sensitive"} {
		t.Run(scenario, func(t *testing.T) {
			s := newStore()
			backend := s.(store.SummaryStore)
			ctx := context.Background()
			a := contractAccess
			if err := s.Create(ctx, a, &store.Session{ID: "summary", TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, Version: 1, Status: store.SessionStatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()}); err != nil {
				t.Fatal(err)
			}
			content := "early objective"
			if scenario == "fragments" {
				content = strings.Repeat("中文目标与未完成事项", 150)
			}
			if scenario == "sensitive" {
				content = strings.Repeat("x", 250) + " password=must-never-be-derived " + strings.Repeat("y", 1000)
			}
			first, err := s.PrepareInteraction(ctx, a, "summary", "first", "first", time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.AddHistoryToTurn(ctx, a, "summary", first.TurnID, ai.NewUserTextMessage(content)); err != nil {
				t.Fatal(err)
			}
			if scenario == "tools" {
				ledger := s.(store.CompilationStore)
				b := store.ToolBatch{ID: "summary-tools", TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, SessionID: "summary", TurnID: first.TurnID, InteractionID: first.InteractionID, Sequence: 1, Version: 1, SchemaVersion: "v1", Calls: []store.ToolCall{{ID: "call-one", Name: "lookup", Input: json.RawMessage(`{}`), Status: "pending"}, {ID: "call-two", Name: "lookup", Input: json.RawMessage(`{}`), Status: "pending"}}}
				if err := ledger.CreateToolBatch(ctx, a, b); err != nil {
					t.Fatal(err)
				}
				call := b.Calls[0]
				call.Status = "succeeded"
				call.Output = json.RawMessage(`{"ok":true}`)
				if err := ledger.CompleteToolCall(ctx, a, "summary", b.ID, 1, 0, call); err != nil {
					t.Fatal(err)
				}
				if err := s.AddHistoryToTurn(ctx, a, "summary", first.TurnID, ai.NewModelTextMessage("must follow tool evidence")); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.CompleteInteraction(ctx, a, "summary", first.InteractionID, first.TurnID, "done"); err != nil {
				t.Fatal(err)
			}
			// An empty failed attempt must remain an explicit position, despite removed messages.
			failed, err := s.PrepareInteraction(ctx, a, "summary", "failed", "failed", time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.FailInteraction(ctx, a, "summary", failed.InteractionID, failed.TurnID); err != nil {
				t.Fatal(err)
			}
			second, err := s.PrepareInteraction(ctx, a, "summary", "second", "second", time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.AddHistoryToTurn(ctx, a, "summary", second.TurnID, ai.NewUserTextMessage("recent")); err != nil {
				t.Fatal(err)
			}
			if err := s.CompleteInteraction(ctx, a, "summary", second.InteractionID, second.TurnID, "done"); err != nil {
				t.Fatal(err)
			}
			active, err := s.PrepareInteraction(ctx, a, "summary", "active", "active", time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			snap, err := backend.OpenHistorySnapshot(ctx, a, "summary", active.TurnID, 0)
			if err != nil || snap.Count != 3 {
				t.Fatalf("snapshot %+v: %v", snap, err)
			}
			read := store.SummaryReadRequest{Snapshot: snap, MaxBytes: 256}
			batch, err := backend.ReadSummaryBatch(ctx, a, read)
			if err != nil {
				t.Fatal(err)
			}
			if !utf8.ValidString(batch.Text) || len(batch.Text) > 512 || batch.Entry.TurnID != first.TurnID {
				t.Fatalf("invalid bounded fragment: %+v", batch)
			}
			if scenario == "tools" {
				next, err := backend.ReadSummaryBatch(ctx, a, store.SummaryReadRequest{Snapshot: snap, Cursor: batch.Next, MaxBytes: store.MaxSummaryFragmentBytes})
				if err != nil {
					t.Fatal(err)
				}
				if !strings.HasPrefix(next.Source, "tool_uncertain:") || !strings.Contains(next.Text, `"Status":"pending"`) || !strings.Contains(next.Text, `"Status":"succeeded"`) {
					t.Fatalf("canonical uncertainty/order lost: %+v", next)
				}
			}
			payload := store.SummaryPayload{Goals: []store.SummaryStatement{{Text: "retain early objective", Sources: []string{batch.Source}}}}
			value := store.Summary{TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, SessionID: "summary", Version: 1, SourceUpper: snap.Upper, Target: snap.Target, SchemaVersion: store.SummarySchemaVersion, TemplateVersion: "test-v1", Model: "test"}
			if batch.Done {
				value.Covered = batch.Next.After
				value.Payload = payload
				value.Sources = store.SummarySources(payload)
			} else {
				value.Progress = &store.SummaryProgress{Cursor: batch.Next, Payload: payload, Sources: store.SummarySources(payload)}
			}
			commit := store.SummaryCommit{Read: read, Digest: batch.Digest, Value: value}
			switch scenario {
			case "delete race":
				var wg sync.WaitGroup
				var deleteErr, commitErr error
				wg.Add(2)
				go func() { defer wg.Done(); deleteErr = s.Delete(ctx, a, "summary") }()
				go func() { defer wg.Done(); commitErr = backend.CommitSummary(ctx, a, commit) }()
				wg.Wait()
				if deleteErr != nil {
					t.Fatal(deleteErr)
				}
				if _, err := backend.GetSummary(ctx, a, "summary"); err == nil {
					t.Fatal("summary readable after delete race")
				}
				artifacts, err := s.ListDerivedArtifactsForCleanup(ctx, security.SystemAccessContext(), time.Now(), 10)
				if err != nil {
					t.Fatal(err)
				}
				if commitErr == nil && len(artifacts) != 1 {
					t.Fatal("winning commit lost cleanup registration")
				}
				if commitErr != nil && len(artifacts) != 0 {
					t.Fatal("losing commit left an orphan artifact")
				}
			case "CAS":
				var wg sync.WaitGroup
				errs := make(chan error, 2)
				for range 2 {
					wg.Add(1)
					go func() { defer wg.Done(); errs <- backend.CommitSummary(ctx, a, commit) }()
				}
				wg.Wait()
				close(errs)
				wins := 0
				for err := range errs {
					if err == nil {
						wins++
					} else if !errors.Is(err, store.ErrContextConflict) {
						t.Fatal(err)
					}
				}
				if wins != 1 {
					t.Fatalf("CAS winners %d", wins)
				}
			case "deleted":
				if err := s.Delete(ctx, a, "summary"); err != nil {
					t.Fatal(err)
				}
				if err := backend.CommitSummary(ctx, a, commit); err == nil {
					t.Fatal("deleted source resurrected")
				}
				if _, err := backend.GetSummary(ctx, a, "summary"); err == nil {
					t.Fatal("deleted source readable")
				}
			case "released":
				if err := s.FailInteraction(ctx, a, "summary", active.InteractionID, active.TurnID); err != nil {
					t.Fatal(err)
				}
				if err := backend.CommitSummary(ctx, a, commit); err == nil {
					t.Fatal("released worker committed")
				}
			case "ownership":
				for _, other := range []security.AccessContext{security.MustAccessContext(security.Subject{TenantID: a.Subject.TenantID, UserID: "other", AuthMethod: "test"}), security.MustAccessContext(security.Subject{TenantID: "other", UserID: a.Subject.UserID, AuthMethod: "test"})} {
					if _, err := backend.OpenHistorySnapshot(ctx, other, "summary", active.TurnID, 0); err == nil {
						t.Fatal("cross-owner snapshot")
					}
					if _, err := backend.ReadSummaryBatch(ctx, other, read); err == nil {
						t.Fatal("cross-owner evidence")
					}
					if err := backend.CommitSummary(ctx, other, commit); err == nil {
						t.Fatal("cross-owner summary")
					}
				}
			case "invalid":
				bad := commit
				bad.Value.Covered = snap.Upper + 1
				if err := backend.CommitSummary(ctx, a, bad); err == nil {
					t.Fatal("forged coverage accepted")
				}
				bad = commit
				bad.Digest = "wrong"
				if err := backend.CommitSummary(ctx, a, bad); err == nil {
					t.Fatal("forged source version accepted")
				}
				bad = commit
				bad.Value.Sources = []string{"invented"}
				if err := backend.CommitSummary(ctx, a, bad); err == nil {
					t.Fatal("forged source reference accepted")
				}
				if _, err := backend.GetSummary(ctx, a, "summary"); !errors.Is(err, store.ErrSummaryNotFound) {
					t.Fatalf("partial invalid commit: %v", err)
				}
			case "sensitive":
				if strings.Contains(batch.Text, "must-never-be-derived") || !strings.Contains(batch.Text, "Sensitive canonical record excluded") {
					t.Fatalf("sensitive record not excluded: %s", batch.Text)
				}
			default:
				if err := backend.CommitSummary(ctx, a, commit); err != nil {
					t.Fatal(err)
				}
				v, err := backend.GetSummary(ctx, a, "summary")
				if err != nil {
					t.Fatal(err)
				}
				if scenario == "fragments" && (v.Covered != 0 || v.Progress == nil) {
					t.Fatal("partial Turn claimed complete coverage")
				}
				cfg := summary.Config{Enabled: true, Model: "test", FragmentBytes: 256, MaxCalls: 2, RecoveryMaxCalls: 8, TriggerHeadroomTurns: 1}
				calls := 0
				fail := scenario == "failure recovery"
				generator := summary.GenerateFunc(func(ctx context.Context, r summary.GenerateRequest) (string, error) {
					calls++
					if fail {
						return "", errors.New("fixture timeout")
					}
					if !utf8.ValidString(r.Batch.Text) {
						t.Fatal("fragment split UTF-8")
					}
					p := r.Previous
					if len(p.Goals) == 0 {
						p.Goals = []store.SummaryStatement{{Text: "retain early objective", Sources: []string{r.Batch.Source}}}
					}
					raw, _ := json.Marshal(p)
					return string(raw), nil
				})
				service := summary.Service{Store: backend, Config: cfg, Generator: generator}
				if fail {
					result, err := service.Advance(ctx, a, "summary", active.InteractionID, active.TurnID, 1)
					if err != nil || result.Reason == "" || result.Summary.Version != 1 {
						t.Fatalf("failed summary advanced: %+v %v", result, err)
					}
					fail = false
				}
				var result summary.Result
				for i := 0; i < 20; i++ {
					result, err = service.Advance(ctx, a, "summary", active.InteractionID, active.TurnID, 1)
					if err != nil {
						t.Fatal(err)
					}
					if result.Calls > 8 {
						t.Fatal("unbounded calls")
					}
					if !result.Backlog {
						break
					}
				}
				if result.Backlog || result.Summary.Covered != snap.Target || result.Summary.Progress != nil {
					t.Fatalf("did not catch up: %+v", result)
				}
				before := calls
				result, err = service.Advance(ctx, a, "summary", active.InteractionID, active.TurnID, 1)
				if err != nil || calls != before {
					t.Fatal("called model without new history")
				}
				// Returned payloads must not alias persisted state.
				result.Summary.Payload.Goals[0].Text = "mutated"
				again, err := backend.GetSummary(ctx, a, "summary")
				if err != nil || again.Payload.Goals[0].Text == "mutated" {
					t.Fatal("mutable summary storage")
				}
				if err := s.Delete(ctx, a, "summary"); err != nil {
					t.Fatal(err)
				}
				artifacts, err := s.ListDerivedArtifactsForCleanup(ctx, security.SystemAccessContext(), time.Now(), 10)
				if err != nil || len(artifacts) != 1 {
					t.Fatalf("missing atomic artifact: %+v %v", artifacts, err)
				}
				maintenance := s.(store.SummaryMaintenanceStore)
				if err := maintenance.DeleteSummary(ctx, a, artifacts[0]); err == nil {
					t.Fatal("business identity removed a summary")
				}
				if err := maintenance.DeleteSummary(ctx, security.SystemAccessContext(), artifacts[0]); err != nil {
					t.Fatal(err)
				}
				if err := maintenance.DeleteSummary(ctx, security.SystemAccessContext(), artifacts[0]); err != nil {
					t.Fatal("cleanup not idempotent", err)
				}
				if err := s.MarkDerivedArtifactDeleted(ctx, security.SystemAccessContext(), artifacts[0].ID); err != nil {
					t.Fatal(err)
				}
				if _, err := s.DeleteExpired(ctx, security.SystemAccessContext(), time.Now(), 10); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func runSummaryPackedTurnContractTests(t *testing.T, newStore func() store.BackendStore) {
	for _, tools := range []bool{false, true} {
		name := "packed user answer"
		if tools {
			name = "packed user tools answer"
		}
		t.Run(name, func(t *testing.T) {
			s := newStore()
			ctx, a := context.Background(), contractAccess
			if err := s.Create(ctx, a, &store.Session{ID: "packed", TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, Version: 1, Status: store.SessionStatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()}); err != nil {
				t.Fatal(err)
			}
			p, err := s.PrepareInteraction(ctx, a, "packed", "seed", "seed", time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.AddHistoryToTurn(ctx, a, "packed", p.TurnID, ai.NewUserTextMessage("find the outage")); err != nil {
				t.Fatal(err)
			}
			if tools {
				ledger := s.(store.CompilationStore)
				b := store.ToolBatch{ID: "packed-batch", TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, SessionID: "packed", TurnID: p.TurnID, InteractionID: p.InteractionID, Sequence: 1, Version: 1, SchemaVersion: "v1", Calls: []store.ToolCall{{ID: "lookup", Name: "lookup", Input: json.RawMessage(`{}`), Status: "pending"}}}
				if err := ledger.CreateToolBatch(ctx, a, b); err != nil {
					t.Fatal(err)
				}
				call := b.Calls[0]
				call.Status, call.Output = "succeeded", json.RawMessage(`{"observed":"unavailable"}`)
				if err := ledger.CompleteToolCall(ctx, a, "packed", b.ID, 1, 0, call); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.AddHistoryToTurn(ctx, a, "packed", p.TurnID, ai.NewModelTextMessage("possible cause")); err != nil {
				t.Fatal(err)
			}
			if err := s.CompleteInteraction(ctx, a, "packed", p.InteractionID, p.TurnID, "possible cause"); err != nil {
				t.Fatal(err)
			}
			active, err := s.PrepareInteraction(ctx, a, "packed", "active", "active", time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			generator := summary.GenerateFunc(func(_ context.Context, r summary.GenerateRequest) (string, error) {
				calls++
				want := 2
				if tools {
					want = 3
				}
				if !r.Batch.Done || len(r.Batch.Sources) != want || len(r.Sources) != want || len(r.Batch.Text) > 8192 {
					t.Fatalf("short Turn was fragmented or lost citations: %+v", r.Batch)
				}
				for i, ref := range r.Batch.Sources {
					class := "model:"
					if i == 0 {
						class = "user:"
					} else if tools && i == 1 {
						class = "tool_success:"
					}
					if !strings.HasPrefix(ref, class) || !strings.Contains(r.Batch.Text, "["+ref+"]") {
						t.Fatalf("packed evidence lost independent provenance: %s", ref)
					}
				}
				payload := store.SummaryPayload{Goals: []store.SummaryStatement{{Text: "find the outage", Sources: []string{r.Batch.Sources[0]}}}, Evidence: []store.SummaryStatement{{Text: "possible cause remains unconfirmed", Sources: []string{r.Batch.Sources[want-1]}}}}
				if tools {
					payload.ConfirmedFacts = []store.SummaryStatement{{Text: "observed unavailable", Sources: []string{r.Batch.Sources[1]}, Basis: "tool_evidence"}}
				}
				data, _ := json.Marshal(payload)
				return string(data), nil
			})
			result, err := (summary.Service{Store: s.(store.SummaryStore), Generator: generator, Config: summary.Config{Enabled: true, Model: "test", MaxCalls: 1, RecoveryMaxCalls: 1, TriggerHeadroomTurns: 1}}).Advance(ctx, a, "packed", active.InteractionID, active.TurnID, 1)
			if err != nil || calls != 1 || result.Backlog || result.Summary == nil || result.Summary.Progress != nil || result.Summary.Covered != result.Snapshot.Target {
				t.Fatalf("one short Turn required multiple summary calls: %+v calls=%d err=%v", result, calls, err)
			}
		})
	}
}
