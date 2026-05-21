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

type Source string

const (
	SourceAdmin     Source = "ADMIN"
	SourceUpstream  Source = "UPSTREAM"
	SourceRollback  Source = "ROLLBACK"
	SourceBootstrap Source = "BOOTSTRAP"
)

type Operation string

const (
	OperationCreate Operation = "CREATE"
	OperationUpdate Operation = "UPDATE"
	OperationDelete Operation = "DELETE"
)

var (
	ErrFeatureDisabled  = errors.New("rule versioning is disabled")
	ErrVersionConflict  = errors.New("rule version conflict")
	ErrVersionNotFound  = errors.New("rule version not found")
	ErrRollbackToDelete = errors.New("cannot roll back to a deleted rule version")
)

type Version struct {
	ID               int64                  `json:"id" gorm:"primaryKey;autoIncrement"`
	RuleKind         coremodel.ResourceKind `json:"ruleKind" gorm:"type:varchar(64);not null;index:idx_rk_key_created,priority:1;uniqueIndex:uk_rk_key_ver,priority:1;index:idx_rk_hash,priority:1"`
	Mesh             string                 `json:"mesh" gorm:"type:varchar(128);not null"`
	ResourceKey      string                 `json:"resourceKey" gorm:"type:varchar(512);not null;index:idx_rk_key_created,priority:2;uniqueIndex:uk_rk_key_ver,priority:2"`
	RuleName         string                 `json:"ruleName" gorm:"type:varchar(256);not null"`
	VersionNo        int64                  `json:"versionNo" gorm:"not null;uniqueIndex:uk_rk_key_ver,priority:3"`
	ContentHash      string                 `json:"contentHash" gorm:"type:char(64);not null;index:idx_rk_hash,priority:2"`
	SpecJSON         string                 `json:"specJson" gorm:"type:text;not null"`
	Source           Source                 `json:"source" gorm:"type:varchar(16);not null"`
	Operation        Operation              `json:"operation" gorm:"type:varchar(16);not null"`
	Author           string                 `json:"author" gorm:"type:varchar(128);not null"`
	Reason           string                 `json:"reason,omitempty" gorm:"type:varchar(1024)"`
	RolledBackFromID *int64                 `json:"rolledBackFromId,omitempty"`
	CreatedAt        time.Time              `json:"createdAt" gorm:"not null;index:idx_rk_key_created,priority:3,sort:desc"`
	IsCurrent        bool                   `json:"isCurrent" gorm:"-"`
}

func (Version) TableName() string {
	return "rule_version"
}

type Meta struct {
	RuleKind       coremodel.ResourceKind `json:"ruleKind" gorm:"type:varchar(64);primaryKey"`
	ResourceKey    string                 `json:"resourceKey" gorm:"type:varchar(512);primaryKey"`
	CurrentVersion *int64                 `json:"currentVersion"`
	LastVersionNo  int64                  `json:"lastVersionNo" gorm:"not null;default:0"`
	UpdatedAt      time.Time              `json:"updatedAt" gorm:"not null"`
}

func (Meta) TableName() string {
	return "rule_version_meta"
}

type InsertRequest struct {
	RuleKind         coremodel.ResourceKind
	Mesh             string
	ResourceKey      string
	RuleName         string
	SpecJSON         string
	ContentHash      string
	Source           Source
	Operation        Operation
	Author           string
	Reason           string
	RolledBackFromID *int64
	CreatedAt        time.Time
}

type ListResult struct {
	Items []Version `json:"items"`
	Total int64     `json:"total"`
}

type DiffResult struct {
	Left  DiffSide `json:"left"`
	Right DiffSide `json:"right"`
}

type DiffSide struct {
	ID        int64  `json:"id"`
	VersionNo int64  `json:"versionNo"`
	SpecJSON  string `json:"specJson"`
}

type ConflictError struct {
	CurrentVersionID *int64
}

func (e *ConflictError) Error() string {
	return ErrVersionConflict.Error()
}
