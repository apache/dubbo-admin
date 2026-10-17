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
	"time"
)

const (
	SessionStatusActive          = "active"
	SessionStatusDeletionPending = "deletion_pending"
	SessionStatusDerivedCleanup  = "derived_cleanup"
	SessionStatusDeleted         = "deleted"

	InteractionStatusRunning   = "running"
	InteractionStatusCompleted = "completed"
	InteractionStatusFailed    = "failed"
)

// Session is the persisted conversation session metadata.
type Session struct {
	ID        string
	TenantID  string
	OwnerID   string
	Version   uint64
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt *time.Time
	Status    string
}

// Turn describes one conversation turn. A nil CompletedAt identifies the
// active turn.
type Turn struct {
	ID            uint64
	SessionID     string
	InteractionID string
	Status        string
	CreatedAt     time.Time
	CompletedAt   *time.Time
}

// Interaction is the durable identity and lifecycle record for one logical
// chat request. Retries reuse this record instead of creating another Turn.
type Interaction struct {
	ID             string
	TenantID       string
	OwnerID        string
	SessionID      string
	IdempotencyKey string
	RequestHash    string
	TurnID         uint64
	Status         string
	Attempt        uint32
	Result         string
	LeaseExpiresAt time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CompletedAt    *time.Time
}

// PreparedInteraction is returned by the atomic request preflight. Replayed
// interactions are already complete and must not invoke the model again.
type PreparedInteraction struct {
	InteractionID string
	TurnID        uint64
	Replayed      bool
	Result        string
}

type DerivedArtifact struct {
	ID                string
	TenantID          string
	CanonicalSourceID string
	Kind              string
	Locator           string
	Status            string
	CleanupAttempts   uint32
	LastErrorCode     string
	// NextRetryAt is the authoritative eligibility time used by cleanup scans.
	NextRetryAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type DerivedArtifactCleanupHandler interface {
	CleanupDerivedArtifact(ctx context.Context, artifact DerivedArtifact) error
}

type DerivedArtifactCleanupFunc func(ctx context.Context, artifact DerivedArtifact) error

func (f DerivedArtifactCleanupFunc) CleanupDerivedArtifact(ctx context.Context, artifact DerivedArtifact) error {
	return f(ctx, artifact)
}

type CleanupJob struct {
	ID                string
	TenantID          string
	CanonicalSourceID string
	Status            string
	Attempts          uint32
	LastErrorCode     string
	// NextRetryAt projects the earliest retry across remaining artifacts. It is
	// observable progress metadata, not an independent scheduling authority.
	NextRetryAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CleanupCursor is an opaque keyset-pagination position. SortTime is the
// primary ordering value for the page being scanned; TenantID and ID provide
// deterministic tie breakers.
type CleanupCursor struct {
	SortTime time.Time
	TenantID string
	ID       string
	Valid    bool
}

type OutboxEvent struct {
	ID          string
	TenantID    string
	AggregateID string
	EventType   string
	CreatedAt   time.Time
	PublishedAt *time.Time
}
