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

package summary

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"dubbo-admin-ai/security"
	"dubbo-admin-ai/store"
	memorystore "dubbo-admin-ai/store/memory"
	"dubbo-admin-ai/testutils"
	"github.com/firebase/genkit/go/ai"
)

func TestSummaryRecoversWhileNewTurnsArrive(t *testing.T) {
	ctx := context.Background()
	a := testutils.TestAccess()
	backend := memorystore.NewMemoryStore(30)
	if err := backend.Create(ctx, a, testutils.TestSession("growth", time.Now())); err != nil {
		t.Fatal(err)
	}
	fail := true
	service := Service{Store: backend, Config: Config{Enabled: true, Model: "test", MaxCalls: 2, RecoveryMaxCalls: 8, TriggerHeadroomTurns: 1}, Generator: GenerateFunc(func(ctx context.Context, r GenerateRequest) (string, error) {
		if fail {
			return "", fmt.Errorf("fixture unavailable")
		}
		p := r.Previous
		if len(p.Goals) == 0 {
			p.Goals = []store.SummaryStatement{{Text: "first objective", Sources: []string{r.Batch.Source}}}
		}
		encoded, _ := json.Marshal(p)
		return string(encoded), nil
	})}
	var last Result
	for i := 0; i < 12; i++ {
		p, err := backend.PrepareInteraction(ctx, a, "growth", fmt.Sprint(i), fmt.Sprint(i), time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		if err := backend.AddHistoryToTurn(ctx, a, "growth", p.TurnID, ai.NewUserTextMessage("new question")); err != nil {
			t.Fatal(err)
		}
		fail = i < 6
		last, err = service.Advance(ctx, a, "growth", p.InteractionID, p.TurnID, 2)
		if err != nil {
			t.Fatal(err)
		}
		if fail && last.Summary != nil {
			t.Fatal("failed generation advanced coverage")
		}
		if last.Calls > 8 {
			t.Fatal("unbounded recovery calls")
		}
		if err := backend.AddHistoryToTurn(ctx, a, "growth", p.TurnID, ai.NewModelTextMessage("answer")); err != nil {
			t.Fatal(err)
		}
		if err := backend.CompleteInteraction(ctx, a, "growth", p.InteractionID, p.TurnID, "answer"); err != nil {
			t.Fatal(err)
		}
	}
	if last.Backlog || last.Summary == nil || last.Summary.Covered != last.Snapshot.Target {
		t.Fatalf("recovery did not outpace new history: %+v", last)
	}
}

func TestSummaryAdvanceAllowsOwnerAndSessionAdminToAlternate(t *testing.T) {
	for _, order := range []string{"owner-admin", "admin-owner"} {
		t.Run(order, func(t *testing.T) {
			ctx := context.Background()
			owner := testutils.TestAccess()
			admin := security.MustAccessContext(security.Subject{TenantID: owner.Subject.TenantID, UserID: "summary-admin", Scopes: []string{security.ScopeSessionAdmin}, AuthMethod: security.AuthMethodStatic})
			backend := memorystore.NewMemoryStore()
			if err := backend.Create(ctx, owner, testutils.TestSession("alternating", time.Now())); err != nil {
				t.Fatal(err)
			}
			generator := GenerateFunc(func(_ context.Context, r GenerateRequest) (string, error) {
				payload := store.SummaryPayload{Goals: []store.SummaryStatement{{Text: "retain", Sources: []string{r.Batch.Source}}}}
				data, err := json.Marshal(payload)
				return string(data), err
			})
			service := Service{Store: backend, Generator: generator, Config: Config{Enabled: true, Model: "test", FragmentBytes: 256, MaxCalls: 1, RecoveryMaxCalls: 1, TriggerHeadroomTurns: 1}}
			first, err := backend.PrepareInteraction(ctx, owner, "alternating", "first", "first", time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			if err := backend.AddHistoryToTurn(ctx, owner, "alternating", first.TurnID, ai.NewUserTextMessage("first")); err != nil {
				t.Fatal(err)
			}
			if err := backend.CompleteInteraction(ctx, owner, "alternating", first.InteractionID, first.TurnID, "done"); err != nil {
				t.Fatal(err)
			}
			active, err := backend.PrepareInteraction(ctx, owner, "alternating", "active", "active", time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			advance := func(a security.AccessContext, turn uint64) Result {
				result, err := service.Advance(ctx, a, "alternating", "active", turn, 1)
				if err != nil {
					t.Fatal(err)
				}
				return result
			}
			if order == "owner-admin" {
				if got := advance(owner, active.TurnID); got.Backlog {
					t.Fatalf("owner first left backlog: %+v", got)
				}
			} else if got := advance(admin, active.TurnID); got.Backlog {
				t.Fatalf("admin first left backlog: %+v", got)
			}
			if err := backend.CompleteInteraction(ctx, owner, "alternating", active.InteractionID, active.TurnID, "done"); err != nil {
				t.Fatal(err)
			}
			next, err := backend.PrepareInteraction(ctx, owner, "alternating", "next", "next", time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			var got Result
			if order == "owner-admin" {
				got = advance(admin, next.TurnID)
			} else {
				got = advance(owner, next.TurnID)
			}
			if got.Backlog || got.Summary == nil || got.Summary.OwnerID != owner.Subject.UserID || got.Summary.TenantID != owner.Subject.TenantID {
				t.Fatalf("alternating advance failed: %+v", got)
			}
		})
	}
}

func TestSummaryTimeoutAndMalformedOutputKeepCoverage(t *testing.T) {
	for _, scenario := range []string{"timeout", "invalid JSON", "unknown reference", "secret"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			a := testutils.TestAccess()
			backend := memorystore.NewMemoryStore()
			if err := backend.Create(ctx, a, testutils.TestSession("failure", time.Now())); err != nil {
				t.Fatal(err)
			}
			turn, err := backend.BeginTurn(ctx, a, "failure")
			if err != nil {
				t.Fatal(err)
			}
			if err := backend.AddHistoryToTurn(ctx, a, "failure", turn, ai.NewUserTextMessage("goal")); err != nil {
				t.Fatal(err)
			}
			if err := backend.NextTurnForTurn(ctx, a, "failure", turn); err != nil {
				t.Fatal(err)
			}
			active, err := backend.BeginTurn(ctx, a, "failure")
			if err != nil {
				t.Fatal(err)
			}
			service := Service{Store: backend, Config: Config{Enabled: true, Model: "test", TimeoutSeconds: 1, TriggerHeadroomTurns: 1}, Generator: GenerateFunc(func(ctx context.Context, r GenerateRequest) (string, error) {
				if scenario == "timeout" {
					<-ctx.Done()
					return "", ctx.Err()
				}
				if scenario == "invalid JSON" {
					return `{}`, nil
				}
				text := "bad fact"
				ref := "invented"
				if scenario == "secret" {
					text = `{"password":"must-not-persist"}`
					ref = r.Batch.Source
				}
				p := store.SummaryPayload{Goals: []store.SummaryStatement{{Text: text, Sources: []string{ref}}}}
				raw, _ := json.Marshal(p)
				return string(raw), nil
			})}
			start := time.Now()
			result, err := service.Advance(ctx, a, "failure", "interaction", active, 1)
			if err != nil || result.Summary != nil || !result.Backlog || !strings.HasPrefix(result.Reason, "SUMMARY_") {
				t.Fatalf("invalid failure result: %+v %v", result, err)
			}
			if time.Since(start) > 2*time.Second {
				t.Fatal("summary exceeded total time bound")
			}
		})
	}
}
