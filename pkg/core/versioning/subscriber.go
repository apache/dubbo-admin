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
	"time"

	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

const IntentIDEventContextKey = "rule-version-intent-id"

type Subscriber struct {
	kind        coremodel.ResourceKind
	store       Store
	maxVersions int64
	lockMgr     lock.Lock
}

type ParentRef struct {
	Kind        coremodel.ResourceKind
	Mesh        string
	Name        string
	ResourceKey string
}

type normalizedRuleEvent struct {
	Resource         coremodel.Resource
	Parent           ParentRef
	Operation        Operation
	SpecJSON         []byte
	ContentHash      string
	MutationIntentID int64
	Context          map[string]string
}

func NewSubscriber(kind coremodel.ResourceKind, store Store, maxVersions int64, lockMgr lock.Lock) *Subscriber {
	if lockMgr == nil {
		panic("rule version subscriber requires lock manager")
	}
	return &Subscriber{
		kind:        kind,
		store:       store,
		maxVersions: maxVersions,
		lockMgr:     lockMgr,
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

	ctx := event.Context()
	intentID := int64(0)
	if ctx != nil && ctx[IntentIDEventContextKey] != "" {
		parsed, err := strconv.ParseInt(ctx[IntentIDEventContextKey], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid intent token %q", ErrVersionLedgerCorrupt, ctx[IntentIDEventContextKey])
		}
		intentID = parsed
	}

	return &normalizedRuleEvent{
		Resource: res,
		Parent: ParentRef{
			Kind:        res.ResourceKind(),
			Mesh:        res.ResourceMesh(),
			Name:        res.ResourceMeta().Name,
			ResourceKey: res.ResourceKey(),
		},
		Operation:        op,
		SpecJSON:         []byte(specJSON),
		ContentHash:      hash,
		MutationIntentID: intentID,
		Context:          ctx,
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
	if normalized.MutationIntentID == 0 {
		openIntent, err := s.store.OpenIntent(normalized.Parent.Kind, normalized.Parent.ResourceKey)
		if err != nil {
			return err
		}
		if openIntent != nil {
			logger.Infof("skipping un-tokened event for %s while rule version intent %d is open", normalized.Parent.ResourceKey, openIntent.ID)
			return nil
		}
	}
	return withRuleVersionLock(s.lockMgr, normalized.Parent.Kind, normalized.Parent.ResourceKey, func(leaseCtx context.Context) error {
		return s.record(leaseCtx, *normalized)
	})
}

func (s *Subscriber) record(ctx context.Context, event normalizedRuleEvent) error {
	if err := lock.CheckLease(ctx); err != nil {
		return err
	}
	if _, err := s.store.ReconcileMeta(event.Parent.Kind, event.Parent.ResourceKey); err != nil {
		return err
	}
	if err := lock.CheckLease(ctx); err != nil {
		return err
	}
	if committed, err := s.tryCommitIntentFromToken(ctx, event); err != nil {
		return err
	} else if committed {
		return nil
	}
	openIntent, err := s.store.OpenIntent(event.Parent.Kind, event.Parent.ResourceKey)
	if err != nil {
		return err
	}
	if openIntent != nil {
		logger.Infof("skipping un-tokened event for %s while rule version intent %d is open", event.Parent.ResourceKey, openIntent.ID)
		return nil
	}

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

	_, err = s.store.InsertVersion(req, s.maxVersions)
	if err != nil {
		return fmt.Errorf("failed to insert version: %w", err)
	}

	// InsertVersion already handles:
	// - Version number allocation from Meta
	// - RuleVersion resource creation
	// - Meta update (for optimistic locking)
	// - Old version cleanup (trimming)

	return nil
}

func (s *Subscriber) tryCommitIntentFromToken(ctx context.Context, event normalizedRuleEvent) (bool, error) {
	if event.MutationIntentID == 0 {
		return false, nil
	}
	intent, err := s.store.GetIntent(event.MutationIntentID)
	if err != nil {
		return false, err
	}
	if intent.Status != IntentStatusPending && intent.Status != IntentStatusApplied {
		return false, ErrVersionIntentNotOpen
	}
	if intent.RuleKind != event.Parent.Kind ||
		intent.ResourceKey != event.Parent.ResourceKey ||
		intent.ContentHash != event.ContentHash ||
		intent.Operation != event.Operation {
		return false, fmt.Errorf("%w: intent token %d does not match event parent/content/operation", ErrVersionLedgerCorrupt, event.MutationIntentID)
	}
	if intent.Source == SourceRollback && intent.RolledBackFromID == nil {
		return false, fmt.Errorf("%w: rollback intent %d is missing rolledBackFromId", ErrVersionLedgerCorrupt, event.MutationIntentID)
	}
	if intent.Status == IntentStatusPending {
		if err := lock.CheckLease(ctx); err != nil {
			return false, err
		}
		if err := s.store.MarkIntentApplied(intent.ID); err != nil {
			return false, err
		}
	}
	if err := lock.CheckLease(ctx); err != nil {
		return false, err
	}
	if _, err := s.store.CommitIntent(intent.ID, s.maxVersions); err != nil {
		return false, err
	}
	return true, nil
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

// RecordBootstrap creates a baseline version for a rule during bootstrap.
func RecordBootstrap(store Store, maxVersions int64, res coremodel.Resource) error {
	kind := res.ResourceKind()
	if _, err := store.ReconcileMeta(kind, res.ResourceKey()); err != nil {
		return err
	}
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
	if _, err := store.InsertVersion(req, maxVersions); err != nil {
		return fmt.Errorf("bootstrap version for %s failed: %w", res.ResourceKey(), err)
	}
	return nil
}

func RecordBootstrapLocked(store Store, maxVersions int64, res coremodel.Resource, lockMgr lock.Lock) error {
	return withRuleVersionLock(lockMgr, res.ResourceKind(), res.ResourceKey(), func(ctx context.Context) error {
		if err := lock.CheckLease(ctx); err != nil {
			return err
		}
		return RecordBootstrap(store, maxVersions, res)
	})
}

func shortHash(hash string) string {
	if len(hash) <= 8 {
		return hash
	}
	return hash[:8]
}
