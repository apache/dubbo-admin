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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/core/manager"
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
	PutAdminHint(res coremodel.Resource, op Operation, source Source, author, reason string, rolledBackFromID *int64) error
	Rollback(ctx context.Context, rm manager.ResourceManager, kind coremodel.ResourceKind, mesh, ruleName string, versionID int64, reason string, expected *int64, author string) (*Version, error)
}

type service struct {
	enabled        bool
	maxVersions    int64
	hintTTL        time.Duration
	coalesceWindow time.Duration
	rollbackWait   time.Duration
	store          Store
	hints          *AdminHintRegistry
}

func NewService(enabled bool, maxVersions int64, coalesceWindow, hintTTL time.Duration, store Store, hints *AdminHintRegistry) Service {
	return NewServiceWithRollbackWait(enabled, maxVersions, coalesceWindow, hintTTL, 0, store, hints)
}

func NewServiceWithRollbackWait(enabled bool, maxVersions int64, coalesceWindow, hintTTL, rollbackWait time.Duration, store Store, hints *AdminHintRegistry) Service {
	return &service{
		enabled:        enabled,
		maxVersions:    maxVersions,
		hintTTL:        hintTTL,
		coalesceWindow: coalesceWindow,
		rollbackWait:   rollbackWait,
		store:          store,
		hints:          hints,
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
	return s.store.CheckExpectedVersion(kind, coremodel.BuildResourceKey(mesh, ruleName), expected)
}

func (s *service) PutAdminHint(res coremodel.Resource, op Operation, source Source, author, reason string, rolledBackFromID *int64) error {
	if err := s.ensureEnabled(); err != nil {
		return nil
	}
	hash, specJSON, err := NormalizeResource(res)
	if op == OperationDelete {
		hash = HashSpecJSON(DeleteSpecJSON)
		specJSON = DeleteSpecJSON
		err = nil
	}
	if err != nil {
		return err
	}
	if strings.TrimSpace(author) == "" {
		author = "system:unknown"
	}
	s.hints.Put(res.ResourceKind(), res.ResourceKey(), hash, AdminHint{
		RuleKind:         res.ResourceKind(),
		Mesh:             res.ResourceMesh(),
		ResourceKey:      res.ResourceKey(),
		RuleName:         res.ResourceMeta().Name,
		ContentHash:      hash,
		SpecJSON:         specJSON,
		Source:           source,
		Author:           author,
		Reason:           reason,
		Operation:        op,
		RolledBackFromID: rolledBackFromID,
		ExpiresAt:        time.Now().Add(s.hintTTL),
	})
	return nil
}

func (s *service) Rollback(ctx context.Context, rm manager.ResourceManager, kind coremodel.ResourceKind, mesh, ruleName string, versionID int64, reason string, expected *int64, author string) (*Version, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, bizerror.New(bizerror.InvalidArgument, "rollback reason is required")
	}
	if err := s.store.CheckExpectedVersion(kind, coremodel.BuildResourceKey(mesh, ruleName), expected); err != nil {
		return nil, err
	}
	target, err := s.store.GetVersion(kind, coremodel.BuildResourceKey(mesh, ruleName), versionID)
	if err != nil {
		return nil, err
	}
	if target.Operation == OperationDelete {
		return nil, ErrRollbackToDelete
	}
	res, err := ResourceFromSpecJSON(kind, mesh, ruleName, target.SpecJSON)
	if err != nil {
		return nil, err
	}
	fromID := target.ID
	if err := s.PutAdminHint(res, OperationUpdate, SourceRollback, author, reason, &fromID); err != nil {
		return nil, err
	}
	if err := rm.Upsert(res); err != nil {
		return nil, err
	}
	wait := s.rollbackWait
	if wait == 0 {
		wait = s.coalesceWindow + 500*time.Millisecond
	}
	deadline := time.NewTimer(wait)
	defer deadline.Stop()
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, fmt.Errorf("timeout waiting for rollback version row")
		case <-tick.C:
			latest, err := s.store.LatestVersion(kind, res.ResourceKey())
			if err != nil {
				return nil, err
			}
			if latest != nil && latest.Source == SourceRollback && latest.RolledBackFromID != nil && *latest.RolledBackFromID == fromID {
				return latest, nil
			}
		}
	}
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
