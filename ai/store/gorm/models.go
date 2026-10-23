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

import "time"

// SessionModel is the database representation of a conversation session.
// Associations are intentionally represented by ordinary indexed columns;
// cleanup is performed explicitly by the Store without database foreign keys.
type SessionModel struct {
	TemplateRef         string     `gorm:"size:256"`
	ID                  string     `gorm:"primaryKey;size:64"`
	TenantID            string     `gorm:"size:128;index:idx_ai_session_owner,priority:1"`
	OwnerID             string     `gorm:"size:128;index:idx_ai_session_owner,priority:2"`
	Version             uint64     `gorm:"not null;default:1"`
	CreatedAt           time.Time  `gorm:"not null"`
	UpdatedAt           time.Time  `gorm:"not null;index"`
	ExpiresAt           *time.Time `gorm:"index"`
	Status              string     `gorm:"size:24;not null;index"`
	ActiveInteractionID *string    `gorm:"size:64;index"`
	StoredBytes         int64      `gorm:"not null;default:0"`
}

func (SessionModel) TableName() string { return "ai_sessions" }

type ToolBatchModel struct {
	ID        string `gorm:"primaryKey;size:64"`
	SessionID string `gorm:"size:64;not null;index"`
	TurnID    uint64 `gorm:"not null;uniqueIndex:idx_ai_tool_sequence"`
	Sequence  uint64 `gorm:"not null;uniqueIndex:idx_ai_tool_sequence"`
	Version   uint64 `gorm:"not null"`
	Payload   []byte `gorm:"not null"`
}

func (ToolBatchModel) TableName() string { return "ai_tool_batches" }

// TurnModel is one persisted conversation turn. A nil CompletedAt identifies
// the current active turn for a session.
type TurnModel struct {
	ID            uint64     `gorm:"primaryKey;autoIncrement"`
	SessionID     string     `gorm:"size:64;not null;index"`
	InteractionID string     `gorm:"size:64;index"`
	Status        string     `gorm:"size:24;not null;default:running;index"`
	EventCount    uint64     `gorm:"not null;default:0"`
	StoredBytes   int64      `gorm:"not null;default:0"`
	CreatedAt     time.Time  `gorm:"not null;index"`
	CompletedAt   *time.Time `gorm:"index"`
}

func (TurnModel) TableName() string { return "ai_turns" }

// MessageModel stores the complete Genkit message as JSON. Sequence is
// scoped to a turn and provides deterministic message ordering.
type MessageModel struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	TurnID    uint64    `gorm:"not null;uniqueIndex:uidx_ai_message_turn_sequence"`
	Sequence  uint64    `gorm:"not null;uniqueIndex:uidx_ai_message_turn_sequence"`
	Payload   []byte    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (MessageModel) TableName() string { return "ai_messages" }

type InteractionModel struct {
	ID             string     `gorm:"primaryKey;size:64"`
	TenantID       string     `gorm:"size:128;not null;uniqueIndex:uidx_ai_interaction_key,priority:1"`
	OwnerID        string     `gorm:"size:128;not null;uniqueIndex:uidx_ai_interaction_key,priority:2"`
	SessionID      string     `gorm:"size:64;not null;uniqueIndex:uidx_ai_interaction_key,priority:3;index"`
	IdempotencyKey string     `gorm:"size:128;not null;uniqueIndex:uidx_ai_interaction_key,priority:4"`
	RequestHash    string     `gorm:"size:64;not null"`
	TurnID         uint64     `gorm:"not null;index"`
	Status         string     `gorm:"size:24;not null;index"`
	Attempt        uint32     `gorm:"not null"`
	Result         string     `gorm:"type:text"`
	LeaseExpiresAt time.Time  `gorm:"not null;index"`
	CreatedAt      time.Time  `gorm:"not null"`
	UpdatedAt      time.Time  `gorm:"not null"`
	CompletedAt    *time.Time `gorm:"index"`
}

func (InteractionModel) TableName() string { return "ai_interactions" }

type DerivedArtifactModel struct {
	ID                string     `gorm:"primaryKey;size:64"`
	TenantID          string     `gorm:"size:128;not null;index"`
	CanonicalSourceID string     `gorm:"size:128;not null;index"`
	SourceSessionID   string     `gorm:"size:64;not null;default:'';index"`
	Kind              string     `gorm:"size:64;not null"`
	Locator           string     `gorm:"size:512;not null"`
	Status            string     `gorm:"size:24;not null;index"`
	CleanupAttempts   uint32     `gorm:"not null;default:0"`
	LastErrorCode     string     `gorm:"size:64"`
	NextRetryAt       *time.Time `gorm:"index"`
	CreatedAt         time.Time  `gorm:"not null"`
	UpdatedAt         time.Time  `gorm:"not null"`
}

func (DerivedArtifactModel) TableName() string { return "ai_derived_artifacts" }

type CleanupJobModel struct {
	ID                string     `gorm:"primaryKey;size:64"`
	TenantID          string     `gorm:"size:128;not null;index"`
	CanonicalSourceID string     `gorm:"size:128;not null;uniqueIndex"`
	Status            string     `gorm:"size:24;not null;index"`
	Attempts          uint32     `gorm:"not null"`
	LastErrorCode     string     `gorm:"size:64"`
	NextRetryAt       *time.Time `gorm:"index"`
	CreatedAt         time.Time  `gorm:"not null"`
	UpdatedAt         time.Time  `gorm:"not null"`
}

func (CleanupJobModel) TableName() string { return "ai_cleanup_jobs" }

type OutboxEventModel struct {
	ID          string     `gorm:"primaryKey;size:64"`
	TenantID    string     `gorm:"size:128;not null;index"`
	AggregateID string     `gorm:"size:128;not null;index"`
	EventType   string     `gorm:"size:64;not null"`
	CreatedAt   time.Time  `gorm:"not null"`
	PublishedAt *time.Time `gorm:"index"`
}

func (OutboxEventModel) TableName() string { return "ai_outbox_events" }

type TenantUsageModel struct {
	TenantID     string `gorm:"primaryKey;size:128"`
	SessionCount int64  `gorm:"not null;default:0"`
	StoredBytes  int64  `gorm:"not null;default:0"`
}

func (TenantUsageModel) TableName() string { return "ai_tenant_usage" }

type UsageEventModel struct {
	EstimatedInputTokens *int
	TemplateRef          string `gorm:"size:256"`
	CompilerVersion      string `gorm:"size:64"`
	EstimatorVersion     string `gorm:"size:64"`
	ID                   string `gorm:"primaryKey;size:64"`
	SchemaVersion        string `gorm:"size:32"`
	TenantID             string `gorm:"size:128;not null;index"`
	UserID               string `gorm:"size:128;not null;index"`
	SessionID            string `gorm:"size:64;index"`
	InteractionID        string `gorm:"size:64;index"`
	CallID               string `gorm:"size:128;index"`
	CallType             string `gorm:"size:32;not null"`
	Provider             string `gorm:"size:128"`
	Model                string `gorm:"size:128"`
	ProviderVersion      string `gorm:"size:64"`
	ModelVersion         string `gorm:"size:64"`
	PolicyVersion        string `gorm:"size:64"`
	AmountSource         string `gorm:"size:24;not null"`
	InputTokens          *int
	OutputTokens         *int
	TotalTokens          *int
	PricingVersion       string `gorm:"size:64"`
	CostStatus           string `gorm:"size:24;not null"`
	Cost                 *float64
	Status               string    `gorm:"size:32"`
	ReasonCode           string    `gorm:"size:64"`
	Attempt              int       `gorm:"not null;default:0"`
	Retry                bool      `gorm:"not null;default:false"`
	Auxiliary            bool      `gorm:"not null;default:false"`
	CreatedAt            time.Time `gorm:"not null;index"`
}

func (UsageEventModel) TableName() string { return "ai_usage_events" }

type AuditEventModel struct {
	ID               string    `gorm:"primaryKey;size:64"`
	SchemaVersion    string    `gorm:"size:32"`
	TenantID         string    `gorm:"size:128;not null;index"`
	UserID           string    `gorm:"size:128;not null;index"`
	SessionID        string    `gorm:"size:64;index"`
	InteractionID    string    `gorm:"size:64;index"`
	CallID           string    `gorm:"size:128;index"`
	ToolCallID       string    `gorm:"size:128;index"`
	EventType        string    `gorm:"size:64;not null;index"`
	Status           string    `gorm:"size:32"`
	ReasonCode       string    `gorm:"size:64"`
	PolicyVersion    string    `gorm:"size:64"`
	ComponentVersion string    `gorm:"size:64"`
	CreatedAt        time.Time `gorm:"not null;index"`
}

func (AuditEventModel) TableName() string { return "ai_audit_events" }

// EventCapacityModel serializes reservations across Store instances and processes.
type EventCapacityModel struct {
	Kind       string `gorm:"primaryKey;size:16"`
	EventCount int64  `gorm:"not null;default:0"`
}

func (EventCapacityModel) TableName() string { return "ai_event_capacity" }
