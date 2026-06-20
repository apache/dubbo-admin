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
	"time"

	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type Subscriber struct {
	kind        coremodel.ResourceKind
	store       Store
	maxVersions int64
	lockMgr     lock.Lock
	appCtx      context.Context
}

type ParentRef struct {
	Kind        coremodel.ResourceKind
	Mesh        string
	Name        string
	ResourceKey string
}

type normalizedRuleEvent struct {
	Resource    coremodel.Resource
	Parent      ParentRef
	Operation   Operation
	SpecJSON    []byte
	ContentHash string
	Context     map[string]string
}

func NewSubscriber(kind coremodel.ResourceKind, store Store, maxVersions int64, lockMgr lock.Lock, appCtx context.Context) *Subscriber {
	return &Subscriber{
		kind:        kind,
		store:       store,
		maxVersions: maxVersions,
		lockMgr:     lockMgr,
		appCtx:      appCtx,
	}
}

func (s *Subscriber) ResourceKind() coremodel.ResourceKind {
	return s.kind
}

func (s *Subscriber) Name() string {
	return "rule-version-" + s.kind.ToString()
}

func (s *Subscriber) AsyncEnabled() bool {
	return false
}

func normalizeRuleEvent(event events.Event) (*normalizedRuleEvent, error) {
	var res coremodel.Resource
	var op Operation
	switch event.Type() {
	case cache.Added:
		res = event.NewObj()
		op = OperationCreate
	case cache.Updated:
		res = event.NewObj()
		op = OperationUpdate
	case cache.Deleted:
		res = event.OldObj()
		op = OperationDelete
	default:
		return nil, nil
	}
	if res == nil {
		return nil, nil
	}

	hash, specJSON, err := NormalizeResource(res)
	if op == OperationDelete {
		hash = HashSpecJSON(DeleteSpecJSON)
		specJSON = DeleteSpecJSON
		err = nil
	}
	if err != nil {
		return nil, err
	}

	return &normalizedRuleEvent{
		Resource: res,
		Parent: ParentRef{
			Kind:        res.ResourceKind(),
			Mesh:        res.ResourceMesh(),
			Name:        res.ResourceMeta().Name,
			ResourceKey: res.ResourceKey(),
		},
		Operation:   op,
		SpecJSON:    []byte(specJSON),
		ContentHash: hash,
		Context:     event.Context(),
	}, nil
}

func (s *Subscriber) ProcessEvent(event events.Event) error {
	normalized, err := normalizeRuleEvent(event)
	if err != nil {
		return err
	}
	if normalized == nil {
		return nil
	}
	openIntent, err := s.store.OpenIntent(normalized.Parent.Kind, normalized.Parent.ResourceKey)
	if err != nil {
		return err
	}
	if openIntent != nil {
		return s.handleOpenIntentEvent(openIntent, *normalized)
	}
	if s.lockMgr == nil {
		return lock.ErrLockUnavailable
	}
	if s.appCtx == nil {
		return context.Canceled
	}
	return withRuleVersionLock(s.appCtx, s.lockMgr, normalized.Parent.Kind, normalized.Parent.ResourceKey, func(leaseCtx context.Context) error {
		return s.record(leaseCtx, *normalized)
	})
}

func (s *Subscriber) record(ctx context.Context, event normalizedRuleEvent) error {
	if err := lock.CheckLease(ctx); err != nil {
		return err
	}
	openIntent, err := s.store.OpenIntent(event.Parent.Kind, event.Parent.ResourceKey)
	if err != nil {
		return err
	}
	if openIntent != nil {
		return s.handleOpenIntentEvent(openIntent, event)
	}
	return s.recordVersion(ctx, event)
}

func (s *Subscriber) recordVersion(ctx context.Context, event normalizedRuleEvent) error {
	source := SourceUpstream
	author := "system:upstream"
	if event.Context != nil {
		if registry := event.Context[events.SourceRegistryContextKey]; registry != "" {
			author = "system:" + registry
		}
	}

	if exists, err := s.checkDuplicate(event.Parent.Kind, event.Parent.ResourceKey, event.Operation, event.ContentHash); err != nil {
		return fmt.Errorf("failed to check duplicate hash: %w", err)
	} else if exists {
		logger.Infof("skipping duplicate version for %s (operation=%s hash=%s)", event.Parent.ResourceKey, event.Operation, shortHash(event.ContentHash))
		return nil
	}

	if err := lock.CheckLease(ctx); err != nil {
		return err
	}
	req := InsertRequest{
		RuleKind:    event.Parent.Kind,
		Mesh:        event.Parent.Mesh,
		ResourceKey: event.Parent.ResourceKey,
		RuleName:    event.Parent.Name,
		SpecJSON:    string(event.SpecJSON),
		ContentHash: event.ContentHash,
		Operation:   event.Operation,
		Source:      source,
		Author:      author,
		CreatedAt:   time.Now(),
	}

	_, err := s.store.InsertVersion(ctx, req, s.maxVersions)
	if err != nil {
		return fmt.Errorf("failed to insert version: %w", err)
	}

	return nil
}

func (s *Subscriber) handleOpenIntentEvent(openIntent *Intent, event normalizedRuleEvent) error {
	if intentMatchesEvent(openIntent, event) {
		logger.Infof("skipping admin echo rule event for %s while rule version intent %d is open", event.Parent.ResourceKey, openIntent.ID)
		return nil
	}
	logger.Infof("recording non-matching rule event for %s while rule version intent %d is open; intent close will reconcile actual state", event.Parent.ResourceKey, openIntent.ID)
	return s.markIntentObservedOrRecord(openIntent, event)
}

func (s *Subscriber) markIntentObservedOrRecord(openIntent *Intent, event normalizedRuleEvent) error {
	if s.appCtx == nil {
		return context.Canceled
	}
	current := openIntent
	for attempt := 0; attempt < maxIntentCASRetries; attempt++ {
		if err := s.appCtx.Err(); err != nil {
			return err
		}
		if current == nil || current.Status == IntentStatusCommitting {
			return s.recordAfterIntentClosed(event)
		}
		err := s.store.MarkIntentObserved(s.appCtx, current.ID, event.Operation, event.ContentHash, string(event.SpecJSON))
		if err == nil {
			return nil
		}
		if errors.Is(err, ErrVersionIntentNotFound) || errors.Is(err, ErrVersionIntentNotOpen) {
			return s.recordAfterIntentClosed(event)
		}
		if !errors.Is(err, ErrVersionIntentConflict) {
			return err
		}
		refreshed, refreshErr := s.store.OpenIntent(event.Parent.Kind, event.Parent.ResourceKey)
		if refreshErr != nil {
			return refreshErr
		}
		if refreshed != nil && intentMatchesEvent(refreshed, event) {
			logger.Infof("skipping admin echo rule event for %s after intent refresh; rule version intent %d is open", event.Parent.ResourceKey, refreshed.ID)
			return nil
		}
		current = refreshed
	}
	return s.recordAfterIntentClosed(event)
}

func (s *Subscriber) recordAfterIntentClosed(event normalizedRuleEvent) error {
	if s.lockMgr == nil {
		return lock.ErrLockUnavailable
	}
	if s.appCtx == nil {
		return context.Canceled
	}
	return withRuleVersionLock(s.appCtx, s.lockMgr, event.Parent.Kind, event.Parent.ResourceKey, func(leaseCtx context.Context) error {
		for attempt := 0; attempt < maxIntentCASRetries; attempt++ {
			if err := lock.CheckLease(leaseCtx); err != nil {
				return err
			}
			openIntent, err := s.store.OpenIntent(event.Parent.Kind, event.Parent.ResourceKey)
			if err != nil {
				return err
			}
			if openIntent == nil || openIntent.Status == IntentStatusCommitting {
				return s.recordVersion(leaseCtx, event)
			}
			if intentMatchesEvent(openIntent, event) {
				logger.Infof("skipping admin echo rule event for %s after lock reacquire; rule version intent %d is open", event.Parent.ResourceKey, openIntent.ID)
				return nil
			}
			err = s.store.MarkIntentObserved(leaseCtx, openIntent.ID, event.Operation, event.ContentHash, string(event.SpecJSON))
			if err == nil {
				return nil
			}
			if errors.Is(err, ErrVersionIntentNotFound) || errors.Is(err, ErrVersionIntentNotOpen) {
				continue
			}
			if errors.Is(err, ErrVersionIntentConflict) {
				continue
			}
			return err
		}
		return s.recordVersion(leaseCtx, event)
	})
}

func intentMatchesEvent(intent *Intent, event normalizedRuleEvent) bool {
	return intent != nil &&
		intent.RuleKind == event.Parent.Kind &&
		intent.ResourceKey == event.Parent.ResourceKey &&
		intent.Operation == event.Operation &&
		intent.ContentHash == event.ContentHash &&
		intent.SpecJSON == string(event.SpecJSON)
}

func (s *Subscriber) checkDuplicate(kind coremodel.ResourceKind, resourceKey string, op Operation, hash string) (bool, error) {
	latest, err := s.store.LatestVersion(kind, resourceKey)
	if errors.Is(err, ErrVersionNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if latest.Operation == OperationDelete && op == OperationDelete {
		return true, nil
	}
	if latest.Operation == OperationDelete {
		return false, nil
	}
	return op != OperationDelete && latest.ContentHash == hash, nil
}

// recordBootstrapState creates a baseline version for a rule during bootstrap.
func recordBootstrapState(ctx context.Context, store Store, maxVersions int64, res coremodel.Resource) error {
	kind := res.ResourceKind()
	versions, err := store.ListVersions(kind, res.ResourceKey())
	if err != nil {
		return err
	}
	if len(versions) > 0 {
		return nil
	}

	hash, specJSON, err := NormalizeResource(res)
	if err != nil {
		return err
	}

	req := InsertRequest{
		RuleKind:    kind,
		Mesh:        res.ResourceMesh(),
		ResourceKey: res.ResourceKey(),
		RuleName:    res.ResourceMeta().Name,
		SpecJSON:    specJSON,
		ContentHash: hash,
		Source:      SourceBootstrap,
		Operation:   OperationCreate,
		Author:      "system:bootstrap",
		CreatedAt:   time.Now(),
	}
	if _, err := store.InsertVersion(ctx, req, maxVersions); err != nil {
		return fmt.Errorf("bootstrap version for %s failed: %w", res.ResourceKey(), err)
	}
	return nil
}

func RecordBootstrapLocked(ctx context.Context, store Store, maxVersions int64, kind coremodel.ResourceKind, resourceKey string, rm manager.ResourceManager, lockMgr lock.Lock) error {
	return withRuleVersionLock(ctx, lockMgr, kind, resourceKey, func(ctx context.Context) error {
		if err := lock.CheckLease(ctx); err != nil {
			return err
		}
		current, exists, err := rm.GetByKey(kind, resourceKey)
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}
		return recordBootstrapState(ctx, store, maxVersions, current)
	})
}

func shortHash(hash string) string {
	if len(hash) <= 8 {
		return hash
	}
	return hash[:8]
}
