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

package gormstore_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"dubbo-admin-ai/component/memory/summary"
	"dubbo-admin-ai/store"
	"dubbo-admin-ai/testutils"
	"github.com/firebase/genkit/go/ai"
	"gorm.io/gorm"
)

func TestSummaryFragmentsSurviveRestartAndAtomicArtifactFailure(t *testing.T) {
	for _, scenario := range []string{"restart", "artifact rollback"} {
		t.Run(scenario, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "restart.db")
			s := newSQLiteStore(t, path, 20)
			ctx := context.Background()
			a := testutils.TestAccess()
			if err := s.Create(ctx, a, testutils.TestSession("summary-restart", time.Now())); err != nil {
				t.Fatal(err)
			}
			p, err := s.PrepareInteraction(ctx, a, "summary-restart", "first", "first", time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.AddHistoryToTurn(ctx, a, "summary-restart", p.TurnID, ai.NewUserTextMessage(strings.Repeat("中文目标", 500))); err != nil {
				t.Fatal(err)
			}
			if err := s.CompleteInteraction(ctx, a, "summary-restart", p.InteractionID, p.TurnID, "done"); err != nil {
				t.Fatal(err)
			}
			active, err := s.PrepareInteraction(ctx, a, "summary-restart", "active", "active", time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			generator := summary.GenerateFunc(func(_ context.Context, r summary.GenerateRequest) (string, error) {
				payload := r.Previous
				if len(payload.Goals) == 0 {
					payload.Goals = []store.SummaryStatement{{Text: "early goal", Sources: []string{r.Batch.Source}}}
				}
				data, _ := json.Marshal(payload)
				return string(data), nil
			})
			cfg := summary.Config{Enabled: true, Model: "test", FragmentBytes: 256, MaxCalls: 1, RecoveryMaxCalls: 1, TriggerHeadroomTurns: 1}
			if scenario == "artifact rollback" {
				if err := s.DB().Exec("CREATE TRIGGER reject_summary_artifact BEFORE INSERT ON ai_derived_artifacts BEGIN SELECT RAISE(ABORT, 'fixture'); END").Error; err != nil {
					t.Fatal(err)
				}
			}
			result, err := (summary.Service{Store: s, Generator: generator, Config: cfg}).Advance(ctx, a, "summary-restart", active.InteractionID, active.TurnID, 1)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "artifact rollback" {
				if result.Summary != nil || result.Reason != "SUMMARY_COMMIT_CONFLICT" {
					t.Fatalf("unexpected rollback %+v", result)
				}
				if _, err := s.GetSummary(ctx, a, "summary-restart"); !errors.Is(err, store.ErrSummaryNotFound) {
					t.Fatal("orphan summary after transaction failure", err)
				}
				var bytes int64
				if err := s.DB().Table("ai_sessions").Select("stored_bytes").Where("id = ?", "summary-restart").Scan(&bytes).Error; err != nil {
					t.Fatal(err)
				}
				// The read transaction durably inspected the canonical record before
				// generation. Only the summary transaction's reservation rolls back.
				if bytes != int64(lenMustMessage(t, strings.Repeat("中文目标", 500)))+4+store.SummaryInspectionBytes {
					t.Fatalf("capacity changed after rollback: %d", bytes)
				}
				return
			}
			if result.Summary == nil || result.Summary.Covered != 0 || result.Summary.Progress == nil {
				t.Fatalf("missing partial checkpoint: %+v", result)
			}
			cursor := result.Summary.Progress.Cursor
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			s = newSQLiteStore(t, path, 20)
			v, err := s.GetSummary(ctx, a, "summary-restart")
			if err != nil || v.Progress.Cursor != cursor {
				t.Fatalf("checkpoint lost after restart: %+v %v", v, err)
			}
			cfg.MaxCalls = 2
			cfg.RecoveryMaxCalls = 8
			for i := 0; i < 10; i++ {
				result, err = (summary.Service{Store: s, Generator: generator, Config: cfg}).Advance(ctx, a, "summary-restart", active.InteractionID, active.TurnID, 1)
				if err != nil {
					t.Fatal(err)
				}
				if !result.Backlog {
					break
				}
			}
			if result.Backlog || result.Summary.Progress != nil || result.Summary.Covered == 0 {
				t.Fatalf("restart did not recover: %+v", result)
			}
		})
	}
}

func lenMustMessage(t *testing.T, text string) int {
	t.Helper()
	b, err := json.Marshal(ai.NewUserTextMessage(text))
	if err != nil {
		t.Fatal(err)
	}
	return len(b)
}

func TestSummaryInspectionAvoidsRepeatedFullRecordReads(t *testing.T) {
	ctx, a := context.Background(), testutils.TestAccess()
	path := filepath.Join(t.TempDir(), "inspection.db")
	s := newSQLiteStore(t, path, 20)
	if err := s.Create(ctx, a, testutils.TestSession("inspection", time.Now())); err != nil {
		t.Fatal(err)
	}
	p, err := s.PrepareInteraction(ctx, a, "inspection", "seed", "seed", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Repeat("ordinary diagnostic evidence. ", 40000)[:1<<20]
	if err := s.AddHistoryToTurn(ctx, a, "inspection", p.TurnID, ai.NewUserTextMessage(text)); err != nil {
		t.Fatal(err)
	}
	if err := s.CompleteInteraction(ctx, a, "inspection", p.InteractionID, p.TurnID, "done"); err != nil {
		t.Fatal(err)
	}
	active, err := s.PrepareInteraction(ctx, a, "inspection", "active", "active", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := s.OpenHistorySnapshot(ctx, a, "inspection", active.TurnID, 0)
	if err != nil {
		t.Fatal(err)
	}
	var reads atomic.Int64
	installCounter := func() {
		t.Helper()
		if err := s.DB().Callback().Row().After("gorm:row").Register("count_summary_payload_reads", func(tx *gorm.DB) {
			if strings.Contains(strings.ToUpper(tx.Statement.SQL.String()), "SUBSTR(PAYLOAD") {
				reads.Add(1)
			}
		}); err != nil {
			t.Fatal(err)
		}
	}
	installCounter()
	read := store.SummaryReadRequest{Snapshot: snap, MaxBytes: 8192}
	first, err := s.ReadSummaryBatch(ctx, a, read)
	if err != nil || first.Done || reads.Load() < 128 || reads.Load() > 130 {
		t.Fatalf("initial inspection/read count: %d, batch=%+v err=%v", reads.Load(), first.Next, err)
	}
	// The inspection and its capacity reservation must survive process restart.
	var beforeBytes int64
	if err := s.DB().Table("ai_sessions").Select("stored_bytes").Where("id = ?", "inspection").Scan(&beforeBytes).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = newSQLiteStore(t, path, 20)
	installCounter()
	var afterBytes int64
	if err := s.DB().Table("ai_sessions").Select("stored_bytes").Where("id = ?", "inspection").Scan(&afterBytes).Error; err != nil || beforeBytes != afterBytes {
		t.Fatalf("inspection accounting changed on restart: %d -> %d, %v", beforeBytes, afterBytes, err)
	}
	before := reads.Load()
	if _, err := s.ReadSummaryBatch(ctx, a, store.SummaryReadRequest{Snapshot: snap, Cursor: first.Next, MaxBytes: 8192}); err != nil || reads.Load()-before != 1 {
		t.Fatalf("next fragment rescanned immutable record: reads=%d err=%v", reads.Load()-before, err)
	}
	service := summary.Service{Store: s, Config: summary.Config{Enabled: true, Model: "test", FragmentBytes: 8192, MaxCalls: 32, RecoveryMaxCalls: 64, TimeoutSeconds: 30, TriggerHeadroomTurns: 1}, Generator: summary.GenerateFunc(func(_ context.Context, r summary.GenerateRequest) (string, error) {
		payload := r.Previous
		if len(payload.Goals) == 0 {
			payload.Goals = []store.SummaryStatement{{Text: "original objective", Sources: []string{r.Batch.Source}}}
		}
		data, _ := json.Marshal(payload)
		return string(data), nil
	})}
	var result summary.Result
	for range 4 {
		result, err = service.Advance(ctx, a, "inspection", active.InteractionID, active.TurnID, 1)
		if err != nil {
			t.Fatal(err)
		}
		if !result.Backlog {
			break
		}
	}
	if result.Backlog || result.Summary == nil || result.Summary.Covered != snap.Target || reads.Load() > 450 {
		t.Fatalf("read/commit amplification: reads=%d result=%+v", reads.Load(), result)
	}
	t.Logf("1 MiB record, including restart and all fragment commits: %d payload reads", reads.Load())
	// A detector update requires exactly one new full scan and no second charge.
	if err := s.DB().Table("ai_messages").Where("turn_id = ?", p.TurnID).Update("summary_checked_rule", "old-rule").Error; err != nil {
		t.Fatal(err)
	}
	before = reads.Load()
	if _, err := s.ReadSummaryBatch(ctx, a, read); err != nil || reads.Load()-before < 128 || reads.Load()-before > 130 {
		t.Fatalf("old detector result was reused: reads=%d err=%v", reads.Load()-before, err)
	}
}
