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
	"fmt"
	"strings"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
)

type RuleMutationOptions struct {
	ExpectedVersionID *int64
	Author            string
}

func ruleVersioning(ctx consolectx.Context) versioning.Service {
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

func prepareRuleMutation(ctx consolectx.Context, kindName RuleKindName, opts RuleMutationOptions) error {
	if err := repairPendingIntent(ctx, kindName); err != nil {
		return err
	}
	return checkExpectedVersion(ctx, kindName, opts)
}

func repairPendingIntent(ctx consolectx.Context, kindName RuleKindName) error {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil
	}
	resourceKey := coremodel.BuildResourceKey(kindName.Mesh, kindName.Name)
	current, exists, err := ctx.ResourceManager().GetByKey(kindName.Kind, resourceKey)
	if err != nil {
		return err
	}
	_, err = svc.RepairIntent(kindName.Kind, resourceKey, current, !exists)
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

func applyAdminMutation(ctx consolectx.Context, res coremodel.Resource, op versioning.Operation, opts RuleMutationOptions, mutate func() error) error {
	_, err := applyRuleMutationIntent(ctx, res, op, versioning.SourceAdmin, opts.Author, "", opts.ExpectedVersionID, nil, mutate)
	return err
}

func applyRuleMutationIntent(ctx consolectx.Context, res coremodel.Resource, op versioning.Operation, source versioning.Source, author, reason string, expected *int64, rolledBackFromID *int64, mutate func() error) (*versioning.Version, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, mutate()
	}
	intent, err := svc.BeginMutationIntent(res, op, source, author, reason, expected, rolledBackFromID)
	if err != nil {
		return nil, err
	}
	if intent == nil {
		return nil, mutate()
	}
	if err := mutate(); err != nil {
		if markErr := svc.FailMutationIntent(intent.ID, err.Error()); markErr != nil {
			return nil, fmt.Errorf("%w; failed to mark version intent failed: %v", err, markErr)
		}
		return nil, err
	}
	if err := svc.MarkMutationIntentApplied(intent.ID); err != nil {
		return nil, err
	}
	return svc.CommitMutationIntent(intent.ID)
}

func ListRuleVersions(ctx consolectx.Context, kindName RuleKindName) (*versioning.ListResult, error) {
	return ctx.RuleVersioning().List(kindName.Kind, kindName.Mesh, kindName.Name)
}

func GetRuleVersion(ctx consolectx.Context, kindName RuleKindName, versionID int64) (*versioning.Version, error) {
	return ctx.RuleVersioning().Get(kindName.Kind, kindName.Mesh, kindName.Name, versionID)
}

func DiffRuleVersion(ctx consolectx.Context, kindName RuleKindName, versionID int64, against string) (*versioning.DiffResult, error) {
	return ctx.RuleVersioning().Diff(kindName.Kind, kindName.Mesh, kindName.Name, versionID, against)
}

func RepairRuleVersionIntent(ctx consolectx.Context, intentID int64) (*versioning.Version, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, versioning.ErrFeatureDisabled
	}
	intent, err := svc.Store().GetIntent(intentID)
	if err != nil {
		return nil, err
	}
	kindName := ruleKindNameFromIntent(intent)
	var repaired *versioning.Version
	err = withRuleLock(ctx, kindName, func() error {
		current, deleted, err := currentResourceForIntent(ctx, intentID)
		if err != nil {
			return err
		}
		repaired, err = svc.RepairIntentByID(intentID, current, deleted)
		return err
	})
	return repaired, err
}

func AbandonRuleVersionIntent(ctx consolectx.Context, intentID int64, reason string) error {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return versioning.ErrFeatureDisabled
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return bizerror.New(bizerror.InvalidArgument, "abandon reason is required")
	}
	intent, err := svc.Store().GetIntent(intentID)
	if err != nil {
		return err
	}
	return withRuleLock(ctx, ruleKindNameFromIntent(intent), func() error {
		intent, err := svc.Store().GetIntent(intentID)
		if err != nil {
			return err
		}
		if intent.Status != versioning.IntentStatusPending {
			return bizerror.New(bizerror.InvalidArgument, "only pending rule version intent can be abandoned")
		}
		current, exists, err := ctx.ResourceManager().GetByKey(intent.RuleKind, intent.ResourceKey)
		if err != nil {
			return err
		}
		if versioning.IntentMatchesResource(intent, current, !exists) {
			return bizerror.New(bizerror.InvalidArgument, "rule version intent matches the current resource; repair it instead")
		}
		return svc.Store().MarkIntentFailedWithReason(intent.ID, reason)
	})
}

func RollbackRuleVersion(ctx consolectx.Context, kindName RuleKindName, versionID int64, reason string, expected *int64, author string) (*versioning.Version, error) {
	var rollback *versioning.Version
	err := withRuleLock(ctx, kindName, func() error {
		var err error
		rollback, err = rollbackRuleVersionUnsafe(ctx, kindName, versionID, reason, expected, author)
		return err
	})
	return rollback, err
}

func rollbackRuleVersionUnsafe(ctx consolectx.Context, kindName RuleKindName, versionID int64, reason string, expected *int64, author string) (*versioning.Version, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, versioning.ErrFeatureDisabled
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, bizerror.New(bizerror.InvalidArgument, "rollback reason is required")
	}
	if err := prepareRuleMutation(ctx, kindName, RuleMutationOptions{ExpectedVersionID: expected, Author: author}); err != nil {
		return nil, err
	}
	target, err := svc.Get(kindName.Kind, kindName.Mesh, kindName.Name, versionID)
	if err != nil {
		return nil, err
	}
	if target.Operation == versioning.OperationDelete {
		return nil, versioning.ErrRollbackToDelete
	}
	if target.IsCurrent {
		return nil, versioning.ErrRollbackToCurrent
	}
	resourceKey := coremodel.BuildResourceKey(kindName.Mesh, kindName.Name)
	meta, err := svc.Store().CurrentMeta(kindName.Kind, resourceKey)
	if err != nil {
		return nil, err
	}
	if meta != nil && meta.CurrentVersion != nil {
		current, err := svc.Store().GetVersion(kindName.Kind, resourceKey, *meta.CurrentVersion)
		if err != nil {
			return nil, err
		}
		if current.ID == target.ID || current.ContentHash == target.ContentHash {
			return nil, versioning.ErrRollbackToCurrent
		}
	}
	res, err := versioning.ResourceFromSpecJSON(kindName.Kind, kindName.Mesh, kindName.Name, target.SpecJSON)
	if err != nil {
		return nil, err
	}
	fromID := target.ID
	return applyRuleMutationIntent(ctx, res, versioning.OperationUpdate, versioning.SourceRollback, author, reason, expected, &fromID, func() error {
		return ctx.ResourceManager().Upsert(res)
	})
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
	intent, err := ctx.RuleVersioning().Store().GetIntent(intentID)
	if err != nil {
		return nil, false, err
	}
	current, exists, err := ctx.ResourceManager().GetByKey(intent.RuleKind, intent.ResourceKey)
	return current, !exists, err
}

func withRuleLock(ctx consolectx.Context, kindName RuleKindName, fn func() error) error {
	lockMgr := ctx.LockManager()
	if lockMgr == nil {
		return fn()
	}
	lockKey, err := ruleLockKey(kindName)
	if err != nil {
		return err
	}
	return lockMgr.WithLock(ctx.AppContext(), lockKey, constants.DefaultLockTimeout, fn)
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
