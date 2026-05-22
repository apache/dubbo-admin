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
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type GormStore struct {
	db *gorm.DB
	mu sync.Mutex
}

func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

func (s *GormStore) AutoMigrate() error {
	return s.db.AutoMigrate(&Version{}, &Meta{})
}

func (s *GormStore) InsertVersion(req InsertRequest, maxVersions int64) (*Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var inserted Version
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var meta Meta
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("rule_kind = ? AND resource_key = ?", req.RuleKind, req.ResourceKey).
			First(&meta).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			meta = Meta{RuleKind: req.RuleKind, ResourceKey: req.ResourceKey}
			if err := tx.Create(&meta).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		var latest Version
		err = tx.Where("rule_kind = ? AND resource_key = ?", req.RuleKind, req.ResourceKey).
			Order("version_no DESC").
			First(&latest).Error
		if err == nil && shouldDedupVersion(&latest, req) {
			inserted = latest
			return nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		meta.LastVersionNo++
		inserted = Version{
			RuleKind:         req.RuleKind,
			Mesh:             req.Mesh,
			ResourceKey:      req.ResourceKey,
			RuleName:         req.RuleName,
			VersionNo:        meta.LastVersionNo,
			ContentHash:      req.ContentHash,
			SpecJSON:         req.SpecJSON,
			Source:           req.Source,
			Operation:        req.Operation,
			Author:           req.Author,
			Reason:           req.Reason,
			RolledBackFromID: req.RolledBackFromID,
			CreatedAt:        req.CreatedAt,
		}
		if err := tx.Create(&inserted).Error; err != nil {
			return err
		}
		if req.Operation == OperationDelete {
			meta.CurrentVersion = nil
		} else {
			current := inserted.ID
			meta.CurrentVersion = &current
		}
		meta.UpdatedAt = req.CreatedAt
		if err := tx.Save(&meta).Error; err != nil {
			return err
		}
		return trimGorm(tx, req.RuleKind, req.ResourceKey, maxVersions)
	})
	if err != nil {
		return nil, err
	}
	meta, err := s.CurrentMeta(inserted.RuleKind, inserted.ResourceKey)
	if err == nil && meta != nil && meta.CurrentVersion != nil {
		inserted.IsCurrent = *meta.CurrentVersion == inserted.ID
	}
	return &inserted, nil
}

func (s *GormStore) ListVersions(kind coremodel.ResourceKind, resourceKey string) ([]Version, error) {
	var items []Version
	if err := s.db.Where("rule_kind = ? AND resource_key = ?", kind, resourceKey).
		Order("version_no DESC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	meta, err := s.CurrentMeta(kind, resourceKey)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].IsCurrent = meta != nil && meta.CurrentVersion != nil && *meta.CurrentVersion == items[i].ID
	}
	return items, nil
}

func (s *GormStore) GetVersion(kind coremodel.ResourceKind, resourceKey string, id int64) (*Version, error) {
	var v Version
	err := s.db.Where("id = ? AND rule_kind = ? AND resource_key = ?", id, kind, resourceKey).First(&v).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrVersionNotFound
	}
	if err != nil {
		return nil, err
	}
	meta, err := s.CurrentMeta(kind, resourceKey)
	if err != nil {
		return nil, err
	}
	v.IsCurrent = meta != nil && meta.CurrentVersion != nil && *meta.CurrentVersion == v.ID
	return &v, nil
}

func (s *GormStore) GetVersionByID(id int64) (*Version, error) {
	var v Version
	err := s.db.Where("id = ?", id).First(&v).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrVersionNotFound
	}
	if err != nil {
		return nil, err
	}
	meta, err := s.CurrentMeta(v.RuleKind, v.ResourceKey)
	if err != nil {
		return nil, err
	}
	v.IsCurrent = meta != nil && meta.CurrentVersion != nil && *meta.CurrentVersion == v.ID
	return &v, nil
}

func (s *GormStore) CurrentMeta(kind coremodel.ResourceKind, resourceKey string) (*Meta, error) {
	var meta Meta
	err := s.db.Where("rule_kind = ? AND resource_key = ?", kind, resourceKey).First(&meta).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *GormStore) LatestVersion(kind coremodel.ResourceKind, resourceKey string) (*Version, error) {
	var v Version
	err := s.db.Where("rule_kind = ? AND resource_key = ?", kind, resourceKey).
		Order("version_no DESC").
		First(&v).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *GormStore) CheckExpectedVersion(kind coremodel.ResourceKind, resourceKey string, expected *int64) error {
	if expected == nil {
		return nil
	}
	meta, err := s.CurrentMeta(kind, resourceKey)
	if err != nil {
		return err
	}
	if meta == nil || meta.CurrentVersion == nil || *meta.CurrentVersion != *expected {
		var current *int64
		if meta != nil {
			current = meta.CurrentVersion
		}
		return &ConflictError{CurrentVersionID: current}
	}
	return nil
}

func trimGorm(tx *gorm.DB, kind coremodel.ResourceKind, resourceKey string, maxVersions int64) error {
	if maxVersions <= 0 {
		return nil
	}
	var keepIDs []int64
	if err := tx.Model(&Version{}).
		Where("rule_kind = ? AND resource_key = ?", kind, resourceKey).
		Order("version_no DESC").
		Limit(int(maxVersions)).
		Pluck("id", &keepIDs).Error; err != nil {
		return err
	}
	if len(keepIDs) == 0 {
		return nil
	}
	return tx.Where("rule_kind = ? AND resource_key = ? AND id NOT IN ?", kind, resourceKey, keepIDs).
		Delete(&Version{}).Error
}
