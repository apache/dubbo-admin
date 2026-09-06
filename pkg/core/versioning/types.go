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
	"errors"
	"time"

	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// Source identifies where a rule mutation originated.
// Used for auditing and distinguishing user-initiated changes from system-generated ones.
type Source string

const (
	SourceAdmin Source = "ADMIN" // User edit via Admin UI/API
	// SourceUpstream marks a version imported from an upstream registry event.
	SourceUpstream Source = "UPSTREAM"
	// SourceBootstrap represents a baseline record for an existing rule when it
	// is first brought under version tracking.
	SourceBootstrap Source = "BOOTSTRAP"
	// SourceRollback marks a version produced by re-publishing a historical
	// snapshot. Rollback records a new version and does not rewrite history.
	SourceRollback Source = "ROLLBACK"
)

type Operation string

const (
	OperationCreate Operation = "CREATE"
	OperationUpdate Operation = "UPDATE"
	OperationDelete Operation = "DELETE"
)

var (
	ErrVersionNotFound   = errors.New("rule version not found")
	ErrVersionStoreError = errors.New("rule version store error")
	ErrRollbackToDelete  = errors.New("cannot roll back to a DELETE marker")
	ErrRollbackToCurrent = errors.New("cannot roll back to a version identical to current")
)

// Version represents a snapshot of a rule's spec at a point in time. Version
// entries are immutable after creation. Rollback appends a new version, while
// retention may delete the oldest entries. IsLatestRecorded is derived from
// history only; it is not proof that this snapshot equals the live rule.
type Version struct {
	RuleKind    coremodel.ResourceKind `json:"ruleKind"`
	Mesh        string                 `json:"mesh"`
	ResourceKey string                 `json:"resourceKey"`
	RuleName    string                 `json:"ruleName"`
	VersionNo   int64                  `json:"versionNo"`
	ContentHash string                 `json:"contentHash"`
	SpecJSON    string                 `json:"specJson"`
	Source      Source                 `json:"source"`
	Operation   Operation              `json:"operation"`
	Author      string                 `json:"author"`
	Reason      string                 `json:"reason,omitempty"`
	// RolledBackFromVersionNo records the historical version whose snapshot was
	// re-published to produce this version. It is audit metadata only.
	RolledBackFromVersionNo *int64    `json:"rolledBackFromVersionNo,omitempty"`
	CreatedAt               time.Time `json:"createdAt"`
	RecordedAt              time.Time `json:"recordedAt"`
	IsLatestRecorded        bool      `json:"isLatestRecorded"`
}

type historyState struct {
	Versions     []Version
	Latest       *Version
	MaxVersionNo int64
}

type HistorySnapshot struct {
	Versions []Version
	Head     *Version
	Deleted  bool
}

type InsertRequest struct {
	RuleKind                coremodel.ResourceKind
	Mesh                    string
	ResourceKey             string
	RuleName                string
	SpecJSON                string
	ContentHash             string
	Source                  Source
	Operation               Operation
	Author                  string
	Reason                  string
	RolledBackFromVersionNo *int64
	CreatedAt               time.Time
}

type ListResult struct {
	Items                   []Version `json:"items"`
	Total                   int64     `json:"total"`
	LatestRecordedVersionNo int64     `json:"latestRecordedVersionNo,omitempty"`
	LatestRecordedDeleted   bool      `json:"latestRecordedDeleted"`
}

type DiffResult struct {
	Left  DiffSide `json:"left"`
	Right DiffSide `json:"right"`
}

type DiffSide struct {
	VersionNo int64  `json:"versionNo"`
	SpecJSON  string `json:"specJson"`
}
