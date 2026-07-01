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
	"strconv"
	"strings"
	"time"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// Service provides lightweight rule history operations. RuleVersion entries are
// audit records and rollback material; rule writes are still owned by
// ResourceManager and the backing registry.
type Service struct {
	maxVersions int64
	store       Store
}

func NewService(maxVersions int64, store Store) *Service {
	return &Service{
		maxVersions: maxVersions,
		store:       store,
	}
}

func (s *Service) ensureEnabled() error {
	if s == nil || s.store == nil {
		return ErrVersionStoreError
	}
	return nil
}

func (s *Service) List(kind coremodel.ResourceKind, mesh, ruleName string) (*ListResult, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	resourceKey := coremodel.BuildResourceKey(mesh, ruleName)
	snapshot, err := s.store.HistorySnapshot(kind, resourceKey)
	if err != nil {
		return nil, err
	}
	result := &ListResult{Items: snapshot.Versions, Total: int64(len(snapshot.Versions)), Deleted: snapshot.Deleted}
	if snapshot.Head != nil && !snapshot.Deleted {
		currentID := snapshot.Head.ID
		result.CurrentVersionID = &currentID
		result.CurrentVersionNo = snapshot.Head.VersionNo
	}
	return result, nil
}

func (s *Service) Get(kind coremodel.ResourceKind, mesh, ruleName string, id int64) (*Version, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	resourceKey := coremodel.BuildResourceKey(mesh, ruleName)
	version, err := s.store.GetVersion(kind, resourceKey, id)
	if err != nil {
		return nil, err
	}
	snapshot, err := s.store.HistorySnapshot(kind, resourceKey)
	if err != nil {
		return nil, err
	}
	if snapshot.Head != nil && !snapshot.Deleted {
		version.IsCurrent = version.ID == snapshot.Head.ID
	}
	return version, nil
}

func (s *Service) Diff(kind coremodel.ResourceKind, mesh, ruleName string, id int64, against string) (*DiffResult, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	left, err := s.Get(kind, mesh, ruleName, id)
	if err != nil {
		return nil, err
	}
	right, err := s.diffRight(kind, mesh, ruleName, id, against)
	if err != nil {
		return nil, err
	}
	return &DiffResult{
		Left:  DiffSide{ID: left.ID, VersionNo: left.VersionNo, SpecJSON: left.SpecJSON},
		Right: DiffSide{ID: right.ID, VersionNo: right.VersionNo, SpecJSON: right.SpecJSON},
	}, nil
}

func (s *Service) diffRight(kind coremodel.ResourceKind, mesh, ruleName string, id int64, against string) (*Version, error) {
	switch against {
	case "previous":
		list, err := s.store.ListVersions(kind, coremodel.BuildResourceKey(mesh, ruleName))
		if err != nil {
			return nil, err
		}
		for i := range list {
			if list[i].ID != id {
				continue
			}
			if i+1 >= len(list) {
				return nil, ErrVersionNotFound
			}
			return &list[i+1], nil
		}
		return nil, ErrVersionNotFound
	case "", "current":
		resourceKey := coremodel.BuildResourceKey(mesh, ruleName)
		snapshot, err := s.store.HistorySnapshot(kind, resourceKey)
		if err != nil {
			return nil, err
		}
		if snapshot.Head == nil {
			return nil, ErrVersionNotFound
		}
		return snapshot.Head, nil
	default:
		againstID, err := strconv.ParseInt(against, 10, 64)
		if err != nil {
			return nil, bizerror.New(bizerror.InvalidArgument, "against must be 'current', 'previous', or a version ID")
		}
		return s.Get(kind, mesh, ruleName, againstID)
	}
}

func (s *Service) Append(ctx context.Context, res coremodel.Resource, op Operation, source Source, author, reason string, rolledBackFromID *int64) (*Version, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	req, err := BuildInsertRequest(res, op, source, author, reason, rolledBackFromID, time.Now())
	if err != nil {
		return nil, err
	}
	return s.store.InsertVersion(ctx, req, s.maxVersions)
}

func (s *Service) AppendDelete(ctx context.Context, res coremodel.Resource, source Source, author, reason string) (*Version, error) {
	return s.Append(ctx, res, OperationDelete, source, author, reason, nil)
}

func (s *Service) LatestVersion(kind coremodel.ResourceKind, mesh, ruleName string) (*Version, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	return s.store.LatestVersion(kind, coremodel.BuildResourceKey(mesh, ruleName))
}

func (s *Service) HasHistory(kind coremodel.ResourceKind, mesh, ruleName string) (bool, error) {
	if err := s.ensureEnabled(); err != nil {
		return false, err
	}
	_, err := s.store.LatestVersion(kind, coremodel.BuildResourceKey(mesh, ruleName))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrVersionNotFound) {
		return false, nil
	}
	return false, err
}

func BuildInsertRequest(res coremodel.Resource, op Operation, source Source, author, reason string, rolledBackFromID *int64, createdAt time.Time) (InsertRequest, error) {
	if res == nil {
		return InsertRequest{}, bizerror.New(bizerror.InvalidArgument, "rule resource is required")
	}
	hash, specJSON, err := NormalizeResource(res)
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
