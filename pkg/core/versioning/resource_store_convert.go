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
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

func buildVersionResourceKey(kind coremodel.ResourceKind, parentResourceKey string, versionNo int64) string {
	name := fmt.Sprintf("%s-%s-version-%d", kind, extractName(parentResourceKey), versionNo)
	return coremodel.BuildResourceKey(extractMesh(parentResourceKey), name)
}

func buildParentIndexKey(kind coremodel.ResourceKind, resourceKey string) string {
	mesh := extractMesh(resourceKey)
	name := extractName(resourceKey)
	return fmt.Sprintf("%s/%s/%s", kind, mesh, name)
}

func extractMesh(resourceKey string) string {
	mesh, _ := coremodel.ParseResourceKey(resourceKey)
	return mesh
}

func extractName(resourceKey string) string {
	_, name := coremodel.ParseResourceKey(resourceKey)
	return name
}

func protoToVersion(spec *meshproto.RuleVersion) (*Version, error) {
	if spec == nil {
		return nil, bizerror.New(bizerror.InvalidArgument, "RuleVersion spec is nil")
	}

	var rolledBackFromVersionNo *int64
	if spec.RolledBackFromVersionNo != 0 {
		v := spec.RolledBackFromVersionNo
		rolledBackFromVersionNo = &v
	}

	createdAt := timestampAsTime(spec.CreatedAt)
	recordedAt := timestampAsTime(spec.RecordedAt)
	if recordedAt.IsZero() {
		recordedAt = createdAt
	}

	return &Version{
		RuleKind:                coremodel.ResourceKind(spec.ParentRuleKind),
		Mesh:                    spec.ParentRuleMesh,
		ResourceKey:             coremodel.BuildResourceKey(spec.ParentRuleMesh, spec.ParentRuleName),
		RuleName:                spec.ParentRuleName,
		VersionNo:               spec.VersionNo,
		ContentHash:             spec.ContentHash,
		SpecJSON:                spec.SpecJson,
		Operation:               Operation(spec.Operation),
		Source:                  Source(spec.Source),
		Author:                  spec.Author,
		Reason:                  spec.Reason,
		RolledBackFromVersionNo: rolledBackFromVersionNo,
		CreatedAt:               createdAt,
		RecordedAt:              recordedAt,
		IsLatestRecorded:        false,
	}, nil
}

func historySnapshotFromState(state *historyState) *HistorySnapshot {
	if state == nil {
		return &HistorySnapshot{}
	}
	snapshot := &HistorySnapshot{
		Versions: append([]Version(nil), state.Versions...),
	}
	if len(snapshot.Versions) == 0 {
		return snapshot
	}
	head := snapshot.Versions[0]
	snapshot.Head = &head
	snapshot.Deleted = head.Operation == OperationDelete
	snapshot.Versions[0].IsLatestRecorded = true
	return snapshot
}

func duplicateVersionNoError(kind coremodel.ResourceKind, resourceKey string, versionNo int64) error {
	return fmt.Errorf("%w: duplicate version number for kind=%s mesh=%s rule=%s versionNo=%d",
		ErrVersionStoreError,
		kind,
		extractMesh(resourceKey),
		extractName(resourceKey),
		versionNo,
	)
}

func timestampAsTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}
