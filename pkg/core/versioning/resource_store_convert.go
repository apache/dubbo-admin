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
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

const ruleVersionIDAnnotation = "dubbo.apache.org/rule-version-id"

func buildVersionName(kind coremodel.ResourceKind, resourceKey string, id int64) string {
	return fmt.Sprintf("%s-%s-%d", kind, extractName(resourceKey), id)
}

func buildVersionNoName(kind coremodel.ResourceKind, resourceKey string, versionNo int64) string {
	return fmt.Sprintf("%s-%s-version-%d", kind, extractName(resourceKey), versionNo)
}

func buildParentIndexKey(kind coremodel.ResourceKind, resourceKey string) string {
	mesh := extractMesh(resourceKey)
	name := extractName(resourceKey)
	return fmt.Sprintf("%s/%s/%s", kind, mesh, name)
}

func extractMesh(resourceKey string) string {
	// resourceKey format: "mesh/name" or just "name"
	for i := 0; i < len(resourceKey); i++ {
		if resourceKey[i] == '/' {
			return resourceKey[:i]
		}
	}
	return ""
}

func extractName(resourceKey string) string {
	// resourceKey format: "mesh/name" or just "name"
	for i := 0; i < len(resourceKey); i++ {
		if resourceKey[i] == '/' {
			return resourceKey[i+1:]
		}
	}
	return resourceKey
}

func extractIDFromName(name string) (int64, error) {
	idx := strings.LastIndex(name, "-")
	if idx == -1 || idx == len(name)-1 {
		return 0, fmt.Errorf("invalid version name format: %s", name)
	}
	id, err := strconv.ParseInt(name[idx+1:], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid version name format: %s", name)
	}
	return id, nil
}

func versionIDFromResource(rv *meshresource.RuleVersionResource) (int64, error) {
	if rv == nil {
		return 0, fmt.Errorf("RuleVersion resource is nil")
	}
	if rv.Annotations != nil {
		if raw := rv.Annotations[ruleVersionIDAnnotation]; raw != "" {
			id, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid RuleVersion id annotation for %s: %w", rv.Name, err)
			}
			return id, nil
		}
	}
	return extractIDFromName(rv.Name)
}

func protoToVersion(spec *meshproto.RuleVersion, id int64) (*Version, error) {
	if spec == nil {
		return nil, bizerror.New(bizerror.InvalidArgument, "RuleVersion spec is nil")
	}

	var rolledBackFromID *int64
	if spec.RolledBackFromId != 0 {
		v := spec.RolledBackFromId
		rolledBackFromID = &v
	}

	createdAt := timestampAsTime(spec.CreatedAt)
	committedAt := timestampAsTime(spec.CommittedAt)
	if committedAt.IsZero() {
		committedAt = createdAt
	}

	return &Version{
		ID:               id,
		RuleKind:         coremodel.ResourceKind(spec.ParentRuleKind),
		Mesh:             spec.ParentRuleMesh,
		ResourceKey:      coremodel.BuildResourceKey(spec.ParentRuleMesh, spec.ParentRuleName),
		RuleName:         spec.ParentRuleName,
		VersionNo:        spec.VersionNo,
		ContentHash:      spec.ContentHash,
		SpecJSON:         spec.SpecJson,
		Operation:        Operation(spec.Operation),
		Source:           Source(spec.Source),
		Author:           spec.Author,
		Reason:           spec.Reason,
		IntentID:         spec.IntentId,
		RolledBackFromID: rolledBackFromID,
		CreatedAt:        createdAt,
		CommittedAt:      committedAt,
		IsCurrent:        false, // Will be set by caller based on Meta
	}, nil
}

// Intent helper functions

func buildIntentName(kind coremodel.ResourceKind, resourceKey string, id int64) string {
	return fmt.Sprintf("%s-%s-intent-%d", kind, extractName(resourceKey), id)
}

func extractIDFromIntentName(name string) (int64, error) {
	id, err := extractIDFromName(name)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func intentFromResource(res *meshresource.RuleIntentResource, id int64) *Intent {
	spec := res.Spec

	var rolledBackFromID *int64
	if spec.RolledBackFromId != 0 {
		v := spec.RolledBackFromId
		rolledBackFromID = &v
	}

	createdAt := timestampAsTime(spec.CreatedAt)

	return &Intent{
		ID:               id,
		RuleKind:         coremodel.ResourceKind(spec.ParentRuleKind),
		Mesh:             spec.ParentRuleMesh,
		ResourceKey:      coremodel.BuildResourceKey(spec.ParentRuleMesh, spec.ParentRuleName),
		RuleName:         spec.ParentRuleName,
		ContentHash:      spec.ContentHash,
		SpecJSON:         spec.SpecJson,
		Operation:        Operation(spec.Operation),
		Source:           Source(spec.Source),
		Author:           spec.Author,
		Reason:           spec.Reason,
		RolledBackFromID: rolledBackFromID,
		Status:           IntentStatus(spec.Status),
		LastError:        spec.FailureReason,
		CreatedAt:        createdAt,
	}
}

// Meta helper functions

func buildMetaName(kind coremodel.ResourceKind, resourceKey string) string {
	name := extractName(resourceKey)
	return fmt.Sprintf("%s-%s-meta", kind, name)
}

func metaFromRuleMeta(kind coremodel.ResourceKind, resourceKey string, spec *meshproto.RuleMeta) *Meta {
	meta := &Meta{
		RuleKind:      kind,
		ResourceKey:   resourceKey,
		LastVersionNo: spec.CurrentVersionNo,
	}
	if spec.UpdatedAt != nil {
		meta.UpdatedAt = spec.UpdatedAt.AsTime()
	}
	if spec.CurrentVersionId != 0 {
		currentID := spec.CurrentVersionId
		meta.CurrentVersion = &currentID
	}
	return meta
}

func ledgerSnapshotFromState(state *ledgerState) *LedgerSnapshot {
	if state == nil {
		return &LedgerSnapshot{}
	}
	snapshot := &LedgerSnapshot{
		Versions: append([]Version(nil), state.Versions...),
	}
	if len(snapshot.Versions) == 0 {
		return snapshot
	}
	head := snapshot.Versions[0]
	snapshot.Head = &head
	snapshot.Deleted = head.Operation == OperationDelete
	if !snapshot.Deleted {
		for i := range snapshot.Versions {
			snapshot.Versions[i].IsCurrent = snapshot.Versions[i].ID == head.ID
		}
	}
	return snapshot
}

func duplicateVersionNoError(kind coremodel.ResourceKind, resourceKey string, versionNo, firstID, secondID int64) error {
	return fmt.Errorf("%w: duplicate version number for kind=%s mesh=%s rule=%s versionNo=%d conflictingVersionIDs=%d,%d",
		ErrVersionLedgerCorrupt,
		kind,
		extractMesh(resourceKey),
		extractName(resourceKey),
		versionNo,
		firstID,
		secondID,
	)
}

func timestampAsTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}
