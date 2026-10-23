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
	"database/sql"
	"encoding/json"

	"dubbo-admin-ai/security"
	store "dubbo-admin-ai/store"
	"gorm.io/gorm"
)

var _ store.CompilationStore = (*GormStore)(nil)

func (s *GormStore) RecoveryToolBatches(ctx context.Context, a security.AccessContext, id string, active uint64, limit, maxBytes int) (store.ToolRecovery, error) {
	var result store.ToolRecovery
	if limit <= 0 || limit > 64 || maxBytes <= 0 {
		return result, store.ErrResourceExhausted
	}
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		if _, err := s.findActiveSession(tx, a, id, false); err != nil {
			return err
		}
		var current TurnModel
		if err := tx.Where("session_id = ? AND id = ? AND completed_at IS NULL", id, active).First(&current).Error; err != nil {
			return store.ErrTurnNotFound
		}
		var rows []struct {
			ID    string
			Bytes int
		}
		// Read sizes before payloads, including only released source Turns.
		if err := tx.Table("ai_tool_batches AS b").Select("b.id, LENGTH(b.payload) AS bytes").Where("b.session_id = ? AND b.turn_id < ?", id, active).
			Where("NOT EXISTS (SELECT 1 FROM ai_turns t WHERE t.id = b.turn_id AND t.session_id = b.session_id)").
			Order("b.turn_id DESC").Order("b.sequence DESC").Limit(limit + 1).Scan(&rows).Error; err != nil {
			return err
		}
		if len(rows) > limit {
			result.Omitted = true
			rows = rows[:limit]
		}
		size := 0
		for _, row := range rows {
			if size+row.Bytes > maxBytes {
				result.Omitted = true
				break
			}
			var stored ToolBatchModel
			if err := tx.Where("id = ? AND session_id = ?", row.ID, id).First(&stored).Error; err != nil {
				return err
			}
			var b store.ToolBatch
			if err := json.Unmarshal(stored.Payload, &b); err != nil {
				return err
			}
			size += row.Bytes
			result.Batches = append(result.Batches, b)
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	for i, j := 0, len(result.Batches)-1; i < j; i, j = i+1, j-1 {
		result.Batches[i], result.Batches[j] = result.Batches[j], result.Batches[i]
	}
	return result, err
}

func (s *GormStore) PinTemplate(ctx context.Context, a security.AccessContext, id, ref string) (string, error) {
	if err := store.ValidateTemplateRef(ref); err != nil {
		return "", err
	}
	var result string
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		v, err := s.findActiveSession(tx, a, id, true)
		if err != nil {
			return err
		}
		result = v.TemplateRef
		if result == "" {
			result = ref
			return tx.Model(v).Update("template_ref", ref).Error
		}
		return nil
	})
	return result, err
}

func readBatches(tx *gorm.DB, id string, turn uint64, limit int) ([]store.ToolBatch, error) {
	if limit <= 0 {
		return nil, store.ErrResourceExhausted
	}
	var rows []ToolBatchModel
	if err := tx.Where("session_id = ? AND turn_id = ?", id, turn).Order("sequence ASC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) > limit {
		return nil, store.ErrResourceExhausted
	}
	var result []store.ToolBatch
	for _, r := range rows {
		var b store.ToolBatch
		if err := json.Unmarshal(r.Payload, &b); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, nil
}

func (s *GormStore) ToolBatches(ctx context.Context, a security.AccessContext, id string, turn uint64, limit int) ([]store.ToolBatch, error) {
	var result []store.ToolBatch
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		if _, err := s.findActiveSession(tx, a, id, false); err != nil {
			return err
		}
		var err error
		result, err = readBatches(tx, id, turn, limit)
		return err
	})
	return result, err
}

func (s *GormStore) reserveToolBytes(tx *gorm.DB, tenant, id string, delta int64) error {
	if delta == 0 {
		return nil
	}
	r := tx.Model(&SessionModel{}).Where("id = ? AND stored_bytes + ? <= ?", id, delta, s.quota.MaxSessionBytes).Update("stored_bytes", gorm.Expr("stored_bytes + ?", delta))
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected != 1 {
		return store.ErrResourceExhausted
	}
	r = tx.Model(&TenantUsageModel{}).Where("tenant_id = ? AND stored_bytes + ? <= ?", tenant, delta, s.quota.MaxTenantBytes).Update("stored_bytes", gorm.Expr("stored_bytes + ?", delta))
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected != 1 {
		return store.ErrResourceExhausted
	}
	return nil
}

func (s *GormStore) CreateToolBatch(ctx context.Context, a security.AccessContext, b store.ToolBatch) error {
	if err := store.ValidateToolBatch(a, b); err != nil {
		return err
	}
	data, err := json.Marshal(b)
	if err != nil {
		return err
	}
	if len(data) > store.MaxToolBatchBytes {
		return store.ErrResourceExhausted
	}
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		session, err := s.findActiveSession(tx, a, b.SessionID, true)
		if err != nil {
			return err
		}
		if session.TenantID != b.TenantID || session.OwnerID != b.OwnerID {
			return security.ErrResourceNotAccessible
		}
		var turn TurnModel
		if err := tx.Where("id = ? AND session_id = ? AND interaction_id = ? AND completed_at IS NULL", b.TurnID, b.SessionID, b.InteractionID).First(&turn).Error; err != nil {
			return store.ErrTurnNotFound
		}
		var count int64
		if err := tx.Model(&ToolBatchModel{}).Where("turn_id = ?", b.TurnID).Count(&count).Error; err != nil {
			return err
		}
		if count >= int64(s.quota.MaxEventsPerTurn) {
			return store.ErrResourceExhausted
		}
		if err := s.reserveToolBytes(tx, b.TenantID, b.SessionID, int64(len(data))); err != nil {
			return err
		}
		return tx.Create(&ToolBatchModel{ID: b.ID, SessionID: b.SessionID, TurnID: b.TurnID, Sequence: b.Sequence, Version: b.Version, Payload: data}).Error
	})
}

func (s *GormStore) CompleteToolCall(ctx context.Context, a security.AccessContext, id, batch string, version uint64, index int, call store.ToolCall) error {
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		if _, err := s.findActiveSession(tx, a, id, true); err != nil {
			return err
		}
		var row ToolBatchModel
		if err := tx.Where("id = ? AND session_id = ?", batch, id).First(&row).Error; err != nil {
			return store.ErrContextConflict
		}
		var count int64
		if err := tx.Model(&TurnModel{}).Where("id = ? AND session_id = ? AND completed_at IS NULL", row.TurnID, id).Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return store.ErrContextConflict
		}
		var b store.ToolBatch
		if err := json.Unmarshal(row.Payload, &b); err != nil {
			return err
		}
		if err := store.UpdateToolCall(&b, version, index, call); err != nil {
			return err
		}
		data, err := json.Marshal(b)
		if err != nil {
			return err
		}
		if len(data) > store.MaxToolBatchBytes {
			return store.ErrResourceExhausted
		}
		if err := s.reserveToolBytes(tx, b.TenantID, id, int64(len(data)-len(row.Payload))); err != nil {
			return err
		}
		r := tx.Model(&ToolBatchModel{}).Where("id = ? AND version = ?", batch, version).Updates(map[string]any{"version": b.Version, "payload": data})
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected != 1 {
			return store.ErrContextConflict
		}
		return nil
	})
}

func (s *GormStore) ContextTurns(ctx context.Context, a security.AccessContext, id string, active uint64, limit, maxBytes int) ([]store.TurnContext, error) {
	if limit < 0 || maxBytes <= 0 {
		return nil, store.ErrResourceExhausted
	}
	if limit > s.quota.MaxTurnsPerSession {
		limit = s.quota.MaxTurnsPerSession
	}
	var result []store.TurnContext
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		session, err := s.findActiveSession(tx, a, id, false)
		if err != nil {
			return err
		}
		var current TurnModel
		if err := tx.Where("session_id = ? AND id = ? AND completed_at IS NULL", id, active).First(&current).Error; err != nil {
			return store.ErrTurnNotFound
		}
		var turns []TurnModel
		if limit > 0 {
			if err := tx.Where("session_id = ? AND completed_at IS NOT NULL", id).Order("completed_at DESC").Order("id DESC").Limit(limit).Find(&turns).Error; err != nil {
				return err
			}
		}
		for i, j := 0, len(turns)-1; i < j; i, j = i+1, j-1 {
			turns[i], turns[j] = turns[j], turns[i]
		}
		turns = append(turns, current)
		size := 0
		blocked := false
		for i := len(turns) - 1; i >= 0; i-- {
			t := turns[i]
			item := store.TurnContext{TenantID: session.TenantID, OwnerID: session.OwnerID, ID: t.ID, Status: t.Status}
			if blocked {
				item.ByteLimited = true
				result = append(result, item)
				continue
			}
			var bytes int64
			if err := tx.Model(&MessageModel{}).Select("COALESCE(SUM(LENGTH(payload)),0)").Where("turn_id = ?", t.ID).Scan(&bytes).Error; err != nil {
				return err
			}
			var toolBytes int64
			if err := tx.Model(&ToolBatchModel{}).Select("COALESCE(SUM(LENGTH(payload)),0)").Where("turn_id = ? AND session_id = ?", t.ID, id).Scan(&toolBytes).Error; err != nil {
				return err
			}
			if bytes+toolBytes > int64(maxBytes-size) {
				if t.ID == active {
					return store.ErrResourceExhausted
				}
				blocked = true
				item.ByteLimited = true
				result = append(result, item)
				continue
			}
			messages, err := readTurnMessages(tx, normalizeContext(ctx), t.ID)
			if err != nil {
				return err
			}
			batches, err := readBatches(tx, id, t.ID, s.quota.MaxEventsPerTurn)
			if err != nil {
				return err
			}
			item.Messages = messages
			item.Batches = batches
			data, _ := json.Marshal(item)
			if size+len(data) > maxBytes {
				if t.ID == active {
					return store.ErrResourceExhausted
				}
				blocked = true
				item.ByteLimited = true
				item.Messages = nil
				item.Batches = nil
			} else {
				size += len(data)
			}
			result = append(result, item)
		}
		for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
			result[i], result[j] = result[j], result[i]
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	return result, err
}
