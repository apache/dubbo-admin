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

package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
)

const ruleLockTTL = 30 * time.Second

// RuleMutationOptions carries version control metadata for rule mutations.
// ExpectedVersionID is a weak CAS guard supplied by the UI; it prevents a user
// from mutating over another user's newer rule change.
type RuleMutationOptions struct {
	ExpectedVersionID *int64
	Author            string
	leaseCtx          context.Context
}

func (o RuleMutationOptions) WithLeaseContext(ctx context.Context) RuleMutationOptions {
	o.leaseCtx = ctx
	return o
}

func ensureMutationContext(ctx consolectx.Context, opts RuleMutationOptions) RuleMutationOptions {
	if opts.leaseCtx != nil {
		return opts
	}
	if ctx != nil {
		opts.leaseCtx = ctx.AppContext()
	}
	return opts
}

func ruleVersioning(ctx consolectx.Context) *versioning.Service {
	if ctx == nil {
		return nil
	}
	return ctx.RuleVersioning()
}

func checkExpectedVersion(ctx consolectx.Context, kindName RuleKindName, opts RuleMutationOptions) error {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil
	}
	return svc.CheckExpected(kindName.Kind, kindName.Mesh, kindName.Name, opts.ExpectedVersionID)
}

// prepareRuleMutation repairs stale intents before applying the weak CAS guard,
// so expectedVersionId is compared against the latest committed ledger state.
func prepareRuleMutation(ctx consolectx.Context, kindName RuleKindName, opts RuleMutationOptions) error {
	if err := checkMutationLease(opts); err != nil {
		return err
	}
	if err := repairPendingIntent(ctx, kindName, opts); err != nil {
		return err
	}
	if err := checkMutationLease(opts); err != nil {
		return err
	}
	return checkExpectedVersion(ctx, kindName, opts)
}

// repairPendingIntent commits an open intent only when ResourceManager state
// already reflects that mutation; otherwise the pending ledger blocks writes.
func repairPendingIntent(ctx consolectx.Context, kindName RuleKindName, opts RuleMutationOptions) error {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil
	}
	resourceKey := coremodel.BuildResourceKey(kindName.Mesh, kindName.Name)
	if err := checkMutationLease(opts); err != nil {
		return err
	}
	current, exists, err := ctx.ResourceManager().GetByKey(kindName.Kind, resourceKey)
	if err != nil {
		return err
	}
	if err := checkMutationLease(opts); err != nil {
		return err
	}
	_, err = svc.RepairIntent(opts.leaseCtx, kindName.Kind, resourceKey, current, !exists)
	return err
}

type RuleKindName struct {
	Kind coremodel.ResourceKind
	Mesh string
	Name string
}

func getExistingRule(ctx consolectx.Context, kindName RuleKindName) (coremodel.Resource, error) {
	key := coremodel.BuildResourceKey(kindName.Mesh, kindName.Name)
	res, exists, err := ctx.ResourceManager().GetByKey(kindName.Kind, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("%s %s does not exist", kindName.Kind, key)
	}
	return res, nil
}

func withRuleMutation(
	ctx consolectx.Context,
	kindName RuleKindName,
	opts RuleMutationOptions,
	loadResource func(RuleMutationOptions) (coremodel.Resource, error),
	op versioning.Operation,
	mutate func(RuleMutationOptions) error,
) error {
	execute := func(scoped RuleMutationOptions) error {
		scoped = ensureMutationContext(ctx, scoped)
		if err := prepareRuleMutation(ctx, kindName, scoped); err != nil {
			return err
		}
		res, err := loadResource(scoped)
		if err != nil {
			return err
		}
		return applyAdminMutation(ctx, res, op, scoped, func() error {
			if err := checkMutationLease(scoped); err != nil {
				return err
			}
			return mutate(scoped)
		})
	}

	lockMgr := ctx.LockManager()
	if lockMgr == nil {
		if ruleVersioning(ctx) != nil {
			return lock.ErrLockUnavailable
		}
		return execute(opts)
	}
	lockKey, err := ruleLockKey(kindName)
	if err != nil {
		return err
	}
	return lock.WithLock(ctx.AppContext(), lockMgr, lockKey, ruleLockTTL, func(leaseCtx context.Context) error {
		return execute(opts.WithLeaseContext(leaseCtx))
	})
}

// applyAdminMutation is a convenience wrapper for admin-initiated mutations.
func applyAdminMutation(ctx consolectx.Context, res coremodel.Resource, op versioning.Operation, opts RuleMutationOptions, mutate func() error) error {
	_, err := applyRuleMutationIntentWithOptions(ctx, res, op, versioning.SourceAdmin, opts, "", nil, mutate)
	return err
}

type MutationCommit struct {
	Intent  *versioning.Intent
	Version *versioning.Version
}

func applyRuleMutationIntentWithOptions(ctx consolectx.Context, res coremodel.Resource, op versioning.Operation, source versioning.Source, opts RuleMutationOptions, reason string, rolledBackFromID *int64, mutate func() error) (*MutationCommit, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		if err := checkMutationLease(opts); err != nil {
			return nil, err
		}
		return nil, mutate()
	}
	if err := checkMutationLease(opts); err != nil {
		return nil, err
	}
	intent, err := svc.BeginMutation(opts.leaseCtx, res, op, source, opts.Author, reason, rolledBackFromID)
	if err != nil {
		return nil, err
	}
	if intent == nil {
		if err := checkMutationLease(opts); err != nil {
			return nil, err
		}
		return nil, mutate()
	}
	if err := checkMutationLease(opts); err != nil {
		return nil, err
	}
	if err := mutate(); err != nil {
		// A registry error, timeout, or lease loss does not prove the remote
		// mutation failed. Keep the durable intent and reconcile actual state
		// under the same canonical rule lock before reporting the outcome.
		if markErr := svc.MarkIntentOutcomeUnknown(opts.leaseCtx, intent, err.Error()); markErr != nil && !errors.Is(markErr, versioning.ErrVersionIntentNotFound) {
			return nil, markErr
		}
		if version, finalizeErr := ensureMutationIntentCommitted(ctx, svc, intent, opts); finalizeErr == nil {
			return &MutationCommit{Intent: intent, Version: version}, nil
		}
		return nil, pendingLedgerError(intent.ID, err)
	}
	if err := checkMutationLease(opts); err != nil {
		return nil, err
	}
	version, err := ensureMutationIntentCommitted(ctx, svc, intent, opts)
	if err != nil {
		return nil, pendingLedgerError(intent.ID, err)
	}
	return &MutationCommit{Intent: intent, Version: version}, nil
}

func checkMutationLease(opts RuleMutationOptions) error {
	return lock.CheckLease(opts.leaseCtx)
}

func ensureMutationIntentCommitted(ctx consolectx.Context, svc *versioning.Service, intent *versioning.Intent, opts RuleMutationOptions) (*versioning.Version, error) {
	if err := checkMutationLease(opts); err != nil {
		return nil, err
	}
	current, exists, err := ctx.ResourceManager().GetByKey(intent.RuleKind, intent.ResourceKey)
	if err != nil {
		return nil, err
	}
	return svc.FinalizeMutation(opts.leaseCtx, intent, current, !exists)
}

func abandonIntentAndReconcile(ctx consolectx.Context, svc *versioning.Service, leaseCtx context.Context, intent *versioning.Intent, reason string) error {
	if err := svc.AbandonIntent(leaseCtx, intent, reason); err != nil {
		return err
	}
	if err := lock.CheckLease(leaseCtx); err != nil {
		return err
	}
	current, exists, err := ctx.ResourceManager().GetByKey(intent.RuleKind, intent.ResourceKey)
	if err != nil {
		return err
	}
	if err := lock.CheckLease(leaseCtx); err != nil {
		return err
	}
	_, err = svc.ReconcileActualState(leaseCtx, intent.RuleKind, intent.ResourceKey, current, !exists, "system:reconcile")
	return err
}

func pendingLedgerError(intentID int64, cause error) error {
	if errors.Is(cause, versioning.ErrVersionLedgerCorrupt) || errors.Is(cause, versioning.ErrIntentOutcomeMismatch) {
		return cause
	}
	return fmt.Errorf("%w: %v", &versioning.IntentPendingError{IntentID: intentID}, cause)
}

func ListRuleVersions(ctx consolectx.Context, kindName RuleKindName) (*versioning.ListResult, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, versioning.ErrVersionLedgerCorrupt
	}
	return svc.List(kindName.Kind, kindName.Mesh, kindName.Name)
}

func GetRuleVersion(ctx consolectx.Context, kindName RuleKindName, versionID int64) (*versioning.Version, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, versioning.ErrVersionLedgerCorrupt
	}
	return svc.Get(kindName.Kind, kindName.Mesh, kindName.Name, versionID)
}

func DiffRuleVersion(ctx consolectx.Context, kindName RuleKindName, versionID int64, against string) (*versioning.DiffResult, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, versioning.ErrVersionLedgerCorrupt
	}
	return svc.Diff(kindName.Kind, kindName.Mesh, kindName.Name, versionID, against)
}

func RepairRuleVersionIntent(ctx consolectx.Context, intentID int64) (*versioning.Version, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, versioning.ErrVersionLedgerCorrupt
	}
	intent, err := svc.GetIntent(intentID)
	if err != nil {
		return nil, err
	}
	kindName := ruleKindNameFromIntent(intent)
	var repaired *versioning.Version
	err = withRuleLock(ctx, kindName, func(leaseCtx context.Context) error {
		current, deleted, err := currentResourceForIntent(ctx, intentID)
		if err != nil {
			return err
		}
		if err := lock.CheckLease(leaseCtx); err != nil {
			return err
		}
		intent, err := svc.GetIntent(intentID)
		if err != nil {
			return err
		}
		repaired, err = svc.FinalizeMutation(leaseCtx, intent, current, deleted)
		return err
	})
	return repaired, err
}

func AbandonRuleVersionIntent(ctx consolectx.Context, intentID int64, reason string) error {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return versioning.ErrVersionLedgerCorrupt
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return bizerror.New(bizerror.InvalidArgument, "abandon reason is required")
	}
	intent, err := svc.GetIntent(intentID)
	if err != nil {
		return err
	}
	return withRuleLock(ctx, ruleKindNameFromIntent(intent), func(leaseCtx context.Context) error {
		intent, err := svc.GetIntent(intentID)
		if err != nil {
			return err
		}
		if intent.Status != versioning.IntentStatusPending &&
			intent.Status != versioning.IntentStatusApplied &&
			intent.Status != versioning.IntentStatusOutcomeUnknown {
			return bizerror.New(bizerror.InvalidArgument, "only open rule version intent can be abandoned")
		}
		current, exists, err := ctx.ResourceManager().GetByKey(intent.RuleKind, intent.ResourceKey)
		if err != nil {
			return err
		}
		if versioning.IntentMatchesResource(intent, current, !exists) {
			return bizerror.New(bizerror.InvalidArgument, "rule version intent matches the current resource; repair it instead")
		}
		if err := lock.CheckLease(leaseCtx); err != nil {
			return err
		}
		return abandonIntentAndReconcile(ctx, svc, leaseCtx, intent, reason)
	})
}

// RollbackResult summarizes a committed rollback for the API response.
type RollbackResult struct {
	RolledBackFromID int64  `json:"rolledBackFromId"`
	VersionID        int64  `json:"versionId"`
	VersionNo        int64  `json:"versionNo"`
	Source           string `json:"source"`
	Committed        bool   `json:"committed"`
}

// RollbackRuleVersion re-publishes the spec of a historical version as a new
// rule mutation. It does not modify historical versions; the resulting rule
// change is observed through the normal versioning flow and recorded as a new
// SourceRollback version.
func RollbackRuleVersion(ctx consolectx.Context, kindName RuleKindName, targetVersionID int64, reason string, expectedVersionID *int64, author string) (*RollbackResult, error) {
	var result *RollbackResult
	err := withRuleLock(ctx, kindName, func(leaseCtx context.Context) error {
		var inner error
		result, inner = rollbackRuleVersionLocked(ctx, kindName, targetVersionID, reason, expectedVersionID, author, leaseCtx)
		return inner
	})
	return result, err
}

func rollbackRuleVersionLocked(ctx consolectx.Context, kindName RuleKindName, targetVersionID int64, reason string, expectedVersionID *int64, author string, leaseCtx context.Context) (*RollbackResult, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, versioning.ErrVersionLedgerCorrupt
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, bizerror.New(bizerror.InvalidArgument, "rollback reason is required")
	}

	target, err := svc.Get(kindName.Kind, kindName.Mesh, kindName.Name, targetVersionID)
	if err != nil {
		return nil, err
	}
	if target.Operation == versioning.OperationDelete {
		// A delete marker represents absence of a rule. Treating it as a
		// rollback target would turn rollback into a delete operation, which is
		// intentionally kept out of scope for this endpoint.
		return nil, versioning.ErrRollbackToDelete
	}

	// Repair stale intents and enforce optimistic locking before touching state.
	opts := RuleMutationOptions{ExpectedVersionID: expectedVersionID, Author: author}.WithLeaseContext(leaseCtx)
	if err := prepareRuleMutation(ctx, kindName, opts); err != nil {
		return nil, err
	}

	resourceKey := coremodel.BuildResourceKey(kindName.Mesh, kindName.Name)
	current, currentDeleted, err := svc.CurrentLedgerHead(kindName.Kind, resourceKey)
	if err != nil {
		return nil, err
	}
	if current != nil && !currentDeleted {
		if current.ContentHash == target.ContentHash {
			return nil, versioning.ErrRollbackToCurrent
		}
	}

	res, err := versioning.ResourceFromSpecJSON(kindName.Kind, kindName.Mesh, kindName.Name, target.SpecJSON)
	if err != nil {
		return nil, err
	}

	fromID := target.ID
	operation := versioning.OperationUpdate
	if currentDeleted {
		operation = versioning.OperationCreate
	}
	commit, err := applyRuleMutationIntentWithOptions(ctx, res, operation, versioning.SourceRollback, opts, reason, &fromID, func() error {
		if err := checkMutationLease(opts); err != nil {
			return err
		}
		return ctx.ResourceManager().Upsert(opts.leaseCtx, res)
	})
	if err != nil {
		return nil, err
	}
	if commit == nil || commit.Intent == nil || commit.Version == nil {
		return nil, fmt.Errorf("rollback intent was not created for %s", resourceKey)
	}

	committed, err := validateRollbackCommit(commit.Version, kindName.Kind, resourceKey, commit.Intent.ID, fromID, target)
	if err != nil {
		return nil, err
	}

	return &RollbackResult{
		RolledBackFromID: fromID,
		VersionID:        committed.ID,
		VersionNo:        committed.VersionNo,
		Source:           string(committed.Source),
		Committed:        true,
	}, nil
}

func validateRollbackCommit(current *versioning.Version, kind coremodel.ResourceKind, resourceKey string, intentID, rolledBackFromID int64, target *versioning.Version) (*versioning.Version, error) {
	if current == nil ||
		current.IntentID != intentID ||
		current.RuleKind != kind ||
		current.ResourceKey != resourceKey ||
		current.Source != versioning.SourceRollback ||
		(current.Operation != versioning.OperationUpdate && current.Operation != versioning.OperationCreate) ||
		current.RolledBackFromID == nil ||
		*current.RolledBackFromID != rolledBackFromID ||
		target == nil ||
		current.ContentHash != target.ContentHash ||
		current.SpecJSON != target.SpecJSON {
		return nil, fmt.Errorf("rollback version commit was not observed for %s", resourceKey)
	}
	current.IsCurrent = true
	return current, nil
}

func ruleKindNameFromIntent(intent *versioning.Intent) RuleKindName {
	if intent == nil {
		return RuleKindName{}
	}
	return RuleKindName{
		Kind: intent.RuleKind,
		Mesh: intent.Mesh,
		Name: intent.RuleName,
	}
}

func currentResourceForIntent(ctx consolectx.Context, intentID int64) (coremodel.Resource, bool, error) {
	intent, err := ctx.RuleVersioning().GetIntent(intentID)
	if err != nil {
		return nil, false, err
	}
	current, exists, err := ctx.ResourceManager().GetByKey(intent.RuleKind, intent.ResourceKey)
	// Repair APIs pass deleted=true when the resource manager no longer has the rule.
	return current, !exists, err
}

func withRuleLock(ctx consolectx.Context, kindName RuleKindName, fn func(context.Context) error) error {
	lockMgr := ctx.LockManager()
	if lockMgr == nil {
		return lock.ErrLockUnavailable
	}
	lockKey, err := ruleLockKey(kindName)
	if err != nil {
		return err
	}
	return lock.WithLock(ctx.AppContext(), lockMgr, lockKey, ruleLockTTL, fn)
}

func ruleLockKey(kindName RuleKindName) (string, error) {
	switch kindName.Kind {
	case meshresource.ConditionRouteKind:
		return lock.BuildConditionRuleLockKey(kindName.Mesh, kindName.Name), nil
	case meshresource.TagRouteKind:
		return lock.BuildTagRouteLockKey(kindName.Mesh, kindName.Name), nil
	case meshresource.DynamicConfigKind:
		return lock.BuildConfiguratorRuleLockKey(kindName.Mesh, kindName.Name), nil
	default:
		return "", bizerror.New(bizerror.InvalidArgument, "unsupported rule kind")
	}
}
