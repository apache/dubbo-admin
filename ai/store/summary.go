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

package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"dubbo-admin-ai/security"
)

const SummarySchemaVersion = "summary-v1"
const SummaryArtifactKind = "conversation_summary"
const MaxSummaryBytes = 64 << 10
const MaxSummaryFragmentBytes = 64 << 10

// Terminal journal metadata is independently count-bounded, including failed
// attempts that no longer consume the completed-Turn storage quota.
const MaxHistoryEntriesPerSession = 10000

var ErrSummaryNotFound = errors.New("summary not found")

func SummaryArtifactID(session string) string {
	digest := sha256.Sum256([]byte(session))
	return "summary:" + hex.EncodeToString(digest[:24])
}

// HistorySnapshot fixes a terminal-history upper bound for one active attempt.
// Sequence values are opaque keyset positions, never Turn IDs or offsets.
type HistorySnapshot struct {
	SessionID  string
	ActiveTurn uint64
	Upper      uint64
	Target     uint64
	Count      int
}

type HistoryEntry struct {
	Dependencies []MemoryDependency
	Sequence     uint64
	SessionID    string
	TurnID       uint64
	Status       string
}

// SummaryCursor advances only after a whole terminal Turn has been processed.
// Record/Offset describe resumable input within the next Turn, not coverage.
type SummaryCursor struct {
	After  uint64
	Record int
	Offset int
}

type SummaryStatement struct {
	Text    string   `json:"text"`
	Sources []string `json:"sources"`
	// Confirmed facts must distinguish explicit user assertions from tool evidence.
	Basis string `json:"basis,omitempty"`
}

type SummaryPayload struct {
	Goals                 []SummaryStatement `json:"goals"`
	ConfirmedFacts        []SummaryStatement `json:"confirmed_facts"`
	Evidence              []SummaryStatement `json:"evidence"`
	CompletedActions      []SummaryStatement `json:"completed_actions"`
	UnresolvedWork        []SummaryStatement `json:"unresolved_work"`
	SupersededConclusions []SummaryStatement `json:"superseded_conclusions"`
}

// MemoryDependency is server-owned provenance propagated through answers,
// summaries and intermediate fragments. PR 3 resolves these versions at read/write.
type MemoryDependency struct {
	ID         string
	Version    uint64
	Generation uint64
}

type SummaryProgress struct {
	Cursor  SummaryCursor
	Payload SummaryPayload
	Sources []string
}

type Summary struct {
	SourceUpper                           uint64
	Target                                uint64
	TenantID, OwnerID, SessionID          string
	Version                               uint64
	SchemaVersion, TemplateVersion, Model string
	Covered                               uint64
	Payload                               SummaryPayload
	Sources                               []string
	Dependencies                          []MemoryDependency
	Progress                              *SummaryProgress
}

type SummaryReadRequest struct {
	Snapshot HistorySnapshot
	Cursor   SummaryCursor
	MaxBytes int
}

type SummaryBatch struct {
	Entry HistoryEntry
	Text  string
	// Sources retains independent authority for each record in a packed Turn.
	Sources []string
	// Source is the first citation, retained for single-fragment consumers.
	Source string
	Digest string
	Next   SummaryCursor
	Done   bool
}

type SummaryCommit struct {
	Read            SummaryReadRequest
	Digest          string
	ExpectedVersion uint64
	Value           Summary
}

type SummaryStore interface {
	OpenHistorySnapshot(context.Context, security.AccessContext, string, uint64, int) (HistorySnapshot, error)
	ReadSummaryBatch(context.Context, security.AccessContext, SummaryReadRequest) (SummaryBatch, error)
	GetSummary(context.Context, security.AccessContext, string) (*Summary, error)
	CommitSummary(context.Context, security.AccessContext, SummaryCommit) error
}

type SummaryMaintenanceStore interface {
	DeleteSummary(context.Context, security.AccessContext, DerivedArtifact) error
}

// BuildSummaryBatch packs records from the next Turn into one byte-bounded
// evidence batch. Only oversized Turns use Record/Offset checkpoints. The reader
// must bound payload reads in storage before materialization.
func BuildSummaryBatch(req SummaryReadRequest, entry HistoryEntry, records int, read func(int, int, int) (string, int, error), classifications ...func(int) string) (SummaryBatch, error) {
	var b SummaryBatch
	if req.MaxBytes < 256 || req.MaxBytes > MaxSummaryFragmentBytes || req.Cursor.Record < 0 || req.Cursor.Offset < 0 || entry.Sequence <= req.Cursor.After || entry.Sequence > req.Snapshot.Upper {
		return b, ErrContextConflict
	}
	b.Entry = entry
	b.Entry.Dependencies = append([]MemoryDependency(nil), entry.Dependencies...)
	b.Next = req.Cursor
	b.Text = fmt.Sprintf("Turn %d status=%s; uncertain outcomes are not success.\n", entry.TurnID, entry.Status)
	sourceAt := func(class string, record, offset int) string {
		return fmt.Sprintf("%s:turn:%s:%d:record:%d:offset:%d", class, entry.SessionID, entry.TurnID, record, offset)
	}
	if records == 0 {
		if req.Cursor.Record != 0 || req.Cursor.Offset != 0 {
			return b, ErrContextConflict
		}
		b.Next = SummaryCursor{After: entry.Sequence}
		b.Sources = []string{sourceAt("unknown", 0, 0)}
	} else {
		if req.Cursor.Record >= records {
			return b, ErrContextConflict
		}
		for b.Next.Record < records {
			record, offset := b.Next.Record, b.Next.Offset
			// Reserve the longest server classification before invoking the reader,
			// which also determines uncertain/successful tool provenance.
			room := req.MaxBytes - len(b.Text) - len(sourceAt("tool_uncertain", record, offset)) - 4
			if room < utf8.UTFMax {
				if len(b.Sources) == 0 {
					return b, ErrResourceExhausted
				}
				break
			}
			text, total, err := read(record, offset, room)
			if err != nil {
				return b, err
			}
			if offset >= total || len(text) == 0 || len(text) > room {
				return b, ErrContextConflict
			}
			if offset+len(text) < total {
				for len(text) > 0 && !utf8.ValidString(text) {
					text = text[:len(text)-1]
				}
			}
			if len(text) == 0 || !utf8.ValidString(text) {
				return b, ErrContextConflict
			}
			class := "unknown"
			if len(classifications) > 0 {
				class = classifications[0](record)
			}
			source := sourceAt(class, record, offset)
			b.Sources = append(b.Sources, source)
			b.Text += "\n[" + source + "]\n" + text
			b.Next.Offset += len(text)
			if b.Next.Offset < total {
				break
			}
			b.Next.Offset = 0
			b.Next.Record++
		}
		if b.Next.Record == records {
			b.Next = SummaryCursor{After: entry.Sequence}
		}
	}
	b.Done = b.Next.After == entry.Sequence
	b.Source = b.Sources[0]
	encoded, _ := json.Marshal(struct {
		Entry   HistoryEntry
		Cursor  SummaryCursor
		Text    string
		Next    SummaryCursor
		Sources []string
	}{entry, req.Cursor, b.Text, b.Next, b.Sources})
	h := sha256.Sum256(encoded)
	b.Digest = hex.EncodeToString(h[:])
	return b, nil
}

func ValidateSummaryPayload(p SummaryPayload, sources []string) error {
	encoded, err := json.Marshal(p)
	if err != nil || len(encoded) > MaxSummaryBytes {
		return ErrResourceExhausted
	}
	if _, found := security.DefaultSanitizer().ContainsSecret(string(encoded)); found {
		return security.ErrSensitiveDerivedData
	}
	if summarySensitiveKey.Match(encoded) {
		return security.ErrSensitiveDerivedData
	}
	allowed := make(map[string]bool, len(sources))
	for _, s := range sources {
		allowed[s] = true
	}
	for _, group := range [][]SummaryStatement{p.Goals, p.ConfirmedFacts, p.Evidence, p.CompletedActions, p.UnresolvedWork, p.SupersededConclusions} {
		if len(group) > 32 {
			return ErrResourceExhausted
		}
		for _, item := range group {
			if strings.TrimSpace(item.Text) == "" || len(item.Text) > 4096 || len(item.Sources) == 0 || len(item.Sources) > 16 {
				return ErrContextConflict
			}
			for _, ref := range item.Sources {
				if !allowed[ref] {
					return ErrContextConflict
				}
			}
		}
	}
	for _, item := range p.ConfirmedFacts {
		if item.Basis != "user_assertion" && item.Basis != "tool_evidence" {
			return ErrContextConflict
		}
		for _, ref := range item.Sources {
			if item.Basis == "user_assertion" && !strings.HasPrefix(ref, "user:turn:") {
				return ErrContextConflict
			}
			if item.Basis == "tool_evidence" && !strings.HasPrefix(ref, "tool_success:turn:") {
				return ErrContextConflict
			}
		}
	}
	for _, item := range p.CompletedActions {
		for _, ref := range item.Sources {
			if !strings.HasPrefix(ref, "user:turn:") && !strings.HasPrefix(ref, "tool_success:turn:") {
				return ErrContextConflict
			}
		}
	}
	return nil
}

func DecodeSummaryPayload(text string, sources []string) (SummaryPayload, error) {
	var p SummaryPayload
	if len(text) > MaxSummaryBytes {
		return p, ErrResourceExhausted
	}
	var shape map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &shape); err != nil {
		return p, err
	}
	if len(shape) != 6 {
		return p, ErrContextConflict
	}
	for _, key := range []string{"goals", "confirmed_facts", "evidence", "completed_actions", "unresolved_work", "superseded_conclusions"} {
		raw, ok := shape[key]
		if !ok || len(raw) == 0 || (raw[0] != '[' && string(raw) != "null") {
			return p, ErrContextConflict
		}
	}
	d := json.NewDecoder(strings.NewReader(text))
	d.DisallowUnknownFields()
	if err := d.Decode(&p); err != nil {
		return p, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return p, ErrContextConflict
	}
	return p, ValidateSummaryPayload(p, sources)
}

func SummaryCursorFor(v *Summary) SummaryCursor {
	if v == nil {
		return SummaryCursor{}
	}
	if v.Progress != nil {
		return v.Progress.Cursor
	}
	return SummaryCursor{After: v.Covered}
}

func ValidateSummaryCommit(a security.AccessContext, c SummaryCommit, old *Summary, batch SummaryBatch) error {
	v := c.Value
	version := uint64(0)
	if old != nil {
		version = old.Version
	}
	if c.ExpectedVersion != version || v.Version != version+1 || v.SchemaVersion != SummarySchemaVersion || v.TemplateVersion == "" || len(v.TemplateVersion) > 128 || v.Model == "" || len(v.Model) > 256 || v.TenantID != a.Subject.TenantID || v.OwnerID != a.Subject.UserID || v.SessionID != c.Read.Snapshot.SessionID || c.Digest != batch.Digest || c.Read.Cursor != SummaryCursorFor(old) || v.SourceUpper != c.Read.Snapshot.Upper || v.Target != c.Read.Snapshot.Target {
		return ErrContextConflict
	}
	// References are derived from validated output, keeping provenance bounded.
	allowed := append([]string(nil), batch.Sources...)
	if old != nil {
		allowed = append(allowed, old.Sources...)
		if old.Progress != nil {
			allowed = append(allowed, old.Progress.Sources...)
		}
	}
	if batch.Done {
		if v.Progress != nil || v.Covered != batch.Next.After {
			return ErrContextConflict
		}
		if err := ValidateSummaryPayload(v.Payload, allowed); err != nil {
			return err
		}
		if !reflect.DeepEqual(v.Sources, SummarySources(v.Payload)) {
			return ErrContextConflict
		}
	} else {
		covered := uint64(0)
		if old != nil {
			covered = old.Covered
		}
		if v.Covered != covered || v.Progress == nil || v.Progress.Cursor != batch.Next {
			return ErrContextConflict
		}
		if err := ValidateSummaryPayload(v.Progress.Payload, allowed); err != nil {
			return err
		}
		if !reflect.DeepEqual(v.Progress.Sources, SummarySources(v.Progress.Payload)) {
			return ErrContextConflict
		}
		if old == nil {
			if len(v.Sources) > 0 {
				return ErrContextConflict
			}
			if string(summaryJSON(v.Payload)) != string(summaryJSON(SummaryPayload{})) {
				return ErrContextConflict
			}
		} else if string(summaryJSON(v.Payload)) != string(summaryJSON(old.Payload)) {
			return ErrContextConflict
		}
		if old != nil && !reflect.DeepEqual(v.Sources, old.Sources) {
			return ErrContextConflict
		}
	}
	var oldDeps []MemoryDependency
	if old != nil {
		oldDeps = old.Dependencies
	}
	expectedDeps := MergeMemoryDependencies(oldDeps, batch.Entry.Dependencies)
	if len(expectedDeps) > 256 {
		return ErrResourceExhausted
	}
	if !reflect.DeepEqual(v.Dependencies, expectedDeps) {
		return ErrContextConflict
	}
	if len(summaryJSON(v)) > MaxSummaryBytes*2 {
		return ErrResourceExhausted
	}
	return nil
}

func SummarySources(p SummaryPayload) []string {
	var out []string
	seen := map[string]bool{}
	for _, group := range [][]SummaryStatement{p.Goals, p.ConfirmedFacts, p.Evidence, p.CompletedActions, p.UnresolvedWork, p.SupersededConclusions} {
		for _, item := range group {
			for _, ref := range item.Sources {
				if !seen[ref] {
					seen[ref] = true
					out = append(out, ref)
				}
			}
		}
	}
	return out
}

func summaryJSON(v any) []byte { b, _ := json.Marshal(v); return b }

func MergeMemoryDependencies(groups ...[]MemoryDependency) []MemoryDependency {
	var out []MemoryDependency
	seen := map[MemoryDependency]bool{}
	for _, group := range groups {
		for _, d := range group {
			if !seen[d] {
				seen[d] = true
				out = append(out, d)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		if a.Generation != b.Generation {
			return a.Generation < b.Generation
		}
		return a.Version < b.Version
	})
	return out
}
