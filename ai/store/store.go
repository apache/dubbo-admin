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

	"dubbo-admin-ai/security"

	"github.com/firebase/genkit/go/ai"
)

// SessionStore persists Session metadata.
type SessionStore interface {
	Create(ctx context.Context, access security.AccessContext, session *Session) error
	Get(ctx context.Context, access security.AccessContext, sessionID string) (*Session, error)
	List(ctx context.Context, access security.AccessContext) ([]*Session, error)
	Touch(ctx context.Context, access security.AccessContext, sessionID string, updatedAt time.Time, retentions ...time.Duration) error
	Delete(ctx context.Context, access security.AccessContext, sessionID string) error
}

// MaintenanceStore exposes privileged lifecycle operations separately from
// the business-facing SessionStore contract.
type MaintenanceStore interface {
	DeleteExpired(ctx context.Context, access security.AccessContext, now time.Time, batchSizes ...int) (int, error)
	AdoptLegacySessions(ctx context.Context, access security.AccessContext, tenantID, ownerID string) (int, error)
	RegisterDerivedArtifact(ctx context.Context, access security.AccessContext, artifact DerivedArtifact) error
	// ListDerivedArtifactsForCleanup uses each artifact's NextRetryAt as the
	// authoritative retry schedule and includes orphaned or inconsistent sources.
	ListDerivedArtifactsForCleanup(ctx context.Context, access security.AccessContext, now time.Time, limit int) ([]DerivedArtifact, error)
	MarkDerivedArtifactDeleted(ctx context.Context, access security.AccessContext, artifactID string) error
	MarkDerivedArtifactCleanupFailed(ctx context.Context, access security.AccessContext, artifactID, errorCode string, failedAt time.Time) error
	DeleteExpiredPage(ctx context.Context, access security.AccessContext, now time.Time, cursor CleanupCursor, limit int) (int, CleanupCursor, error)
	ListDerivedArtifactsForCleanupPage(ctx context.Context, access security.AccessContext, now time.Time, cursor CleanupCursor, limit int) ([]DerivedArtifact, CleanupCursor, error)
}

// MessageStore persists conversation Turns and Genkit messages.
type MessageStore interface {
	PrepareInteraction(ctx context.Context, access security.AccessContext, sessionID, idempotencyKey, requestHash string, leaseDuration time.Duration) (*PreparedInteraction, error)
	CompleteInteraction(ctx context.Context, access security.AccessContext, sessionID, interactionID string, turnID uint64, result string) error
	FailInteraction(ctx context.Context, access security.AccessContext, sessionID, interactionID string, turnID uint64) error
	BeginTurn(ctx context.Context, access security.AccessContext, sessionID string) (uint64, error)
	AddHistoryToTurn(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64, messages ...*ai.Message) error
	IsTurnEmpty(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64) (bool, error)
	// ContextWindowForTurn returns independent message snapshots from the last
	// completedLimit completed turns followed by the active turn. A zero limit
	// includes only the active turn; negative limits are invalid. A missing or
	// completed turn returns ErrTurnNotFound.
	ContextWindowForTurn(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64, completedLimit int) ([]*ai.Message, error)
	WindowMemoryForTurn(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64) ([]*ai.Message, error)
	AllMemory(ctx context.Context, access security.AccessContext, sessionID string) ([]*ai.Message, error)
	NextTurnForTurn(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64) error
	AbortTurnForTurn(ctx context.Context, access security.AccessContext, sessionID string, turnID uint64) error
}

// Store combines only the business-facing Session and conversation history
// contracts.
type Store interface {
	SessionStore
	MessageStore
}

// BackendStore is implemented by concrete storage backends and is used only
// where both business and privileged lifecycle operations are required.
type BackendStore interface {
	Store
	MaintenanceStore
}
