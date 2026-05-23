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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"google.golang.org/protobuf/encoding/protojson"
)

type Service interface {
	Store() Store
	Hints() *AdminHintRegistry
	List(kind coremodel.ResourceKind, mesh, ruleName string) (*ListResult, error)
	Get(kind coremodel.ResourceKind, mesh, ruleName string, id int64) (*Version, error)
	Diff(kind coremodel.ResourceKind, mesh, ruleName string, id int64, against string) (*DiffResult, error)
	CheckExpected(kind coremodel.ResourceKind, mesh, ruleName string, expected *int64) error
	BeginMutationIntent(res coremodel.Resource, op Operation, source Source, author, reason string, expected *int64, rolledBackFromID *int64) (*Intent, error)
	MarkMutationIntentApplied(id int64) error
	FailMutationIntent(id int64, message string) error
	CommitMutationIntent(id int64) (*Version, error)
	RepairIntent(kind coremodel.ResourceKind, resourceKey string, current coremodel.Resource, deleted bool) (*Version, error)
	RepairIntentByID(id int64, current coremodel.Resource, deleted bool) (*Version, error)
	CommitMatchingIntent(kind coremodel.ResourceKind, resourceKey, contentHash string) (*Version, bool, error)
	RecordMutation(res coremodel.Resource, op Operation, source Source, author, reason string, rolledBackFromID *int64) (*Version, error)
	PutAdminHint(res coremodel.Resource, op Operation, source Source, author, reason string, rolledBackFromID *int64) error
}

type service struct {
	enabled     bool
	maxVersions int64
	hintTTL     time.Duration
	store       Store
	hints       *AdminHintRegistry
}

func NewService(enabled bool, maxVersions int64, coalesceWindow, hintTTL time.Duration, store Store, hints *AdminHintRegistry) Service {
	return NewServiceWithRollbackWait(enabled, maxVersions, coalesceWindow, hintTTL, 0, store, hints)
}

func NewServiceWithRollbackWait(enabled bool, maxVersions int64, coalesceWindow, hintTTL, rollbackWait time.Duration, store Store, hints *AdminHintRegistry) Service {
	_ = coalesceWindow
	_ = rollbackWait
	return &service{
		enabled:     enabled,
		maxVersions: maxVersions,
		hintTTL:     hintTTL,
		store:       store,
		hints:       hints,
	}
}

func (s *service) Store() Store {
	return s.store
}

func (s *service) Hints() *AdminHintRegistry {
	return s.hints
}

func (s *service) ensureEnabled() error {
	if !s.enabled {
		return ErrFeatureDisabled
	}
	return nil
}

func (s *service) List(kind coremodel.ResourceKind, mesh, ruleName string) (*ListResult, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	items, err := s.store.ListVersions(kind, coremodel.BuildResourceKey(mesh, ruleName))
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: items, Total: int64(len(items))}, nil
}

func (s *service) Get(kind coremodel.ResourceKind, mesh, ruleName string, id int64) (*Version, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	return s.store.GetVersion(kind, coremodel.BuildResourceKey(mesh, ruleName), id)
}

func (s *service) Diff(kind coremodel.ResourceKind, mesh, ruleName string, id int64, against string) (*DiffResult, error) {
	left, err := s.Get(kind, mesh, ruleName, id)
	if err != nil {
		return nil, err
	}
	var right *Version
	if against == "" || against == "current" {
		meta, err := s.store.CurrentMeta(kind, coremodel.BuildResourceKey(mesh, ruleName))
		if err != nil {
			return nil, err
		}
		if meta == nil || meta.CurrentVersion == nil {
			return nil, ErrVersionNotFound
		}
		right, err = s.store.GetVersion(kind, coremodel.BuildResourceKey(mesh, ruleName), *meta.CurrentVersion)
		if err != nil {
			return nil, err
		}
	} else {
		var againstID int64
		if _, err := fmt.Sscan(against, &againstID); err != nil {
			return nil, bizerror.New(bizerror.InvalidArgument, "against must be a version id or current")
		}
		right, err = s.Get(kind, mesh, ruleName, againstID)
		if err != nil {
			return nil, err
		}
	}
	return &DiffResult{
		Left:  DiffSide{ID: left.ID, VersionNo: left.VersionNo, SpecJSON: left.SpecJSON},
		Right: DiffSide{ID: right.ID, VersionNo: right.VersionNo, SpecJSON: right.SpecJSON},
	}, nil
}

func (s *service) CheckExpected(kind coremodel.ResourceKind, mesh, ruleName string, expected *int64) error {
	if err := s.ensureEnabled(); err != nil {
		return nil
	}
	resourceKey := coremodel.BuildResourceKey(mesh, ruleName)
	intent, err := s.store.OpenIntent(kind, resourceKey)
	if err != nil {
		return err
	}
	if intent != nil {
		return &IntentPendingError{IntentID: intent.ID}
	}
	return s.store.CheckExpectedVersion(kind, resourceKey, expected)
}

func (s *service) BeginMutationIntent(res coremodel.Resource, op Operation, source Source, author, reason string, expected *int64, rolledBackFromID *int64) (*Intent, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, nil
	}
	req, err := buildMutationInsertRequest(res, op, source, author, reason, rolledBackFromID, time.Now())
	if err != nil {
		return nil, err
	}
	return s.store.CreateIntent(req, expected)
}

func (s *service) MarkMutationIntentApplied(id int64) error {
	if err := s.ensureEnabled(); err != nil {
		return nil
	}
	return s.store.MarkIntentApplied(id)
}

func (s *service) FailMutationIntent(id int64, message string) error {
	if err := s.ensureEnabled(); err != nil {
		return nil
	}
	return s.store.MarkIntentFailed(id, message)
}

func (s *service) CommitMutationIntent(id int64) (*Version, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, nil
	}
	return s.store.CommitIntent(id, s.maxVersions)
}

func (s *service) RepairIntent(kind coremodel.ResourceKind, resourceKey string, current coremodel.Resource, deleted bool) (*Version, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, nil
	}
	intent, err := s.store.OpenIntent(kind, resourceKey)
	if err != nil || intent == nil {
		return nil, err
	}
	return s.repairIntent(intent, current, deleted)
}

func (s *service) RepairIntentByID(id int64, current coremodel.Resource, deleted bool) (*Version, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, nil
	}
	intent, err := s.store.GetIntent(id)
	if err != nil {
		return nil, err
	}
	return s.repairIntent(intent, current, deleted)
}

func (s *service) repairIntent(intent *Intent, current coremodel.Resource, deleted bool) (*Version, error) {
	if intent == nil {
		return nil, nil
	}
	if intent.Status == IntentStatusCommitted {
		if intent.VersionID == nil {
			return nil, ErrVersionIntentNotOpen
		}
		return s.store.GetVersionByID(*intent.VersionID)
	}
	if intent.Status == IntentStatusFailed {
		return nil, ErrVersionIntentNotOpen
	}
	if intent.Status != IntentStatusPending && intent.Status != IntentStatusApplied {
		return nil, ErrVersionIntentNotOpen
	}
	if intent.Status == IntentStatusApplied || IntentMatchesResource(intent, current, deleted) {
		if intent.Status == IntentStatusPending {
			if err := s.store.MarkIntentApplied(intent.ID); err != nil {
				return nil, err
			}
		}
		return s.store.CommitIntent(intent.ID, s.maxVersions)
	}
	return nil, &IntentPendingError{IntentID: intent.ID}
}

func (s *service) CommitMatchingIntent(kind coremodel.ResourceKind, resourceKey, contentHash string) (*Version, bool, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, false, nil
	}
	intent, err := s.store.FindOpenIntentByHash(kind, resourceKey, contentHash)
	if err != nil || intent == nil {
		return nil, false, err
	}
	if intent.Status == IntentStatusPending {
		if err := s.store.MarkIntentApplied(intent.ID); err != nil {
			return nil, false, err
		}
	}
	version, err := s.store.CommitIntent(intent.ID, s.maxVersions)
	return version, true, err
}

func (s *service) RecordMutation(res coremodel.Resource, op Operation, source Source, author, reason string, rolledBackFromID *int64) (*Version, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, nil
	}
	req, err := buildMutationInsertRequest(res, op, source, author, reason, rolledBackFromID, time.Now())
	if err != nil {
		return nil, err
	}
	return s.store.InsertVersion(req, s.maxVersions)
}

func (s *service) PutAdminHint(res coremodel.Resource, op Operation, source Source, author, reason string, rolledBackFromID *int64) error {
	if err := s.ensureEnabled(); err != nil {
		return nil
	}
	now := time.Now()
	req, err := buildMutationInsertRequest(res, op, source, author, reason, rolledBackFromID, now)
	if err != nil {
		return err
	}
	s.hints.Put(req.RuleKind, req.ResourceKey, req.ContentHash, AdminHint{
		RuleKind:         req.RuleKind,
		Mesh:             req.Mesh,
		ResourceKey:      req.ResourceKey,
		RuleName:         req.RuleName,
		ContentHash:      req.ContentHash,
		SpecJSON:         req.SpecJSON,
		Source:           req.Source,
		Author:           req.Author,
		Reason:           req.Reason,
		Operation:        req.Operation,
		RolledBackFromID: req.RolledBackFromID,
		ExpiresAt:        now.Add(s.hintTTL),
	})
	return nil
}

func buildMutationInsertRequest(res coremodel.Resource, op Operation, source Source, author, reason string, rolledBackFromID *int64, createdAt time.Time) (InsertRequest, error) {
	if res == nil {
		return InsertRequest{}, bizerror.New(bizerror.InvalidArgument, "rule resource is required")
	}
	hash, specJSON, err := NormalizeResource(res)
	if op == OperationDelete {
		hash = HashSpecJSON(DeleteSpecJSON)
		specJSON = DeleteSpecJSON
		err = nil
	}
	if err != nil {
		return InsertRequest{}, err
	}
	if strings.TrimSpace(author) == "" {
		author = "system:unknown"
	} else {
		author = strings.TrimSpace(author)
	}
	if source == "" {
		source = SourceAdmin
	}
	return InsertRequest{
		RuleKind:         res.ResourceKind(),
		Mesh:             res.ResourceMesh(),
		ResourceKey:      res.ResourceKey(),
		RuleName:         res.ResourceMeta().Name,
		SpecJSON:         specJSON,
		ContentHash:      hash,
		Source:           source,
		Operation:        op,
		Author:           author,
		Reason:           reason,
		RolledBackFromID: rolledBackFromID,
		CreatedAt:        createdAt,
	}, nil
}

func IntentMatchesResource(intent *Intent, current coremodel.Resource, deleted bool) bool {
	if intent == nil {
		return false
	}
	if deleted || current == nil {
		return intent.Operation == OperationDelete && intent.ContentHash == HashSpecJSON(DeleteSpecJSON)
	}
	hash, _, err := NormalizeResource(current)
	return err == nil && hash == intent.ContentHash
}

func ResourceFromSpecJSON(kind coremodel.ResourceKind, mesh, ruleName, specJSON string) (coremodel.Resource, error) {
	switch kind {
	case meshresource.ConditionRouteKind:
		res := meshresource.NewConditionRouteResourceWithAttributes(ruleName, mesh)
		var spec meshproto.ConditionRoute
		if err := protojson.Unmarshal([]byte(specJSON), &spec); err != nil {
			if err := json.Unmarshal([]byte(specJSON), &spec); err != nil {
				return nil, err
			}
		}
		res.Spec = &spec
		return res, nil
	case meshresource.TagRouteKind:
		res := meshresource.NewTagRouteResourceWithAttributes(ruleName, mesh)
		var spec meshproto.TagRoute
		if err := protojson.Unmarshal([]byte(specJSON), &spec); err != nil {
			if err := json.Unmarshal([]byte(specJSON), &spec); err != nil {
				return nil, err
			}
		}
		res.Spec = &spec
		return res, nil
	case meshresource.DynamicConfigKind:
		res := meshresource.NewDynamicConfigResourceWithAttributes(ruleName, mesh)
		var spec meshproto.DynamicConfig
		if err := protojson.Unmarshal([]byte(specJSON), &spec); err != nil {
			if err := json.Unmarshal([]byte(specJSON), &spec); err != nil {
				return nil, err
			}
		}
		res.Spec = &spec
		return res, nil
	default:
		return nil, bizerror.New(bizerror.InvalidArgument, "unsupported rule kind")
	}
}
