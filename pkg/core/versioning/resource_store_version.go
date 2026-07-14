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
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

func (a *ResourceStoreAdapter) GetVersion(kind coremodel.ResourceKind, resourceKey string, versionNo int64) (*Version, error) {
	if err := a.ensureStores(); err != nil {
		return nil, err
	}
	rv, err := a.getVersionResourceForRule(kind, resourceKey, versionNo)
	if err != nil {
		return nil, err
	}
	return protoToVersion(rv.Spec)
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
	// ListLatestVersions builds the global latest list, so it scans all stored
	// RuleVersion resources and groups by parent. ByParentRule is only suitable
	// for a single parent query; this pass intentionally avoids a new index.
	keys := a.versionStore.ListKeys()
	objs, err := a.versionStore.GetByKeys(keys)
	if err != nil {
		return nil, err
	}
	versions, err := versionsFromResources(objs, kind, "", false, "")
	if err != nil {
		return nil, err
	}
	byParent := make(map[string][]Version)
	for _, version := range versions {
		byParent[version.ResourceKey] = append(byParent[version.ResourceKey], version)
	}

	latest := make([]Version, 0, len(byParent))
	for resourceKey, versions := range byParent {
		if err := validateAndSortVersions(kind, resourceKey, versions); err != nil {
			return nil, err
		}
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

	versions, err := versionsFromIndexObjects(objs, kind, resourceKey, true, "parent "+parentKey)
	if err != nil {
		return nil, err
	}
	if err := validateAndSortVersions(kind, resourceKey, versions); err != nil {
		return nil, err
	}

	state := &historyState{Versions: versions}
	if len(versions) > 0 {
		state.Latest = &versions[0]
		state.MaxVersionNo = versions[0].VersionNo
	}
	return state, nil
}

func versionsFromResources(objs []coremodel.Resource, kind coremodel.ResourceKind, resourceKey string, filterParent bool, specContext string) ([]Version, error) {
	versions := make([]Version, 0, len(objs))
	for _, obj := range objs {
		version, include, err := versionFromObject(obj, kind, resourceKey, filterParent, specContext)
		if err != nil {
			return nil, err
		}
		if include {
			versions = append(versions, version)
		}
	}
	return versions, nil
}

func versionsFromIndexObjects(objs []interface{}, kind coremodel.ResourceKind, resourceKey string, filterParent bool, specContext string) ([]Version, error) {
	versions := make([]Version, 0, len(objs))
	for _, obj := range objs {
		version, include, err := versionFromObject(obj, kind, resourceKey, filterParent, specContext)
		if err != nil {
			return nil, err
		}
		if include {
			versions = append(versions, version)
		}
	}
	return versions, nil
}

func versionFromObject(obj interface{}, kind coremodel.ResourceKind, resourceKey string, filterParent bool, specContext string) (Version, bool, error) {
	rv, ok := obj.(*meshresource.RuleVersionResource)
	if !ok {
		return Version{}, false, fmt.Errorf("%w: expected RuleVersionResource, got %T", ErrVersionStoreError, obj)
	}
	if rv.Spec == nil {
		if specContext == "" {
			specContext = rv.ResourceKey()
		}
		return Version{}, false, fmt.Errorf("%w: RuleVersion spec is nil for %s", ErrVersionStoreError, specContext)
	}
	if rv.Spec.ParentRuleKind != string(kind) {
		return Version{}, false, nil
	}
	if filterParent && !versionResourceMatchesParent(rv, kind, resourceKey) {
		return Version{}, false, nil
	}
	v, err := protoToVersion(rv.Spec)
	if err != nil {
		return Version{}, false, err
	}
	return *v, true, nil
}

func validateAndSortVersions(kind coremodel.ResourceKind, resourceKey string, versions []Version) error {
	seenVersionNo := make(map[int64]struct{}, len(versions))
	for _, version := range versions {
		if _, ok := seenVersionNo[version.VersionNo]; ok {
			return duplicateVersionNoError(kind, resourceKey, version.VersionNo)
		}
		seenVersionNo[version.VersionNo] = struct{}{}
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].VersionNo > versions[j].VersionNo
	})
	return nil
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

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var rv *meshresource.RuleVersionResource
	var addErr error
	for attempt := 0; attempt < maxVersionAllocateAttempts; attempt++ {
		rv = newRuleVersionResource(req, versionNo, createdAt, recordedAt)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		addErr = a.versionStore.Add(rv)
		if addErr == nil {
			break
		}

		// A concurrent database writer can win the deterministic ResourceKey.
		// Confirm that key now exists, then re-read the parent history before
		// allocating the next monotonically increasing VersionNo.
		if _, err := a.getVersionResourceForRule(req.RuleKind, req.ResourceKey, versionNo); err != nil {
			if errors.Is(err, ErrVersionNotFound) {
				return nil, fmt.Errorf("failed to add rule version %d: %w", versionNo, addErr)
			}
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		state, err = a.historyState(req.RuleKind, req.ResourceKey)
		if err != nil {
			return nil, err
		}
		nextVersionNo := state.MaxVersionNo + 1
		if nextVersionNo <= versionNo {
			return nil, fmt.Errorf("failed to allocate unique rule version number %d: %w", versionNo, addErr)
		}
		versionNo = nextVersionNo
	}
	if addErr != nil {
		return nil, fmt.Errorf("failed to allocate unique rule version number after %d attempts: %w", maxVersionAllocateAttempts, addErr)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if maxVersions > 0 {
		if err := a.trimVersionsLocked(ctx, req.RuleKind, req.ResourceKey, maxVersions); err != nil {
			logger.Warnf("rule version retention cleanup failed for kind=%s resourceKey=%s recordedVersionNo=%d: %v", req.RuleKind, req.ResourceKey, versionNo, err)
		}
	}

	return protoToVersion(rv.Spec)
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
		rv, err := a.getVersionResourceForRule(kind, resourceKey, v.VersionNo)
		if err != nil {
			return err
		}
		if err := a.versionStore.Delete(rv); err != nil {
			return err
		}
	}

	return nil
}

func (a *ResourceStoreAdapter) getVersionResourceForRule(kind coremodel.ResourceKind, resourceKey string, versionNo int64) (*meshresource.RuleVersionResource, error) {
	versionResourceKey := buildVersionResourceKey(kind, resourceKey, versionNo)
	obj, exists, err := a.versionStore.GetByKey(versionResourceKey)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrVersionNotFound
	}
	rv, ok := obj.(*meshresource.RuleVersionResource)
	if !ok {
		return nil, fmt.Errorf("%w: expected RuleVersionResource, got %T", ErrVersionStoreError, obj)
	}
	if rv.Spec == nil {
		return nil, fmt.Errorf("%w: RuleVersion spec is nil for key %s", ErrVersionStoreError, versionResourceKey)
	}
	if !versionResourceMatchesParent(rv, kind, resourceKey) || rv.Spec.VersionNo != versionNo {
		return nil, fmt.Errorf("%w: RuleVersion key %s does not match parent or version number", ErrVersionStoreError, versionResourceKey)
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

func newRuleVersionResource(req InsertRequest, versionNo int64, createdAt, recordedAt time.Time) *meshresource.RuleVersionResource {
	_, name := coremodel.ParseResourceKey(buildVersionResourceKey(req.RuleKind, req.ResourceKey, versionNo))
	rv := meshresource.NewRuleVersionResourceWithAttributes(
		name,
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
		CreatedAt:      timestamppb.New(createdAt),
		RecordedAt:     timestamppb.New(recordedAt),
	}
	if req.RolledBackFromVersionNo != nil {
		rv.Spec.RolledBackFromVersionNo = *req.RolledBackFromVersionNo
	}
	return rv
}
