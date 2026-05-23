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

package versioning

import (
	"sort"
	"sync"
	"time"

	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type Store interface {
	InsertVersion(req InsertRequest, maxVersions int64) (*Version, error)
	CreateIntent(req InsertRequest, expected *int64) (*Intent, error)
	GetIntent(id int64) (*Intent, error)
	OpenIntent(kind coremodel.ResourceKind, resourceKey string) (*Intent, error)
	FindOpenIntentByHash(kind coremodel.ResourceKind, resourceKey, contentHash string) (*Intent, error)
	MarkIntentApplied(id int64) error
	MarkIntentFailed(id int64, message string) error
	MarkIntentFailedWithReason(id int64, reason string) error
	CommitIntent(id int64, maxVersions int64) (*Version, error)
	ListOpenIntents() ([]Intent, error)
	ListVersions(kind coremodel.ResourceKind, resourceKey string) ([]Version, error)
	GetVersion(kind coremodel.ResourceKind, resourceKey string, id int64) (*Version, error)
	GetVersionByID(id int64) (*Version, error)
	CurrentMeta(kind coremodel.ResourceKind, resourceKey string) (*Meta, error)
	LatestVersion(kind coremodel.ResourceKind, resourceKey string) (*Version, error)
	CheckExpectedVersion(kind coremodel.ResourceKind, resourceKey string, expected *int64) error
}

type MemoryStore struct {
	mu           sync.Mutex
	nextID       int64
	nextIntentID int64
	versions     map[int64]*Version
	byRule       map[ruleKey][]int64
	meta         map[ruleKey]*Meta
	intents      map[int64]*Intent
	byIntentRule map[ruleKey][]int64
}

type ruleKey struct {
	kind        coremodel.ResourceKind
	resourceKey string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		nextID:       1,
		nextIntentID: 1,
		versions:     make(map[int64]*Version),
		byRule:       make(map[ruleKey][]int64),
		meta:         make(map[ruleKey]*Meta),
		intents:      make(map[int64]*Intent),
		byIntentRule: make(map[ruleKey][]int64),
	}
}

func (s *MemoryStore) InsertVersion(req InsertRequest, maxVersions int64) (*Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.insertVersionLocked(req, maxVersions)
}

func (s *MemoryStore) insertVersionLocked(req InsertRequest, maxVersions int64) (*Version, error) {
	key := ruleKey{kind: req.RuleKind, resourceKey: req.ResourceKey}
	meta := s.meta[key]
	if meta == nil {
		meta = &Meta{RuleKind: req.RuleKind, ResourceKey: req.ResourceKey}
		s.meta[key] = meta
	}
	if ids := s.byRule[key]; len(ids) > 0 {
		latest := s.versions[ids[len(ids)-1]]
		if shouldDedupVersion(latest, req) {
			cp := *latest
			if meta.CurrentVersion != nil && *meta.CurrentVersion == cp.ID {
				cp.IsCurrent = true
			}
			return &cp, nil
		}
	}
	now := req.CreatedAt
	meta.LastVersionNo++
	id := s.nextID
	s.nextID++
	v := &Version{
		ID:               id,
		RuleKind:         req.RuleKind,
		Mesh:             req.Mesh,
		ResourceKey:      req.ResourceKey,
		RuleName:         req.RuleName,
		VersionNo:        meta.LastVersionNo,
		ContentHash:      req.ContentHash,
		SpecJSON:         req.SpecJSON,
		Source:           req.Source,
		Operation:        req.Operation,
		Author:           req.Author,
		Reason:           req.Reason,
		RolledBackFromID: req.RolledBackFromID,
		CreatedAt:        now,
	}
	s.versions[id] = v
	s.byRule[key] = append(s.byRule[key], id)
	if req.Operation == OperationDelete {
		meta.CurrentVersion = nil
	} else {
		current := id
		meta.CurrentVersion = &current
	}
	meta.UpdatedAt = now
	s.trimLocked(key, maxVersions)
	cp := *v
	cp.IsCurrent = meta.CurrentVersion != nil && *meta.CurrentVersion == cp.ID
	return &cp, nil
}

func (s *MemoryStore) CreateIntent(req InsertRequest, expected *int64) (*Intent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := req.CreatedAt
	if now.IsZero() {
		now = time.Now()
	}
	id := s.nextIntentID
	s.nextIntentID++
	intent := &Intent{
		ID:                id,
		RuleKind:          req.RuleKind,
		Mesh:              req.Mesh,
		ResourceKey:       req.ResourceKey,
		RuleName:          req.RuleName,
		ContentHash:       req.ContentHash,
		SpecJSON:          req.SpecJSON,
		Source:            req.Source,
		Operation:         req.Operation,
		Author:            req.Author,
		Reason:            req.Reason,
		RolledBackFromID:  req.RolledBackFromID,
		ExpectedVersionID: expected,
		Status:            IntentStatusPending,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	s.intents[id] = intent
	key := ruleKey{kind: req.RuleKind, resourceKey: req.ResourceKey}
	s.byIntentRule[key] = append(s.byIntentRule[key], id)
	return copyIntent(intent), nil
}

func (s *MemoryStore) GetIntent(id int64) (*Intent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	intent := s.intents[id]
	if intent == nil {
		return nil, ErrVersionIntentNotFound
	}
	return copyIntent(intent), nil
}

func (s *MemoryStore) OpenIntent(kind coremodel.ResourceKind, resourceKey string) (*Intent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return copyIntent(s.openIntentLocked(ruleKey{kind: kind, resourceKey: resourceKey})), nil
}

func (s *MemoryStore) FindOpenIntentByHash(kind coremodel.ResourceKind, resourceKey, contentHash string) (*Intent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := ruleKey{kind: kind, resourceKey: resourceKey}
	for _, id := range s.byIntentRule[key] {
		intent := s.intents[id]
		if isOpenIntent(intent) && intent.ContentHash == contentHash {
			return copyIntent(intent), nil
		}
	}
	return nil, nil
}

func (s *MemoryStore) MarkIntentApplied(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	intent := s.intents[id]
	if intent == nil {
		return ErrVersionIntentPending
	}
	if intent.Status == IntentStatusPending {
		intent.Status = IntentStatusApplied
		intent.UpdatedAt = time.Now()
	}
	return nil
}

func (s *MemoryStore) MarkIntentFailed(id int64, message string) error {
	return s.MarkIntentFailedWithReason(id, message)
}

func (s *MemoryStore) MarkIntentFailedWithReason(id int64, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	intent := s.intents[id]
	if intent == nil {
		return ErrVersionIntentNotFound
	}
	if intent.Status != IntentStatusPending {
		return ErrVersionIntentNotOpen
	}
	intent.Status = IntentStatusFailed
	intent.LastError = reason
	intent.UpdatedAt = time.Now()
	return nil
}

func (s *MemoryStore) CommitIntent(id int64, maxVersions int64) (*Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	intent := s.intents[id]
	if intent == nil || !isOpenIntent(intent) {
		if intent != nil && intent.Status == IntentStatusCommitted && intent.VersionID != nil {
			return s.copyVersionLocked(*intent.VersionID)
		}
		return nil, ErrVersionIntentPending
	}
	version, err := s.insertVersionLocked(intentInsertRequest(intent), maxVersions)
	if err != nil {
		return nil, err
	}
	intent.Status = IntentStatusCommitted
	intent.VersionID = &version.ID
	intent.UpdatedAt = time.Now()
	return version, nil
}

func (s *MemoryStore) ListOpenIntents() ([]Intent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]Intent, 0)
	for _, intent := range s.intents {
		if isOpenIntent(intent) {
			items = append(items, *copyIntent(intent))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items, nil
}

func shouldDedupVersion(latest *Version, req InsertRequest) bool {
	if latest == nil || latest.ContentHash != req.ContentHash {
		return false
	}
	if latest.Operation == OperationDelete || req.Operation == OperationDelete {
		return latest.Operation == req.Operation
	}
	return true
}

func (s *MemoryStore) ListVersions(kind coremodel.ResourceKind, resourceKey string) ([]Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := ruleKey{kind: kind, resourceKey: resourceKey}
	ids := append([]int64(nil), s.byRule[key]...)
	sort.Slice(ids, func(i, j int) bool {
		return s.versions[ids[i]].VersionNo > s.versions[ids[j]].VersionNo
	})
	meta := s.meta[key]
	items := make([]Version, 0, len(ids))
	for _, id := range ids {
		if v := s.versions[id]; v != nil {
			cp := *v
			cp.IsCurrent = meta != nil && meta.CurrentVersion != nil && *meta.CurrentVersion == cp.ID
			items = append(items, cp)
		}
	}
	return items, nil
}

func (s *MemoryStore) GetVersion(kind coremodel.ResourceKind, resourceKey string, id int64) (*Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, err := s.copyVersionLocked(id)
	if err != nil {
		return nil, err
	}
	if v.RuleKind != kind || v.ResourceKey != resourceKey {
		return nil, ErrVersionNotFound
	}
	return v, nil
}

func (s *MemoryStore) copyVersionLocked(id int64) (*Version, error) {
	v := s.versions[id]
	if v == nil {
		return nil, ErrVersionNotFound
	}
	cp := *v
	if meta := s.meta[ruleKey{kind: v.RuleKind, resourceKey: v.ResourceKey}]; meta != nil && meta.CurrentVersion != nil {
		cp.IsCurrent = *meta.CurrentVersion == cp.ID
	}
	return &cp, nil
}

func (s *MemoryStore) GetVersionByID(id int64) (*Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.copyVersionLocked(id)
}

func (s *MemoryStore) CurrentMeta(kind coremodel.ResourceKind, resourceKey string) (*Meta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta := s.meta[ruleKey{kind: kind, resourceKey: resourceKey}]
	if meta == nil {
		return nil, nil
	}
	cp := *meta
	return &cp, nil
}

func (s *MemoryStore) LatestVersion(kind coremodel.ResourceKind, resourceKey string) (*Version, error) {
	items, err := s.ListVersions(kind, resourceKey)
	if err != nil || len(items) == 0 {
		return nil, err
	}
	return &items[0], nil
}

func (s *MemoryStore) CheckExpectedVersion(kind coremodel.ResourceKind, resourceKey string, expected *int64) error {
	if expected == nil {
		return nil
	}
	meta, err := s.CurrentMeta(kind, resourceKey)
	if err != nil {
		return err
	}
	if meta == nil || meta.CurrentVersion == nil || *meta.CurrentVersion != *expected {
		var current *int64
		if meta != nil {
			current = meta.CurrentVersion
		}
		return &ConflictError{CurrentVersionID: current}
	}
	return nil
}

func (s *MemoryStore) trimLocked(key ruleKey, maxVersions int64) {
	if maxVersions <= 0 {
		return
	}
	ids := s.byRule[key]
	if int64(len(ids)) <= maxVersions {
		return
	}
	remove := ids[:int64(len(ids))-maxVersions]
	s.byRule[key] = ids[int64(len(ids))-maxVersions:]
	for _, id := range remove {
		delete(s.versions, id)
	}
}

func (s *MemoryStore) openIntentLocked(key ruleKey) *Intent {
	for _, id := range s.byIntentRule[key] {
		intent := s.intents[id]
		if isOpenIntent(intent) {
			return intent
		}
	}
	return nil
}

func isOpenIntent(intent *Intent) bool {
	return intent != nil && (intent.Status == IntentStatusPending || intent.Status == IntentStatusApplied)
}

func copyIntent(intent *Intent) *Intent {
	if intent == nil {
		return nil
	}
	cp := *intent
	return &cp
}

func intentInsertRequest(intent *Intent) InsertRequest {
	return InsertRequest{
		RuleKind:         intent.RuleKind,
		Mesh:             intent.Mesh,
		ResourceKey:      intent.ResourceKey,
		RuleName:         intent.RuleName,
		SpecJSON:         intent.SpecJSON,
		ContentHash:      intent.ContentHash,
		Source:           intent.Source,
		Operation:        intent.Operation,
		Author:           intent.Author,
		Reason:           intent.Reason,
		RolledBackFromID: intent.RolledBackFromID,
		CreatedAt:        intent.CreatedAt,
	}
}
