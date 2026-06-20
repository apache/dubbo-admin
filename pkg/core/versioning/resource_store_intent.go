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

	"google.golang.org/protobuf/types/known/timestamppb"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

func (a *ResourceStoreAdapter) CreateIntent(ctx context.Context, req InsertRequest) (*Intent, error) {
	if err := a.ensureStores(); err != nil {
		return nil, err
	}
	var intent *Intent
	err := a.withParentLock(req.RuleKind, req.ResourceKey, func() error {
		if err := lock.CheckLease(ctx); err != nil {
			return err
		}
		var inner error
		intent, inner = a.createIntentLocked(ctx, req)
		return inner
	})
	return intent, err
}

func (a *ResourceStoreAdapter) createIntentLocked(ctx context.Context, req InsertRequest) (*Intent, error) {
	open, err := a.OpenIntent(req.RuleKind, req.ResourceKey)
	if err != nil {
		return nil, err
	}
	if open != nil {
		return nil, &IntentPendingError{IntentID: open.ID}
	}

	var intentRes *meshresource.RuleIntentResource
	var id int64
	var addErr error
	for attempt := 0; attempt < maxIDGenerateAttempts; attempt++ {
		generated, err := a.idGenerator.Next()
		if err != nil {
			return nil, err
		}
		id = generated
		if _, _, err := a.getIntentResourceByID(id); err == nil {
			addErr = store.ErrorResourceAlreadyExists(meshresource.RuleIntentKind.ToString(), buildIntentName(req.RuleKind, req.ResourceKey, id), req.Mesh)
			continue
		} else if !errors.Is(err, ErrVersionIntentNotFound) {
			return nil, err
		}
		if _, err := a.getVersionResourceByGlobalID(id); err == nil {
			addErr = store.ErrorResourceAlreadyExists(meshresource.RuleIntentKind.ToString(), buildIntentName(req.RuleKind, req.ResourceKey, id), req.Mesh)
			continue
		} else if !errors.Is(err, ErrVersionNotFound) {
			return nil, err
		}
		intentRes = newRuleIntentResource(req, id)
		if err := lock.CheckLease(ctx); err != nil {
			return nil, err
		}
		addErr = a.intentStore.Add(intentRes)
		if addErr == nil {
			break
		}
		if !isAddConflict(addErr) {
			return nil, addErr
		}
	}
	if addErr != nil {
		return nil, fmt.Errorf("failed to allocate unique rule intent id after %d attempts: %w", maxIDGenerateAttempts, addErr)
	}

	return intentFromResource(intentRes, id), nil
}

func (a *ResourceStoreAdapter) GetIntent(id int64) (*Intent, error) {
	if err := a.ensureStores(); err != nil {
		return nil, err
	}
	intentRes, parsedID, err := a.getIntentResourceByID(id)
	if err != nil {
		return nil, err
	}
	return intentFromResource(intentRes, parsedID), nil
}

func (a *ResourceStoreAdapter) OpenIntent(kind coremodel.ResourceKind, resourceKey string) (*Intent, error) {
	if err := a.ensureStores(); err != nil {
		return nil, err
	}
	intents, err := a.openIntentResources(kind, resourceKey)
	if err != nil {
		return nil, err
	}
	switch len(intents) {
	case 0:
		return nil, nil
	case 1:
		id, err := extractIDFromIntentName(intents[0].Name)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrVersionLedgerCorrupt, err)
		}
		return intentFromResource(intents[0], id), nil
	default:
		return nil, a.multipleOpenIntentsError(kind, resourceKey, intents)
	}
}

func (a *ResourceStoreAdapter) MarkIntentApplied(ctx context.Context, id int64) error {
	if err := a.ensureStores(); err != nil {
		return err
	}
	return a.updateIntentStatus(ctx, id, IntentStatusApplied, "")
}

func (a *ResourceStoreAdapter) MarkIntentOutcomeUnknown(ctx context.Context, id int64, message string) error {
	if err := a.ensureStores(); err != nil {
		return err
	}
	intentRes, _, err := a.getIntentResourceByID(id)
	if err != nil {
		return err
	}
	return a.withParentLock(coremodel.ResourceKind(intentRes.Spec.ParentRuleKind), intentResourceKey(intentRes), func() error {
		if err := lock.CheckLease(ctx); err != nil {
			return err
		}
		fresh, _, err := a.getIntentResourceByID(id)
		if err != nil {
			return err
		}
		return updateIntentResourceStatus(a.intentStore, fresh, IntentStatusOutcomeUnknown, message)
	})
}

func (a *ResourceStoreAdapter) MarkIntentObserved(ctx context.Context, id int64, op Operation, contentHash, specJSON string) error {
	if err := a.ensureStores(); err != nil {
		return err
	}
	intentRes, _, err := a.getIntentResourceByID(id)
	if err != nil {
		return err
	}
	return a.withParentLock(coremodel.ResourceKind(intentRes.Spec.ParentRuleKind), intentResourceKey(intentRes), func() error {
		for {
			if err := lock.CheckLease(ctx); err != nil {
				return err
			}
			fresh, _, err := a.getIntentResourceByID(id)
			if err != nil {
				return err
			}
			err = updateIntentResourceObserved(a.intentStore, fresh, op, contentHash, specJSON)
			if errors.Is(err, ErrVersionIntentConflict) {
				continue
			}
			return err
		}
	})
}

func (a *ResourceStoreAdapter) MarkIntentFailed(ctx context.Context, id int64, message string) error {
	if err := a.ensureStores(); err != nil {
		return err
	}
	if err := a.updateIntentStatus(ctx, id, IntentStatusFailed, message); err != nil {
		return err
	}
	return a.cleanupIntent(id, IntentStatusFailed)
}

func (a *ResourceStoreAdapter) CommitIntent(ctx context.Context, id int64, maxVersions int64) (*Version, error) {
	if err := a.ensureStores(); err != nil {
		return nil, err
	}
	if err := lock.CheckLease(ctx); err != nil {
		return nil, err
	}
	intentRes, _, err := a.getIntentResourceByID(id)
	if err != nil {
		return nil, err
	}
	var version *Version
	err = a.withParentLock(coremodel.ResourceKind(intentRes.Spec.ParentRuleKind), intentResourceKey(intentRes), func() error {
		if err := lock.CheckLease(ctx); err != nil {
			return err
		}
		freshRes, parsedID, err := a.getIntentResourceByID(id)
		if err != nil {
			return err
		}
		intent := intentFromResource(freshRes, parsedID)
		switch intent.Status {
		case IntentStatusCommitted:
			committed, err := a.committedVersionForIntentLocked(intent)
			if err != nil {
				return err
			}
			version = committed
			return a.cleanupIntentLocked(id, IntentStatusCommitted)
		case IntentStatusApplied:
			if intent.ReconcileRequired {
				return &IntentPendingError{IntentID: intent.ID}
			}
		default:
			return ErrVersionIntentNotOpen
		}

		// CommitIntent appends the intended state only when no durable
		// subscriber marker has modified the intent since APPLIED. The fixed
		// version ID makes retries idempotent if the process crashes after Add.
		committed, err := a.insertVersionLocked(ctx, InsertRequest{
			RuleKind:         intent.RuleKind,
			Mesh:             intent.Mesh,
			ResourceKey:      intent.ResourceKey,
			RuleName:         intent.RuleName,
			SpecJSON:         intent.SpecJSON,
			ContentHash:      intent.ContentHash,
			Source:           intent.Source,
			Operation:        intent.Operation,
			Author:           intent.Author,
			Reason:           intent.Reason,
			IntentID:         intent.ID,
			RolledBackFromID: intent.RolledBackFromID,
			CreatedAt:        intent.CreatedAt,
			FixedVersionID:   &intent.ID,
		}, maxVersions)
		if err != nil {
			return err
		}

		if err := lock.CheckLease(ctx); err != nil {
			return err
		}
		refreshed, _, err := a.getIntentResourceByID(id)
		if err != nil {
			return err
		}
		if refreshed.Spec.Revision != intent.Revision || refreshed.Spec.ReconcileRequired {
			return &IntentPendingError{IntentID: intent.ID}
		}
		if err := updateIntentResourceStatus(a.intentStore, refreshed, IntentStatusCommitted, ""); err != nil {
			return err
		}
		if err := a.cleanupIntentLocked(id, IntentStatusCommitted); err != nil {
			return err
		}
		version = committed
		return nil
	})
	return version, err
}

func (a *ResourceStoreAdapter) CleanupIntent(id int64, terminalStatus IntentStatus) error {
	if err := a.ensureStores(); err != nil {
		return err
	}
	return a.cleanupIntent(id, terminalStatus)
}

func (a *ResourceStoreAdapter) ListOpenIntents() ([]Intent, error) {
	if err := a.ensureStores(); err != nil {
		return nil, err
	}
	return a.listIntentsByStatuses(openIntentStatuses())
}

func (a *ResourceStoreAdapter) ListTerminalIntents() ([]Intent, error) {
	if err := a.ensureStores(); err != nil {
		return nil, err
	}
	return a.listIntentsByStatuses(terminalIntentStatuses())
}

func (a *ResourceStoreAdapter) listIntentsByStatuses(statuses []IntentStatus) ([]Intent, error) {
	var result []Intent
	for _, status := range statuses {
		objects, err := a.intentStore.ByIndex(index.ByRuleIntentStatusIndexName, string(status))
		if err != nil {
			return nil, err
		}
		for _, obj := range objects {
			intentRes, ok := obj.(*meshresource.RuleIntentResource)
			if !ok {
				return nil, fmt.Errorf("%w: expected RuleIntentResource, got %T", ErrVersionLedgerCorrupt, obj)
			}
			if intentRes.Spec == nil {
				return nil, fmt.Errorf("%w: RuleIntent spec is nil for %s", ErrVersionLedgerCorrupt, intentRes.Name)
			}
			id, err := extractIDFromIntentName(intentRes.Name)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrVersionLedgerCorrupt, err)
			}
			result = append(result, *intentFromResource(intentRes, id))
		}
	}

	return result, nil
}

func isOpenIntentStatus(status IntentStatus) bool {
	switch status {
	case IntentStatusPending, IntentStatusApplied, IntentStatusOutcomeUnknown:
		return true
	default:
		return false
	}
}

func openIntentStatuses() []IntentStatus {
	return []IntentStatus{IntentStatusPending, IntentStatusApplied, IntentStatusOutcomeUnknown}
}

func terminalIntentStatuses() []IntentStatus {
	return []IntentStatus{IntentStatusCommitted, IntentStatusFailed}
}

func (a *ResourceStoreAdapter) getIntentResourceByID(id int64) (*meshresource.RuleIntentResource, int64, error) {
	objects, err := a.intentStore.ByIndex(index.ByRuleIntentIDIndexName, strconv.FormatInt(id, 10))
	if err != nil {
		return nil, 0, err
	}
	switch len(objects) {
	case 0:
		return nil, 0, ErrVersionIntentNotFound
	case 1:
		intentRes, ok := objects[0].(*meshresource.RuleIntentResource)
		if !ok {
			return nil, 0, fmt.Errorf("%w: expected RuleIntentResource, got %T", ErrVersionLedgerCorrupt, objects[0])
		}
		if intentRes.Spec == nil {
			return nil, 0, fmt.Errorf("%w: RuleIntent spec is nil for id %d", ErrVersionLedgerCorrupt, id)
		}
		parsedID, err := extractIDFromIntentName(intentRes.Name)
		if err != nil {
			return nil, 0, fmt.Errorf("%w: %v", ErrVersionLedgerCorrupt, err)
		}
		if parsedID != id {
			return nil, 0, fmt.Errorf("%w: RuleIntent id index mismatch: requested %d, got %d", ErrVersionLedgerCorrupt, id, parsedID)
		}
		return intentRes, parsedID, nil
	default:
		return nil, 0, fmt.Errorf("%w: multiple RuleIntent resources indexed by id %d", ErrVersionLedgerCorrupt, id)
	}
}

func (a *ResourceStoreAdapter) openIntentResources(kind coremodel.ResourceKind, resourceKey string) ([]*meshresource.RuleIntentResource, error) {
	mesh := extractMesh(resourceKey)
	name := extractName(resourceKey)
	var intents []*meshresource.RuleIntentResource
	for _, status := range openIntentStatuses() {
		indexKey := fmt.Sprintf("%s/%s/%s/%s", kind, mesh, name, status)
		objects, err := a.intentStore.ByIndex(index.ByRuleIntentParentAndStatus, indexKey)
		if err != nil {
			return nil, err
		}
		for _, obj := range objects {
			intentRes, ok := obj.(*meshresource.RuleIntentResource)
			if !ok {
				return nil, fmt.Errorf("%w: expected RuleIntentResource, got %T", ErrVersionLedgerCorrupt, obj)
			}
			if intentRes.Spec == nil {
				return nil, fmt.Errorf("%w: RuleIntent spec is nil for %s", ErrVersionLedgerCorrupt, intentRes.Name)
			}
			if isOpenIntentStatus(IntentStatus(intentRes.Spec.Status)) {
				intents = append(intents, intentRes)
			}
		}
	}
	return intents, nil
}

func (a *ResourceStoreAdapter) updateIntentStatus(ctx context.Context, id int64, status IntentStatus, failureReason string) error {
	if err := lock.CheckLease(ctx); err != nil {
		return err
	}
	intentRes, _, err := a.getIntentResourceByID(id)
	if err != nil {
		return err
	}
	return a.withParentLock(coremodel.ResourceKind(intentRes.Spec.ParentRuleKind), intentResourceKey(intentRes), func() error {
		if err := lock.CheckLease(ctx); err != nil {
			return err
		}
		fresh, _, err := a.getIntentResourceByID(id)
		if err != nil {
			return err
		}
		if err := lock.CheckLease(ctx); err != nil {
			return err
		}
		return updateIntentResourceStatus(a.intentStore, fresh, status, failureReason)
	})
}

func updateIntentResourceStatus(intentStore store.ResourceStore, intentRes *meshresource.RuleIntentResource, status IntentStatus, failureReason string) error {
	currentStatus := IntentStatus(intentRes.Spec.Status)
	switch status {
	case IntentStatusApplied:
		if currentStatus == IntentStatusCommitted {
			return nil
		}
		if currentStatus != IntentStatusPending &&
			currentStatus != IntentStatusApplied &&
			currentStatus != IntentStatusOutcomeUnknown {
			return ErrVersionIntentNotOpen
		}
	case IntentStatusOutcomeUnknown:
		if currentStatus == IntentStatusCommitted {
			return ErrVersionIntentNotOpen
		}
		if !isOpenIntentStatus(currentStatus) {
			return ErrVersionIntentNotOpen
		}
	case IntentStatusCommitted:
		if currentStatus == IntentStatusCommitted {
			return nil
		}
		if currentStatus != IntentStatusApplied {
			return ErrVersionIntentNotOpen
		}
	case IntentStatusFailed:
		if currentStatus == IntentStatusCommitted {
			return ErrVersionIntentNotOpen
		}
	}

	updated, err := prepareIntentUpdate(intentStore, intentRes)
	if err != nil {
		return err
	}
	updated.Spec.Status = string(status)
	if failureReason != "" {
		updated.Spec.FailureReason = failureReason
	}

	now := timestamppb.New(time.Now())
	switch status {
	case IntentStatusApplied:
		updated.Spec.AppliedAt = now
	case IntentStatusCommitted:
		updated.Spec.CommittedAt = now
	}

	if err := intentStore.Update(updated); err != nil {
		return err
	}

	return nil
}

func updateIntentResourceObserved(intentStore store.ResourceStore, intentRes *meshresource.RuleIntentResource, op Operation, contentHash, specJSON string) error {
	currentStatus := IntentStatus(intentRes.Spec.Status)
	if !isOpenIntentStatus(currentStatus) {
		return ErrVersionIntentNotOpen
	}

	updated, err := prepareIntentUpdate(intentStore, intentRes)
	if err != nil {
		return err
	}
	updated.Spec.ReconcileRequired = true
	updated.Spec.ObservedOperation = string(op)
	updated.Spec.ObservedContentHash = contentHash
	updated.Spec.ObservedSpecJson = specJSON
	updated.Spec.ObservedAt = timestamppb.New(time.Now())
	if currentStatus == IntentStatusPending {
		updated.Spec.Status = string(IntentStatusOutcomeUnknown)
	}
	if err := intentStore.Update(updated); err != nil {
		return err
	}
	return nil
}

func prepareIntentUpdate(intentStore store.ResourceStore, intentRes *meshresource.RuleIntentResource) (*meshresource.RuleIntentResource, error) {
	if intentRes == nil || intentRes.Spec == nil {
		return nil, ErrVersionLedgerCorrupt
	}
	currentObj, exists, err := intentStore.GetByKey(intentRes.ResourceKey())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrVersionIntentNotFound
	}
	current, ok := currentObj.(*meshresource.RuleIntentResource)
	if !ok {
		return nil, fmt.Errorf("%w: expected RuleIntentResource, got %T", ErrVersionLedgerCorrupt, currentObj)
	}
	if current.Spec == nil {
		return nil, fmt.Errorf("%w: RuleIntent spec is nil for %s", ErrVersionLedgerCorrupt, current.Name)
	}
	if current.Spec.Revision != intentRes.Spec.Revision {
		return nil, ErrVersionIntentConflict
	}
	updated := current.DeepCopyObject().(*meshresource.RuleIntentResource)
	updated.Spec.Revision++
	return updated, nil
}

func intentResourceKey(intentRes *meshresource.RuleIntentResource) string {
	if intentRes == nil || intentRes.Spec == nil {
		return ""
	}
	return coremodel.BuildResourceKey(intentRes.Spec.ParentRuleMesh, intentRes.Spec.ParentRuleName)
}

func (a *ResourceStoreAdapter) cleanupIntent(id int64, terminalStatus IntentStatus) error {
	intentRes, _, err := a.getIntentResourceByID(id)
	if errors.Is(err, ErrVersionIntentNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read terminal rule version intent %d for cleanup: %w", id, err)
	}
	return a.withParentLock(coremodel.ResourceKind(intentRes.Spec.ParentRuleKind), intentResourceKey(intentRes), func() error {
		return a.cleanupIntentLocked(id, terminalStatus)
	})
}

func (a *ResourceStoreAdapter) cleanupIntentLocked(id int64, terminalStatus IntentStatus) error {
	intentRes, _, err := a.getIntentResourceByID(id)
	if errors.Is(err, ErrVersionIntentNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read terminal rule version intent %d for cleanup: %w", id, err)
	}
	if intentRes.Spec == nil || IntentStatus(intentRes.Spec.Status) != terminalStatus {
		return nil
	}
	if terminalStatus == IntentStatusCommitted {
		intent := intentFromResource(intentRes, id)
		if _, err := a.committedVersionForIntentLocked(intent); err != nil {
			return err
		}
	}
	if err := a.intentStore.Delete(intentRes); err != nil {
		return fmt.Errorf("failed to cleanup terminal rule version intent %d: %w", id, err)
	}
	return nil
}

func (a *ResourceStoreAdapter) committedVersionForIntentLocked(intent *Intent) (*Version, error) {
	if intent == nil {
		return nil, ErrVersionIntentNotFound
	}
	versions, err := a.ledgerState(intent.RuleKind, intent.ResourceKey)
	if err != nil {
		return nil, err
	}
	var found *Version
	for i := range versions.Versions {
		if versions.Versions[i].IntentID != intent.ID {
			continue
		}
		if found != nil && found.ID != versions.Versions[i].ID {
			return nil, fmt.Errorf("%w: multiple RuleVersion resources committed for intent %d", ErrVersionLedgerCorrupt, intent.ID)
		}
		v := versions.Versions[i]
		found = &v
	}
	if found == nil {
		return nil, ErrVersionNotFound
	}
	return validateCommittedIntentVersion(found, intent)
}

func (a *ResourceStoreAdapter) multipleOpenIntentsError(kind coremodel.ResourceKind, resourceKey string, intents []*meshresource.RuleIntentResource) error {
	ids := make([]string, 0, len(intents))
	for _, intentRes := range intents {
		id, err := extractIDFromIntentName(intentRes.Name)
		if err != nil {
			ids = append(ids, fmt.Sprintf("%s(parse-error:%v)", intentRes.Name, err))
			continue
		}
		ids = append(ids, strconv.FormatInt(id, 10))
	}
	msg := fmt.Sprintf("multiple open intents for parentKind=%s resourceKey=%s intentIDs=%s", kind, resourceKey, strings.Join(ids, ","))
	logger.Warnf("%s", msg)
	return fmt.Errorf("%w: %s", ErrVersionLedgerCorrupt, msg)
}

func newRuleIntentResource(req InsertRequest, id int64) *meshresource.RuleIntentResource {
	intentRes := meshresource.NewRuleIntentResourceWithAttributes(buildIntentName(req.RuleKind, req.ResourceKey, id), req.Mesh)
	intentRes.Spec = &meshproto.RuleIntent{
		ParentRuleKind: string(req.RuleKind),
		ParentRuleMesh: req.Mesh,
		ParentRuleName: req.RuleName,
		ContentHash:    req.ContentHash,
		SpecJson:       req.SpecJSON,
		Operation:      string(req.Operation),
		Source:         string(req.Source),
		Author:         req.Author,
		Reason:         req.Reason,
		Status:         string(IntentStatusPending),
		CreatedAt:      timestamppb.New(req.CreatedAt),
		Revision:       1,
	}
	if req.RolledBackFromID != nil {
		intentRes.Spec.RolledBackFromId = *req.RolledBackFromID
	}
	return intentRes
}
