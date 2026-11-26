/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
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
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"dubbo-admin-ai/security"
	"github.com/google/uuid"
)

var (
	ErrMemoryInvalid    = errors.New("MEMORY_INVALID")
	ErrMemoryNotFound   = errors.New("MEMORY_NOT_FOUND")
	ErrMemoryConflict   = errors.New("MEMORY_VERSION_CONFLICT")
	ErrMemoryDisabled   = errors.New("MEMORY_DISABLED")
	ErrMemoryDependency = errors.New("MEMORY_DEPENDENCY_INVALID")
)

const MemoryArtifactKind = "explicit-memory"

type MemoryScopeType string

const MemoryScopeUser MemoryScopeType = "user"

// MatchContext contains applicability hints, never resource authorization.
// ProjectRef must be resolved by a server-owned validator before collection.
type MatchContext struct {
	ProjectRef string `json:"projectRef,omitempty"`
	TaskType   string `json:"taskType,omitempty"`
}

func NormalizeMatchContext(m MatchContext) (MatchContext, error) {
	m.ProjectRef = strings.TrimSpace(m.ProjectRef)
	m.TaskType = strings.TrimSpace(m.TaskType)
	if len(m.ProjectRef) > 128 || strings.ContainsAny(m.ProjectRef, "\r\n\x00") {
		return m, ErrMemoryInvalid
	}
	switch m.TaskType {
	case "", "diagnosis", "configuration", "inspection":
	default:
		return m, ErrMemoryInvalid
	}
	return m, nil
}

type MemoryPolicy struct {
	MaxRecords    int `json:"max_records" yaml:"max_records"`
	MaxValueBytes int `json:"max_value_bytes" yaml:"max_value_bytes"`
	MaxUserBytes  int `json:"max_user_bytes" yaml:"max_user_bytes"`
	MaxTTLSeconds int `json:"max_ttl_seconds" yaml:"max_ttl_seconds"`
}

func (p MemoryPolicy) WithDefaults() MemoryPolicy {
	if p.MaxRecords == 0 {
		p.MaxRecords = 128
	}
	if p.MaxValueBytes == 0 {
		p.MaxValueBytes = 4096
	}
	if p.MaxUserBytes == 0 {
		p.MaxUserBytes = 512 << 10
	}
	if p.MaxTTLSeconds == 0 {
		p.MaxTTLSeconds = 365 * 24 * 3600
	}
	return p
}
func (p MemoryPolicy) Validate() error {
	p = p.WithDefaults()
	if p.MaxRecords < 1 || p.MaxRecords > 256 || p.MaxValueBytes < 1 || p.MaxValueBytes > 8192 || p.MaxUserBytes < p.MaxValueBytes || p.MaxUserBytes > 2<<20 || p.MaxTTLSeconds < 1 || p.MaxTTLSeconds > 5*365*24*3600 {
		return ErrMemoryInvalid
	}
	return nil
}

type ExplicitMemory struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	OwnerID         string          `json:"owner_id"`
	ScopeType       MemoryScopeType `json:"scope_type"`
	ScopeID         string          `json:"scope_id"`
	MemoryType      string          `json:"memory_type"`
	FactKey         string          `json:"fact_key"`
	KeyVersion      string          `json:"key_version"`
	CanonicalKey    string          `json:"canonical_key"`
	Value           string          `json:"value"`
	Applicability   MatchContext    `json:"applicability"`
	Status          string          `json:"status"`
	Version         uint64          `json:"version"`
	Generation      uint64          `json:"generation"`
	SourceSessionID string          `json:"source_session_id,omitempty"`
	Provenance      string          `json:"provenance"`
	Supersedes      uint64          `json:"supersedes,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty"`
}

type MemorySettings struct {
	Generation   uint64 `json:"generation"`
	StateVersion uint64 `json:"state_version"`
	Enabled      bool   `json:"enabled"`
}

// MemoryLedger is bounded by MemoryPolicy. Backends serialize all operations on
// this user aggregate, including source checks and relationship registration.
// Deleted IDs are never reused; updates cannot implicitly create records.
type MemoryLedger struct {
	TenantID string `json:"tenant_id"`
	OwnerID  string `json:"owner_id"`
	MemorySettings
	Records []ExplicitMemory `json:"records"`
}

func NewMemoryLedger() MemoryLedger {
	return MemoryLedger{MemorySettings: MemorySettings{Generation: 1, StateVersion: 1, Enabled: true}, Records: []ExplicitMemory{}}
}
func MemoryOwnerKey(tenant, owner string) string {
	b, _ := json.Marshal([]string{tenant, owner})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// MemoryContentBytes counts only record payloads for tenant content quotas.
// Per-user control state (generation, state version, enabled) is deliberately
// excluded so disabling remains available even when the tenant is full.
func MemoryContentBytes(l MemoryLedger) int64 {
	if len(l.Records) == 0 {
		return 0
	}
	b, _ := json.Marshal(l.Records)
	return int64(len(b))
}

type MemoryWrite struct {
	ID              string          `json:"-"`
	Generation      uint64          `json:"generation"`
	StateVersion    uint64          `json:"state_version"`
	ExpectedVersion uint64          `json:"version,omitempty"`
	ScopeType       MemoryScopeType `json:"scope_type"`
	MemoryType      string          `json:"memory_type"`
	FactKey         string          `json:"fact_key"`
	Value           string          `json:"value"`
	Applicability   MatchContext    `json:"applicability"`
	SourceSessionID string          `json:"source_session_id,omitempty"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty"`
}

type MemoryMutation struct {
	Generation   uint64 `json:"generation"`
	StateVersion uint64 `json:"state_version"`
	Version      uint64 `json:"version,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

type MemoryPage struct {
	MemorySettings
	Records    []ExplicitMemory `json:"records"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

type ExplicitMemoryStore interface {
	GetMemorySettings(context.Context, security.AccessContext) (MemorySettings, error)
	ListMemories(context.Context, security.AccessContext, string, int) (MemoryPage, error)
	SaveMemory(context.Context, security.AccessContext, MemoryWrite) (*ExplicitMemory, error)
	DeleteMemory(context.Context, security.AccessContext, string, MemoryMutation) error
	ChangeMemorySettings(context.Context, security.AccessContext, MemoryMutation, bool) (MemorySettings, error)
	RecallMemories(context.Context, security.AccessContext, MatchContext, int, int) ([]ExplicitMemory, MemorySettings, error)
	ValidateMemoryDependencies(context.Context, security.AccessContext, string, []MemoryDependency) error
	RecordTurnMemoryDependencies(context.Context, security.AccessContext, string, uint64, []MemoryDependency) error
}
type ExplicitMemoryMaintenanceStore interface {
	DeleteSourceMemory(context.Context, security.AccessContext, DerivedArtifact) error
}

func (l MemoryLedger) Check(generation, stateVersion uint64, writing bool) error {
	if l.Generation != generation || l.StateVersion != stateVersion {
		return ErrMemoryConflict
	}
	if writing && !l.Enabled {
		return ErrMemoryDisabled
	}
	return nil
}

// NormalizeMemoryWrite supplies the only identity algorithm used by both stores.
func NormalizeMemoryWrite(w MemoryWrite, p MemoryPolicy, now time.Time) (MemoryWrite, string, error) {
	p = p.WithDefaults()
	var err error
	w.Applicability, err = NormalizeMatchContext(w.Applicability)
	if err != nil {
		return w, "", err
	}
	w.Value = strings.TrimSpace(w.Value)
	if w.ScopeType != MemoryScopeUser || w.Value == "" || !utf8.ValidString(w.Value) || len(w.Value) > p.MaxValueBytes || len(w.SourceSessionID) > 64 {
		return w, "", ErrMemoryInvalid
	}
	switch w.MemoryType {
	case "preference":
		if w.Applicability != (MatchContext{}) {
			return w, "", ErrMemoryInvalid
		}
		switch w.FactKey {
		case "response.language", "response.style":
		default:
			return w, "", ErrMemoryInvalid
		}
	case "task_context":
		if w.FactKey != "project.background" || w.Applicability.ProjectRef == "" {
			return w, "", ErrMemoryInvalid
		}
	case "diagnostic_constraint":
		if w.FactKey != "diagnostic.priority" && w.FactKey != "diagnostic.read_only" {
			return w, "", ErrMemoryInvalid
		}
		if w.Applicability.TaskType != "diagnosis" {
			return w, "", ErrMemoryInvalid
		}
		if w.FactKey == "diagnostic.read_only" && w.Value != "true" {
			return w, "", ErrMemoryInvalid
		}
	default:
		return w, "", ErrMemoryInvalid
	}
	if w.ExpiresAt != nil && (!w.ExpiresAt.After(now) || w.ExpiresAt.After(now.Add(time.Duration(p.MaxTTLSeconds)*time.Second))) {
		return w, "", ErrMemoryInvalid
	}
	if clean := security.DefaultSanitizer().SanitizeText(w.Value); len(clean.Findings) > 0 {
		return w, "", ErrMemoryInvalid
	}
	// Structured, explicit facts remain untrusted model data. Reject common role/
	// policy override syntax; no text classifier is treated as authorization.
	lower := strings.ToLower(w.Value)
	for _, v := range []string{"ignore previous", "ignore all", "system prompt", "<|system", "[system]", "忽略之前", "忽略所有", "覆盖系统", "绕过权限"} {
		if strings.Contains(lower, v) {
			return w, "", ErrMemoryInvalid
		}
	}
	b, _ := json.Marshal(struct {
		Type          string       `json:"memory_type"`
		Applicability MatchContext `json:"applicability"`
		FactKey       string       `json:"fact_key"`
	}{w.MemoryType, w.Applicability, w.FactKey})
	h := sha256.Sum256(b)
	return w, "v1:" + hex.EncodeToString(h[:]), nil
}

func (l *MemoryLedger) Save(a security.AccessContext, w MemoryWrite, p MemoryPolicy, now time.Time) (*ExplicitMemory, error) {
	if err := l.Check(w.Generation, w.StateVersion, true); err != nil {
		return nil, err
	}
	w, key, err := NormalizeMemoryWrite(w, p, now)
	if err != nil {
		return nil, err
	}
	index := -1
	target := -1
	for i, r := range l.Records {
		if r.ID == w.ID {
			index = i
		}
		if r.CanonicalKey == key {
			target = i
		}
	}
	if w.ID != "" && (index < 0 || l.Records[index].Version != w.ExpectedVersion) {
		return nil, ErrMemoryConflict
	}
	if w.ID == "" && w.ExpectedVersion != 0 {
		return nil, ErrMemoryInvalid
	}
	if target >= 0 && target != index {
		r := l.Records[target]
		if w.ID == "" && r.Value == w.Value && r.SourceSessionID == w.SourceSessionID && sameMemoryTime(r.ExpiresAt, w.ExpiresAt) && r.MemoryType == w.MemoryType && r.FactKey == w.FactKey && r.Applicability == w.Applicability {
			return &r, nil
		}
		return nil, ErrMemoryConflict
	}
	r := ExplicitMemory{ID: uuid.NewString(), TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, ScopeType: "user", ScopeID: a.Subject.UserID, MemoryType: w.MemoryType, FactKey: w.FactKey, KeyVersion: "v1", CanonicalKey: key, Value: w.Value, Applicability: w.Applicability, Status: "confirmed", Version: 1, Generation: l.Generation, SourceSessionID: w.SourceSessionID, Provenance: "explicit-user-confirmation", CreatedAt: now, UpdatedAt: now, ExpiresAt: w.ExpiresAt}
	if index >= 0 {
		old := l.Records[index]
		if old.SourceSessionID != w.SourceSessionID {
			return nil, ErrMemoryInvalid
		}
		if old.CanonicalKey == key && old.Value == r.Value && old.SourceSessionID == r.SourceSessionID && sameMemoryTime(old.ExpiresAt, r.ExpiresAt) {
			return &old, nil
		}
		r.ID = old.ID
		r.Version = old.Version + 1
		r.Supersedes = old.Version
		r.CreatedAt = old.CreatedAt
	}
	copyRecords := append([]ExplicitMemory(nil), l.Records...)
	if index >= 0 {
		copyRecords[index] = r
	} else {
		copyRecords = append(copyRecords, r)
	}
	p = p.WithDefaults()
	encoded, _ := json.Marshal(copyRecords)
	if len(copyRecords) > p.MaxRecords || len(encoded) > p.MaxUserBytes {
		return nil, ErrResourceExhausted
	}
	l.Records = copyRecords
	return &r, nil
}
func sameMemoryTime(a, b *time.Time) bool {
	return a == nil && b == nil || a != nil && b != nil && a.Equal(*b)
}

func (l *MemoryLedger) Delete(id string, m MemoryMutation) error {
	if err := l.Check(m.Generation, m.StateVersion, false); err != nil {
		return err
	}
	for i, r := range l.Records {
		if r.ID == id {
			if r.Version != m.Version {
				return ErrMemoryConflict
			}
			l.Records = append(l.Records[:i], l.Records[i+1:]...)
			return nil
		}
	}
	return ErrMemoryNotFound
}
func (l *MemoryLedger) Change(m MemoryMutation, clear bool) error {
	if err := l.Check(m.Generation, m.StateVersion, false); err != nil {
		return err
	}
	if clear {
		l.Generation++
		l.Records = []ExplicitMemory{}
	} else {
		if m.Enabled == nil {
			return ErrMemoryInvalid
		}
		if l.Enabled != *m.Enabled {
			l.Enabled = *m.Enabled
			l.StateVersion++
		}
	}
	return nil
}
func MemoryMatches(r ExplicitMemory, m MatchContext) bool {
	return (r.Applicability.ProjectRef == "" || r.Applicability.ProjectRef == m.ProjectRef) && (r.Applicability.TaskType == "" || r.Applicability.TaskType == m.TaskType)
}
func MemoryLive(l MemoryLedger, r ExplicitMemory, now time.Time) bool {
	return r.Status == "confirmed" && r.Generation == l.Generation && (r.ExpiresAt == nil || r.ExpiresAt.After(now))
}

// MemoryManageable includes records whose own expiry has passed so users can
// inspect and remove records that still consume quota. Source-invalid records
// are returned as metadata-only tombstones by MemoryList.
func MemoryManageable(l MemoryLedger, r ExplicitMemory) bool {
	return r.Status == "confirmed" && r.Generation == l.Generation
}
func SortMemories(records []ExplicitMemory) {
	sort.Slice(records, func(i, j int) bool {
		if records[i].CanonicalKey != records[j].CanonicalKey {
			return records[i].CanonicalKey < records[j].CanonicalKey
		}
		return records[i].ID < records[j].ID
	})
}

func MemoryDependencyValid(l MemoryLedger, d MemoryDependency, now time.Time, sourceLive func(string) bool) bool {
	if !l.Enabled || l.Generation != d.Generation || l.StateVersion != d.StateVersion {
		return false
	}
	for _, r := range l.Records {
		if r.ID == d.ID && r.Version == d.Version && MemoryLive(l, r, now) && (r.SourceSessionID == "" || sourceLive(r.SourceSessionID)) {
			return true
		}
	}
	return false
}

// FilterDependentTurn never edits canonical records. Turn-level provenance is
// conservative: suppress all model content and tool groups from a stale Turn.
// Independent user statements remain history, not resurrected Memory facts.
func FilterDependentTurn(t TurnContext) TurnContext {
	t.Messages = append(t.Messages[:0:0], t.Messages...)
	out := t.Messages[:0]
	for _, m := range t.Messages {
		if m != nil && string(m.Role) == "user" {
			out = append(out, m)
		}
	}
	t.Messages = out
	t.Batches = nil
	t.Dependencies = nil
	t.MemoryFiltered = true
	return t
}

func MemoryArtifactID(id string) string { return "memory:" + id }

func MemoryList(l MemoryLedger, cursor string, limit int, now time.Time, sourceLive func(string) bool) (MemoryPage, error) {
	out := MemoryPage{MemorySettings: l.MemorySettings, Records: []ExplicitMemory{}}
	if limit < 1 || limit > 128 || len(cursor) > 128 {
		return out, ErrMemoryInvalid
	}
	records := append([]ExplicitMemory(nil), l.Records...)
	SortMemories(records)
	for _, r := range records {
		position := r.CanonicalKey + ":" + r.ID
		if position <= cursor || !MemoryManageable(l, r) {
			continue
		}
		if len(out.Records) == limit {
			last := out.Records[len(out.Records)-1]
			out.NextCursor = last.CanonicalKey + ":" + last.ID
			break
		}
		if r.SourceSessionID != "" && !sourceLive(r.SourceSessionID) {
			// Keep the ID/version so the owner can delete the quota-consuming
			// record, but never expose content after its source is inaccessible.
			r.Value = ""
			r.Status = "source_unavailable"
		}
		out.Records = append(out.Records, r)
	}
	return out, nil
}

func MemoryRecall(l MemoryLedger, m MatchContext, limit, maxBytes int, now time.Time, sourceLive func(string) bool) ([]ExplicitMemory, error) {
	m, err := NormalizeMatchContext(m)
	if err != nil {
		return nil, err
	}
	if limit < 1 || limit > 128 || maxBytes < 1 || maxBytes > 2<<20 {
		return nil, ErrMemoryInvalid
	}
	out := []ExplicitMemory{}
	if !l.Enabled {
		return out, nil
	}
	records := append([]ExplicitMemory(nil), l.Records...)
	SortMemories(records)
	bytes := 0
	for _, r := range records {
		if !MemoryLive(l, r, now) || !MemoryMatches(r, m) || (r.SourceSessionID != "" && !sourceLive(r.SourceSessionID)) {
			continue
		}
		b, _ := json.Marshal(r)
		if bytes+len(b) > maxBytes {
			continue
		}
		out = append(out, r)
		bytes += len(b)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}
