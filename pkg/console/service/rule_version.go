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

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
)

type RuleMutationOptions struct {
	Author string
}

type RuleKindName struct {
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

func appendRuleHistory(ctx consolectx.Context, res coremodel.Resource, op versioning.Operation, source versioning.Source, author, reason string, rolledBackFromID *int64) (*versioning.Version, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, nil
	}
	return svc.Append(ctx.AppContext(), res, op, source, author, reason, rolledBackFromID)
}

func appendRuleHistoryBestEffort(ctx consolectx.Context, res coremodel.Resource, op versioning.Operation, source versioning.Source, author, reason string, rolledBackFromID *int64) *versioning.Version {
	version, err := appendRuleHistory(ctx, res, op, source, author, reason, rolledBackFromID)
	if err != nil {
		logger.Warnf("append rule history failed for kind=%s key=%s operation=%s: %v", res.ResourceKind(), res.ResourceKey(), op, err)
		return nil
	}
	return version
}

func ensureBaselineHistoryBestEffort(ctx consolectx.Context, res coremodel.Resource) {
	svc := ruleVersioning(ctx)
	if svc == nil || res == nil {
		return
	}
	hasHistory, err := svc.HasHistory(res.ResourceKind(), res.ResourceMesh(), res.ResourceMeta().Name)
	if err != nil {
		logger.Warnf("check rule history baseline failed for kind=%s key=%s: %v", res.ResourceKind(), res.ResourceKey(), err)
		return
	}
	if hasHistory {
		return
	}
	if _, err := svc.Append(ctx.AppContext(), res, versioning.OperationCreate, versioning.SourceBootstrap, "system:baseline", "import existing rule before first edit", nil); err != nil {
		logger.Warnf("append rule history baseline failed for kind=%s key=%s: %v", res.ResourceKind(), res.ResourceKey(), err)
	}
}

func createRule(ctx consolectx.Context, res coremodel.Resource, opts RuleMutationOptions) error {
	if err := ctx.ResourceManager().Add(ctx.AppContext(), res); err != nil {
		return err
	}
	appendRuleHistoryBestEffort(ctx, res, versioning.OperationCreate, versioning.SourceAdmin, opts.Author, "", nil)
	return nil
}

func updateRule(ctx consolectx.Context, res coremodel.Resource, opts RuleMutationOptions) error {
	existing, err := getExistingRule(ctx, RuleKindName{Kind: res.ResourceKind(), Mesh: res.ResourceMesh(), Name: res.ResourceMeta().Name})
	if err != nil {
		return err
	}
	ensureBaselineHistoryBestEffort(ctx, existing)
	if err := ctx.ResourceManager().Update(ctx.AppContext(), res); err != nil {
		return err
	}
	appendRuleHistoryBestEffort(ctx, res, versioning.OperationUpdate, versioning.SourceAdmin, opts.Author, "", nil)
	return nil
}

func deleteRule(ctx consolectx.Context, kindName RuleKindName, opts RuleMutationOptions) error {
	snapshot, err := getExistingRule(ctx, kindName)
	if err != nil {
		return err
	}
	ensureBaselineHistoryBestEffort(ctx, snapshot)
	if err := ctx.ResourceManager().DeleteByKey(ctx.AppContext(), kindName.Kind, kindName.Mesh, coremodel.BuildResourceKey(kindName.Mesh, kindName.Name)); err != nil {
		return err
	}
	appendRuleHistoryBestEffort(ctx, snapshot, versioning.OperationDelete, versioning.SourceAdmin, opts.Author, "", nil)
	return nil
}

func ListRuleVersions(ctx consolectx.Context, kindName RuleKindName) (*versioning.ListResult, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, versioning.ErrVersionStoreError
	}
	return svc.List(kindName.Kind, kindName.Mesh, kindName.Name)
}

func GetRuleVersion(ctx consolectx.Context, kindName RuleKindName, versionID int64) (*versioning.Version, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, versioning.ErrVersionStoreError
	}
	return svc.Get(kindName.Kind, kindName.Mesh, kindName.Name, versionID)
}

func DiffRuleVersion(ctx consolectx.Context, kindName RuleKindName, versionID int64, against string) (*versioning.DiffResult, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, versioning.ErrVersionStoreError
	}
	if against != "" && against != "current" {
		return svc.Diff(kindName.Kind, kindName.Mesh, kindName.Name, versionID, against)
	}
	left, err := svc.Get(kindName.Kind, kindName.Mesh, kindName.Name, versionID)
	if err != nil {
		return nil, err
	}
	current, exists, err := ctx.ResourceManager().GetByKey(kindName.Kind, coremodel.BuildResourceKey(kindName.Mesh, kindName.Name))
	if err != nil {
		return nil, err
	}
	specJSON := versioning.DeleteSpecJSON
	if exists && current != nil {
		_, specJSON, err = versioning.NormalizeResource(current)
		if err != nil {
			return nil, err
		}
	}
	return &versioning.DiffResult{
		Left:  versioning.DiffSide{ID: left.ID, VersionNo: left.VersionNo, SpecJSON: left.SpecJSON},
		Right: versioning.DiffSide{ID: 0, VersionNo: 0, SpecJSON: specJSON},
	}, nil
}

// RollbackResult summarizes a rollback write for the API response.
type RollbackResult struct {
	RolledBackFromID int64  `json:"rolledBackFromId"`
	VersionID        int64  `json:"versionId"`
	VersionNo        int64  `json:"versionNo"`
	Source           string `json:"source"`
	HistoryRecorded  bool   `json:"historyRecorded"`
}

func RollbackRuleVersion(ctx consolectx.Context, kindName RuleKindName, targetVersionID int64, reason string, author string) (*RollbackResult, error) {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil, versioning.ErrVersionStoreError
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
		return nil, versioning.ErrRollbackToDelete
	}
	if target.Operation != versioning.OperationCreate && target.Operation != versioning.OperationUpdate {
		return nil, bizerror.New(bizerror.InvalidArgument, "only CREATE or UPDATE rule versions can be rolled back")
	}

	resourceKey := coremodel.BuildResourceKey(kindName.Mesh, kindName.Name)
	current, exists, err := ctx.ResourceManager().GetByKey(kindName.Kind, resourceKey)
	if err != nil {
		return nil, err
	}
	if exists && current != nil {
		hash, _, err := versioning.NormalizeResource(current)
		if err != nil {
			return nil, err
		}
		if hash == target.ContentHash {
			return nil, versioning.ErrRollbackToCurrent
		}
	}

	res, err := versioning.ResourceFromSpecJSON(kindName.Kind, kindName.Mesh, kindName.Name, target.SpecJSON)
	if err != nil {
		return nil, err
	}

	operation := versioning.OperationUpdate
	if !exists {
		operation = versioning.OperationCreate
	}
	if err := ctx.ResourceManager().Upsert(ctx.AppContext(), res); err != nil {
		return nil, err
	}

	fromID := target.ID
	appended := appendRuleHistoryBestEffort(ctx, res, operation, versioning.SourceRollback, author, reason, &fromID)
	result := &RollbackResult{
		RolledBackFromID: fromID,
		Source:           string(versioning.SourceRollback),
		HistoryRecorded:  appended != nil,
	}
	if appended != nil {
		result.VersionID = appended.ID
		result.VersionNo = appended.VersionNo
	}
	return result, nil
}

func AppContextOrBackground(ctx consolectx.Context) context.Context {
	if ctx == nil || ctx.AppContext() == nil {
		return context.Background()
	}
	return ctx.AppContext()
}

func IsRuleVersionNotFound(err error) bool {
	return errors.Is(err, versioning.ErrVersionNotFound)
}

func SupportedRuleKind(kind coremodel.ResourceKind) bool {
	switch kind {
	case meshresource.ConditionRouteKind, meshresource.TagRouteKind, meshresource.DynamicConfigKind:
		return true
	default:
		return false
	}
}
