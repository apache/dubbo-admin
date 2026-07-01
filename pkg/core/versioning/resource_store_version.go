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
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

func (a *ResourceStoreAdapter) GetVersion(kind coremodel.ResourceKind, resourceKey string, id int64) (*Version, error) {
	if err := a.ensureStores(); err != nil {
		return nil, err
	}
	rv, err := a.getVersionResourceForRule(kind, resourceKey, id)
	if err != nil {
		return nil, err
	}
	return protoToVersion(rv.Spec, id)
}

func (a *ResourceStoreAdapter) ListVersions(kind coremodel.ResourceKind, resourceKey string) ([]Version, error) {
	if err := a.ensureStores(); err != nil {
		return nil, err
	}
	snapshot, err := a.HistorySnapshot(kind, resourceKey)
	if err != nil {
		return nil, err
	}
	return snapshot.Versions, nil
}

func (a *ResourceStoreAdapter) ListLatestVersions(kind coremodel.ResourceKind) ([]Version, error) {
	if err := a.ensureStores(); err != nil {
		return nil, err
	}
	keys := a.versionStore.ListKeys()
	objs, err := a.versionStore.GetByKeys(keys)
	if err != nil {
		return nil, err
	}
	byParent := make(map[string][]Version)
	for _, obj := range objs {
		rv, ok := obj.(*meshresource.RuleVersionResource)
		if !ok {
			return nil, fmt.Errorf("%w: expected RuleVersionResource, got %T", ErrVersionStoreError, obj)
		}
		if rv.Spec == nil {
			return nil, fmt.Errorf("%w: RuleVersion spec is nil for %s", ErrVersionStoreError, rv.ResourceKey())
		}
		if rv.Spec.ParentRuleKind != string(kind) {
			continue
		}
		id, err := versionIDFromResource(rv)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrVersionStoreError, err)
		}
		v, err := protoToVersion(rv.Spec, id)
		if err != nil {
			return nil, err
		}
		byParent[v.ResourceKey] = append(byParent[v.ResourceKey], *v)
	}

	latest := make([]Version, 0, len(byParent))
	for resourceKey, versions := range byParent {
		seenVersionNo := make(map[int64]int64, len(versions))
		for _, version := range versions {
			if previousID, ok := seenVersionNo[version.VersionNo]; ok && previousID != version.ID {
				return nil, duplicateVersionNoError(kind, resourceKey, version.VersionNo, previousID, version.ID)
			}
			seenVersionNo[version.VersionNo] = version.ID
		}
		sort.Slice(versions, func(i, j int) bool {
			return versions[i].VersionNo > versions[j].VersionNo
		})
		if len(versions) > 0 {
			latest = append(latest, versions[0])
		}
	}
	sort.Slice(latest, func(i, j int) bool {
		if latest[i].ResourceKey == latest[j].ResourceKey {
			return latest[i].VersionNo > latest[j].VersionNo
		}
		return latest[i].ResourceKey < latest[j].ResourceKey
	})
	return latest, nil
}

func (a *ResourceStoreAdapter) HistorySnapshot(kind coremodel.ResourceKind, resourceKey string) (*HistorySnapshot, error) {
	if err := a.ensureStores(); err != nil {
		return nil, err
	}
	var snapshot *HistorySnapshot
	err := a.withParentLock(kind, resourceKey, func() error {
		state, err := a.historyState(kind, resourceKey)
		if err != nil {
			return err
		}
		snapshot = historySnapshotFromState(state)
		return nil
	})
	return snapshot, err
}

func (a *ResourceStoreAdapter) latestVersionLocked(kind coremodel.ResourceKind, resourceKey string) (*Version, error) {
	state, err := a.historyState(kind, resourceKey)
	if err != nil {
		return nil, err
	}
	if state.Latest == nil {
		return nil, ErrVersionNotFound
	}
	return state.Latest, nil
}

func (a *ResourceStoreAdapter) historyState(kind coremodel.ResourceKind, resourceKey string) (*historyState, error) {
	parentKey := buildParentIndexKey(kind, resourceKey)
	objs, err := a.versionStore.ByIndex(index.ByParentRuleIndexName, parentKey)
	if err != nil {
		return nil, err
	}

	versions := make([]Version, 0, len(objs))
	for _, obj := range objs {
		rv, ok := obj.(*meshresource.RuleVersionResource)
		if !ok {
			return nil, fmt.Errorf("%w: expected RuleVersionResource, got %T", ErrVersionStoreError, obj)
		}
		if rv.Spec == nil {
			return nil, fmt.Errorf("%w: RuleVersion spec is nil for parent %s", ErrVersionStoreError, parentKey)
		}
		id, err := versionIDFromResource(rv)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrVersionStoreError, err)
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

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].VersionNo > versions[j].VersionNo
	})

	state := &historyState{Versions: versions}
	if len(versions) > 0 {
		state.Latest = &versions[0]
		state.MaxVersionNo = versions[0].VersionNo
	}
	return state, nil
}

func (a *ResourceStoreAdapter) InsertVersion(ctx context.Context, req InsertRequest, maxVersions int64) (*Version, error) {
	if err := a.ensureStores(); err != nil {
		return nil, err
	}
	var version *Version
	err := a.withParentLock(req.RuleKind, req.ResourceKey, func() error {
		var inner error
		version, inner = a.insertVersionLocked(ctx, req, maxVersions)
		return inner
	})
	return version, err
}

func (a *ResourceStoreAdapter) insertVersionLocked(ctx context.Context, req InsertRequest, maxVersions int64) (*Version, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	state, err := a.historyState(req.RuleKind, req.ResourceKey)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	versionNo := state.MaxVersionNo + 1

	createdAt := req.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	recordedAt := time.Now()

	var rv *meshresource.RuleVersionResource
	var id int64

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	attempts := maxIDGenerateAttempts
	var addErr error
	for attempt := 0; attempt < attempts; attempt++ {
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
		rv = newRuleVersionResource(req, id, versionNo, createdAt, recordedAt)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		addErr = a.versionStore.Add(rv)
		if addErr == nil {
			break
		}
		if !isAddConflict(addErr) {
			return nil, fmt.Errorf("failed to add version resource id=%d: %w", id, addErr)
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		state, err = a.historyState(req.RuleKind, req.ResourceKey)
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

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if maxVersions > 0 {
		if err := a.trimVersionsLocked(ctx, req.RuleKind, req.ResourceKey, maxVersions); err != nil {
			logger.Warnf("rule version retention cleanup failed for kind=%s resourceKey=%s recordedVersion=%d: %v", req.RuleKind, req.ResourceKey, id, err)
		}
	}

	return protoToVersion(rv.Spec, id)
}

func (a *ResourceStoreAdapter) trimVersionsLocked(ctx context.Context, kind coremodel.ResourceKind, resourceKey string, keep int64) error {
	state, err := a.historyState(kind, resourceKey)
	if err != nil {
		return err
	}
	versions := state.Versions
	if int64(len(versions)) <= keep {
		return nil
	}

	// Retention runs after the new version is durable and only removes entries
	// beyond the configured window. Cleanup failure is reported to logs by the
	// caller and does not roll back the already-written rule mutation.
	toDelete := versions[int(keep):]
	for _, v := range toDelete {
		if err := ctx.Err(); err != nil {
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
		return nil, fmt.Errorf("%w: multiple RuleVersion resources indexed by id %d", ErrVersionStoreError, id)
	}
	rv, ok := objects[0].(*meshresource.RuleVersionResource)
	if !ok {
		return nil, fmt.Errorf("%w: expected RuleVersionResource, got %T", ErrVersionStoreError, objects[0])
	}
	if rv.Spec == nil {
		return nil, fmt.Errorf("%w: RuleVersion spec is nil for id %d", ErrVersionStoreError, id)
	}
	indexedID, err := versionIDFromResource(rv)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVersionStoreError, err)
	}
	if indexedID != id {
		return nil, fmt.Errorf("%w: RuleVersion id index mismatch: requested %d, got %d", ErrVersionStoreError, id, indexedID)
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

func validateExistingVersionForRequest(existing *meshresource.RuleVersionResource, existingID int64, req InsertRequest) error {
	spec := existing.Spec
	if spec.ParentRuleKind != string(req.RuleKind) ||
		spec.ParentRuleMesh != extractMesh(req.ResourceKey) ||
		spec.ParentRuleName != extractName(req.ResourceKey) ||
		spec.ContentHash != req.ContentHash ||
		spec.SpecJson != req.SpecJSON ||
		spec.Source != string(req.Source) ||
		spec.Operation != string(req.Operation) ||
		spec.Author != req.Author ||
		spec.Reason != req.Reason ||
		spec.RolledBackFromId != rolledBackFromIDValue(req.RolledBackFromID) {
		return fmt.Errorf("%w: RuleVersion id %d already exists with different content", ErrVersionStoreError, existingID)
	}
	if !req.CreatedAt.IsZero() && !timestampAsTime(spec.CreatedAt).Equal(req.CreatedAt) {
		return fmt.Errorf("%w: RuleVersion id %d already exists with different content", ErrVersionStoreError, existingID)
	}
	return nil
}

func newRuleVersionResource(req InsertRequest, id, versionNo int64, createdAt, recordedAt time.Time) *meshresource.RuleVersionResource {
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
		CreatedAt:      timestamppb.New(createdAt),
		RecordedAt:     timestamppb.New(recordedAt),
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
