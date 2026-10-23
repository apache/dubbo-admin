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
	"errors"
	"fmt"
	"strings"
	"time"

	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"

	"github.com/firebase/genkit/go/ai"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const expirationCleanupBatchSize = 100

// DefaultMaxTurns matches MemorySpec's default conversation limit.
const DefaultMaxTurns = 100

// GormStore persists sessions, turns, and messages in a relational database.
// It deliberately does not use database foreign keys: relationship checks and
// deletion ordering are handled explicitly by the store transaction.
type GormStore struct {
	db          *gorm.DB
	limit       int
	quota       conversationstore.QuotaPolicy
	lifecycle   conversationstore.LifecyclePolicy
	usagePolicy conversationstore.UsageAuditPolicy
}

var _ conversationstore.BackendStore = (*GormStore)(nil)
var _ conversationstore.UsageAuditMaintenanceStore = (*GormStore)(nil)

// NewGormStore creates a store around an already opened Gorm database. The
// optional limit exists for runtime configuration and tests; the default
// matches MemorySpec.
func NewGormStore(db *gorm.DB, limits ...int) (*GormStore, error) {
	quota := conversationstore.DefaultQuotaPolicy()
	if db == nil {
		return nil, fmt.Errorf("gorm database is nil")
	}
	// The Store owns relationship validation and deletion ordering. Keep Gorm
	// from creating database foreign-key constraints if models gain fields in
	// the future.
	if db.Config == nil {
		db.Config = &gorm.Config{}
	}
	db.Config.DisableForeignKeyConstraintWhenMigrating = true
	limit := DefaultMaxTurns
	if len(limits) > 0 && limits[0] > 0 {
		limit = limits[0]
	}
	quota.MaxTurnsPerSession = limit
	return NewGormStoreWithPolicy(db, quota)
}

func NewGormStoreWithPolicy(db *gorm.DB, quota conversationstore.QuotaPolicy) (*GormStore, error) {
	return NewGormStoreWithPolicies(db, quota, conversationstore.DefaultLifecyclePolicy())
}

func NewGormStoreWithPolicies(db *gorm.DB, quota conversationstore.QuotaPolicy, lifecycle conversationstore.LifecyclePolicy) (*GormStore, error) {
	if db == nil {
		return nil, fmt.Errorf("gorm database is nil")
	}
	if db.Config == nil {
		db.Config = &gorm.Config{}
	}
	db.Config.DisableForeignKeyConstraintWhenMigrating = true
	quota = quota.WithDefaults()
	lifecycle = lifecycle.WithDefaults()
	return &GormStore{db: db, limit: quota.MaxTurnsPerSession, quota: quota, lifecycle: lifecycle, usagePolicy: conversationstore.DefaultUsageAuditPolicy()}, nil
}

func (s *GormStore) SetUsageAuditPolicy(policy conversationstore.UsageAuditPolicy) {
	if s != nil {
		s.usagePolicy = policy.WithDefaults()
	}
}

// Migrate creates or updates the Store tables. Gorm's model associations are
// not declared, so this migration does not create foreign-key constraints.
func (s *GormStore) Migrate(ctx context.Context) error {
	if err := s.checkContext(ctx); err != nil {
		return err
	}
	db := s.db.WithContext(normalizeContext(ctx))
	if err := db.AutoMigrate(
		&HistoryEntryModel{}, &SummaryModel{},
		&ToolBatchModel{},
		&SessionModel{}, &TurnModel{}, &MessageModel{}, &InteractionModel{},
		&DerivedArtifactModel{}, &CleanupJobModel{}, &OutboxEventModel{}, &TenantUsageModel{},
		&UsageEventModel{}, &AuditEventModel{}, &EventCapacityModel{},
	); err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, entry := range []struct {
			kind  string
			model any
		}{{"usage", &UsageEventModel{}}, {"audit", &AuditEventModel{}}} {
			// Only initialize a missing counter. Never recount over a live counter:
			// existing writers may be reserving capacity in other processes.
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&EventCapacityModel{Kind: entry.kind, EventCount: -1})
			if result.Error != nil {
				return result.Error
			}
			// Lock before examining the initialization sentinel. Do not infer
			// insertion from RowsAffected: MySQL clientFoundRows changes its meaning.
			if err := tx.Model(&EventCapacityModel{}).Where("kind = ?", entry.kind).
				Update("event_count", gorm.Expr("event_count + 0")).Error; err != nil {
				return err
			}
			var capacity EventCapacityModel
			if err := tx.First(&capacity, "kind = ?", entry.kind).Error; err != nil {
				return err
			}
			if capacity.EventCount < 0 {
				var count int64
				if err := tx.Model(entry.model).Count(&count).Error; err != nil {
					return err
				}
				if err := tx.Model(&EventCapacityModel{}).Where("kind = ?", entry.kind).Update("event_count", count).Error; err != nil {
					return err
				}
			}
		}
		var artifacts []DerivedArtifactModel
		if err := tx.Where("source_session_id = ?", "").Find(&artifacts).Error; err != nil {
			return err
		}
		for i := range artifacts {
			sessionID, err := conversationstore.SessionIDFromCanonicalSource(artifacts[i].CanonicalSourceID)
			if err != nil {
				continue
			}
			if err := tx.Model(&DerivedArtifactModel{}).Where("id = ?", artifacts[i].ID).Update("source_session_id", sessionID).Error; err != nil {
				return err
			}
		}
		// Pre-PR3 active Turns have no durable Interaction or lease and cannot be
		// resumed safely after deployment.
		var orphaned []TurnModel
		if err := tx.Where("completed_at IS NULL AND (interaction_id IS NULL OR interaction_id = ?)", "").Find(&orphaned).Error; err != nil {
			return err
		}
		for i := range orphaned {
			if err := tx.Where("turn_id = ?", orphaned[i].ID).Delete(&MessageModel{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("completed_at IS NULL AND (interaction_id IS NULL OR interaction_id = ?)", "").Delete(&TurnModel{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&TurnModel{}).Where("completed_at IS NOT NULL").Update("status", conversationstore.InteractionStatusCompleted).Error; err != nil {
			return err
		}
		// Backfill metadata only, preserving the old completed-at/id ordering.
		var legacy []TurnModel
		if err := tx.Where("completed_at IS NOT NULL AND NOT EXISTS (SELECT 1 FROM ai_history_entries h WHERE h.session_id = ai_turns.session_id AND h.turn_id = ai_turns.id)").Order("completed_at ASC").Order("id ASC").Find(&legacy).Error; err != nil {
			return err
		}
		for _, t := range legacy {
			if err := recordSummaryHistory(tx, t.SessionID, t.ID, conversationstore.InteractionStatusCompleted); err != nil {
				return err
			}
		}
		var released []ToolBatchModel
		if err := tx.Select("session_id, turn_id").Where("NOT EXISTS (SELECT 1 FROM ai_turns t WHERE t.id = ai_tool_batches.turn_id) AND NOT EXISTS (SELECT 1 FROM ai_history_entries h WHERE h.session_id = ai_tool_batches.session_id AND h.turn_id = ai_tool_batches.turn_id)").Group("session_id, turn_id").Order("turn_id ASC").Find(&released).Error; err != nil {
			return err
		}
		for _, b := range released {
			if err := recordSummaryHistory(tx, b.SessionID, b.TurnID, conversationstore.InteractionStatusFailed); err != nil {
				return err
			}
		}
		var sessions []SessionModel
		if err := tx.Find(&sessions).Error; err != nil {
			return err
		}
		type usage struct{ sessions, bytes int64 }
		byTenant := make(map[string]usage)
		for i := range sessions {
			if sessions[i].ExpiresAt == nil {
				expiresAt := sessions[i].UpdatedAt.Add(s.lifecycle.Retention)
				if err := tx.Model(&SessionModel{}).Where("id = ?", sessions[i].ID).Update("expires_at", expiresAt).Error; err != nil {
					return err
				}
				sessions[i].ExpiresAt = &expiresAt
			}
			var messageBytes, resultBytes int64
			if err := tx.Table("ai_messages").Select("COALESCE(SUM(LENGTH(ai_messages.payload) + CASE WHEN ai_messages.summary_checked_rule IS NOT NULL AND ai_messages.summary_checked_rule <> '' THEN ? ELSE 0 END), 0)", conversationstore.SummaryInspectionBytes).Joins("JOIN ai_turns ON ai_turns.id = ai_messages.turn_id").Where("ai_turns.session_id = ?", sessions[i].ID).Scan(&messageBytes).Error; err != nil {
				return err
			}
			if err := tx.Model(&InteractionModel{}).Select("COALESCE(SUM(LENGTH(result)), 0)").Where("session_id = ?", sessions[i].ID).Scan(&resultBytes).Error; err != nil {
				return err
			}
			var toolBytes int64
			if err := tx.Model(&ToolBatchModel{}).Select("COALESCE(SUM(LENGTH(payload) + CASE WHEN summary_checked_rule IS NOT NULL AND summary_checked_rule <> '' THEN ? ELSE 0 END), 0)", conversationstore.SummaryInspectionBytes).Where("session_id = ?", sessions[i].ID).Scan(&toolBytes).Error; err != nil {
				return err
			}
			var summaryBytes int64
			if err := tx.Model(&SummaryModel{}).Select("COALESCE(SUM(LENGTH(payload)),0)").Where("session_id = ?", sessions[i].ID).Scan(&summaryBytes).Error; err != nil {
				return err
			}
			storedBytes := messageBytes + resultBytes + toolBytes + summaryBytes
			if sessions[i].Status == conversationstore.SessionStatusDeleted {
				storedBytes = 0
			}
			if err := tx.Model(&SessionModel{}).Where("id = ?", sessions[i].ID).Update("stored_bytes", storedBytes).Error; err != nil {
				return err
			}
			current := byTenant[sessions[i].TenantID]
			if sessions[i].Status != conversationstore.SessionStatusDeleted {
				current.sessions++
			}
			current.bytes += storedBytes
			byTenant[sessions[i].TenantID] = current
		}
		if err := tx.Model(&TenantUsageModel{}).Where("1 = 1").Updates(map[string]any{"session_count": 0, "stored_bytes": 0}).Error; err != nil {
			return err
		}
		for tenantID, current := range byTenant {
			model := TenantUsageModel{TenantID: tenantID}
			if err := tx.Where("tenant_id = ?", tenantID).FirstOrCreate(&model).Error; err != nil {
				return err
			}
			if err := tx.Model(&TenantUsageModel{}).Where("tenant_id = ?", tenantID).Updates(map[string]any{"session_count": current.sessions, "stored_bytes": current.bytes}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// DB returns the underlying database for connection-pool configuration and
// test inspection. Callers must not replace the database instance.
func (s *GormStore) DB() *gorm.DB { return s.db }

// Close closes the underlying SQL database connection.
func (s *GormStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *GormStore) Create(ctx context.Context, access security.AccessContext, session *conversationstore.Session) error {
	if err := s.checkContext(ctx); err != nil {
		return err
	}
	if err := access.ValidateBusiness(); err != nil {
		return err
	}
	if err := conversationstore.ValidateSession(session); err != nil {
		return err
	}
	if !access.CanAccessOwner(session.TenantID, session.OwnerID) {
		return security.ErrResourceNotAccessible
	}
	model := sessionModelFromDomain(session)
	if model.ExpiresAt == nil {
		expiresAt := model.UpdatedAt.Add(s.lifecycle.Retention)
		model.ExpiresAt = &expiresAt
	}
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		usage := TenantUsageModel{TenantID: session.TenantID}
		if err := tx.Where("tenant_id = ?", session.TenantID).FirstOrCreate(&usage).Error; err != nil {
			return err
		}
		reserved := tx.Model(&TenantUsageModel{}).Where("tenant_id = ? AND session_count < ?", session.TenantID, s.quota.MaxSessionsPerTenant).Update("session_count", gorm.Expr("session_count + 1"))
		if reserved.Error != nil {
			return reserved.Error
		}
		if reserved.RowsAffected != 1 {
			return conversationstore.ErrResourceExhausted
		}
		return tx.Create(&model).Error
	})
}

func (s *GormStore) Get(ctx context.Context, access security.AccessContext, sessionID string) (*conversationstore.Session, error) {
	if err := s.checkContext(ctx); err != nil {
		return nil, err
	}
	if err := access.ValidateBusiness(); err != nil {
		return nil, err
	}
	model, err := s.findSession(s.db.WithContext(normalizeContext(ctx)), access, sessionID, false)
	if err != nil {
		return nil, err
	}
	if model.Status != conversationstore.SessionStatusActive {
		return nil, conversationstore.ErrSessionNotFound
	}
	if s.isExpired(model, time.Now()) {
		return nil, conversationstore.ErrSessionExpired
	}
	return sessionDomainFromModel(model), nil
}

func (s *GormStore) List(ctx context.Context, access security.AccessContext) ([]*conversationstore.Session, error) {
	if err := s.checkContext(ctx); err != nil {
		return nil, err
	}
	if err := access.ValidateBusiness(); err != nil {
		return nil, err
	}
	var models []SessionModel
	now := time.Now()
	cutoff := now.Add(-s.lifecycle.Retention)
	err := scopeSessions(s.db.WithContext(normalizeContext(ctx)), access).
		Where("status = ? AND ((expires_at IS NOT NULL AND expires_at > ?) OR (expires_at IS NULL AND updated_at >= ?))", "active", now, cutoff).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	result := make([]*conversationstore.Session, 0, len(models))
	for i := range models {
		if !access.CanAccessOwner(models[i].TenantID, models[i].OwnerID) {
			continue
		}
		result = append(result, sessionDomainFromModel(&models[i]))
	}
	return result, nil
}

func (s *GormStore) Touch(ctx context.Context, access security.AccessContext, sessionID string, updatedAt time.Time, retentions ...time.Duration) error {
	if err := s.checkContext(ctx); err != nil {
		return err
	}
	if err := access.ValidateBusiness(); err != nil {
		return err
	}
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		_, err := s.findActiveSession(tx, access, sessionID, true)
		if err != nil {
			return err
		}
		retention := s.lifecycle.Retention
		if len(retentions) > 0 && retentions[0] > 0 {
			retention = retentions[0]
		}
		expiresAt := updatedAt.Add(retention)
		return scopeSessions(tx.Model(&SessionModel{}), access).Where("id = ?", sessionID).Updates(map[string]any{
			"updated_at": updatedAt,
			"expires_at": expiresAt,
			"version":    gorm.Expr("version + 1"),
		}).Error
	})
}

func (s *GormStore) Delete(ctx context.Context, access security.AccessContext, sessionID string) error {
	if err := s.checkContext(ctx); err != nil {
		return err
	}
	if err := access.ValidateBusiness(); err != nil {
		return err
	}
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		if _, err := s.findActiveSession(tx, access, sessionID, true); err != nil {
			return err
		}
		return markSessionForDeletion(tx, sessionID, access.Subject.TenantID, time.Now())
	})
}

func (s *GormStore) DeleteExpired(ctx context.Context, access security.AccessContext, now time.Time, batchSizes ...int) (int, error) {
	if err := s.checkContext(ctx); err != nil {
		return 0, err
	}
	if err := access.Validate(); err != nil {
		return 0, err
	}
	if !access.IsSystem() {
		return 0, security.ErrResourceNotAccessible
	}
	cutoff := now.Add(-s.lifecycle.Retention)
	batchLimit := expirationCleanupBatchSize
	if len(batchSizes) > 0 && batchSizes[0] > 0 {
		batchLimit = batchSizes[0]
	}
	deleted := 0
	for {
		batchDeleted := 0
		err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
			query := tx.Where("((status = ?) AND ((expires_at IS NOT NULL AND expires_at <= ?) OR (expires_at IS NULL AND updated_at < ?))) OR status IN ?", conversationstore.SessionStatusActive, now, cutoff, []string{conversationstore.SessionStatusDeletionPending, conversationstore.SessionStatusDerivedCleanup}).Order("updated_at ASC, tenant_id ASC, id ASC").Limit(batchLimit)
			if supportsRowLock(tx) {
				query = query.Clauses(clause.Locking{Strength: "UPDATE"})
			}
			var sessions []SessionModel
			if err := query.Find(&sessions).Error; err != nil {
				return err
			}
			for i := range sessions {
				// Re-read the row after acquiring the transaction lock. This prevents
				// cleanup from deleting a session refreshed by another instance.
				current, err := s.findSession(tx, access, sessions[i].ID, true)
				if err != nil {
					if errors.Is(err, conversationstore.ErrSessionNotFound) {
						continue
					}
					return err
				}
				if current.Status == conversationstore.SessionStatusActive && !s.isExpired(current, now) {
					continue
				}
				if current.Status == conversationstore.SessionStatusActive {
					if err := markSessionForDeletion(tx, current.ID, current.TenantID, now); err != nil {
						return err
					}
				}
				if err := cleanupSessionData(tx, current.ID, current.TenantID, now); err != nil {
					return err
				}
				var cleaned SessionModel
				if err := tx.Select("status").Where("id = ?", current.ID).First(&cleaned).Error; err != nil {
					return err
				}
				if cleaned.Status == conversationstore.SessionStatusDeleted {
					batchDeleted++
				}
			}
			return nil
		})
		if err != nil {
			return deleted, err
		}
		deleted += batchDeleted
		return deleted, nil
	}
}

func (s *GormStore) DeleteExpiredPage(ctx context.Context, access security.AccessContext, now time.Time, cursor conversationstore.CleanupCursor, limit int) (int, conversationstore.CleanupCursor, error) {
	if err := s.checkContext(ctx); err != nil {
		return 0, cursor, err
	}
	if err := access.Validate(); err != nil {
		return 0, cursor, err
	}
	if !access.IsSystem() {
		return 0, cursor, security.ErrResourceNotAccessible
	}
	if limit <= 0 {
		limit = expirationCleanupBatchSize
	}
	cutoff := now.Add(-s.lifecycle.Retention)
	var sessions []SessionModel
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		query := tx.Where("((status = ?) AND ((expires_at IS NOT NULL AND expires_at <= ?) OR (expires_at IS NULL AND updated_at < ?))) OR status IN ?", conversationstore.SessionStatusActive, now, cutoff, []string{conversationstore.SessionStatusDeletionPending, conversationstore.SessionStatusDerivedCleanup})
		if cursor.Valid {
			query = query.Where("(updated_at > ?) OR (updated_at = ? AND tenant_id > ?) OR (updated_at = ? AND tenant_id = ? AND id > ?)", cursor.SortTime, cursor.SortTime, cursor.TenantID, cursor.SortTime, cursor.TenantID, cursor.ID)
		}
		query = query.Order("updated_at ASC, tenant_id ASC, id ASC").Limit(limit)
		if supportsRowLock(tx) {
			query = query.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		return query.Find(&sessions).Error
	})
	if err != nil {
		return 0, cursor, err
	}
	next := conversationstore.CleanupCursor{}
	if len(sessions) > 0 {
		last := sessions[len(sessions)-1]
		next = conversationstore.CleanupCursor{SortTime: last.UpdatedAt, TenantID: last.TenantID, ID: last.ID, Valid: true}
	}
	deleted := 0
	for i := range sessions {
		if err := ctx.Err(); err != nil {
			return deleted, next, err
		}
		if err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
			current, err := s.findSession(tx, access, sessions[i].ID, true)
			if err != nil {
				return err
			}
			if current.Status == conversationstore.SessionStatusActive && !s.isExpired(current, now) {
				return nil
			}
			if current.Status == conversationstore.SessionStatusActive {
				if err := markSessionForDeletion(tx, current.ID, current.TenantID, now); err != nil {
					return err
				}
			}
			if err := cleanupSessionData(tx, current.ID, current.TenantID, now); err != nil {
				return err
			}
			deleted++
			return nil
		}); err != nil {
			return deleted, next, err
		}
	}
	return deleted, next, nil
}

func markSessionForDeletion(tx *gorm.DB, sessionID, tenantID string, now time.Time) error {
	result := tx.Model(&SessionModel{}).Where("id = ? AND status = ?", sessionID, conversationstore.SessionStatusActive).Updates(map[string]any{
		"status": conversationstore.SessionStatusDeletionPending, "version": gorm.Expr("version + 1"),
		"updated_at": now, "active_interaction_id": nil,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}
	canonicalID := "session:" + sessionID
	job := CleanupJobModel{ID: uuid.NewString(), TenantID: tenantID, CanonicalSourceID: canonicalID, Status: "pending", CreatedAt: now, UpdatedAt: now}
	if err := tx.Where("canonical_source_id = ?", canonicalID).FirstOrCreate(&job).Error; err != nil {
		return err
	}
	return tx.Create(&OutboxEventModel{ID: uuid.NewString(), TenantID: tenantID, AggregateID: canonicalID, EventType: "session.deletion_requested", CreatedAt: now}).Error
}

func cleanupSessionData(tx *gorm.DB, sessionID, tenantID string, now time.Time) error {
	canonicalID := "session:" + sessionID
	var session SessionModel
	if err := tx.Select("stored_bytes").Where("id = ?", sessionID).First(&session).Error; err != nil {
		return err
	}
	if err := tx.Model(&SessionModel{}).Where("id = ? AND status = ?", sessionID, conversationstore.SessionStatusDeletionPending).Update("status", conversationstore.SessionStatusDerivedCleanup).Error; err != nil {
		return err
	}
	var artifactCount int64
	if err := tx.Model(&DerivedArtifactModel{}).Where("tenant_id = ? AND canonical_source_id = ? AND status <> ?", tenantID, canonicalID, "deleted").Count(&artifactCount).Error; err != nil {
		return err
	}
	if artifactCount > 0 {
		return tx.Model(&CleanupJobModel{}).Where("canonical_source_id = ?", canonicalID).Updates(map[string]any{"status": "cleanup_failed", "attempts": gorm.Expr("attempts + 1"), "last_error_code": "HANDLER_REQUIRED", "updated_at": now}).Error
	}
	var turns []TurnModel
	if err := tx.Where("session_id = ?", sessionID).Find(&turns).Error; err != nil {
		return err
	}
	for i := range turns {
		if err := tx.Where("turn_id = ?", turns[i].ID).Delete(&MessageModel{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("session_id = ?", sessionID).Delete(&TurnModel{}).Error; err != nil {
		return err
	}
	if err := tx.Where("session_id = ?", sessionID).Delete(&InteractionModel{}).Error; err != nil {
		return err
	}
	if err := tx.Where("session_id = ?", sessionID).Delete(&ToolBatchModel{}).Error; err != nil {
		return err
	}
	if err := tx.Where("session_id = ?", sessionID).Delete(&HistoryEntryModel{}).Error; err != nil {
		return err
	}
	if err := tx.Where("session_id = ?", sessionID).Delete(&SummaryModel{}).Error; err != nil {
		return err
	}
	if err := tx.Model(&TenantUsageModel{}).Where("tenant_id = ?", tenantID).Updates(map[string]any{
		"session_count": gorm.Expr("CASE WHEN session_count > 0 THEN session_count - 1 ELSE 0 END"),
		"stored_bytes":  gorm.Expr("CASE WHEN stored_bytes >= ? THEN stored_bytes - ? ELSE 0 END", session.StoredBytes, session.StoredBytes),
	}).Error; err != nil {
		return err
	}
	if err := tx.Model(&SessionModel{}).Where("id = ?", sessionID).Updates(map[string]any{"status": conversationstore.SessionStatusDeleted, "updated_at": now, "active_interaction_id": nil, "stored_bytes": 0}).Error; err != nil {
		return err
	}
	return tx.Model(&CleanupJobModel{}).Where("canonical_source_id = ?", canonicalID).Updates(map[string]any{"status": "completed", "attempts": gorm.Expr("attempts + 1"), "last_error_code": "", "next_retry_at": nil, "updated_at": now}).Error
}

func (s *GormStore) RegisterDerivedArtifact(ctx context.Context, access security.AccessContext, artifact conversationstore.DerivedArtifact) error {
	if err := s.checkContext(ctx); err != nil {
		return err
	}
	if err := access.Validate(); err != nil {
		return err
	}
	if !access.IsSystem() || artifact.ID == "" || artifact.TenantID == "" || artifact.CanonicalSourceID == "" || artifact.Kind == "" || artifact.Locator == "" {
		return security.ErrResourceNotAccessible
	}
	sessionID, err := conversationstore.SessionIDFromCanonicalSource(artifact.CanonicalSourceID)
	if err != nil {
		return err
	}
	now := time.Now()
	model := DerivedArtifactModel{ID: artifact.ID, TenantID: artifact.TenantID, CanonicalSourceID: artifact.CanonicalSourceID, SourceSessionID: sessionID, Kind: artifact.Kind, Locator: artifact.Locator, Status: "active", CreatedAt: now, UpdatedAt: now}
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		query := tx.Where("id = ? AND tenant_id = ? AND status = ?", sessionID, artifact.TenantID, conversationstore.SessionStatusActive)
		if supportsRowLock(tx) {
			query = query.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		var session SessionModel
		if err := query.First(&session).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return conversationstore.ErrCanonicalSourceNotWritable
		} else if err != nil {
			return err
		}
		return tx.Create(&model).Error
	})
}

func (s *GormStore) ListDerivedArtifactsForCleanup(ctx context.Context, access security.AccessContext, now time.Time, limit int) ([]conversationstore.DerivedArtifact, error) {
	if err := s.checkContext(ctx); err != nil {
		return nil, err
	}
	if err := access.Validate(); err != nil {
		return nil, err
	}
	if !access.IsSystem() {
		return nil, security.ErrResourceNotAccessible
	}
	query := s.db.WithContext(normalizeContext(ctx)).Table("ai_derived_artifacts AS artifact").
		Select("artifact.*").
		Joins("LEFT JOIN ai_sessions AS session ON artifact.source_session_id = session.id").
		Where("artifact.status <> ? AND (artifact.next_retry_at IS NULL OR artifact.next_retry_at <= ?) AND (session.id IS NULL OR session.tenant_id <> artifact.tenant_id OR session.status IN ?)", "deleted", now, []string{conversationstore.SessionStatusDeletionPending, conversationstore.SessionStatusDerivedCleanup, conversationstore.SessionStatusDeleted}).
		Order("COALESCE(artifact.next_retry_at, artifact.created_at) ASC, artifact.tenant_id ASC, artifact.id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	var models []DerivedArtifactModel
	if err := query.Scan(&models).Error; err != nil {
		return nil, err
	}
	result := make([]conversationstore.DerivedArtifact, 0, len(models))
	for i := range models {
		result = append(result, conversationstore.DerivedArtifact{
			ID: models[i].ID, TenantID: models[i].TenantID, CanonicalSourceID: models[i].CanonicalSourceID,
			Kind: models[i].Kind, Locator: models[i].Locator, Status: models[i].Status,
			CleanupAttempts: models[i].CleanupAttempts, LastErrorCode: models[i].LastErrorCode,
			NextRetryAt: models[i].NextRetryAt, CreatedAt: models[i].CreatedAt, UpdatedAt: models[i].UpdatedAt,
		})
	}
	return result, nil
}

func (s *GormStore) ListDerivedArtifactsForCleanupPage(ctx context.Context, access security.AccessContext, now time.Time, cursor conversationstore.CleanupCursor, limit int) ([]conversationstore.DerivedArtifact, conversationstore.CleanupCursor, error) {
	if err := s.checkContext(ctx); err != nil {
		return nil, cursor, err
	}
	if err := access.Validate(); err != nil {
		return nil, cursor, err
	}
	if !access.IsSystem() {
		return nil, cursor, security.ErrResourceNotAccessible
	}
	if limit <= 0 {
		limit = expirationCleanupBatchSize
	}
	query := s.db.WithContext(normalizeContext(ctx)).Table("ai_derived_artifacts AS artifact").Select("artifact.*").Joins("LEFT JOIN ai_sessions AS session ON artifact.source_session_id = session.id").Where("artifact.status <> ? AND (artifact.next_retry_at IS NULL OR artifact.next_retry_at <= ?) AND (session.id IS NULL OR session.tenant_id <> artifact.tenant_id OR session.status IN ?)", "deleted", now, []string{conversationstore.SessionStatusDeletionPending, conversationstore.SessionStatusDerivedCleanup, conversationstore.SessionStatusDeleted})
	if cursor.Valid {
		query = query.Where("(COALESCE(artifact.next_retry_at, artifact.created_at) > ?) OR (COALESCE(artifact.next_retry_at, artifact.created_at) = ? AND artifact.tenant_id > ?) OR (COALESCE(artifact.next_retry_at, artifact.created_at) = ? AND artifact.tenant_id = ? AND artifact.id > ?)", cursor.SortTime, cursor.SortTime, cursor.TenantID, cursor.SortTime, cursor.TenantID, cursor.ID)
	}
	query = query.Order("COALESCE(artifact.next_retry_at, artifact.created_at) ASC, artifact.tenant_id ASC, artifact.id ASC").Limit(limit)
	var models []DerivedArtifactModel
	if err := query.Scan(&models).Error; err != nil {
		return nil, cursor, err
	}
	result := make([]conversationstore.DerivedArtifact, 0, len(models))
	for i := range models {
		result = append(result, conversationstore.DerivedArtifact{ID: models[i].ID, TenantID: models[i].TenantID, CanonicalSourceID: models[i].CanonicalSourceID, Kind: models[i].Kind, Locator: models[i].Locator, Status: models[i].Status, CleanupAttempts: models[i].CleanupAttempts, LastErrorCode: models[i].LastErrorCode, NextRetryAt: models[i].NextRetryAt, CreatedAt: models[i].CreatedAt, UpdatedAt: models[i].UpdatedAt})
	}
	next := conversationstore.CleanupCursor{}
	if len(result) > 0 {
		last := result[len(result)-1]
		at := last.CreatedAt
		if last.NextRetryAt != nil {
			at = *last.NextRetryAt
		}
		next = conversationstore.CleanupCursor{SortTime: at, TenantID: last.TenantID, ID: last.ID, Valid: true}
	}
	return result, next, nil
}

func (s *GormStore) MarkDerivedArtifactDeleted(ctx context.Context, access security.AccessContext, artifactID string) error {
	if err := s.checkContext(ctx); err != nil {
		return err
	}
	if err := access.Validate(); err != nil {
		return err
	}
	if !access.IsSystem() {
		return security.ErrResourceNotAccessible
	}
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		var artifact DerivedArtifactModel
		query := tx.Where("id = ?", artifactID)
		if supportsRowLock(tx) {
			query = query.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if err := query.First(&artifact).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return conversationstore.ErrDerivedArtifactNotFound
		} else if err != nil {
			return err
		}
		now := time.Now()
		if err := tx.Model(&DerivedArtifactModel{}).Where("id = ?", artifactID).Updates(map[string]any{
			"status": "deleted", "last_error_code": "", "next_retry_at": nil, "updated_at": now,
		}).Error; err != nil {
			return err
		}
		return refreshCleanupJobRetryProjection(tx, artifact.CanonicalSourceID, now)
	})
}

func (s *GormStore) MarkDerivedArtifactCleanupFailed(ctx context.Context, access security.AccessContext, artifactID, errorCode string, failedAt time.Time) error {
	if err := s.checkContext(ctx); err != nil {
		return err
	}
	if err := access.Validate(); err != nil {
		return err
	}
	if !access.IsSystem() {
		return security.ErrResourceNotAccessible
	}
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		var artifact DerivedArtifactModel
		query := tx.Where("id = ?", artifactID)
		if supportsRowLock(tx) {
			query = query.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if err := query.First(&artifact).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return conversationstore.ErrDerivedArtifactNotFound
		} else if err != nil {
			return err
		}
		if artifact.Status == "deleted" {
			return nil
		}
		attempt := artifact.CleanupAttempts + 1
		nextRetryAt := conversationstore.DerivedArtifactRetryAt(failedAt, attempt)
		if err := tx.Model(&DerivedArtifactModel{}).Where("id = ? AND status <> ?", artifactID, "deleted").Updates(map[string]any{
			"cleanup_attempts": attempt, "last_error_code": errorCode, "next_retry_at": nextRetryAt, "updated_at": failedAt,
		}).Error; err != nil {
			return err
		}
		var job CleanupJobModel
		err := tx.Where("canonical_source_id = ?", artifact.CanonicalSourceID).First(&job).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			job = CleanupJobModel{
				ID:                uuid.NewString(),
				TenantID:          artifact.TenantID,
				CanonicalSourceID: artifact.CanonicalSourceID,
				Status:            "cleanup_failed",
				CreatedAt:         failedAt,
				UpdatedAt:         failedAt,
			}
			if err := tx.Create(&job).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		}
		result := tx.Model(&CleanupJobModel{}).Where("canonical_source_id = ?", artifact.CanonicalSourceID).Updates(map[string]any{
			"status": "cleanup_failed", "attempts": gorm.Expr("attempts + 1"), "last_error_code": errorCode, "updated_at": failedAt,
		})
		if result.Error != nil {
			return result.Error
		}
		return refreshCleanupJobRetryProjection(tx, artifact.CanonicalSourceID, failedAt)
	})
}

// refreshCleanupJobRetryProjection keeps the canonical-source job useful for
// observation without making it a second scheduler. A nil projection means at
// least one pending artifact is immediately eligible (or none remain).
func refreshCleanupJobRetryProjection(tx *gorm.DB, canonicalSourceID string, updatedAt time.Time) error {
	var job CleanupJobModel
	jobQuery := tx.Select("id").Where("canonical_source_id = ?", canonicalSourceID)
	if supportsRowLock(tx) {
		jobQuery = jobQuery.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := jobQuery.First(&job).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}
	var immediate int64
	if err := tx.Model(&DerivedArtifactModel{}).
		Where("canonical_source_id = ? AND status <> ? AND next_retry_at IS NULL", canonicalSourceID, "deleted").
		Count(&immediate).Error; err != nil {
		return err
	}
	var nextRetryAt *time.Time
	if immediate == 0 {
		var artifact DerivedArtifactModel
		err := tx.Select("next_retry_at").
			Where("canonical_source_id = ? AND status <> ?", canonicalSourceID, "deleted").
			Order("next_retry_at ASC").First(&artifact).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil {
			nextRetryAt = artifact.NextRetryAt
		}
	}
	return tx.Model(&CleanupJobModel{}).
		Where("canonical_source_id = ?", canonicalSourceID).
		Updates(map[string]any{"next_retry_at": nextRetryAt, "updated_at": updatedAt}).Error
}

func (s *GormStore) AdoptLegacySessions(ctx context.Context, access security.AccessContext, tenantID, ownerID string) (int, error) {
	if err := s.checkContext(ctx); err != nil {
		return 0, err
	}
	if err := access.Validate(); err != nil {
		return 0, err
	}
	if !access.IsSystem() {
		return 0, security.ErrResourceNotAccessible
	}
	if tenantID == "" || ownerID == "" {
		return 0, errors.New("legacy session target tenant_id and owner_id are required")
	}
	var legacy []SessionModel
	if err := s.db.WithContext(normalizeContext(ctx)).
		Where("(tenant_id IS NULL OR tenant_id = ?) AND (owner_id IS NULL OR owner_id = ?)", "", "").
		Find(&legacy).Error; err != nil {
		return 0, err
	}
	updated := 0
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		for i := range legacy {
			version := legacy[i].Version
			if version == 0 {
				version = 1
			}
			expiresAt := legacy[i].ExpiresAt
			if expiresAt == nil {
				value := legacy[i].UpdatedAt.Add(s.lifecycle.Retention)
				expiresAt = &value
			}
			result := tx.Model(&SessionModel{}).
				Where("id = ? AND (tenant_id IS NULL OR tenant_id = ?) AND (owner_id IS NULL OR owner_id = ?)", legacy[i].ID, "", "").
				Updates(map[string]any{
					"tenant_id":  tenantID,
					"owner_id":   ownerID,
					"version":    version,
					"expires_at": expiresAt,
				})
			if result.Error != nil {
				return result.Error
			}
			updated += int(result.RowsAffected)
		}
		return nil
	})
	return updated, err
}

func (s *GormStore) PrepareInteraction(ctx context.Context, access security.AccessContext, sessionID, idempotencyKey, requestHash string, leaseDuration time.Duration) (*conversationstore.PreparedInteraction, error) {
	if err := s.checkContext(ctx); err != nil {
		return nil, err
	}
	if idempotencyKey == "" || requestHash == "" || leaseDuration <= 0 {
		return nil, errors.New("idempotency key, request hash, and positive lease are required")
	}
	var prepared conversationstore.PreparedInteraction
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		session, err := s.findActiveSession(tx, access, sessionID, true)
		if err != nil {
			return err
		}
		now := time.Now()
		var existing InteractionModel
		err = tx.Where("tenant_id = ? AND owner_id = ? AND session_id = ? AND idempotency_key = ?", access.Subject.TenantID, access.Subject.UserID, sessionID, idempotencyKey).First(&existing).Error
		if err == nil {
			if existing.RequestHash != requestHash {
				return conversationstore.ErrIdempotencyConflict
			}
			if existing.Status == conversationstore.InteractionStatusCompleted {
				prepared = conversationstore.PreparedInteraction{InteractionID: existing.ID, TurnID: existing.TurnID, Replayed: true, Result: existing.Result}
				return nil
			}
			if existing.Status == conversationstore.InteractionStatusRunning && existing.LeaseExpiresAt.After(now) {
				return conversationstore.ErrInteractionInProgress
			}
			var completedTurnCount int64
			if err := tx.Model(&TurnModel{}).Where("session_id = ? AND completed_at IS NOT NULL", sessionID).Count(&completedTurnCount).Error; err != nil {
				return err
			}
			if completedTurnCount >= int64(s.limit) {
				return conversationstore.ErrResourceExhausted
			}
			if existing.TurnID != 0 {
				if err := s.deleteTurnAndReleaseUsage(tx, access.Subject.TenantID, sessionID, existing.TurnID); err != nil && !errors.Is(err, conversationstore.ErrTurnNotFound) {
					return err
				}
			}
			turn := TurnModel{SessionID: sessionID, InteractionID: existing.ID, Status: conversationstore.InteractionStatusRunning, CreatedAt: now}
			if err := checkSummaryHistoryCapacity(tx, sessionID); err != nil {
				return err
			}
			if err := tx.Create(&turn).Error; err != nil {
				return err
			}
			updates := map[string]any{"turn_id": turn.ID, "status": conversationstore.InteractionStatusRunning, "attempt": gorm.Expr("attempt + 1"), "result": "", "lease_expires_at": now.Add(leaseDuration), "updated_at": now, "completed_at": nil}
			if err := tx.Model(&InteractionModel{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
				return err
			}
			if err := tx.Model(&SessionModel{}).Where("id = ?", sessionID).Update("active_interaction_id", existing.ID).Error; err != nil {
				return err
			}
			prepared = conversationstore.PreparedInteraction{InteractionID: existing.ID, TurnID: turn.ID}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if session.ActiveInteractionID != nil && *session.ActiveInteractionID != "" {
			var active InteractionModel
			activeErr := tx.Where("id = ?", *session.ActiveInteractionID).First(&active).Error
			if activeErr == nil && active.Status == conversationstore.InteractionStatusRunning && active.LeaseExpiresAt.After(now) {
				return conversationstore.ErrSessionBusy
			}
			if activeErr == nil {
				if active.TurnID != 0 {
					if err := s.deleteTurnAndReleaseUsage(tx, access.Subject.TenantID, sessionID, active.TurnID); err != nil && !errors.Is(err, conversationstore.ErrTurnNotFound) {
						return err
					}
				}
				if err := tx.Model(&InteractionModel{}).Where("id = ?", active.ID).Updates(map[string]any{"status": conversationstore.InteractionStatusFailed, "updated_at": now}).Error; err != nil {
					return err
				}
			} else if !errors.Is(activeErr, gorm.ErrRecordNotFound) {
				return activeErr
			}
			if err := tx.Model(&SessionModel{}).Where("id = ?", sessionID).Update("active_interaction_id", nil).Error; err != nil {
				return err
			}
		}
		var completedTurnCount int64
		if err := tx.Model(&TurnModel{}).Where("session_id = ? AND completed_at IS NOT NULL", sessionID).Count(&completedTurnCount).Error; err != nil {
			return err
		}
		if completedTurnCount >= int64(s.limit) {
			return conversationstore.ErrResourceExhausted
		}
		interactionID := uuid.NewString()
		if err := checkSummaryHistoryCapacity(tx, sessionID); err != nil {
			return err
		}
		turn := TurnModel{SessionID: sessionID, InteractionID: interactionID, Status: conversationstore.InteractionStatusRunning, CreatedAt: now}
		if err := tx.Create(&turn).Error; err != nil {
			return err
		}
		interaction := InteractionModel{
			ID: interactionID, TenantID: access.Subject.TenantID, OwnerID: access.Subject.UserID,
			SessionID: sessionID, IdempotencyKey: idempotencyKey, RequestHash: requestHash,
			TurnID: turn.ID, Status: conversationstore.InteractionStatusRunning, Attempt: 1,
			LeaseExpiresAt: now.Add(leaseDuration), CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&interaction).Error; err != nil {
			return err
		}
		result := tx.Model(&SessionModel{}).Where("id = ? AND active_interaction_id IS NULL", sessionID).Update("active_interaction_id", interactionID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return conversationstore.ErrSessionBusy
		}
		prepared = conversationstore.PreparedInteraction{InteractionID: interactionID, TurnID: turn.ID}
		return nil
	})
	if err != nil {
		// A concurrent insert can win the composite idempotency constraint
		// between our lookup and create. Resolve that race to the same stable
		// errors as the non-racing path instead of leaking a driver error.
		var existing InteractionModel
		lookupErr := s.db.WithContext(normalizeContext(ctx)).Where("tenant_id = ? AND owner_id = ? AND session_id = ? AND idempotency_key = ?", access.Subject.TenantID, access.Subject.UserID, sessionID, idempotencyKey).First(&existing).Error
		if lookupErr == nil {
			if existing.RequestHash != requestHash {
				return nil, conversationstore.ErrIdempotencyConflict
			}
			if existing.Status == conversationstore.InteractionStatusCompleted {
				return &conversationstore.PreparedInteraction{InteractionID: existing.ID, TurnID: existing.TurnID, Replayed: true, Result: existing.Result}, nil
			}
			if existing.Status == conversationstore.InteractionStatusRunning {
				return nil, conversationstore.ErrInteractionInProgress
			}
		}
		return nil, err
	}
	return &prepared, nil
}

func (s *GormStore) CompleteInteraction(ctx context.Context, access security.AccessContext, sessionID, interactionID string, turnID uint64, result string) error {
	if err := s.checkContext(ctx); err != nil {
		return err
	}
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		if _, err := s.findActiveSession(tx, access, sessionID, true); err != nil {
			return err
		}
		resultBytes := int64(len([]byte(result)))
		sessionReserved := tx.Model(&SessionModel{}).Where("id = ? AND stored_bytes + ? <= ?", sessionID, resultBytes, s.quota.MaxSessionBytes).Update("stored_bytes", gorm.Expr("stored_bytes + ?", resultBytes))
		if sessionReserved.Error != nil {
			return sessionReserved.Error
		}
		if sessionReserved.RowsAffected != 1 {
			return conversationstore.ErrResourceExhausted
		}
		tenantReserved := tx.Model(&TenantUsageModel{}).Where("tenant_id = ? AND stored_bytes + ? <= ?", access.Subject.TenantID, resultBytes, s.quota.MaxTenantBytes).Update("stored_bytes", gorm.Expr("stored_bytes + ?", resultBytes))
		if tenantReserved.Error != nil {
			return tenantReserved.Error
		}
		if tenantReserved.RowsAffected != 1 {
			return conversationstore.ErrResourceExhausted
		}
		now := time.Now()
		updated := tx.Model(&InteractionModel{}).Where("id = ? AND session_id = ? AND turn_id = ? AND status = ?", interactionID, sessionID, turnID, conversationstore.InteractionStatusRunning).Updates(map[string]any{"status": conversationstore.InteractionStatusCompleted, "result": result, "updated_at": now, "completed_at": now})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return conversationstore.ErrTurnNotFound
		}
		if err := tx.Model(&TurnModel{}).Where("id = ? AND session_id = ?", turnID, sessionID).Updates(map[string]any{"status": conversationstore.InteractionStatusCompleted, "completed_at": now}).Error; err != nil {
			return err
		}
		if err := recordSummaryHistory(tx, sessionID, turnID, conversationstore.InteractionStatusCompleted); err != nil {
			return err
		}
		return tx.Model(&SessionModel{}).Where("id = ? AND active_interaction_id = ?", sessionID, interactionID).Update("active_interaction_id", nil).Error
	})
}

func (s *GormStore) FailInteraction(ctx context.Context, access security.AccessContext, sessionID, interactionID string, turnID uint64) error {
	if err := s.checkContext(ctx); err != nil {
		return err
	}
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		if _, err := s.findActiveSession(tx, access, sessionID, true); err != nil {
			return err
		}
		if err := s.deleteTurnAndReleaseUsage(tx, access.Subject.TenantID, sessionID, turnID); err != nil {
			return err
		}
		updated := tx.Model(&InteractionModel{}).Where("id = ? AND session_id = ? AND turn_id = ?", interactionID, sessionID, turnID).Updates(map[string]any{"status": conversationstore.InteractionStatusFailed, "updated_at": time.Now()})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return conversationstore.ErrTurnNotFound
		}
		return tx.Model(&SessionModel{}).Where("id = ? AND active_interaction_id = ?", sessionID, interactionID).Update("active_interaction_id", nil).Error
	})
}

func (s *GormStore) BeginTurn(ctx context.Context, access security.AccessContext, sessionID string) (uint64, error) {
	if err := s.checkContext(ctx); err != nil {
		return 0, err
	}
	var turnID uint64
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		session, err := s.findActiveSession(tx, access, sessionID, true)
		if err != nil {
			return err
		}
		if err := checkSummaryHistoryCapacity(tx, sessionID); err != nil {
			return err
		}
		if session.ActiveInteractionID != nil && *session.ActiveInteractionID != "" {
			return conversationstore.ErrSessionBusy
		}
		var activeCount int64
		if err := tx.Model(&TurnModel{}).Where("session_id = ? AND completed_at IS NULL", sessionID).Count(&activeCount).Error; err != nil {
			return err
		}
		if activeCount > 0 {
			return conversationstore.ErrSessionBusy
		}

		var turnCount int64
		if err := tx.Model(&TurnModel{}).
			Where("session_id = ?", sessionID).
			Count(&turnCount).Error; err != nil {
			return err
		}
		if turnCount >= int64(s.limit) {
			return fmt.Errorf("%w: current session's context is full, please create a new session", conversationstore.ErrTurnLimitReached)
		}
		turn := TurnModel{SessionID: sessionID, Status: conversationstore.InteractionStatusRunning, CreatedAt: time.Now()}
		if err := tx.Create(&turn).Error; err != nil {
			return err
		}
		turnID = turn.ID
		return nil
	})
	return turnID, err
}

func (s *GormStore) AddHistoryToTurn(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64, messages ...*ai.Message) error {
	if err := s.checkContext(ctx); err != nil {
		return err
	}
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		_, err := s.findActiveSession(tx, access, sessionID, true)
		if err != nil {
			return err
		}

		var turn TurnModel
		if err := tx.Where("id = ? AND session_id = ?", turnID, sessionID).First(&turn).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return conversationstore.ErrTurnNotFound
		} else if err != nil {
			return err
		}
		if turn.CompletedAt != nil {
			return conversationstore.ErrTurnNotFound
		}

		return s.insertMessages(tx, access, sessionID, turn.ID, messages)
	})
	if err != nil {
		// BeginTurn is intentionally separate from message persistence. If this
		// first write failed before any message was stored, remove only the empty
		// Turn so a failed interaction does not leave an orphan active Turn.
		_ = s.deleteEmptyTurn(ctx, access, sessionID, turnID)
	}
	return err
}

func (s *GormStore) deleteEmptyTurn(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64) error {
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		_, err := s.findActiveSession(tx, access, sessionID, false)
		if err != nil {
			return nil
		}
		var count int64
		if err := tx.Model(&MessageModel{}).Where("turn_id = ?", turnID).Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			return nil
		}
		return tx.Where("id = ? AND session_id = ? AND completed_at IS NULL", turnID, sessionID).Delete(&TurnModel{}).Error
	})
}

func (s *GormStore) IsTurnEmpty(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64) (bool, error) {
	if err := s.checkContext(ctx); err != nil {
		return false, err
	}
	if _, err := s.findActiveSession(s.db.WithContext(normalizeContext(ctx)), access, sessionID, false); err != nil {
		return false, err
	}
	var turn TurnModel
	err := s.db.WithContext(normalizeContext(ctx)).
		Where("id = ? AND session_id = ? AND completed_at IS NULL", turnID, sessionID).
		First(&turn).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, conversationstore.ErrTurnNotFound
	}
	if err != nil {
		return false, err
	}
	// An active Turn exists even when no supported messages were supplied.
	return false, nil
}

func (s *GormStore) WindowMemoryForTurn(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64) ([]*ai.Message, error) {
	return s.ContextWindowForTurn(ctx, access, sessionID, turnID, 0)
}

// ContextWindowForTurn returns the most recent completed turns in chronological
// order followed by the selected active turn. The transaction makes the set of
// turns and their messages a consistent snapshot for this interaction.
func (s *GormStore) ContextWindowForTurn(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64, completedLimit int) ([]*ai.Message, error) {
	if err := s.checkContext(ctx); err != nil {
		return nil, err
	}
	if completedLimit < 0 {
		return nil, fmt.Errorf("completed turn limit must not be negative")
	}

	var result []*ai.Message
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		if _, err := s.findActiveSession(tx, access, sessionID, false); err != nil {
			return err
		}
		var active TurnModel
		err := tx.Where("id = ? AND session_id = ? AND completed_at IS NULL", turnID, sessionID).
			First(&active).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return conversationstore.ErrTurnNotFound
		}
		if err != nil {
			return err
		}

		var completed []TurnModel
		if completedLimit > 0 {
			if err := tx.Where("session_id = ? AND completed_at IS NOT NULL", sessionID).
				Order("completed_at DESC").Order("id DESC").Limit(completedLimit).Find(&completed).Error; err != nil {
				return err
			}
		}
		for left, right := 0, len(completed)-1; left < right; left, right = left+1, right-1 {
			completed[left], completed[right] = completed[right], completed[left]
		}
		completed = append(completed, active)
		for i := range completed {
			messages, err := readTurnMessages(tx, normalizeContext(ctx), completed[i].ID)
			if err != nil {
				return err
			}
			result = append(result, messages...)
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	return result, err
}

func (s *GormStore) AllMemory(ctx context.Context, access security.AccessContext, sessionID string) ([]*ai.Message, error) {
	if err := s.checkContext(ctx); err != nil {
		return nil, err
	}
	var result []*ai.Message
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		if _, err := s.findActiveSession(tx, access, sessionID, false); err != nil {
			return err
		}
		var active []TurnModel
		if err := tx.Where("session_id = ? AND completed_at IS NULL", sessionID).
			Order("id ASC").Find(&active).Error; err != nil {
			return err
		}
		var completed []TurnModel
		if err := tx.Where("session_id = ? AND completed_at IS NOT NULL", sessionID).
			Order("id ASC").Find(&completed).Error; err != nil {
			return err
		}
		turns := make([]TurnModel, 0, len(completed)+len(active))
		turns = append(turns, active...)
		turns = append(turns, completed...)
		for i := range turns {
			messages, err := readTurnMessages(tx, normalizeContext(ctx), turns[i].ID)
			if err != nil {
				return err
			}
			result = append(result, messages...)
		}
		return nil
	})
	return result, err
}

func (s *GormStore) NextTurnForTurn(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64) error {
	if err := s.checkContext(ctx); err != nil {
		return err
	}
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		_, err := s.findActiveSession(tx, access, sessionID, true)
		if err != nil {
			return err
		}
		var active TurnModel
		if err := tx.Where("id = ? AND session_id = ? AND completed_at IS NULL", turnID, sessionID).First(&active).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return conversationstore.ErrTurnNotFound
		} else if err != nil {
			return err
		}
		var completedCount int64
		if err := tx.Model(&TurnModel{}).
			Where("session_id = ? AND completed_at IS NOT NULL", sessionID).
			Count(&completedCount).Error; err != nil {
			return err
		}
		if completedCount >= int64(s.limit) {
			return fmt.Errorf("%w: current session's context is full, please create a new session", conversationstore.ErrTurnLimitReached)
		}
		completedAt := time.Now()
		if err := recordSummaryHistory(tx, sessionID, turnID, conversationstore.InteractionStatusCompleted); err != nil {
			return err
		}
		return tx.Model(&TurnModel{}).Where("id = ?", active.ID).Updates(map[string]any{"completed_at": completedAt, "status": conversationstore.InteractionStatusCompleted}).Error
	})
}

// AbortTurnForTurn removes an unfinished interaction and its messages in one
// transaction. Aborted Turns do not count toward the configured limit.
func (s *GormStore) AbortTurnForTurn(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64) error {
	if err := s.checkContext(ctx); err != nil {
		return err
	}
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		if _, err := s.findActiveSession(tx, access, sessionID, true); err != nil {
			return err
		}
		var turn TurnModel
		if err := tx.Where("id = ? AND session_id = ? AND completed_at IS NULL", turnID, sessionID).First(&turn).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return conversationstore.ErrTurnNotFound
		} else if err != nil {
			return err
		}
		return s.deleteTurnAndReleaseUsage(tx, access.Subject.TenantID, sessionID, turn.ID)
	})
}

func (s *GormStore) deleteTurnAndReleaseUsage(tx *gorm.DB, tenantID, sessionID string, turnID uint64) error {
	var turn TurnModel
	if err := tx.Where("id = ? AND session_id = ?", turnID, sessionID).First(&turn).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return conversationstore.ErrTurnNotFound
	} else if err != nil {
		return err
	}
	if err := recordSummaryHistory(tx, sessionID, turnID, conversationstore.InteractionStatusFailed); err != nil {
		return err
	}
	if err := tx.Where("turn_id = ?", turnID).Delete(&MessageModel{}).Error; err != nil {
		return err
	}
	if err := tx.Delete(&turn).Error; err != nil {
		return err
	}
	if turn.StoredBytes == 0 {
		return nil
	}
	if err := tx.Model(&SessionModel{}).Where("id = ?", sessionID).Update("stored_bytes", gorm.Expr("CASE WHEN stored_bytes >= ? THEN stored_bytes - ? ELSE 0 END", turn.StoredBytes, turn.StoredBytes)).Error; err != nil {
		return err
	}
	return tx.Model(&TenantUsageModel{}).Where("tenant_id = ?", tenantID).Update("stored_bytes", gorm.Expr("CASE WHEN stored_bytes >= ? THEN stored_bytes - ? ELSE 0 END", turn.StoredBytes, turn.StoredBytes)).Error
}

// AddHistory is retained for callers of the pre-Store API. Production code
// must use BeginTurn and AddHistoryToTurn to isolate concurrent interactions.
func (s *GormStore) AddHistory(ctx context.Context, access security.AccessContext, sessionID string, messages ...*ai.Message) error {
	if err := s.checkContext(ctx); err != nil {
		return err
	}
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		_, err := s.findActiveSession(tx, access, sessionID, true)
		if err != nil {
			return err
		}
		var active []TurnModel
		if err := tx.Where("session_id = ? AND completed_at IS NULL", sessionID).Order("id ASC").Find(&active).Error; err != nil {
			return err
		}
		var turn TurnModel
		switch len(active) {
		case 0:
			var completedCount int64
			if err := tx.Model(&TurnModel{}).Where("session_id = ? AND completed_at IS NOT NULL", sessionID).Count(&completedCount).Error; err != nil {
				return err
			}
			if completedCount >= int64(s.limit) {
				return fmt.Errorf("%w: current session's context is full, please create a new session", conversationstore.ErrTurnLimitReached)
			}
			turn = TurnModel{SessionID: sessionID, CreatedAt: time.Now()}
			if err := tx.Create(&turn).Error; err != nil {
				return err
			}
		case 1:
			turn = active[0]
		default:
			return fmt.Errorf("session %q has multiple active turns; use AddHistoryToTurn", sessionID)
		}
		return s.insertMessages(tx, access, sessionID, turn.ID, messages)
	})
}

func (s *GormStore) insertMessages(tx *gorm.DB, access security.AccessContext, sessionID string, turnID uint64, messages []*ai.Message) error {
	var last MessageModel
	err := tx.Where("turn_id = ?", turnID).Order("sequence DESC").First(&last).Error
	nextSequence := uint64(0)
	if err == nil {
		nextSequence = last.Sequence + 1
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	models := make([]MessageModel, 0, len(messages))
	var addedBytes int64
	for _, message := range messages {
		if message == nil || !supportedRole(message.Role) {
			continue
		}
		payload, err := encodeMessage(message)
		if err != nil {
			return fmt.Errorf("failed to encode message: %w", err)
		}
		models = append(models, MessageModel{Role: string(message.Role), TurnID: turnID, Sequence: nextSequence, Payload: payload, CreatedAt: time.Now()})
		addedBytes += int64(len(payload))
		nextSequence++
	}
	turnReserved := tx.Model(&TurnModel{}).Where("id = ? AND event_count + ? <= ?", turnID, len(models), s.quota.MaxEventsPerTurn).Updates(map[string]any{"event_count": gorm.Expr("event_count + ?", len(models)), "stored_bytes": gorm.Expr("stored_bytes + ?", addedBytes)})
	if turnReserved.Error != nil {
		return turnReserved.Error
	}
	if turnReserved.RowsAffected != 1 {
		return conversationstore.ErrResourceExhausted
	}
	sessionReserved := tx.Model(&SessionModel{}).Where("id = ? AND stored_bytes + ? <= ?", sessionID, addedBytes, s.quota.MaxSessionBytes).Update("stored_bytes", gorm.Expr("stored_bytes + ?", addedBytes))
	if sessionReserved.Error != nil {
		return sessionReserved.Error
	}
	if sessionReserved.RowsAffected != 1 {
		return conversationstore.ErrResourceExhausted
	}
	tenantReserved := tx.Model(&TenantUsageModel{}).Where("tenant_id = ? AND stored_bytes + ? <= ?", access.Subject.TenantID, addedBytes, s.quota.MaxTenantBytes).Update("stored_bytes", gorm.Expr("stored_bytes + ?", addedBytes))
	if tenantReserved.Error != nil {
		return tenantReserved.Error
	}
	if tenantReserved.RowsAffected != 1 {
		return conversationstore.ErrResourceExhausted
	}
	for i := range models {
		if err := tx.Create(&models[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *GormStore) IsEmpty(ctx context.Context, access security.AccessContext, sessionID string) (bool, error) {
	turnID, err := s.activeTurnID(ctx, access, sessionID)
	if err != nil {
		if errors.Is(err, conversationstore.ErrNoActiveTurn) {
			return true, nil
		}
		return false, err
	}
	if _, err := s.IsTurnEmpty(ctx, access, sessionID, turnID); err != nil {
		return false, err
	}
	return false, nil
}

func (s *GormStore) WindowMemory(ctx context.Context, access security.AccessContext, sessionID string) ([]*ai.Message, error) {
	turnID, err := s.activeTurnID(ctx, access, sessionID)
	if err != nil {
		if errors.Is(err, conversationstore.ErrNoActiveTurn) {
			return nil, nil
		}
		return nil, err
	}
	return s.WindowMemoryForTurn(ctx, access, sessionID, turnID)
}

func (s *GormStore) NextTurn(ctx context.Context, access security.AccessContext, sessionID string) error {
	turnID, err := s.activeTurnID(ctx, access, sessionID)
	if err != nil {
		return err
	}
	return s.NextTurnForTurn(ctx, access, sessionID, turnID)
}

func (s *GormStore) activeTurnID(ctx context.Context, access security.AccessContext, sessionID string) (uint64, error) {
	if _, err := s.findActiveSession(s.db.WithContext(normalizeContext(ctx)), access, sessionID, false); err != nil {
		return 0, err
	}
	var turns []TurnModel
	err := s.db.WithContext(normalizeContext(ctx)).Where("session_id = ? AND completed_at IS NULL", sessionID).Order("id ASC").Find(&turns).Error
	if err != nil {
		return 0, err
	}
	if len(turns) == 0 {
		return 0, conversationstore.ErrNoActiveTurn
	}
	if len(turns) > 1 {
		return 0, fmt.Errorf("session %q has multiple active turns", sessionID)
	}
	return turns[0].ID, nil
}

func readTurnMessages(db *gorm.DB, ctx context.Context, turnID uint64) ([]*ai.Message, error) {
	var models []MessageModel
	if err := db.WithContext(normalizeContext(ctx)).Where("turn_id = ?", turnID).Order("sequence ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	return decodeMessages(models)
}

func decodeMessages(models []MessageModel) ([]*ai.Message, error) {
	result := make([]*ai.Message, 0, len(models))
	for i := range models {
		message, err := decodeMessage(models[i].ID, models[i].Payload)
		if err != nil {
			return nil, err
		}
		switch message.Role {
		case ai.RoleSystem, ai.RoleUser, ai.RoleModel:
			result = append(result, message)
		}
	}
	return result, nil
}

func (s *GormStore) findSession(tx *gorm.DB, access security.AccessContext, sessionID string, lock bool) (*SessionModel, error) {
	if err := access.Validate(); err != nil {
		return nil, err
	}
	query := scopeSessions(tx, access).Where("id = ?", sessionID)
	if lock && supportsRowLock(tx) {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var model SessionModel
	if err := query.First(&model).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, conversationstore.ErrSessionNotFound
	} else if err != nil {
		return nil, err
	}
	if !access.IsSystem() && !access.CanAccessOwner(model.TenantID, model.OwnerID) {
		return nil, conversationstore.ErrSessionNotFound
	}
	return &model, nil
}

func (s *GormStore) findActiveSession(tx *gorm.DB, access security.AccessContext, sessionID string, lock bool) (*SessionModel, error) {
	if err := access.ValidateBusiness(); err != nil {
		return nil, err
	}
	model, err := s.findSession(tx, access, sessionID, lock)
	if err != nil {
		return nil, err
	}
	if err := conversationstore.ValidateSessionActive(sessionID, model.Status, s.isExpired(model, time.Now())); err != nil {
		return nil, err
	}
	return model, nil
}

func scopeSessions(query *gorm.DB, access security.AccessContext) *gorm.DB {
	if access.IsSystem() {
		return query
	}
	query = query.Where("tenant_id = ?", access.Subject.TenantID)
	return query
}

func deleteSessionData(tx *gorm.DB, sessionID string) error {
	if err := tx.Where("session_id = ?", sessionID).Delete(&ToolBatchModel{}).Error; err != nil {
		return err
	}
	var turns []TurnModel
	if err := tx.Where("session_id = ?", sessionID).Find(&turns).Error; err != nil {
		return err
	}
	if len(turns) > 0 {
		ids := make([]uint64, 0, len(turns))
		for i := range turns {
			ids = append(ids, turns[i].ID)
		}
		if err := tx.Where("turn_id IN ?", ids).Delete(&MessageModel{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("session_id = ?", sessionID).Delete(&TurnModel{}).Error; err != nil {
		return err
	}
	return tx.Where("id = ?", sessionID).Delete(&SessionModel{}).Error
}

func sessionModelFromDomain(session *conversationstore.Session) SessionModel {
	return SessionModel{
		TemplateRef: session.TemplateRef,
		ID:          session.ID, TenantID: session.TenantID, OwnerID: session.OwnerID, Version: session.Version,
		CreatedAt: session.CreatedAt, UpdatedAt: session.UpdatedAt, ExpiresAt: session.ExpiresAt, Status: session.Status,
	}
}

func sessionDomainFromModel(model *SessionModel) *conversationstore.Session {
	return &conversationstore.Session{
		TemplateRef: model.TemplateRef,
		ID:          model.ID, TenantID: model.TenantID, OwnerID: model.OwnerID, Version: model.Version,
		CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt, ExpiresAt: model.ExpiresAt, Status: model.Status,
	}
}

func (s *GormStore) checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func supportsRowLock(tx *gorm.DB) bool {
	name := strings.ToLower(tx.Name())
	return name == "mysql" || name == "postgres"
}

func supportedRole(role ai.Role) bool {
	return role == ai.RoleSystem || role == ai.RoleUser || role == ai.RoleModel
}

func (s *GormStore) isExpired(session *SessionModel, now time.Time) bool {
	return conversationstore.IsSessionExpired(session.ExpiresAt, session.UpdatedAt, now, s.lifecycle.Retention)
}

func (s *GormStore) RecordUsage(ctx context.Context, event conversationstore.UsageEvent) error {
	access, err := security.AccessContextFromContext(ctx)
	if err != nil || access.ValidateBusiness() != nil || event.TenantID == "" || event.TenantID != access.Subject.TenantID || (event.UserID != "" && event.UserID != access.Subject.UserID) {
		return security.ErrResourceNotAccessible
	}
	event.UserID = access.Subject.UserID
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	return s.recordEvent(ctx, "usage", s.usagePolicy.MaxUsageEvents, &UsageEventModel{ID: event.ID, SchemaVersion: event.SchemaVersion, TenantID: event.TenantID, UserID: event.UserID, SessionID: event.SessionID, InteractionID: event.InteractionID, CallID: event.CallID, CallType: event.CallType, Provider: event.Provider, Model: event.Model, ProviderVersion: event.ProviderVersion, ModelVersion: event.ModelVersion, PolicyVersion: event.PolicyVersion, AmountSource: event.AmountSource, InputTokens: event.InputTokens, OutputTokens: event.OutputTokens, TotalTokens: event.TotalTokens, PricingVersion: event.PricingVersion, CostStatus: event.CostStatus, Cost: event.Cost, Status: event.Status, ReasonCode: event.ReasonCode, Attempt: event.Attempt, Retry: event.Retry, Auxiliary: event.Auxiliary, CreatedAt: event.CreatedAt, EstimatedInputTokens: event.EstimatedInputTokens, TemplateRef: event.TemplateRef, CompilerVersion: event.CompilerVersion, EstimatorVersion: event.EstimatorVersion})
}

func (s *GormStore) RecordAudit(ctx context.Context, event conversationstore.AuditEvent) error {
	access, err := security.AccessContextFromContext(ctx)
	if err != nil || access.ValidateBusiness() != nil || event.TenantID == "" || event.TenantID != access.Subject.TenantID || (event.UserID != "" && event.UserID != access.Subject.UserID) {
		return security.ErrResourceNotAccessible
	}
	event.UserID = access.Subject.UserID
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	return s.recordEvent(ctx, "audit", s.usagePolicy.MaxAuditEvents, &AuditEventModel{ID: event.ID, SchemaVersion: event.SchemaVersion, TenantID: event.TenantID, UserID: event.UserID, SessionID: event.SessionID, InteractionID: event.InteractionID, CallID: event.CallID, ToolCallID: event.ToolCallID, EventType: event.EventType, Status: event.Status, ReasonCode: event.ReasonCode, PolicyVersion: event.PolicyVersion, ComponentVersion: event.ComponentVersion, CreatedAt: event.CreatedAt})
}

// recordEvent reserves a slot and inserts atomically. The conditional UPDATE
// serializes writers in the database, including independent Store processes.
func (s *GormStore) recordEvent(ctx context.Context, kind string, maximum int, model any) error {
	return s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&EventCapacityModel{}).Where("kind = ? AND event_count >= 0 AND event_count < ?", kind, maximum).
			Update("event_count", gorm.Expr("event_count + 1"))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return conversationstore.ErrResourceExhausted
		}
		return tx.Create(model).Error
	})
}

func (s *GormStore) CleanupUsage(ctx context.Context, access security.AccessContext, cutoff time.Time, limit int) (int, error) {
	return s.cleanupEvents(ctx, access, cutoff, limit, "usage", &UsageEventModel{})
}

func (s *GormStore) CleanupAudit(ctx context.Context, access security.AccessContext, cutoff time.Time, limit int) (int, error) {
	return s.cleanupEvents(ctx, access, cutoff, limit, "audit", &AuditEventModel{})
}

func (s *GormStore) cleanupEvents(ctx context.Context, access security.AccessContext, cutoff time.Time, limit int, kind string, model any) (int, error) {
	if err := access.Validate(); err != nil {
		return 0, err
	}
	if !access.IsSystem() {
		return 0, security.ErrResourceNotAccessible
	}
	if err := s.checkContext(ctx); err != nil {
		return 0, err
	}
	if limit <= 0 || limit > s.usagePolicy.CleanupBatchSize {
		limit = s.usagePolicy.CleanupBatchSize
	}
	cutoff = conversationstore.UsageAuditCleanupCutoff(cutoff, time.Now(), s.usagePolicy)
	removed := 0
	err := s.db.WithContext(normalizeContext(ctx)).Transaction(func(tx *gorm.DB) error {
		// Lock the same capacity row before selecting deletions. A concurrent
		// cleanup must not delete/count the same records twice.
		if err := tx.Model(&EventCapacityModel{}).Where("kind = ?", kind).
			Update("event_count", gorm.Expr("event_count + 0")).Error; err != nil {
			return err
		}
		var ids []string
		if err := tx.Model(model).Where("created_at < ?", cutoff).Order("created_at ASC, id ASC").
			Limit(limit).Pluck("id", &ids).Error; err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		result := tx.Where("id IN ?", ids).Delete(model)
		if result.Error != nil {
			return result.Error
		}
		removed = int(result.RowsAffected)
		return tx.Model(&EventCapacityModel{}).Where("kind = ?", kind).
			Update("event_count", gorm.Expr("event_count - ?", removed)).Error
	})
	if err != nil {
		return 0, err
	}
	return removed, nil
}
