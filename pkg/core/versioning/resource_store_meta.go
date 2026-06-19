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
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

func (a *ResourceStoreAdapter) CheckExpectedVersion(kind coremodel.ResourceKind, resourceKey string, expected *int64) error {
	if expected == nil {
		return nil
	}

	var currentID *int64
	err := a.withParentLock(kind, resourceKey, func() error {
		state, err := a.ledgerState(kind, resourceKey)
		if err != nil {
			return err
		}
		if state.Latest != nil && state.Latest.Operation != OperationDelete {
			id := state.Latest.ID
			currentID = &id
		}
		return nil
	})
	if err != nil {
		return err
	}

	if currentID == nil {
		if *expected == 0 {
			return nil
		}
		return &ConflictError{CurrentVersionID: nil}
	}
	if *expected != *currentID {
		return &ConflictError{CurrentVersionID: currentID}
	}

	return nil
}

func (a *ResourceStoreAdapter) ReconcileMeta(ctx context.Context, kind coremodel.ResourceKind, resourceKey string) (*Meta, error) {
	var meta *Meta
	err := a.withParentLock(kind, resourceKey, func() error {
		if err := lock.CheckLease(ctx); err != nil {
			return err
		}
		var inner error
		meta, inner = a.reconcileMetaFromLedgerLocked(ctx, kind, resourceKey)
		return inner
	})
	return meta, err
}

func (a *ResourceStoreAdapter) reconcileMetaFromLedgerLocked(ctx context.Context, kind coremodel.ResourceKind, resourceKey string) (*Meta, error) {
	state, err := a.ledgerState(kind, resourceKey)
	if err != nil {
		return nil, err
	}
	if state.Latest == nil {
		return nil, nil
	}
	latest := state.Latest
	currentVersionID := latest.ID
	if latest.Operation == OperationDelete {
		currentVersionID = 0
	}

	mesh := extractMesh(resourceKey)
	name := extractName(resourceKey)
	metaName := buildMetaName(kind, resourceKey)
	fullKey := coremodel.BuildResourceKey(mesh, metaName)

	obj, exists, err := a.metaStore.GetByKey(fullKey)
	if err != nil {
		return nil, err
	}
	if !exists {
		metaRes := meshresource.NewRuleMetaResourceWithAttributes(metaName, mesh)
		metaRes.Spec = &meshproto.RuleMeta{
			ParentRuleKind:     string(kind),
			ParentRuleMesh:     mesh,
			ParentRuleName:     name,
			CurrentVersionId:   currentVersionID,
			CurrentVersionNo:   latest.VersionNo,
			CurrentContentHash: latest.ContentHash,
			UpdatedAt:          timestamppb.New(time.Now()),
		}
		if err := lock.CheckLease(ctx); err != nil {
			return nil, err
		}
		if err := a.metaStore.Add(metaRes); err != nil {
			return nil, err
		}
		return metaFromRuleMeta(kind, resourceKey, metaRes.Spec), nil
	}

	metaRes, ok := obj.(*meshresource.RuleMetaResource)
	if !ok {
		return nil, fmt.Errorf("%w: expected RuleMetaResource, got %T", ErrVersionLedgerCorrupt, obj)
	}
	if metaRes.Spec == nil {
		return nil, fmt.Errorf("%w: RuleMeta spec is nil for %s", ErrVersionLedgerCorrupt, fullKey)
	}
	spec := metaRes.Spec
	if spec.ParentRuleKind == string(kind) &&
		spec.ParentRuleMesh == mesh &&
		spec.ParentRuleName == name &&
		spec.CurrentVersionId == currentVersionID &&
		spec.CurrentVersionNo == latest.VersionNo &&
		spec.CurrentContentHash == latest.ContentHash {
		return metaFromRuleMeta(kind, resourceKey, spec), nil
	}

	updated := metaRes.DeepCopyObject().(*meshresource.RuleMetaResource)
	updated.Spec.ParentRuleKind = string(kind)
	updated.Spec.ParentRuleMesh = mesh
	updated.Spec.ParentRuleName = name
	updated.Spec.CurrentVersionId = currentVersionID
	updated.Spec.CurrentVersionNo = latest.VersionNo
	updated.Spec.CurrentContentHash = latest.ContentHash
	updated.Spec.UpdatedAt = timestamppb.New(time.Now())
	if err := lock.CheckLease(ctx); err != nil {
		return nil, err
	}
	if err := a.metaStore.Update(updated); err != nil {
		return nil, err
	}
	return metaFromRuleMeta(kind, resourceKey, updated.Spec), nil
}

func (a *ResourceStoreAdapter) LatestVersion(kind coremodel.ResourceKind, resourceKey string) (*Version, error) {
	snapshot, err := a.LedgerSnapshot(kind, resourceKey)
	if err != nil {
		return nil, err
	}
	if snapshot.Head == nil {
		return nil, ErrVersionNotFound
	}
	return snapshot.Head, nil
}

// Helper functions
