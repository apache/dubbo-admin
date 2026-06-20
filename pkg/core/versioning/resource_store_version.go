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
	"sort"
	"strconv"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

func (a *ResourceStoreAdapter) GetVersion(kind coremodel.ResourceKind, resourceKey string, id int64) (*Version, error) {
	rv, err := a.getVersionResourceForRule(kind, resourceKey, id)
	if err != nil {
		return nil, err
	}
	return protoToVersion(rv.Spec, id)
}

func (a *ResourceStoreAdapter) ListVersions(kind coremodel.ResourceKind, resourceKey string) ([]Version, error) {
	snapshot, err := a.LedgerSnapshot(kind, resourceKey)
	if err != nil {
		return nil, err
	}
	return snapshot.Versions, nil
}

func (a *ResourceStoreAdapter) LedgerSnapshot(kind coremodel.ResourceKind, resourceKey string) (*LedgerSnapshot, error) {
	var snapshot *LedgerSnapshot
	err := a.withParentLock(kind, resourceKey, func() error {
		state, err := a.ledgerState(kind, resourceKey)
		if err != nil {
			return err
		}
		snapshot = ledgerSnapshotFromState(state)
		return nil
	})
	return snapshot, err
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
		id, err := versionIDFromResource(rv)
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

func (a *ResourceStoreAdapter) InsertVersion(ctx context.Context, req InsertRequest, maxVersions int64) (*Version, error) {
	var version *Version
	err := a.withParentLock(req.RuleKind, req.ResourceKey, func() error {
		if err := lock.CheckLease(ctx); err != nil {
			return err
		}
		var inner error
		version, inner = a.insertVersionLocked(ctx, req, maxVersions)
		return inner
	})
	return version, err
}

func (a *ResourceStoreAdapter) insertVersionLocked(ctx context.Context, req InsertRequest, maxVersions int64) (*Version, error) {
	if err := lock.CheckLease(ctx); err != nil {
		return nil, err
	}
	state, err := a.ledgerState(req.RuleKind, req.ResourceKey)
	if err != nil {
		return nil, err
	}
	if err := lock.CheckLease(ctx); err != nil {
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
		existing, err := a.getVersionResourceByGlobalID(id)
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
		if err := lock.CheckLease(ctx); err != nil {
			return nil, err
		}
		if _, err := a.reconcileMetaFromLedgerLocked(ctx, req.RuleKind, req.ResourceKey); err != nil {
			return nil, err
		}
		if err := lock.CheckLease(ctx); err != nil {
			return nil, err
		}

		attempts := maxIDGenerateAttempts
		var addErr error
		for attempt := 0; attempt < attempts; attempt++ {
			if req.FixedVersionID == nil {
				generated, err := a.idGenerator.Next()
				if err != nil {
					return nil, err
				}
				id = generated
				if _, err := a.getVersionResourceByGlobalID(id); err == nil {
					addErr = store.ErrorResourceAlreadyExists(meshresource.RuleVersionKind.ToString(), buildVersionName(req.RuleKind, req.ResourceKey, id), extractMesh(req.ResourceKey))
					continue
				} else if !errors.Is(err, ErrVersionNotFound) {
					return nil, err
				}
			}
			rv = newRuleVersionResource(req, id, versionNo, createdAt, committedAt)
			if err := lock.CheckLease(ctx); err != nil {
				return nil, err
			}
			addErr = a.versionStore.Add(rv)
			if addErr == nil {
				break
			}
			if req.FixedVersionID != nil {
				existing, getErr := a.getVersionResourceByGlobalID(id)
				if getErr == nil {
					if validateErr := validateExistingVersionForRequest(existing, id, req); validateErr != nil {
						return nil, validateErr
					}
					rv = existing
					addErr = nil
					break
				}
				if !errors.Is(getErr, ErrVersionNotFound) {
					return nil, fmt.Errorf("failed to add version resource with fixed id %d: %w", id, addErr)
				}
			}
			if !isAddConflict(addErr) {
				return nil, fmt.Errorf("failed to add version resource id=%d: %w", id, addErr)
			}
			if err := lock.CheckLease(ctx); err != nil {
				return nil, err
			}
			state, err = a.ledgerState(req.RuleKind, req.ResourceKey)
			if err != nil {
				return nil, err
			}
			if state.MaxVersionNo+1 <= versionNo {
				return nil, fmt.Errorf("failed to allocate unique rule version number %d: %w", versionNo, addErr)
			}
			versionNo = state.MaxVersionNo + 1
		}
		if addErr != nil {
			return nil, fmt.Errorf("failed to allocate unique rule version id after %d attempts: %w", attempts, addErr)
		}
	}

	if err := lock.CheckLease(ctx); err != nil {
		return nil, err
	}
	if _, err := a.reconcileMetaFromLedgerLocked(ctx, req.RuleKind, req.ResourceKey); err != nil {
		return nil, fmt.Errorf("failed to reconcile meta after version %d: %w", id, err)
	}

	if err := lock.CheckLease(ctx); err != nil {
		return nil, err
	}
	if maxVersions > 0 {
		if err := a.trimVersionsLocked(ctx, req.RuleKind, req.ResourceKey, maxVersions); err != nil {
			logger.Warnf("rule version retention cleanup failed for kind=%s resourceKey=%s committedVersion=%d: %v", req.RuleKind, req.ResourceKey, id, err)
		}
	}

	return protoToVersion(rv.Spec, id)
}

func (a *ResourceStoreAdapter) trimVersionsLocked(ctx context.Context, kind coremodel.ResourceKind, resourceKey string, keep int64) error {
	state, err := a.ledgerState(kind, resourceKey)
	if err != nil {
		return err
	}
	versions := state.Versions
	if int64(len(versions)) <= keep {
		return nil
	}

	// Delete oldest versions beyond keep limit
	toDelete := versions[int(keep):]
	for _, v := range toDelete {
		if err := lock.CheckLease(ctx); err != nil {
			return err
		}
		rv, err := a.getVersionResourceForRule(kind, resourceKey, v.ID)
		if err != nil {
			return err
		}
		if err := a.versionStore.Delete(rv); err != nil {
			return err
		}
	}

	return nil
}

func (a *ResourceStoreAdapter) getVersionResourceByGlobalID(id int64) (*meshresource.RuleVersionResource, error) {
	objects, err := a.versionStore.ByIndex(index.ByRuleVersionIDIndexName, strconv.FormatInt(id, 10))
	if err != nil {
		return nil, err
	}
	if len(objects) == 0 {
		return nil, ErrVersionNotFound
	}
	if len(objects) > 1 {
		return nil, fmt.Errorf("%w: multiple RuleVersion resources indexed by id %d", ErrVersionLedgerCorrupt, id)
	}
	rv, ok := objects[0].(*meshresource.RuleVersionResource)
	if !ok {
		return nil, fmt.Errorf("%w: expected RuleVersionResource, got %T", ErrVersionLedgerCorrupt, objects[0])
	}
	if rv.Spec == nil {
		return nil, fmt.Errorf("%w: RuleVersion spec is nil for id %d", ErrVersionLedgerCorrupt, id)
	}
	indexedID, err := versionIDFromResource(rv)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVersionLedgerCorrupt, err)
	}
	if indexedID != id {
		return nil, fmt.Errorf("%w: RuleVersion id index mismatch: requested %d, got %d", ErrVersionLedgerCorrupt, id, indexedID)
	}
	return rv, nil
}

func (a *ResourceStoreAdapter) getVersionResourceForRule(kind coremodel.ResourceKind, resourceKey string, id int64) (*meshresource.RuleVersionResource, error) {
	rv, err := a.getVersionResourceByGlobalID(id)
	if err != nil {
		return nil, err
	}
	if !versionResourceMatchesParent(rv, kind, resourceKey) {
		return nil, ErrVersionNotFound
	}
	return rv, nil
}

func versionResourceMatchesParent(rv *meshresource.RuleVersionResource, kind coremodel.ResourceKind, resourceKey string) bool {
	return rv != nil &&
		rv.Spec != nil &&
		rv.Spec.ParentRuleKind == string(kind) &&
		rv.Spec.ParentRuleMesh == extractMesh(resourceKey) &&
		rv.Spec.ParentRuleName == extractName(resourceKey)
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
		id, err := versionIDFromResource(rv)
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
		spec.Author != req.Author ||
		spec.Reason != req.Reason ||
		spec.IntentId != req.IntentID ||
		spec.RolledBackFromId != rolledBackFromIDValue(req.RolledBackFromID) {
		return fmt.Errorf("%w: RuleVersion id %d already exists with different content", ErrVersionLedgerCorrupt, existingID)
	}
	if !req.CreatedAt.IsZero() && !timestampAsTime(spec.CreatedAt).Equal(req.CreatedAt) {
		return fmt.Errorf("%w: RuleVersion id %d already exists with different content", ErrVersionLedgerCorrupt, existingID)
	}
	return nil
}

func newRuleVersionResource(req InsertRequest, id, versionNo int64, createdAt, committedAt time.Time) *meshresource.RuleVersionResource {
	rv := meshresource.NewRuleVersionResourceWithAttributes(
		buildVersionNoName(req.RuleKind, req.ResourceKey, versionNo),
		extractMesh(req.ResourceKey),
	)
	rv.Annotations = map[string]string{
		ruleVersionIDAnnotation: strconv.FormatInt(id, 10),
	}
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
	return errors.As(err, &conflict)
}

func rolledBackFromIDValue(id *int64) int64 {
	if id == nil {
		return 0
	}
	return *id
}
