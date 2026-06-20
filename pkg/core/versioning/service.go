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
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// Service provides rule versioning functionality.
// Use NewService to create an instance.
type Service struct {
	maxVersions int64
	store       Store
}

func NewService(maxVersions int64, store Store) *Service {
	return &Service{
		maxVersions: maxVersions,
		store:       store,
	}
}

func (s *Service) ensureEnabled() error {
	if s == nil || s.store == nil {
		return ErrVersionLedgerCorrupt
	}
	return nil
}

func (s *Service) List(kind coremodel.ResourceKind, mesh, ruleName string) (*ListResult, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	resourceKey := coremodel.BuildResourceKey(mesh, ruleName)
	snapshot, err := s.store.LedgerSnapshot(kind, resourceKey)
	if err != nil {
		return nil, err
	}
	result := &ListResult{Items: snapshot.Versions, Total: int64(len(snapshot.Versions)), Deleted: snapshot.Deleted}
	if snapshot.Head != nil && !snapshot.Deleted {
		currentID := snapshot.Head.ID
		result.CurrentVersionID = &currentID
		result.CurrentVersionNo = snapshot.Head.VersionNo
	}
	return result, nil
}

func (s *Service) Get(kind coremodel.ResourceKind, mesh, ruleName string, id int64) (*Version, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	resourceKey := coremodel.BuildResourceKey(mesh, ruleName)
	version, err := s.store.GetVersion(kind, resourceKey, id)
	if err != nil {
		return nil, err
	}
	snapshot, err := s.store.LedgerSnapshot(kind, resourceKey)
	if err != nil {
		return nil, err
	}
	if snapshot.Head != nil && !snapshot.Deleted {
		version.IsCurrent = version.ID == snapshot.Head.ID
	}
	return version, nil
}

func (s *Service) Diff(kind coremodel.ResourceKind, mesh, ruleName string, id int64, against string) (*DiffResult, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	left, err := s.Get(kind, mesh, ruleName, id)
	if err != nil {
		return nil, err
	}
	var right *Version
	switch against {
	case "", "current":
		resourceKey := coremodel.BuildResourceKey(mesh, ruleName)
		snapshot, err := s.store.LedgerSnapshot(kind, resourceKey)
		if err != nil {
			return nil, err
		}
		if snapshot.Head == nil {
			return nil, ErrVersionNotFound
		}
		right = snapshot.Head
	case "previous":
		list, err := s.store.ListVersions(kind, coremodel.BuildResourceKey(mesh, ruleName))
		if err != nil {
			return nil, err
		}
		for i := range list {
			if list[i].ID != id {
				continue
			}
			if i+1 >= len(list) {
				return nil, ErrVersionNotFound
			}
			right = &list[i+1]
			break
		}
		if right == nil {
			return nil, ErrVersionNotFound
		}
	default:
		var againstID int64
		if parsed, err := strconv.ParseInt(against, 10, 64); err != nil {
			return nil, bizerror.New(bizerror.InvalidArgument, "against must be 'current', 'previous', or a version ID")
		} else {
			againstID = parsed
		}
		right, err = s.Get(kind, mesh, ruleName, againstID)
		if err != nil {
			return nil, err
		}
	}
	return &DiffResult{
		Left:  DiffSide{ID: left.ID, VersionNo: left.VersionNo, SpecJSON: left.SpecJSON},
		Right: DiffSide{ID: right.ID, VersionNo: right.VersionNo, SpecJSON: right.SpecJSON},
	}, nil
}

// CheckExpected applies the UI-supplied expectedVersionId as a weak
// compare-and-set guard. It prevents a mutation from proceeding over a newer
// ledger entry, but it is not a transactional lock by itself.
func (s *Service) CheckExpected(kind coremodel.ResourceKind, mesh, ruleName string, expected *int64) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	resourceKey := coremodel.BuildResourceKey(mesh, ruleName)
	// Check for open intents first before checking version mismatch.
	// Why: If Writer A created an intent at T1, and Writer B checks expected
	// version at T2 (before A's subscriber commits), the ledger head still
	// reflects the old version. Without this guard, B would get VersionConflict
	// instead of IntentPending, masking the real issue (concurrent write).
	intent, err := s.store.OpenIntent(kind, resourceKey)
	if err != nil {
		return err
	}
	if intent != nil {
		return &IntentPendingError{IntentID: intent.ID}
	}
	return s.store.CheckExpectedVersion(kind, resourceKey, expected)
}

// BeginMutation records a user's mutation before the rule is written.
// The immutable Version is created later from the observed rule state, not from
// the request alone. rolledBackFromID is audit metadata for rollback intents.
func (s *Service) BeginMutation(ctx context.Context, res coremodel.Resource, op Operation, source Source, author, reason string, rolledBackFromID *int64) (*Intent, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	if _, err := lock.RequireLease(ctx); err != nil {
		return nil, err
	}
	req, err := buildMutationInsertRequest(res, op, source, author, reason, rolledBackFromID, time.Now())
	if err != nil {
		return nil, err
	}
	return s.store.CreateIntent(ctx, req)
}

func (s *Service) AbandonIntent(ctx context.Context, intent *Intent, reason string) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	if _, err := lock.RequireLease(ctx); err != nil {
		return err
	}
	if intent == nil {
		return bizerror.New(bizerror.InvalidArgument, "rule version intent is required")
	}
	fresh, err := s.store.GetIntent(intent.ID)
	if err != nil {
		return err
	}
	if fresh.Status == IntentStatusFailed {
		return s.store.CleanupIntent(fresh.ID, IntentStatusFailed)
	}
	if fresh.Status != IntentStatusPending &&
		fresh.Status != IntentStatusApplied &&
		fresh.Status != IntentStatusOutcomeUnknown {
		return ErrVersionIntentNotOpen
	}
	return s.store.MarkIntentFailed(ctx, intent.ID, reason)
}

func (s *Service) MarkIntentOutcomeUnknown(ctx context.Context, intent *Intent, reason string) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	if intent == nil {
		return bizerror.New(bizerror.InvalidArgument, "rule version intent is required")
	}
	return s.store.MarkIntentOutcomeUnknown(ctx, intent.ID, reason)
}

// RepairIntent reconciles an open intent when the rule mutation reached
// ResourceManager but the subscriber did not commit the corresponding version.
// It derives the version from current rule state instead of trusting payloads.
func (s *Service) RepairIntent(ctx context.Context, kind coremodel.ResourceKind, resourceKey string, current coremodel.Resource, deleted bool) (*Version, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	if _, err := lock.RequireLease(ctx); err != nil {
		return nil, err
	}
	intent, err := s.store.OpenIntent(kind, resourceKey)
	if err != nil || intent == nil {
		return nil, err
	}
	return s.repairIntent(ctx, intent, current, deleted)
}

func (s *Service) FinalizeMutation(ctx context.Context, intent *Intent, current coremodel.Resource, deleted bool) (*Version, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	if _, err := lock.RequireLease(ctx); err != nil {
		return nil, err
	}
	if intent == nil {
		return nil, bizerror.New(bizerror.InvalidArgument, "rule version intent is required")
	}
	fresh, err := s.store.GetIntent(intent.ID)
	if err != nil {
		if errors.Is(err, ErrVersionIntentNotFound) {
			return s.committedVersionForClosedIntent(intent)
		}
		return nil, err
	}
	switch fresh.Status {
	case IntentStatusCommitted:
		committed, err := s.committedVersionForIntent(fresh)
		if err == nil {
			if cleanupErr := s.store.CleanupIntent(fresh.ID, IntentStatusCommitted); cleanupErr != nil {
				return nil, cleanupErr
			}
		}
		return committed, err
	case IntentStatusFailed:
		if cleanupErr := s.store.CleanupIntent(fresh.ID, IntentStatusFailed); cleanupErr != nil {
			return nil, cleanupErr
		}
		if fresh.LastError != "" {
			return nil, fmt.Errorf("%w: %s", ErrVersionIntentNotOpen, fresh.LastError)
		}
		return nil, ErrVersionIntentNotOpen
	}
	committed, err := s.repairIntent(ctx, fresh, current, deleted)
	if err != nil {
		if errors.Is(err, ErrVersionIntentNotFound) {
			return s.committedVersionForClosedIntent(fresh)
		}
		return nil, err
	}
	if committed != nil {
		return validateCommittedIntentVersion(committed, fresh)
	}
	return s.committedVersionForIntent(fresh)
}

func (s *Service) GetIntent(id int64) (*Intent, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	return s.store.GetIntent(id)
}

func (s *Service) committedVersionForIntent(intent *Intent) (*Version, error) {
	versions, err := s.store.ListVersions(intent.RuleKind, intent.ResourceKey)
	if err != nil {
		return nil, err
	}
	var found *Version
	for i := range versions {
		if versions[i].IntentID != intent.ID {
			continue
		}
		if found != nil && found.ID != versions[i].ID {
			return nil, fmt.Errorf("%w: multiple RuleVersion resources committed for intent %d", ErrVersionLedgerCorrupt, intent.ID)
		}
		v := versions[i]
		found = &v
	}
	if found == nil {
		return nil, ErrVersionNotFound
	}
	return validateCommittedIntentVersion(found, intent)
}

func (s *Service) committedVersionForClosedIntent(intent *Intent) (*Version, error) {
	version, err := s.committedVersionForIntent(intent)
	if errors.Is(err, ErrVersionNotFound) {
		return nil, fmt.Errorf("%w: terminal intent %d has no committed RuleVersion", ErrVersionLedgerCorrupt, intent.ID)
	}
	return version, err
}

func validateCommittedIntentVersion(version *Version, intent *Intent) (*Version, error) {
	if version == nil || intent == nil ||
		version.RuleKind != intent.RuleKind ||
		version.ResourceKey != intent.ResourceKey ||
		version.ContentHash != intent.ContentHash ||
		version.SpecJSON != intent.SpecJSON ||
		version.Operation != intent.Operation ||
		version.Source != intent.Source ||
		version.Author != intent.Author ||
		version.Reason != intent.Reason ||
		version.IntentID != intent.ID ||
		rolledBackFromIDValue(version.RolledBackFromID) != rolledBackFromIDValue(intent.RolledBackFromID) {
		return nil, fmt.Errorf("%w: committed RuleVersion does not match intent %d", ErrVersionLedgerCorrupt, intent.ID)
	}
	return version, nil
}

func (s *Service) ReconcileActualState(ctx context.Context, kind coremodel.ResourceKind, resourceKey string, current coremodel.Resource, deleted bool, author string) (*Version, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	if _, err := lock.RequireLease(ctx); err != nil {
		return nil, err
	}

	operation := OperationUpdate
	mesh := extractMesh(resourceKey)
	ruleName := extractName(resourceKey)
	specJSON := string(DeleteSpecJSON)
	contentHash := HashSpecJSON(DeleteSpecJSON)
	if deleted || current == nil {
		operation = OperationDelete
	} else {
		mesh = current.ResourceMesh()
		ruleName = current.ResourceMeta().Name
		hash, normalized, err := NormalizeResource(current)
		if err != nil {
			return nil, err
		}
		contentHash = hash
		specJSON = normalized
	}

	latest, err := s.store.LatestVersion(kind, resourceKey)
	if err != nil && !errors.Is(err, ErrVersionNotFound) {
		return nil, err
	}
	if latest != nil {
		if latest.Operation == OperationDelete && operation == OperationDelete {
			return nil, nil
		}
		if latest.Operation == OperationDelete && operation != OperationDelete {
			operation = OperationCreate
		}
		if latest.Operation != OperationDelete && operation != OperationDelete && latest.ContentHash == contentHash {
			return nil, nil
		}
	} else if operation != OperationDelete {
		operation = OperationCreate
	}

	if strings.TrimSpace(author) == "" {
		author = "system:reconcile"
	}
	if _, err := lock.RequireLease(ctx); err != nil {
		return nil, err
	}
	return s.store.InsertVersion(ctx, InsertRequest{
		RuleKind:    kind,
		Mesh:        mesh,
		ResourceKey: resourceKey,
		RuleName:    ruleName,
		SpecJSON:    specJSON,
		ContentHash: contentHash,
		Operation:   operation,
		Source:      SourceUpstream,
		Author:      author,
		CreatedAt:   time.Now(),
	}, s.maxVersions)
}

func (s *Service) CurrentLedgerHead(kind coremodel.ResourceKind, resourceKey string) (*Version, bool, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, false, err
	}
	snapshot, err := s.store.LedgerSnapshot(kind, resourceKey)
	if err != nil {
		return nil, false, err
	}
	return snapshot.Head, snapshot.Deleted, nil
}

func (s *Service) GetVersion(kind coremodel.ResourceKind, resourceKey string, id int64) (*Version, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	return s.store.GetVersion(kind, resourceKey, id)
}

// repairIntent attempts to resolve a stale or stuck intent.
// Called during startup to recover from crashes, and before mutations to clear pending state.
//
// Why repair is needed:
//   - If admin crashes after creating an intent but before the subscriber commits,
//     the intent stays PENDING forever, blocking future writes
//   - If the actual resource state matches the intent's desired state, we can
//     safely commit the intent retroactively
//
// How to apply:
// - Repair runs automatically at startup (component.Start)
// - Also runs before each mutation (prepareRuleMutation) to clear stale intents
// - Returns IntentPendingError if the intent genuinely conflicts with current state
func (s *Service) repairIntent(ctx context.Context, intent *Intent, current coremodel.Resource, deleted bool) (*Version, error) {
	if intent == nil {
		return nil, nil
	}
	if _, err := lock.RequireLease(ctx); err != nil {
		return nil, err
	}
	if intent.Status == IntentStatusCommitted || intent.Status == IntentStatusFailed {
		return nil, ErrVersionIntentNotOpen
	}
	if !isOpenIntentStatus(intent.Status) {
		return nil, ErrVersionIntentNotOpen
	}
	if _, err := lock.RequireLease(ctx); err != nil {
		return nil, err
	}
	if intent.ReconcileRequired {
		return s.resolveObservedIntent(ctx, intent, current, deleted)
	}
	matches := IntentMatchesResource(intent, current, deleted)
	if intent.Status == IntentStatusPending || intent.Status == IntentStatusOutcomeUnknown {
		if !matches {
			if intent.Status == IntentStatusOutcomeUnknown {
				return s.failIntentAfterActualReconcile(ctx, intent, current, deleted, "registry mutation outcome did not match intended state")
			}
			return nil, &IntentPendingError{IntentID: intent.ID}
		}
		if _, err := lock.RequireLease(ctx); err != nil {
			return nil, err
		}
		if err := s.store.MarkIntentApplied(ctx, intent.ID); err != nil {
			return nil, err
		}
		if _, err := lock.RequireLease(ctx); err != nil {
			return nil, err
		}
		return s.store.CommitIntent(ctx, intent.ID, s.maxVersions)
	}
	if !matches {
		return nil, ErrIntentOutcomeMismatch
	}
	if _, err := lock.RequireLease(ctx); err != nil {
		return nil, err
	}
	return s.store.CommitIntent(ctx, intent.ID, s.maxVersions)
}

func (s *Service) resolveObservedIntent(ctx context.Context, intent *Intent, current coremodel.Resource, deleted bool) (*Version, error) {
	visible, err := observedStateVisible(intent, current, deleted)
	if err != nil {
		return nil, err
	}
	if !visible {
		return nil, &IntentPendingError{IntentID: intent.ID}
	}
	if IntentMatchesResource(intent, current, deleted) {
		if intent.Status == IntentStatusPending || intent.Status == IntentStatusOutcomeUnknown {
			if _, err := lock.RequireLease(ctx); err != nil {
				return nil, err
			}
			if err := s.store.MarkIntentApplied(ctx, intent.ID); err != nil {
				return nil, err
			}
		}
		if _, err := lock.RequireLease(ctx); err != nil {
			return nil, err
		}
		return s.store.CommitIntent(ctx, intent.ID, s.maxVersions)
	}
	return s.failIntentAfterActualReconcile(ctx, intent, current, deleted, "non-matching rule event superseded the open intent")
}

func (s *Service) failIntentAfterActualReconcile(ctx context.Context, intent *Intent, current coremodel.Resource, deleted bool, reason string) (*Version, error) {
	if _, err := lock.RequireLease(ctx); err != nil {
		return nil, err
	}
	if _, err := s.ReconcileActualState(ctx, intent.RuleKind, intent.ResourceKey, current, deleted, "system:reconcile"); err != nil {
		return nil, err
	}
	if _, err := lock.RequireLease(ctx); err != nil {
		return nil, err
	}
	if err := s.store.MarkIntentFailed(ctx, intent.ID, reason); err != nil {
		return nil, err
	}
	return nil, ErrIntentOutcomeMismatch
}

func observedStateVisible(intent *Intent, current coremodel.Resource, deleted bool) (bool, error) {
	if intent == nil || !intent.ReconcileRequired || intent.ObservedContentHash == "" {
		return true, nil
	}
	op := OperationUpdate
	hash := HashSpecJSON(DeleteSpecJSON)
	if deleted || current == nil {
		op = OperationDelete
	} else {
		normalizedHash, _, err := NormalizeResource(current)
		if err != nil {
			return false, err
		}
		hash = normalizedHash
	}
	return op == intent.ObservedOperation && hash == intent.ObservedContentHash, nil
}

func buildMutationInsertRequest(res coremodel.Resource, op Operation, source Source, author, reason string, rolledBackFromID *int64, createdAt time.Time) (InsertRequest, error) {
	if res == nil {
		return InsertRequest{}, bizerror.New(bizerror.InvalidArgument, "rule resource is required")
	}
	hash, specJSON, err := NormalizeResource(res)
	if op == OperationDelete {
		hash = HashSpecJSON(DeleteSpecJSON)
		specJSON = DeleteSpecJSON
		err = nil
	}
	if err != nil {
		return InsertRequest{}, err
	}
	if strings.TrimSpace(author) == "" {
		author = "system:unknown"
	} else {
		author = strings.TrimSpace(author)
	}
	if source == "" {
		source = SourceAdmin
	}
	return InsertRequest{
		RuleKind:         res.ResourceKind(),
		Mesh:             res.ResourceMesh(),
		ResourceKey:      res.ResourceKey(),
		RuleName:         res.ResourceMeta().Name,
		SpecJSON:         specJSON,
		ContentHash:      hash,
		Source:           source,
		Operation:        op,
		Author:           author,
		Reason:           reason,
		RolledBackFromID: rolledBackFromID,
		CreatedAt:        createdAt,
	}, nil
}

// IntentMatchesResource checks if the intent's desired state matches actual resource state.
// Used by repair logic to decide if a stale intent can be safely committed.
func IntentMatchesResource(intent *Intent, current coremodel.Resource, deleted bool) bool {
	if intent == nil {
		return false
	}
	if deleted || current == nil {
		return intent.Operation == OperationDelete && intent.ContentHash == HashSpecJSON(DeleteSpecJSON)
	}
	hash, _, err := NormalizeResource(current)
	return err == nil && hash == intent.ContentHash
}

func withRuleVersionLock(ctx context.Context, lockMgr lock.Lock, kind coremodel.ResourceKind, resourceKey string, fn func(context.Context) error) error {
	if lockMgr == nil {
		return lock.ErrLockUnavailable
	}
	key := lock.BuildRuleVersioningLockKey(string(kind), extractMesh(resourceKey), extractName(resourceKey))
	return lock.WithLock(ctx, lockMgr, key, constants.DefaultLockTimeout, fn)
}
