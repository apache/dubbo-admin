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

package context

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"dubbo-admin-ai/store"
)

func TestSummarySelectionAndBoundedHistoryFallback(t *testing.T) {
	for _, scenario := range []string{"selected", "token pressure", "byte pressure", "invalid version", "disabled", "partial"} {
		t.Run(scenario, func(t *testing.T) {
			c, req := fixture(t)
			req.Turns[0].Sequence = 1
			baseline, err := c.Compile(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if err := c.Register(SummarySource{}, SourcePolicy{Order: 5, HistoryCoverage: true, OptionalFailure: true, Kinds: []string{"evidence"}}); err != nil {
				t.Fatal(err)
			}
			payload := store.SummaryPayload{Goals: []store.SummaryStatement{{Text: "early objective survives", Sources: []string{"user:turn:session:1:record:0:offset:0"}}}}
			req.HistorySnapshot = store.HistorySnapshot{SessionID: "session", ActiveTurn: 2, Upper: 1, Target: 1, Count: 1}
			req.Summary = &store.Summary{TenantID: "tenant", OwnerID: "owner", SessionID: "session", Version: 1, SchemaVersion: store.SummarySchemaVersion, Covered: 1, Payload: payload, Sources: store.SummarySources(payload)}
			switch scenario {
			case "token pressure":
				req.ContextTokens = baseline.Report.InputTokens + req.OutputTokens + req.SafetyMargin
			case "byte pressure":
				req.MaxBytes = len(JSON(baseline.Messages)) + len(JSON(baseline.Tools)) + 16
			case "invalid version":
				req.Summary.SchemaVersion = "invalid"
			case "disabled":
				req.Summary = nil
			case "partial":
				req.Summary.Covered = 0
				req.Summary.Progress = &store.SummaryProgress{Payload: payload}
			}
			out, err := c.Compile(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			all := string(JSON(out.Messages))
			if scenario == "selected" {
				if !strings.Contains(all, "early objective survives") || strings.Contains(all, "old question") {
					t.Fatalf("summary/history duplicated or lost: %s", all)
				}
				found := false
				for _, d := range out.Report.Decisions {
					if d.SourceID == "history" && d.ReasonCode == "SUMMARY_COVERED" {
						found = true
					}
				}
				if !found {
					t.Fatal("missing coverage decision")
				}
			} else if !reflect.DeepEqual(out.Messages, baseline.Messages) {
				t.Fatalf("summary exclusion lost fitting history: %s", all)
			}
			if out.Report.InputTokens+req.OutputTokens+req.SafetyMargin > req.ContextTokens {
				t.Fatal("fallback exceeded budget")
			}
		})
	}
}

func TestSummaryNeverPromotesAssistantOrUncertainToolEvidence(t *testing.T) {
	for _, source := range []string{"model:turn:s:1:record:0:offset:0", "tool_uncertain:turn:s:1:record:0:offset:0"} {
		p := store.SummaryPayload{ConfirmedFacts: []store.SummaryStatement{{Text: "inferred success", Sources: []string{source}, Basis: "tool_evidence"}}}
		if err := store.ValidateSummaryPayload(p, []string{source}); err == nil {
			t.Fatal("untrusted inference became a confirmed fact")
		}
	}
}

func TestSummaryDeduplicatesRecoveryOnlyAfterFinalSelection(t *testing.T) {
	for _, scenario := range []string{"selected", "token pressure", "byte pressure", "disabled"} {
		t.Run(scenario, func(t *testing.T) {
			c, req := fixture(t)
			// No complete history is loaded: recovery records cannot claim to fill
			// the bounded-history gap, even if their terminal position is known.
			req.Turns = req.Turns[1:]
			req.Turns[0].ID, req.TurnID = 10, 10
			req.Recovery.TerminalSequences = map[uint64]uint64{1: 2, 3: 4}
			for i, id := range []string{"covered", "uncovered", "legacy"} {
				req.Recovery.Batches = append(req.Recovery.Batches, store.ToolBatch{
					ID: id, TenantID: "tenant", OwnerID: "owner", SessionID: "session", TurnID: uint64(i*2 + 1), Sequence: 1, Version: 1, SchemaVersion: "v1",
					Calls: []store.ToolCall{{ID: id, Name: "lookup", Input: json.RawMessage(`{}`), Status: "pending"}},
				})
			}
			baseline, err := c.Compile(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if err := c.Register(SummarySource{}, SourcePolicy{Order: 5, HistoryCoverage: true, Kinds: []string{"evidence"}}); err != nil {
				t.Fatal(err)
			}
			payload := store.SummaryPayload{Goals: []store.SummaryStatement{{Text: "retained goal " + strings.Repeat("ordinary goal detail ", 130), Sources: []string{"user:turn:session:1:record:0:offset:0"}}}}
			req.Summary = &store.Summary{TenantID: "tenant", OwnerID: "owner", SessionID: "session", Version: 1, SchemaVersion: store.SummarySchemaVersion, Covered: 2, Payload: payload, Sources: store.SummarySources(payload)}
			req.HistorySnapshot = store.HistorySnapshot{SessionID: "session", ActiveTurn: 10, Upper: 4, Target: 4}
			switch scenario {
			case "token pressure":
				req.ContextTokens = baseline.Report.InputTokens + req.OutputTokens + req.SafetyMargin
			case "byte pressure":
				req.MaxBytes = len(JSON(baseline.Messages)) + len(JSON(baseline.Tools)) + 16
			case "disabled":
				req.Summary = nil
			}
			out, err := c.Compile(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if scenario != "selected" {
				if !reflect.DeepEqual(out.Messages, baseline.Messages) {
					t.Fatal("summary fallback lost recovery details")
				}
				return
			}
			text := string(JSON(out.Messages))
			if strings.Contains(text, `\"ID\":\"covered\"`) || !strings.Contains(text, `\"ID\":\"uncovered\"`) || !strings.Contains(text, `\"ID\":\"legacy\"`) || !strings.Contains(text, "automatically replay") {
				t.Fatalf("invalid recovery deduplication: %s", text)
			}
			if len(out.Report.HistoryOmissions) != 1 || out.Report.HistoryOmissions[0].After != 2 || out.Report.HistoryOmissions[0].Through != 4 {
				t.Fatalf("recovery changed complete-history gap accounting: %+v", out.Report.HistoryOmissions)
			}
		})
	}
}
