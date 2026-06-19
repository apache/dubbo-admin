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
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

func (a *ResourceStoreAdapter) GetVersion(kind coremodel.ResourceKind, resourceKey string, id int64) (*Version, error) {
	versionName := buildVersionName(kind, resourceKey, id)
	fullKey := coremodel.BuildResourceKey(extractMesh(resourceKey), versionName)
	obj, exists, err := a.versionStore.GetByKey(fullKey)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrVersionNotFound
	}
	rv, ok := obj.(*meshresource.RuleVersionResource)
	if !ok {
		return nil, fmt.Errorf("%w: expected RuleVersionResource, got %T", ErrVersionLedgerCorrupt, obj)
	}
	if rv.Spec == nil {
		return nil, fmt.Errorf("%w: RuleVersion spec is nil for id %d", ErrVersionLedgerCorrupt, id)
	}
	return protoToVersion(rv.Spec, id)
}

func (a *ResourceStoreAdapter) ListVersions(kind coremodel.ResourceKind, resourceKey string) ([]Version, error) {
	state, err := a.ledgerState(kind, resourceKey)
	if err != nil {
		return nil, err
	}
	return state.Versions, nil
}

func (a *ResourceStoreAdapter) ledgerState(kind coremodel.ResourceKind, resourceKey string) (*ledgerState, error) {
	parentKey := buildParentIndexKey(kind, resourceKey)
	objs, err := a.versionStore.ByIndex(index.ByParentRuleIndexName, parentKey)
	if err != nil {
		return nil, err
	}

	versions := make([]Version, 0, len(objs))
	for _, obj := range objs {
		rv, ok := obj.(*meshresource.RuleVersionResource)
		if !ok {
			return nil, fmt.Errorf("%w: expected RuleVersionResource, got %T", ErrVersionLedgerCorrupt, obj)
		}
		if rv.Spec == nil {
			return nil, fmt.Errorf("%w: RuleVersion spec is nil for parent %s", ErrVersionLedgerCorrupt, parentKey)
		}
		id, err := extractIDFromName(rv.Name)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrVersionLedgerCorrupt, err)
		}
		v, err := protoToVersion(rv.Spec, id)
		if err != nil {
			return nil, err
		}
		versions = append(versions, *v)
	}

	seenVersionNo := make(map[int64]int64, len(versions))
	for _, version := range versions {
		if previousID, ok := seenVersionNo[version.VersionNo]; ok && previousID != version.ID {
			return nil, duplicateVersionNoError(kind, resourceKey, version.VersionNo, previousID, version.ID)
		}
		seenVersionNo[version.VersionNo] = version.ID
	}

	// Sort by version number descending (newest first).
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].VersionNo > versions[j].VersionNo
	})

	state := &ledgerState{Versions: versions}
	if len(versions) > 0 {
		state.Latest = &versions[0]
		state.MaxVersionNo = versions[0].VersionNo
	}
	return state, nil
}

func (a *ResourceStoreAdapter) InsertVersion(req InsertRequest, maxVersions int64) (*Version, error) {
	var version *Version
	err := a.withParentLock(req.RuleKind, req.ResourceKey, func() error {
		var inner error
		version, inner = a.insertVersionLocked(req, maxVersions)
		return inner
	})
	return version, err
}

func (a *ResourceStoreAdapter) insertVersionLocked(req InsertRequest, maxVersions int64) (*Version, error) {
	if _, err := a.reconcileMetaFromLedgerLocked(req.RuleKind, req.ResourceKey); err != nil {
		return nil, err
	}
	state, err := a.ledgerState(req.RuleKind, req.ResourceKey)
	if err != nil {
		return nil, err
	}

	versionNo := state.MaxVersionNo + 1

	createdAt := req.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	committedAt := time.Now()

	versionExists := false
	var rv *meshresource.RuleVersionResource
	var id int64
	if req.IntentID != 0 {
		existing, existingID, err := a.getVersionResourceByIntentID(req.IntentID)
		if err == nil {
			if validateErr := validateExistingVersionForRequest(existing, existingID, req); validateErr != nil {
				return nil, validateErr
			}
			rv = existing
			id = existingID
			versionNo = existing.Spec.VersionNo
			versionExists = true
		} else if !errors.Is(err, ErrVersionNotFound) {
			return nil, err
		}
	}
	if req.FixedVersionID != nil {
		if *req.FixedVersionID <= 0 {
			return nil, bizerror.New(bizerror.InvalidArgument, "fixed version ID must be positive")
		}
		id = *req.FixedVersionID
		existing, err := a.getVersionResourceByID(req.RuleKind, req.ResourceKey, id)
		if err == nil {
			if validateErr := validateExistingVersionForRequest(existing, id, req); validateErr != nil {
				return nil, validateErr
			}
			rv = existing
			versionNo = existing.Spec.VersionNo
			versionExists = true
		} else if !errors.Is(err, ErrVersionNotFound) {
			return nil, err
		}
	}

	if !versionExists {
		attempts := maxIDGenerateAttempts
		if req.FixedVersionID != nil {
			attempts = 1
		}
		var addErr error
		for attempt := 0; attempt < attempts; attempt++ {
			if req.FixedVersionID == nil {
				generated, err := a.idGenerator.Next()
				if err != nil {
					return nil, err
				}
				id = generated
				if _, err := a.getVersionResourceByID(req.RuleKind, req.ResourceKey, id); err == nil {
					addErr = store.ErrorResourceAlreadyExists(meshresource.RuleVersionKind.ToString(), buildVersionName(req.RuleKind, req.ResourceKey, id), extractMesh(req.ResourceKey))
					continue
				} else if !errors.Is(err, ErrVersionNotFound) {
					return nil, err
				}
			}
			rv = newRuleVersionResource(req, id, versionNo, createdAt, committedAt)
			addErr = a.versionStore.Add(rv)
			if addErr == nil {
				break
			}
			if req.FixedVersionID != nil {
				existing, getErr := a.getVersionResourceByID(req.RuleKind, req.ResourceKey, id)
				if getErr != nil {
					return nil, fmt.Errorf("failed to add version resource with fixed id %d: %w", id, addErr)
				}
				if validateErr := validateExistingVersionForRequest(existing, id, req); validateErr != nil {
					return nil, validateErr
				}
				rv = existing
				addErr = nil
				break
			}
			if !isAddConflict(addErr) {
				return nil, fmt.Errorf("failed to add version resource id=%d: %w", id, addErr)
			}
		}
		if addErr != nil {
			return nil, fmt.Errorf("failed to allocate unique rule version id after %d attempts: %w", attempts, addErr)
		}
	}

	if _, err := a.reconcileMetaFromLedgerLocked(req.RuleKind, req.ResourceKey); err != nil {
		return nil, fmt.Errorf("failed to reconcile meta after version %d: %w", id, err)
	}

	if maxVersions > 0 {
		if err := a.TrimVersions(req.RuleKind, req.ResourceKey, maxVersions); err != nil {
			logger.Warnf("rule version retention cleanup failed for kind=%s resourceKey=%s committedVersion=%d: %v", req.RuleKind, req.ResourceKey, id, err)
		}
	}

	return protoToVersion(rv.Spec, id)
}

func (a *ResourceStoreAdapter) TrimVersions(kind coremodel.ResourceKind, resourceKey string, keep int64) error {
	versions, err := a.ListVersions(kind, resourceKey)
	if err != nil {
		return err
	}

	if int64(len(versions)) <= keep {
		return nil
	}

	// Delete oldest versions beyond keep limit
	toDelete := versions[int(keep):]
	for _, v := range toDelete {
		versionName := buildVersionName(kind, resourceKey, v.ID)
		rv := meshresource.NewRuleVersionResourceWithAttributes(versionName, extractMesh(resourceKey))
		if err := a.versionStore.Delete(rv); err != nil {
			return err
		}
	}

	return nil
}

func (a *ResourceStoreAdapter) getVersionResourceByID(kind coremodel.ResourceKind, resourceKey string, id int64) (*meshresource.RuleVersionResource, error) {
	versionName := buildVersionName(kind, resourceKey, id)
	fullKey := coremodel.BuildResourceKey(extractMesh(resourceKey), versionName)
	obj, exists, err := a.versionStore.GetByKey(fullKey)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrVersionNotFound
	}
	rv, ok := obj.(*meshresource.RuleVersionResource)
	if !ok {
		return nil, fmt.Errorf("%w: expected RuleVersionResource, got %T", ErrVersionLedgerCorrupt, obj)
	}
	if rv.Spec == nil {
		return nil, fmt.Errorf("%w: RuleVersion spec is nil for id %d", ErrVersionLedgerCorrupt, id)
	}
	return rv, nil
}

func (a *ResourceStoreAdapter) getVersionResourceByIntentID(intentID int64) (*meshresource.RuleVersionResource, int64, error) {
	objects, err := a.versionStore.ByIndex(index.ByRuleVersionIntentIDIndexName, strconv.FormatInt(intentID, 10))
	if err != nil {
		return nil, 0, err
	}
	switch len(objects) {
	case 0:
		return nil, 0, ErrVersionNotFound
	case 1:
		rv, ok := objects[0].(*meshresource.RuleVersionResource)
		if !ok {
			return nil, 0, fmt.Errorf("%w: expected RuleVersionResource, got %T", ErrVersionLedgerCorrupt, objects[0])
		}
		if rv.Spec == nil {
			return nil, 0, fmt.Errorf("%w: RuleVersion spec is nil for intent %d", ErrVersionLedgerCorrupt, intentID)
		}
		id, err := extractIDFromName(rv.Name)
		if err != nil {
			return nil, 0, fmt.Errorf("%w: %v", ErrVersionLedgerCorrupt, err)
		}
		return rv, id, nil
	default:
		return nil, 0, fmt.Errorf("%w: multiple RuleVersion resources indexed by intent id %d", ErrVersionLedgerCorrupt, intentID)
	}
}

func validateExistingVersionForRequest(existing *meshresource.RuleVersionResource, existingID int64, req InsertRequest) error {
	spec := existing.Spec
	if req.FixedVersionID != nil && existingID != *req.FixedVersionID {
		return fmt.Errorf("%w: RuleVersion intent id %d maps to version id %d, expected %d", ErrVersionLedgerCorrupt, req.IntentID, existingID, *req.FixedVersionID)
	}
	if spec.ParentRuleKind != string(req.RuleKind) ||
		spec.ParentRuleMesh != extractMesh(req.ResourceKey) ||
		spec.ParentRuleName != extractName(req.ResourceKey) ||
		spec.ContentHash != req.ContentHash ||
		spec.SpecJson != req.SpecJSON ||
		spec.Source != string(req.Source) ||
		spec.Operation != string(req.Operation) ||
		spec.IntentId != req.IntentID ||
		spec.RolledBackFromId != rolledBackFromIDValue(req.RolledBackFromID) {
		return fmt.Errorf("%w: RuleVersion id %d already exists with different content", ErrVersionLedgerCorrupt, existingID)
	}
	return nil
}

func newRuleVersionResource(req InsertRequest, id, versionNo int64, createdAt, committedAt time.Time) *meshresource.RuleVersionResource {
	rv := meshresource.NewRuleVersionResourceWithAttributes(
		buildVersionName(req.RuleKind, req.ResourceKey, id),
		extractMesh(req.ResourceKey),
	)
	rv.Spec = &meshproto.RuleVersion{
		ParentRuleKind: string(req.RuleKind),
		ParentRuleMesh: extractMesh(req.ResourceKey),
		ParentRuleName: extractName(req.ResourceKey),
		VersionNo:      versionNo,
		ContentHash:    req.ContentHash,
		SpecJson:       req.SpecJSON,
		Operation:      string(req.Operation),
		Source:         string(req.Source),
		Author:         req.Author,
		Reason:         req.Reason,
		IntentId:       req.IntentID,
		CreatedAt:      timestamppb.New(createdAt),
		CommittedAt:    timestamppb.New(committedAt),
	}
	if req.RolledBackFromID != nil {
		rv.Spec.RolledBackFromId = *req.RolledBackFromID
	}
	return rv
}

func isAddConflict(err error) bool {
	var conflict *store.ResourceConflictError
	if errors.As(err, &conflict) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "already exists")
}

func rolledBackFromIDValue(id *int64) int64 {
	if id == nil {
		return 0
	}
	return *id
}
