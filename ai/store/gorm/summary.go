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

package gormstore

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"dubbo-admin-ai/security"
	store "dubbo-admin-ai/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// HistoryEntryModel is a small immutable terminal-attempt journal. Failed
// messages remain deleted; their status and canonical tool records are retained.
type HistoryEntryModel struct {
	Dependencies []byte
	Sequence     uint64 `gorm:"primaryKey;autoIncrement"`
	SessionID    string `gorm:"size:64;not null;index:idx_ai_history_page,priority:1;uniqueIndex:idx_ai_history_turn,priority:1"`
	TurnID       uint64 `gorm:"not null;uniqueIndex:idx_ai_history_turn,priority:2"`
	Status       string `gorm:"size:24;not null"`
}

func (HistoryEntryModel) TableName() string { return "ai_history_entries" }

type SummaryModel struct {
	SessionID string `gorm:"primaryKey;size:64"`
	Version   uint64 `gorm:"not null"`
	Payload   []byte `gorm:"not null"`
}

func (SummaryModel) TableName() string { return "ai_summaries" }

var _ store.SummaryStore = (*GormStore)(nil)

func checkSummaryHistoryCapacity(tx *gorm.DB, id string) error {
	var count int64
	if err := tx.Model(&HistoryEntryModel{}).Where("session_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count >= store.MaxHistoryEntriesPerSession {
		return store.ErrResourceExhausted
	}
	return nil
}

func recordSummaryHistory(tx *gorm.DB, id string, turn uint64, status string) error {
	var source TurnModel
	if err := tx.Select("dependencies").Where("session_id = ? AND id = ?", id, turn).Find(&source).Error; err != nil {
		return err
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&HistoryEntryModel{Dependencies: source.Dependencies, SessionID: id, TurnID: turn, Status: status}).Error
}

func (s *GormStore) validateSummarySnapshot(tx *gorm.DB, a security.AccessContext, snap store.HistorySnapshot, lock bool) error {
	if _, err := s.findActiveSession(tx, a, snap.SessionID, lock); err != nil {
		return err
	}
	var current TurnModel
	if err := tx.Where("id = ? AND session_id = ? AND completed_at IS NULL", snap.ActiveTurn, snap.SessionID).First(&current).Error; err != nil {
		return store.ErrContextConflict
	}
	if current.InteractionID != "" {
		var interaction InteractionModel
		if err := tx.Where("id = ? AND turn_id = ? AND status = ? AND lease_expires_at > ?", current.InteractionID, current.ID, store.InteractionStatusRunning, time.Now()).First(&interaction).Error; err != nil {
			return store.ErrContextConflict
		}
	}
	if snap.Target > snap.Upper {
		return store.ErrContextConflict
	}
	if snap.Upper > 0 {
		var count int64
		if err := tx.Model(&HistoryEntryModel{}).Where("session_id = ? AND sequence = ?", snap.SessionID, snap.Upper).Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return store.ErrContextConflict
		}
	}
	return nil
}

func (s *GormStore) OpenHistorySnapshot(ctx context.Context, a security.AccessContext, id string, active uint64, keep int) (store.HistorySnapshot, error) {
	snap := store.HistorySnapshot{SessionID: id, ActiveTurn: active}
	if keep < 0 || keep > s.quota.MaxTurnsPerSession {
		return snap, store.ErrContextConflict
	}
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		if err := s.validateSummarySnapshot(tx, a, snap, true); err != nil {
			return err
		}
		var upper uint64
		if err := tx.Model(&HistoryEntryModel{}).Select("COALESCE(MAX(sequence),0)").Where("session_id = ?", id).Scan(&upper).Error; err != nil {
			return err
		}
		snap.Upper = upper
		var count int64
		if err := tx.Model(&HistoryEntryModel{}).Where("session_id = ? AND sequence <= ?", id, upper).Count(&count).Error; err != nil {
			return err
		}
		snap.Count = int(count)
		// Metadata only: the N+1 most recent completed Turns identify the boundary.
		var recent []HistoryEntryModel
		if err := tx.Where("session_id = ? AND sequence <= ? AND status = ?", id, upper, store.InteractionStatusCompleted).Order("sequence DESC").Limit(keep + 1).Find(&recent).Error; err != nil {
			return err
		}
		if len(recent) > keep {
			snap.Target = recent[keep].Sequence
		}
		return nil
	})
	return snap, err
}

func (s *GormStore) readSummaryBatch(tx *gorm.DB, a security.AccessContext, req store.SummaryReadRequest) (store.SummaryBatch, error) {
	// Inspection checkpoints are written lazily under the Session lifecycle lock.
	if err := s.validateSummarySnapshot(tx, a, req.Snapshot, true); err != nil {
		return store.SummaryBatch{}, err
	}
	var row HistoryEntryModel
	if err := tx.Where("session_id = ? AND sequence > ? AND sequence <= ?", req.Snapshot.SessionID, req.Cursor.After, req.Snapshot.Target).Order("sequence ASC").First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return store.SummaryBatch{}, store.ErrSummaryNotFound
		}
		return store.SummaryBatch{}, err
	}
	type record struct {
		ID                 uint64
		Bytes              int
		Role               string
		SummaryCheckedRule string
		SummarySensitive   bool
	}
	var messages []record
	if row.Status == store.InteractionStatusCompleted {
		if err := tx.Model(&MessageModel{}).Select("id, role, LENGTH(payload) AS bytes, summary_checked_rule, summary_sensitive").Where("turn_id = ?", row.TurnID).Order("sequence ASC").Limit(s.quota.MaxEventsPerTurn + 1).Scan(&messages).Error; err != nil {
			return store.SummaryBatch{}, err
		}
	}
	type toolRecord struct {
		ID                    string
		Bytes                 int
		Version               uint64
		SummaryCheckedRule    string
		SummaryCheckedVersion uint64
		SummarySensitive      bool
	}
	var batches []toolRecord
	if err := tx.Model(&ToolBatchModel{}).Select("id, version, LENGTH(payload) AS bytes, summary_checked_rule, summary_checked_version, summary_sensitive").Where("session_id = ? AND turn_id = ?", row.SessionID, row.TurnID).Order("sequence ASC").Limit(s.quota.MaxEventsPerTurn + 1).Scan(&batches).Error; err != nil {
		return store.SummaryBatch{}, err
	}
	if len(messages) > s.quota.MaxEventsPerTurn || len(batches) > s.quota.MaxEventsPerTurn {
		return store.SummaryBatch{}, store.ErrResourceExhausted
	}
	entry := store.HistoryEntry{Sequence: row.Sequence, SessionID: row.SessionID, TurnID: row.TurnID, Status: row.Status}
	if len(row.Dependencies) > 0 {
		if err := json.Unmarshal(row.Dependencies, &entry.Dependencies); err != nil {
			return store.SummaryBatch{}, err
		}
	}
	classes := map[int]string{}
	var roles []string
	for _, m := range messages {
		roles = append(roles, m.Role)
	}
	order := store.SummaryRecordOrder(roles, len(batches))
	return store.BuildSummaryBatch(req, entry, len(messages)+len(batches), func(index, offset, limit int) (string, int, error) {
		index = order[index]
		var payload []byte
		var total int
		var query *gorm.DB
		var check store.SummaryInspection
		if index < len(messages) {
			r := messages[index]
			total = r.Bytes
			query = tx.Model(&MessageModel{}).Where("id = ? AND turn_id = ?", r.ID, row.TurnID).Session(&gorm.Session{})
			check = store.SummaryInspection{Rule: r.SummaryCheckedRule, Version: 1, Sensitive: r.SummarySensitive}
		} else {
			r := batches[index-len(messages)]
			if r.Bytes > store.MaxToolBatchBytes {
				return "", 0, store.ErrResourceExhausted
			}
			var stored ToolBatchModel
			if err := tx.Select("payload").Where("id = ? AND session_id = ?", r.ID, row.SessionID).First(&stored).Error; err != nil {
				return "", 0, err
			}
			var batch store.ToolBatch
			if err := json.Unmarshal(stored.Payload, &batch); err != nil {
				return "", 0, err
			}
			class := "tool_success"
			for _, call := range batch.Calls {
				if call.Status != "succeeded" {
					class = "tool_uncertain"
					break
				}
			}
			classes[index] = class
			data, err := store.SummaryToolEvidence(batch)
			if err != nil {
				return "", 0, err
			}
			read := func(start, size int) (string, int, error) {
				if err := tx.Statement.Context.Err(); err != nil {
					return "", 0, err
				}
				if start > len(data) {
					return "", 0, store.ErrContextConflict
				}
				return string(data[start:min(start+size, len(data))]), len(data), nil
			}
			check = store.SummaryInspection{Rule: r.SummaryCheckedRule, Version: r.SummaryCheckedVersion, Sensitive: r.SummarySensitive}
			query = tx.Model(&ToolBatchModel{}).Where("id = ? AND session_id = ? AND version = ?", r.ID, row.SessionID, r.Version).Session(&gorm.Session{})
			return s.readInspectedFragment(tx, a, row.SessionID, query, check, r.Version, true, offset, limit, read)
		}
		// SUBSTR is applied to a binary payload: offsets and limits are bytes on
		// SQLite/MySQL/PostgreSQL, including multibyte and very large messages.
		read := func(start, size int) (string, int, error) {
			if err := tx.Statement.Context.Err(); err != nil {
				return "", 0, err
			}
			if size == 0 {
				return "", total, nil
			}
			if err := query.Select("SUBSTR(payload, ?, ?)", start+1, size).Row().Scan(&payload); err != nil {
				return "", total, err
			}
			return string(payload), total, nil
		}
		return s.readInspectedFragment(tx, a, row.SessionID, query, check, 1, false, offset, limit, read)
	}, func(index int) string {
		index = order[index]
		if index < len(messages) {
			if messages[index].Role != "" {
				return messages[index].Role
			}
			return "unknown"
		}
		return classes[index]
	})
}

func (s *GormStore) readInspectedFragment(tx *gorm.DB, a security.AccessContext, id string, query *gorm.DB, check store.SummaryInspection, version uint64, tool bool, offset, limit int, read func(int, int) (string, int, error)) (string, int, error) {
	if !check.Valid(version) {
		oldRule := check.Rule
		var err error
		check, err = store.InspectSummaryRecord(version, read)
		if err != nil {
			return "", 0, err
		}
		if oldRule == "" {
			if err := s.reserveToolBytes(tx, a.Subject.TenantID, id, store.SummaryInspectionBytes); err != nil {
				return "", 0, err
			}
		}
		updates := map[string]any{"summary_checked_rule": check.Rule, "summary_sensitive": check.Sensitive}
		if tool {
			updates["summary_checked_version"] = version
		}
		result := query.Updates(updates)
		if result.Error != nil {
			return "", 0, result.Error
		}
		if result.RowsAffected != 1 {
			return "", 0, store.ErrContextConflict
		}
	}
	return store.ReadInspectedSummaryFragment(check, version, offset, limit, read)
}

func (s *GormStore) ReadSummaryBatch(ctx context.Context, a security.AccessContext, req store.SummaryReadRequest) (store.SummaryBatch, error) {
	var out store.SummaryBatch
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error { var err error; out, err = s.readSummaryBatch(tx, a, req); return err })
	return out, err
}

func readSummary(tx *gorm.DB, id string) (*store.Summary, error) {
	var row SummaryModel
	if err := tx.Where("session_id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, store.ErrSummaryNotFound
		}
		return nil, err
	}
	var v store.Summary
	if err := json.Unmarshal(row.Payload, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *GormStore) GetSummary(ctx context.Context, a security.AccessContext, id string) (*store.Summary, error) {
	var v *store.Summary
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		if _, err := s.findActiveSession(tx, a, id, false); err != nil {
			return err
		}
		var err error
		v, err = readSummary(tx, id)
		return err
	})
	return v, err
}

func (s *GormStore) CommitSummary(ctx context.Context, a security.AccessContext, c store.SummaryCommit) error {
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		if err := s.validateSummarySnapshot(tx, a, c.Read.Snapshot, true); err != nil {
			return err
		}
		batch, err := s.readSummaryBatch(tx, a, c.Read)
		if err != nil {
			return err
		}
		old, err := readSummary(tx, c.Value.SessionID)
		if err != nil && !errors.Is(err, store.ErrSummaryNotFound) {
			return err
		}
		if err := store.ValidateSummaryCommit(a, c, old, batch); err != nil {
			return err
		}
		data, err := json.Marshal(c.Value)
		if err != nil {
			return err
		}
		delta := int64(len(data))
		if old != nil {
			raw, _ := json.Marshal(old)
			delta -= int64(len(raw))
		}
		if err := s.reserveToolBytes(tx, c.Value.TenantID, c.Value.SessionID, delta); err != nil {
			return err
		}
		if old == nil {
			if err := tx.Create(&SummaryModel{SessionID: c.Value.SessionID, Version: c.Value.Version, Payload: data}).Error; err != nil {
				return err
			}
		} else {
			r := tx.Model(&SummaryModel{}).Where("session_id = ? AND version = ?", c.Value.SessionID, c.ExpectedVersion).Updates(map[string]any{"version": c.Value.Version, "payload": data})
			if r.Error != nil {
				return r.Error
			}
			if r.RowsAffected != 1 {
				return store.ErrContextConflict
			}
		}
		now := time.Now()
		artifact := DerivedArtifactModel{ID: store.SummaryArtifactID(c.Value.SessionID), TenantID: c.Value.TenantID, CanonicalSourceID: "session:" + c.Value.SessionID, SourceSessionID: c.Value.SessionID, Kind: store.SummaryArtifactKind, Locator: c.Value.SessionID, Status: "active", CreatedAt: now, UpdatedAt: now}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&artifact).Error
	})
}

func (s *GormStore) DeleteSummary(ctx context.Context, access security.AccessContext, artifact store.DerivedArtifact) error {
	if err := access.Validate(); err != nil {
		return err
	}
	if !access.IsSystem() {
		return security.ErrResourceNotAccessible
	}
	id, err := store.SessionIDFromCanonicalSource(artifact.CanonicalSourceID)
	if err != nil {
		return err
	}
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		var session SessionModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&session).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		} else if session.TenantID != artifact.TenantID || session.Status == store.SessionStatusActive {
			return security.ErrResourceNotAccessible
		}
		if artifact.Kind != store.SummaryArtifactKind || artifact.Locator != id {
			return security.ErrResourceNotAccessible
		}
		v, err := readSummary(tx, id)
		if err != nil && !errors.Is(err, store.ErrSummaryNotFound) {
			return err
		}
		if v != nil && v.TenantID != artifact.TenantID {
			return security.ErrResourceNotAccessible
		}
		// Source cleanup releases the complete Session byte reservation once.
		return tx.Where("session_id = ?", id).Delete(&SummaryModel{}).Error
	})
}
