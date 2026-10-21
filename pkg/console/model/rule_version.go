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

package model

import (
	"time"

	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
)

type RollbackRuleVersionReq struct {
	Reason string `json:"reason"`
}

type RuleVersionResp struct {
	RuleKind                coremodel.ResourceKind `json:"ruleKind"`
	Mesh                    string                 `json:"mesh"`
	ResourceKey             string                 `json:"resourceKey"`
	RuleName                string                 `json:"ruleName"`
	VersionNo               int64                  `json:"versionNo"`
	ContentHash             string                 `json:"contentHash"`
	SpecJSON                string                 `json:"specJson"`
	Source                  versioning.Source      `json:"source"`
	Operation               versioning.Operation   `json:"operation"`
	Author                  string                 `json:"author"`
	Reason                  string                 `json:"reason,omitempty"`
	RolledBackFromVersionNo *int64                 `json:"rolledBackFromVersionNo,omitempty"`
	CreatedAt               time.Time              `json:"createdAt"`
	RecordedAt              time.Time              `json:"recordedAt"`
	IsLatestRecorded        bool                   `json:"isLatestRecorded"`
}

type RuleVersionListResp struct {
	Items                   []RuleVersionResp `json:"items"`
	Total                   int64             `json:"total"`
	LatestRecordedVersionNo int64             `json:"latestRecordedVersionNo,omitempty"`
	LatestRecordedDeleted   bool              `json:"latestRecordedDeleted"`
}

type RuleVersionDiffResp struct {
	Left  RuleVersionDiffSideResp `json:"left"`
	Right RuleVersionDiffSideResp `json:"right"`
}

type RuleVersionDiffSideResp struct {
	VersionNo int64  `json:"versionNo"`
	SpecJSON  string `json:"specJson"`
}

type RollbackRuleVersionResp struct {
	RolledBackFromVersionNo int64  `json:"rolledBackFromVersionNo"`
	VersionNo               int64  `json:"versionNo"`
	Source                  string `json:"source"`
}
