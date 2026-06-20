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
	"errors"
	"time"

	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// Source identifies where a rule mutation originated.
// Used for auditing and distinguishing user-initiated changes from system-generated ones.
type Source string

const (
	SourceAdmin     Source = "ADMIN"     // User edit via Admin UI/API
	SourceUpstream  Source = "UPSTREAM"  // Registry change detected by subscriber
	SourceBootstrap Source = "BOOTSTRAP" // Initial version recorded at startup
	// SourceRollback marks a version produced by re-publishing a historical
	// snapshot. Rollback records a new version and does not rewrite history.
	SourceRollback Source = "ROLLBACK"
)

type Operation string

const (
	OperationCreate Operation = "CREATE"
	OperationUpdate Operation = "UPDATE"
	OperationDelete Operation = "DELETE"
)

// IntentStatus tracks the lifecycle of a mutation intent.
// Intent workflow:
//   - PENDING: intent is durable, registry mutation is not yet proven.
//   - APPLIED: intended state was observed, but RuleVersion may not be durable.
//   - OUTCOME_UNKNOWN: registry returned an uncertain result or a conflicting
//     event was observed; repair must read actual state before cleanup.
//   - COMMITTING: a clean APPLIED intent won the storage-level CAS for commit;
//     retries must finish the fixed-ID RuleVersion or reconcile actual state.
//   - COMMITTED/FAILED: terminal states, cleaned up after the durable outcome.
//
// RPC errors and context cancellation are not registry-side fencing. They move
// the intent to OUTCOME_UNKNOWN so repair can reconcile actual state later.
type IntentStatus string

const (
	IntentStatusPending        IntentStatus = "PENDING"         // Intent created, mutation not yet applied
	IntentStatusApplied        IntentStatus = "APPLIED"         // Intended state observed, awaiting version commit
	IntentStatusOutcomeUnknown IntentStatus = "OUTCOME_UNKNOWN" // Actual registry outcome must be reconciled
	IntentStatusCommitting     IntentStatus = "COMMITTING"      // Commit ownership acquired before fixed-ID version append
	IntentStatusCommitted      IntentStatus = "COMMITTED"       // Version successfully recorded, intent closed
	IntentStatusFailed         IntentStatus = "FAILED"          // Mutation failed or was rejected
)

var (
	ErrVersionConflict       = errors.New("rule version conflict") // ExpectedVersionID mismatch
	ErrVersionNotFound       = errors.New("rule version not found")
	ErrVersionIntentNotFound = errors.New("rule version intent not found")
	ErrVersionIntentNotOpen  = errors.New("rule version intent is not open") // Intent already committed or failed
	ErrVersionIntentPending  = errors.New("rule version intent is pending")  // Another mutation in progress
	ErrVersionLedgerCorrupt  = errors.New("rule version ledger corruption")
	ErrVersionIntentConflict = errors.New("rule version intent revision conflict")
	ErrIntentOutcomeMismatch = errors.New("rule version intent outcome does not match current resource")
	ErrRollbackToDelete      = errors.New("cannot roll back to a deleted rule version")
	ErrRollbackToCurrent     = errors.New("cannot roll back to a version identical to current")
)

// Version represents a snapshot of a rule's spec at a point in time. Version
// entries are immutable after creation. Rollback appends a new version, while
// retention may delete the oldest entries. IsCurrent is derived from the ledger
// head at query time.
type Version struct {
	ID          int64                  `json:"id"`
	RuleKind    coremodel.ResourceKind `json:"ruleKind"`
	Mesh        string                 `json:"mesh"`
	ResourceKey string                 `json:"resourceKey"`
	RuleName    string                 `json:"ruleName"`
	VersionNo   int64                  `json:"versionNo"`
	ContentHash string                 `json:"contentHash"`
	SpecJSON    string                 `json:"specJson"`
	Source      Source                 `json:"source"`
	Operation   Operation              `json:"operation"`
	Author      string                 `json:"author"`
	Reason      string                 `json:"reason,omitempty"`
	IntentID    int64                  `json:"intentId,omitempty"`
	// RolledBackFromID records the historical version whose snapshot was
	// re-published to produce this version. It is audit metadata only and must
	// not be used to decide the current version.
	RolledBackFromID *int64    `json:"rolledBackFromId,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	CommittedAt      time.Time `json:"committedAt"`
	IsCurrent        bool      `json:"isCurrent"`
}

// Intent represents a pending mutation to a rule. It records the user's
// mutation before the rule is written; the Version is created only after the
// resulting rule state is observed or repaired from ResourceManager.
type Intent struct {
	ID          int64                  `json:"id"`
	RuleKind    coremodel.ResourceKind `json:"ruleKind"`
	Mesh        string                 `json:"mesh"`
	ResourceKey string                 `json:"resourceKey"`
	RuleName    string                 `json:"ruleName"`
	ContentHash string                 `json:"contentHash"`
	SpecJSON    string                 `json:"specJson"`
	Source      Source                 `json:"source"`
	Operation   Operation              `json:"operation"`
	Author      string                 `json:"author"`
	Reason      string                 `json:"reason,omitempty"`
	// RolledBackFromID is carried from a rollback intent to the version it
	// commits. It is audit metadata only, never the current-version pointer.
	RolledBackFromID *int64       `json:"rolledBackFromId,omitempty"`
	Status           IntentStatus `json:"status"`
	LastError        string       `json:"lastError,omitempty"`
	// ReconcileRequired is durable evidence that a non-matching event arrived
	// while the intent was open. The observed fields let repair wait until the
	// ResourceManager snapshot has caught up before deciding the final outcome.
	ReconcileRequired   bool      `json:"reconcileRequired,omitempty"`
	ObservedContentHash string    `json:"observedContentHash,omitempty"`
	ObservedSpecJSON    string    `json:"observedSpecJson,omitempty"`
	ObservedOperation   Operation `json:"observedOperation,omitempty"`
	ObservedAt          time.Time `json:"observedAt,omitempty"`
	Revision            int64     `json:"revision"`
	CreatedAt           time.Time `json:"createdAt"`
}

type ledgerState struct {
	Versions     []Version
	Latest       *Version
	MaxVersionNo int64
}

type LedgerSnapshot struct {
	Versions []Version
	Head     *Version
	Deleted  bool
}

type InsertRequest struct {
	RuleKind         coremodel.ResourceKind
	Mesh             string
	ResourceKey      string
	RuleName         string
	SpecJSON         string
	ContentHash      string
	Source           Source
	Operation        Operation
	Author           string
	Reason           string
	IntentID         int64
	RolledBackFromID *int64
	CreatedAt        time.Time
	FixedVersionID   *int64
}

type ListResult struct {
	Items            []Version `json:"items"`
	Total            int64     `json:"total"`
	CurrentVersionID *int64    `json:"currentVersionId,omitempty"`
	CurrentVersionNo int64     `json:"currentVersionNo,omitempty"`
	Deleted          bool      `json:"deleted"`
}

type DiffResult struct {
	Left  DiffSide `json:"left"`
	Right DiffSide `json:"right"`
}

type DiffSide struct {
	ID        int64  `json:"id"`
	VersionNo int64  `json:"versionNo"`
	SpecJSON  string `json:"specJson"`
}

type ConflictError struct {
	CurrentVersionID *int64
}

func (e *ConflictError) Error() string {
	return ErrVersionConflict.Error()
}

type IntentPendingError struct {
	IntentID int64
}

func (e *IntentPendingError) Error() string {
	return ErrVersionIntentPending.Error()
}

func (e *IntentPendingError) Is(target error) bool {
	return target == ErrVersionIntentPending
}
