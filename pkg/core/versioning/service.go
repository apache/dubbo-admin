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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// Service provides rule history operations. RuleVersion entries are audit
// records and rollback material; live current state is still owned by
// ResourceManager and the backing registry.
type Service struct {
	maxVersions int64
	store       *ResourceStoreAdapter
}

func NewService(maxVersions int64, store *ResourceStoreAdapter) *Service {
	return &Service{
		maxVersions: maxVersions,
		store:       store,
	}
}

func (s *Service) ensureAvailable() error {
	if s == nil || s.store == nil {
		return ErrVersionStoreError
	}
	return nil
}

func (s *Service) List(kind coremodel.ResourceKind, mesh, ruleName string) (*ListResult, error) {
	if err := s.ensureAvailable(); err != nil {
		return nil, err
	}
	resourceKey := coremodel.BuildResourceKey(mesh, ruleName)
	snapshot, err := s.store.HistorySnapshot(kind, resourceKey)
	if err != nil {
		return nil, err
	}
	result := &ListResult{Items: snapshot.Versions, Total: int64(len(snapshot.Versions)), LatestRecordedDeleted: snapshot.Deleted}
	if snapshot.Head != nil {
		result.LatestRecordedVersionNo = snapshot.Head.VersionNo
	}
	return result, nil
}

func (s *Service) Get(kind coremodel.ResourceKind, mesh, ruleName string, versionNo int64) (*Version, error) {
	if err := s.ensureAvailable(); err != nil {
		return nil, err
	}
	resourceKey := coremodel.BuildResourceKey(mesh, ruleName)
	version, err := s.store.GetVersion(kind, resourceKey, versionNo)
	if err != nil {
		return nil, err
	}
	snapshot, err := s.store.HistorySnapshot(kind, resourceKey)
	if err != nil {
		return nil, err
	}
	if snapshot.Head != nil {
		version.IsLatestRecorded = version.VersionNo == snapshot.Head.VersionNo
	}
	return version, nil
}

func (s *Service) DiffHistoryVersions(kind coremodel.ResourceKind, mesh, ruleName string, versionNo int64, against string) (*DiffResult, error) {
	if err := s.ensureAvailable(); err != nil {
		return nil, err
	}
	left, err := s.Get(kind, mesh, ruleName, versionNo)
	if err != nil {
		return nil, err
	}
	right, err := s.diffRight(kind, mesh, ruleName, versionNo, against)
	if err != nil {
		return nil, err
	}
	return &DiffResult{
		Left:  DiffSide{VersionNo: left.VersionNo, SpecJSON: left.SpecJSON},
		Right: DiffSide{VersionNo: right.VersionNo, SpecJSON: right.SpecJSON},
	}, nil
}

func (s *Service) diffRight(kind coremodel.ResourceKind, mesh, ruleName string, versionNo int64, against string) (*Version, error) {
	switch against {
	case "previous":
		list, err := s.store.ListVersions(kind, coremodel.BuildResourceKey(mesh, ruleName))
		if err != nil {
			return nil, err
		}
		sort.Slice(list, func(i, j int) bool {
			return list[i].VersionNo > list[j].VersionNo
		})
		for i := range list {
			if list[i].VersionNo != versionNo {
				continue
			}
			if i+1 >= len(list) {
				return nil, ErrVersionNotFound
			}
			return &list[i+1], nil
		}
		return nil, ErrVersionNotFound
	case "", "current":
		return nil, bizerror.New(bizerror.InvalidArgument, "current diff requires live ResourceManager state and is implemented by the console service")
	default:
		againstVersionNo, err := strconv.ParseInt(against, 10, 64)
		if err != nil || againstVersionNo <= 0 {
			return nil, bizerror.New(bizerror.InvalidArgument, "against must be 'current', 'previous', or a version number")
		}
		return s.Get(kind, mesh, ruleName, againstVersionNo)
	}
}

func (s *Service) Append(ctx context.Context, res coremodel.Resource, op Operation, source Source, author, reason string, rolledBackFromVersionNo *int64) (*Version, error) {
	if err := s.ensureAvailable(); err != nil {
		return nil, err
	}
	req, err := BuildInsertRequest(res, op, source, author, reason, rolledBackFromVersionNo, time.Now())
	if err != nil {
		return nil, err
	}
	return s.store.InsertVersion(ctx, req, s.maxVersions)
}

func (s *Service) LatestVersion(kind coremodel.ResourceKind, mesh, ruleName string) (*Version, error) {
	if err := s.ensureAvailable(); err != nil {
		return nil, err
	}
	return s.store.LatestVersion(kind, coremodel.BuildResourceKey(mesh, ruleName))
}

func (s *Service) HasHistory(kind coremodel.ResourceKind, mesh, ruleName string) (bool, error) {
	if err := s.ensureAvailable(); err != nil {
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

func BuildInsertRequest(res coremodel.Resource, op Operation, source Source, author, reason string, rolledBackFromVersionNo *int64, createdAt time.Time) (InsertRequest, error) {
	if res == nil {
		return InsertRequest{}, bizerror.New(bizerror.InvalidArgument, "rule resource is required")
	}
	hash, specJSON, err := NormalizeResource(res)
	if err != nil {
		return InsertRequest{}, err
	}
	if op == OperationDelete {
		specJSON = DeleteSpecJSON
		hash = HashSpecJSON(specJSON)
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
		RuleKind:                res.ResourceKind(),
		Mesh:                    res.ResourceMesh(),
		ResourceKey:             res.ResourceKey(),
		RuleName:                res.ResourceMeta().Name,
		SpecJSON:                specJSON,
		ContentHash:             hash,
		Source:                  source,
		Operation:               op,
		Author:                  author,
		Reason:                  reason,
		RolledBackFromVersionNo: rolledBackFromVersionNo,
		CreatedAt:               createdAt,
	}, nil
}
