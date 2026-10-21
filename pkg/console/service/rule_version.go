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
	Author string
}

type RuleRef struct {
	Kind coremodel.ResourceKind
	Mesh string
	Name string
}

func ruleVersioning(ctx consolectx.Context) *versioning.Service {
	if ctx == nil {
		return nil
	}
	return ctx.RuleVersioning()
}

func requiredRuleVersioning(ctx consolectx.Context) (*versioning.Service, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, versioning.ErrVersionStoreError
	}
	return svc, nil
}

// getRuleIfExists reads the live ResourceManager state, not recorded history.
func getRuleIfExists(ctx consolectx.Context, ruleRef RuleRef) (coremodel.Resource, bool, error) {
	key := coremodel.BuildResourceKey(ruleRef.Mesh, ruleRef.Name)
	res, exists, err := ctx.ResourceManager().GetByKey(ruleRef.Kind, key)
	if err != nil {
		return nil, false, err
	}
	if !exists || res == nil {
		return nil, false, nil
	}
	return res, true, nil
}

func getExistingRule(ctx consolectx.Context, ruleRef RuleRef) (coremodel.Resource, error) {
	res, exists, err := getRuleIfExists(ctx, ruleRef)
	if err != nil {
		return nil, err
	}
	if !exists {
		key := coremodel.BuildResourceKey(ruleRef.Mesh, ruleRef.Name)
		return nil, fmt.Errorf("%s %s does not exist", ruleRef.Kind, key)
	}
	return res, nil
}

func appendRuleHistory(ctx consolectx.Context, res coremodel.Resource, op versioning.Operation, source versioning.Source, author, reason string, rolledBackFromVersionNo *int64) (*versioning.Version, error) {
	svc, err := requiredRuleVersioning(ctx)
	if err != nil {
		return nil, err
	}
	return svc.Append(ctx.AppContext(), res, op, source, author, reason, rolledBackFromVersionNo)
}

func ensureBaselineHistory(ctx consolectx.Context, res coremodel.Resource) error {
	svc, err := requiredRuleVersioning(ctx)
	if err != nil {
		return err
	}
	if res == nil {
		return nil
	}
	hasHistory, err := svc.HasHistory(res.ResourceKind(), res.ResourceMesh(), res.ResourceMeta().Name)
	if err != nil {
		return err
	}
	if hasHistory {
		return nil
	}
	if _, err := svc.Append(ctx.AppContext(), res, versioning.OperationCreate, versioning.SourceBootstrap, "system:baseline", "import existing rule before first edit", nil); err != nil {
		return err
	}
	return nil
}

func withRuleLock(ctx consolectx.Context, ruleRef RuleRef, fn func() error) error {
	lockMgr := ctx.LockManager()
	if lockMgr == nil {
		return fn()
	}
	lockKey := ruleLockKey(ruleRef)
	return lockMgr.WithLock(ctx.AppContext(), lockKey, constants.DefaultLockTimeout, fn)
}

func ruleLockKey(ruleRef RuleRef) string {
	switch ruleRef.Kind {
	case meshresource.ConditionRouteKind:
		return lock.BuildConditionRuleLockKey(ruleRef.Mesh, ruleRef.Name)
	case meshresource.TagRouteKind:
		return lock.BuildTagRouteLockKey(ruleRef.Mesh, ruleRef.Name)
	case meshresource.DynamicConfigKind:
		return lock.BuildConfiguratorRuleLockKey(ruleRef.Mesh, ruleRef.Name)
	default:
		return lock.BuildLockKey(ruleRef.Kind.ToString(), ruleRef.Mesh, ruleRef.Name)
	}
}

func createRule(ctx consolectx.Context, res coremodel.Resource, opts RuleMutationOptions) error {
	if err := meshresource.ValidateRule(res); err != nil {
		return err
	}
	ruleRef := RuleRef{Kind: res.ResourceKind(), Mesh: res.ResourceMesh(), Name: res.ResourceMeta().Name}
	return withRuleLock(ctx, ruleRef, func() error {
		if _, err := appendRuleHistory(ctx, res, versioning.OperationCreate, versioning.SourceAdmin, opts.Author, "", nil); err != nil {
			return err
		}
		if err := ctx.ResourceManager().Add(res); err != nil {
			return err
		}
		return nil
	})
}

func updateRule(ctx consolectx.Context, res coremodel.Resource, opts RuleMutationOptions) error {
	if err := meshresource.ValidateRule(res); err != nil {
		return err
	}
	ruleRef := RuleRef{Kind: res.ResourceKind(), Mesh: res.ResourceMesh(), Name: res.ResourceMeta().Name}
	return withRuleLock(ctx, ruleRef, func() error {
		existing, err := getExistingRule(ctx, ruleRef)
		if err != nil {
			return err
		}
		if err := ensureBaselineHistory(ctx, existing); err != nil {
			return err
		}
		if _, err := appendRuleHistory(ctx, res, versioning.OperationUpdate, versioning.SourceAdmin, opts.Author, "", nil); err != nil {
			return err
		}
		if err := ctx.ResourceManager().Update(res); err != nil {
			return err
		}
		return nil
	})
}

func deleteRule(ctx consolectx.Context, ruleRef RuleRef, opts RuleMutationOptions) error {
	return withRuleLock(ctx, ruleRef, func() error {
		resourceKey := coremodel.BuildResourceKey(ruleRef.Mesh, ruleRef.Name)
		snapshot, exists, err := ctx.ResourceManager().GetByKey(ruleRef.Kind, resourceKey)
		if err != nil {
			return err
		}
		if !exists || snapshot == nil {
			return nil
		}
		if err := ensureBaselineHistory(ctx, snapshot); err != nil {
			return err
		}
		if _, err := appendRuleHistory(ctx, snapshot, versioning.OperationDelete, versioning.SourceAdmin, opts.Author, "", nil); err != nil {
			return err
		}
		if err := ctx.ResourceManager().DeleteByKey(ruleRef.Kind, ruleRef.Mesh, resourceKey); err != nil {
			return err
		}
		return nil
	})
}

func ListRuleVersions(ctx consolectx.Context, ruleRef RuleRef) (*versioning.ListResult, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, versioning.ErrVersionStoreError
	}
	return svc.List(ruleRef.Kind, ruleRef.Mesh, ruleRef.Name)
}

func GetRuleVersion(ctx consolectx.Context, ruleRef RuleRef, versionNo int64) (*versioning.Version, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, versioning.ErrVersionStoreError
	}
	return svc.Get(ruleRef.Kind, ruleRef.Mesh, ruleRef.Name, versionNo)
}

func DiffRuleVersion(ctx consolectx.Context, ruleRef RuleRef, versionNo int64, against string) (*versioning.DiffResult, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, versioning.ErrVersionStoreError
	}
	if against != "" && against != "current" {
		return svc.DiffHistoryVersions(ruleRef.Kind, ruleRef.Mesh, ruleRef.Name, versionNo, against)
	}
	left, err := svc.Get(ruleRef.Kind, ruleRef.Mesh, ruleRef.Name, versionNo)
	if err != nil {
		return nil, err
	}
	current, exists, err := getRuleIfExists(ctx, ruleRef)
	if err != nil {
		return nil, err
	}
	specJSON := versioning.DeleteSpecJSON
	if exists {
		_, specJSON, err = versioning.NormalizeResource(current)
		if err != nil {
			return nil, err
		}
	}
	return &versioning.DiffResult{
		Left:  versioning.DiffSide{VersionNo: left.VersionNo, SpecJSON: left.SpecJSON},
		Right: versioning.DiffSide{VersionNo: 0, SpecJSON: specJSON},
	}, nil
}

// RollbackResult summarizes a rollback write for the API response.
type RollbackResult struct {
	RolledBackFromVersionNo int64
	VersionNo               int64
	Source                  string
}

func RollbackRuleVersion(ctx consolectx.Context, ruleRef RuleRef, targetVersionNo int64, reason string, author string) (*RollbackResult, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, versioning.ErrVersionStoreError
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, bizerror.New(bizerror.InvalidArgument, "rollback reason is required")
	}

	var result *RollbackResult
	err := withRuleLock(ctx, ruleRef, func() error {
		target, err := svc.Get(ruleRef.Kind, ruleRef.Mesh, ruleRef.Name, targetVersionNo)
		if err != nil {
			return err
		}
		if target.Operation == versioning.OperationDelete {
			return versioning.ErrRollbackToDelete
		}
		if target.Operation != versioning.OperationCreate && target.Operation != versioning.OperationUpdate {
			return bizerror.New(bizerror.InvalidArgument, "only CREATE or UPDATE rule versions can be rolled back")
		}

		current, exists, err := getRuleIfExists(ctx, ruleRef)
		if err != nil {
			return err
		}
		if exists {
			hash, _, err := versioning.NormalizeResource(current)
			if err != nil {
				return err
			}
			if hash == target.ContentHash {
				return versioning.ErrRollbackToCurrent
			}
		}

		res, err := versioning.ResourceFromSpecJSON(ruleRef.Kind, ruleRef.Mesh, ruleRef.Name, target.SpecJSON)
		if err != nil {
			return err
		}

		operation := versioning.OperationUpdate
		if !exists {
			operation = versioning.OperationCreate
		}
		rolledBackFromVersionNo := target.VersionNo
		appended, err := appendRuleHistory(ctx, res, operation, versioning.SourceRollback, author, reason, &rolledBackFromVersionNo)
		if err != nil {
			return err
		}
		if err := ctx.ResourceManager().Upsert(res); err != nil {
			return err
		}

		result = &RollbackResult{
			RolledBackFromVersionNo: rolledBackFromVersionNo,
			Source:                  string(versioning.SourceRollback),
			VersionNo:               appended.VersionNo,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
