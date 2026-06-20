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
	// snapshot. Rollback is append-only and does not rewind RuleMeta.
	SourceRollback Source = "ROLLBACK"
)

type Operation string

const (
	OperationCreate Operation = "CREATE"
	OperationUpdate Operation = "UPDATE"
	OperationDelete Operation = "DELETE"
)

// IntentStatus tracks the lifecycle of a mutation intent.
// Intent workflow: PENDING (created) → APPLIED (mutation succeeded) → COMMITTED (version recorded)
// Or: PENDING → FAILED (mutation failed or conflicted)
type IntentStatus string

const (
	IntentStatusPending   IntentStatus = "PENDING"   // Intent created, mutation not yet applied
	IntentStatusApplied   IntentStatus = "APPLIED"   // Mutation applied to resource store, awaiting commit
	IntentStatusCommitted IntentStatus = "COMMITTED" // Version successfully recorded, intent closed
	IntentStatusFailed    IntentStatus = "FAILED"    // Mutation failed or was rejected
)

var (
	ErrFeatureDisabled       = errors.New("rule versioning is disabled")
	ErrVersionConflict       = errors.New("rule version conflict") // ExpectedVersionID mismatch
	ErrVersionNotFound       = errors.New("rule version not found")
	ErrVersionIntentNotFound = errors.New("rule version intent not found")
	ErrVersionIntentNotOpen  = errors.New("rule version intent is not open") // Intent already committed or failed
	ErrVersionIntentPending  = errors.New("rule version intent is pending")  // Another mutation in progress
	ErrVersionLedgerCorrupt  = errors.New("rule version ledger corruption")
	ErrIntentOutcomeMismatch = errors.New("rule version intent outcome does not match current resource")
	ErrRollbackToDelete      = errors.New("cannot roll back to a deleted rule version")
	ErrRollbackToCurrent     = errors.New("cannot roll back to a version identical to current")
)

// Version represents an immutable snapshot of a rule's spec at a point in time.
// Versions are append-only; rollback creates a new Version instead of changing
// the historical target version.
// The IsCurrent field is derived from the immutable ledger head at query time.
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

// Meta tracks the current committed version and sequence number for a rule.
// Rollback advances CurrentVersion to the newly committed rollback version; it
// must not point back to the historical target.
// CurrentVersion may be nil if the rule was deleted.
type Meta struct {
	RuleKind       coremodel.ResourceKind `json:"ruleKind"`
	ResourceKey    string                 `json:"resourceKey"`
	CurrentVersion *int64                 `json:"currentVersion"`
	LastVersionNo  int64                  `json:"lastVersionNo"`
	UpdatedAt      time.Time              `json:"updatedAt"`
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
	CreatedAt        time.Time    `json:"createdAt"`
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
