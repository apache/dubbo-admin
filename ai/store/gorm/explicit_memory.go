/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with this
 * work for additional information regarding copyright ownership.
 * The ASF licenses this file under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with it.
 * You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by law or agreed in writing, software is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
 */
package gormstore

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"dubbo-admin-ai/security"
	"dubbo-admin-ai/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// One bounded aggregate per tenant/user. SQL predicates enforce identity before
// loading any content; the policy caps both record count and serialized bytes.
// Revision writes serialize across processes, including SQLite (which ignores
// SELECT FOR UPDATE). No process mutex is used for persistence correctness.
type MemoryLedgerModel struct {
	ID          string `gorm:"primaryKey;size:64"`
	TenantID    string `gorm:"size:128;not null;index"`
	OwnerID     string `gorm:"size:128;not null"`
	Revision    uint64 `gorm:"not null"`
	StoredBytes int64  `gorm:"not null"`
	Payload     []byte `gorm:"not null"`
}

func (MemoryLedgerModel) TableName() string { return "ai_memory_ledgers" }

var _ store.ExplicitMemoryStore = (*GormStore)(nil)

func (s *GormStore) memoryLedger(tx *gorm.DB, a security.AccessContext, lock, create bool) (store.MemoryLedger, *MemoryLedgerModel, error) {
	l := store.NewMemoryLedger()
	l.TenantID = a.Subject.TenantID
	l.OwnerID = a.Subject.UserID
	id := store.MemoryOwnerKey(l.TenantID, l.OwnerID)
	if create {
		data, _ := json.Marshal(l)
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&MemoryLedgerModel{ID: id, TenantID: l.TenantID, OwnerID: l.OwnerID, Payload: data}).Error; err != nil {
			return l, nil, err
		}
	}
	q := tx.Model(&MemoryLedgerModel{}).Where("id = ? AND tenant_id = ? AND owner_id = ?", id, l.TenantID, l.OwnerID)
	if lock {
		if err := q.Update("revision", gorm.Expr("revision + 0")).Error; err != nil {
			return l, nil, err
		}
	}
	var row MemoryLedgerModel
	read := tx.Where("id = ? AND tenant_id = ? AND owner_id = ?", id, l.TenantID, l.OwnerID)
	if lock && supportsRowLock(tx) {
		read = read.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := read.First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return l, nil, nil
		}
		return l, nil, err
	}
	if len(row.Payload) > (2<<20)+8192 {
		return l, nil, store.ErrResourceExhausted
	}
	if err := json.Unmarshal(row.Payload, &l); err != nil {
		return l, nil, err
	}
	if l.TenantID != a.Subject.TenantID || l.OwnerID != a.Subject.UserID {
		return l, nil, security.ErrResourceNotAccessible
	}
	return l, &row, nil
}

// Source tombstones and dependent commits serialize on the same user row.
// Removing associated records makes locking reads safe under REPEATABLE READ.
func (s *GormStore) deleteSessionMemories(tx *gorm.DB, session SessionModel) error {
	a := security.MustAccessContext(security.Subject{TenantID: session.TenantID, UserID: session.OwnerID, AuthMethod: "internal-owner"})
	l, row, err := s.memoryLedger(tx, a, true, false)
	if err != nil || row == nil {
		return err
	}
	kept := make([]store.ExplicitMemory, 0, len(l.Records))
	for _, r := range l.Records {
		if r.SourceSessionID != session.ID {
			kept = append(kept, r)
		}
	}
	if len(kept) == len(l.Records) {
		return nil
	}
	l.Records = kept
	return s.putMemoryLedger(tx, a, l, row)
}
func (s *GormStore) putMemoryLedger(tx *gorm.DB, a security.AccessContext, l store.MemoryLedger, row *MemoryLedgerModel) error {
	data, err := json.Marshal(l)
	if err != nil {
		return err
	}
	delta := store.MemoryContentBytes(l) - row.StoredBytes
	// MySQL reports changed rows by default: adding zero affects no rows and
	// must not be mistaken for a failed quota reservation. Still persist the
	// ledger below, since equal-size changes can alter its logical state.
	if delta != 0 {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&TenantUsageModel{TenantID: a.Subject.TenantID}).Error; err != nil {
			return err
		}
		r := tx.Model(&TenantUsageModel{}).Where("tenant_id = ? AND stored_bytes + ? <= ?", a.Subject.TenantID, delta, s.quota.MaxTenantBytes).Update("stored_bytes", gorm.Expr("stored_bytes + ?", delta))
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected != 1 {
			return store.ErrResourceExhausted
		}
	}
	r := tx.Model(&MemoryLedgerModel{}).Where("id = ? AND revision = ?", row.ID, row.Revision).Updates(map[string]any{"payload": data, "stored_bytes": store.MemoryContentBytes(l), "revision": row.Revision + 1})
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected != 1 {
		return store.ErrMemoryConflict
	}
	return nil
}
func (s *GormStore) memorySourceLive(tx *gorm.DB, a security.AccessContext, id string) bool {
	var count int64
	return tx.Model(&SessionModel{}).Where("id = ? AND tenant_id = ? AND owner_id = ? AND status = ? AND (expires_at IS NULL OR expires_at > ?)", id, a.Subject.TenantID, a.Subject.UserID, store.SessionStatusActive, time.Now()).Count(&count).Error == nil && count == 1
}
func (s *GormStore) GetMemorySettings(ctx context.Context, a security.AccessContext) (store.MemorySettings, error) {
	if err := a.ValidateBusiness(); err != nil {
		return store.MemorySettings{}, err
	}
	l, _, err := s.memoryLedger(s.db.WithContext(normalizeContext(ctx)), a, false, false)
	return l.MemorySettings, err
}
func (s *GormStore) ListMemories(ctx context.Context, a security.AccessContext, cursor string, limit int) (store.MemoryPage, error) {
	var out store.MemoryPage
	if err := a.ValidateBusiness(); err != nil {
		return out, err
	}
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		l, _, err := s.memoryLedger(tx, a, false, false)
		if err != nil {
			return err
		}
		out, err = store.MemoryList(l, cursor, limit, time.Now(), func(id string) bool { return s.memorySourceLive(tx, a, id) })
		return err
	})
	return out, err
}
func (s *GormStore) SaveMemory(ctx context.Context, a security.AccessContext, w store.MemoryWrite) (*store.ExplicitMemory, error) {
	if err := a.ValidateBusiness(); err != nil {
		return nil, err
	}
	var out *store.ExplicitMemory
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		if w.SourceSessionID != "" {
			session, err := s.findActiveSession(tx, a, w.SourceSessionID, true)
			if err != nil {
				return err
			}
			if session.OwnerID != a.Subject.UserID {
				return security.ErrResourceNotAccessible
			}
		}
		l, row, err := s.memoryLedger(tx, a, true, true)
		if err != nil {
			return err
		}
		out, err = l.Save(a, w, s.quota.ExplicitMemory, time.Now())
		if err != nil {
			return err
		}
		if err = s.putMemoryLedger(tx, a, l, row); err != nil {
			return err
		}
		if out.SourceSessionID != "" {
			now := time.Now()
			return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&DerivedArtifactModel{ID: store.MemoryArtifactID(out.ID), TenantID: out.TenantID, CanonicalSourceID: "session:" + out.SourceSessionID, SourceSessionID: out.SourceSessionID, Kind: store.MemoryArtifactKind, Locator: out.OwnerID + ":" + out.ID, Status: "active", CreatedAt: now, UpdatedAt: now}).Error
		}
		return nil
	})
	return out, err
}
func (s *GormStore) DeleteMemory(ctx context.Context, a security.AccessContext, id string, mutation store.MemoryMutation) error {
	if err := a.ValidateBusiness(); err != nil {
		return err
	}
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		l, row, err := s.memoryLedger(tx, a, true, true)
		if err != nil {
			return err
		}
		if err = l.Delete(id, mutation); err != nil {
			return err
		}
		if err := s.putMemoryLedger(tx, a, l, row); err != nil {
			return err
		}
		return tx.Where("id = ? AND tenant_id = ? AND kind = ?", store.MemoryArtifactID(id), a.Subject.TenantID, store.MemoryArtifactKind).Delete(&DerivedArtifactModel{}).Error
	})
}
func (s *GormStore) ChangeMemorySettings(ctx context.Context, a security.AccessContext, mutation store.MemoryMutation, clear bool) (store.MemorySettings, error) {
	if err := a.ValidateBusiness(); err != nil {
		return store.MemorySettings{}, err
	}
	var out store.MemorySettings
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		l, row, err := s.memoryLedger(tx, a, true, true)
		if err != nil {
			return err
		}
		previous := append([]store.ExplicitMemory(nil), l.Records...)
		if err = l.Change(mutation, clear); err != nil {
			return err
		}
		out = l.MemorySettings
		if err := s.putMemoryLedger(tx, a, l, row); err != nil {
			return err
		}
		if clear && len(previous) > 0 {
			ids := make([]string, 0, len(previous))
			for _, r := range previous {
				ids = append(ids, store.MemoryArtifactID(r.ID))
			}
			return tx.Where("id IN ? AND tenant_id = ? AND kind = ?", ids, a.Subject.TenantID, store.MemoryArtifactKind).Delete(&DerivedArtifactModel{}).Error
		}
		return nil
	})
	return out, err
}
func (s *GormStore) RecallMemories(ctx context.Context, a security.AccessContext, match store.MatchContext, limit, bytes int) ([]store.ExplicitMemory, store.MemorySettings, error) {
	if err := a.ValidateBusiness(); err != nil {
		return nil, store.MemorySettings{}, err
	}
	var out []store.ExplicitMemory
	var settings store.MemorySettings
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		l, _, err := s.memoryLedger(tx, a, false, false)
		if err != nil {
			return err
		}
		settings = l.MemorySettings
		out, err = store.MemoryRecall(l, match, limit, bytes, time.Now(), func(id string) bool { return s.memorySourceLive(tx, a, id) })
		return err
	})
	return out, settings, err
}
func (s *GormStore) validateMemoryDependencies(tx *gorm.DB, a security.AccessContext, id string, deps []store.MemoryDependency, lock bool) error {
	if len(deps) == 0 {
		return nil
	}
	session, err := s.findActiveSession(tx, a, id, false)
	if err != nil {
		return err
	}
	owner := security.MustAccessContext(security.Subject{TenantID: session.TenantID, UserID: session.OwnerID, AuthMethod: "internal-owner"})
	l, _, err := s.memoryLedger(tx, owner, lock, false)
	if err != nil {
		return err
	}
	for _, d := range deps {
		if !store.MemoryDependencyValid(l, d, time.Now(), func(source string) bool { return s.memorySourceLive(tx, owner, source) }) {
			return store.ErrMemoryDependency
		}
	}
	return nil
}
func (s *GormStore) ValidateMemoryDependencies(ctx context.Context, a security.AccessContext, id string, deps []store.MemoryDependency) error {
	if err := a.ValidateBusiness(); err != nil {
		return err
	}
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		if _, err := s.findActiveSession(tx, a, id, false); err != nil {
			return err
		}
		return s.validateMemoryDependencies(tx, a, id, deps, false)
	})
}
func (s *GormStore) RecordTurnMemoryDependencies(ctx context.Context, a security.AccessContext, id string, turn uint64, deps []store.MemoryDependency) error {
	if err := a.ValidateBusiness(); err != nil {
		return err
	}
	if len(deps) > 256 {
		return store.ErrResourceExhausted
	}
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		if _, err := s.findActiveSession(tx, a, id, true); err != nil {
			return err
		}
		var t TurnModel
		if err := tx.Where("id = ? AND session_id = ? AND completed_at IS NULL", turn, id).First(&t).Error; err != nil {
			return store.ErrTurnNotFound
		}
		var previous []store.MemoryDependency
		if len(t.Dependencies) > 0 {
			if err := json.Unmarshal(t.Dependencies, &previous); err != nil {
				return err
			}
		}
		merged := store.MergeMemoryDependencies(previous, deps)
		if len(merged) > 256 {
			return store.ErrResourceExhausted
		}
		data, _ := json.Marshal(merged)
		delta := int64(len(data) - len(t.Dependencies))
		// One copy on the active Turn and one prepaid terminal-journal copy.
		if err := s.reserveToolBytes(tx, a.Subject.TenantID, id, delta*2); err != nil {
			return err
		}
		return tx.Model(&TurnModel{}).Where("id = ?", turn).Updates(map[string]any{"dependencies": data, "stored_bytes": gorm.Expr("stored_bytes + ?", delta)}).Error
	})
}
func (s *GormStore) DeleteSourceMemory(ctx context.Context, a security.AccessContext, artifact store.DerivedArtifact) error {
	if !a.IsSystem() || artifact.Kind != store.MemoryArtifactKind {
		return security.ErrResourceNotAccessible
	}
	source, err := store.SessionIDFromCanonicalSource(artifact.CanonicalSourceID)
	if err != nil {
		return err
	}
	split := strings.LastIndex(artifact.Locator, ":")
	if split < 1 {
		return store.ErrMemoryInvalid
	}
	ownerID, id := artifact.Locator[:split], artifact.Locator[split+1:]
	owner := security.MustAccessContext(security.Subject{TenantID: artifact.TenantID, UserID: ownerID, AuthMethod: "internal-owner"})
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		var session SessionModel
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND tenant_id = ?", source, artifact.TenantID).First(&session).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil && session.Status == store.SessionStatusActive && (session.ExpiresAt == nil || session.ExpiresAt.After(time.Now())) {
			return security.ErrResourceNotAccessible
		}
		l, row, err := s.memoryLedger(tx, owner, true, false)
		if err != nil {
			return err
		}
		if row == nil {
			return nil
		}
		for i, r := range l.Records {
			if r.ID == id && r.SourceSessionID == source && artifact.ID == store.MemoryArtifactID(id) {
				l.Records = append(l.Records[:i:i], l.Records[i+1:]...)
				return s.putMemoryLedger(tx, owner, l, row)
			}
		}
		return nil
	})
}
