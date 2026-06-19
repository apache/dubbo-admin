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

package gorm

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/store/dbcommon"
)

// Ensure GormLock implements Lock interface
var _ lock.Lock = (*GormLock)(nil)

// GormLock provides distributed locking using database as backend
// It uses GORM for database operations and supports MySQL, PostgreSQL, etc.
type GormLock struct {
	pool *dbcommon.ConnectionPool
	db   *gorm.DB
}

// NewGormLock creates a new GORM-based distributed lock instance
// Deprecated: Use NewGormLockFromDB to avoid circular dependencies
func NewGormLock(pool *dbcommon.ConnectionPool) lock.Lock {
	return &GormLock{
		pool: pool,
		db:   pool.GetDB(),
	}
}

// NewGormLockFromDB creates a new GORM-based distributed lock instance from a DB connection
// This is the preferred constructor to avoid circular dependencies
func NewGormLockFromDB(db *gorm.DB) lock.Lock {
	return &GormLock{
		db: db,
	}
}

// getDB returns the database instance, to prefer direct DB to pool
func (g *GormLock) getDB() *gorm.DB {
	if g.db != nil {
		return g.db
	}
	if g.pool != nil {
		return g.pool.GetDB()
	}
	return nil
}

type lease struct {
	*lock.LeaseState
	lock *GormLock
}

func (g *GormLock) Acquire(ctx context.Context, key string, ttl time.Duration) (lock.Lease, error) {
	ticker := time.NewTicker(constants.DefaultLockRetryInterval)
	defer ticker.Stop()

	for {
		lease, acquired, err := g.TryAcquire(ctx, key, ttl)
		if err != nil {
			return nil, fmt.Errorf("failed to try lock: %w", err)
		}
		if acquired {
			return lease, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (g *GormLock) TryAcquire(ctx context.Context, key string, ttl time.Duration) (lock.Lease, bool, error) {
	db := g.getDB().WithContext(ctx)
	token, err := lock.NewLeaseToken()
	if err != nil {
		return nil, false, err
	}
	expireAt := time.Now().Add(ttl)

	var acquired bool
	err = db.Transaction(func(tx *gorm.DB) error {
		// Clean up only this key's expired lock to improve performance
		now := time.Now()
		if err := tx.Where("lock_key = ? AND expire_at <= ?", key, now).
			Delete(&LockRecord{}).Error; err != nil {
			return fmt.Errorf("failed to clean expired lock for key %s: %w", key, err)
		}

		lock := &LockRecord{
			LockKey:  key,
			Owner:    token,
			ExpireAt: expireAt,
		}

		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "lock_key"}},
			DoNothing: true,
		}).Create(lock)

		if result.Error != nil {
			return fmt.Errorf("failed to insert lock record: %w", result.Error)
		}

		// Check if the insertion was successful
		if result.RowsAffected == 0 {
			// The lock already exists
			acquired = false
			return nil
		}

		// New row inserted successfully, lock acquired successfully
		acquired = true
		return nil
	})

	if err != nil {
		return nil, false, err
	}

	if !acquired {
		return nil, false, nil
	}
	return &lease{
		LeaseState: lock.NewLeaseState(key, token),
		lock:       g,
	}, true, nil
}

func (l *lease) Unlock(ctx context.Context) error {
	db := l.lock.getDB().WithContext(ctx)

	result := db.Where("lock_key = ? AND owner = ?", l.Key(), l.Token()).
		Delete(&LockRecord{})

	if result.Error != nil {
		return fmt.Errorf("failed to release lock: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return bizerror.New(bizerror.LockNotHeld, "lock not held by this owner")
	}

	return nil
}

func (l *lease) Renew(ctx context.Context, ttl time.Duration) error {
	db := l.lock.getDB().WithContext(ctx)
	now := time.Now()
	newExpireAt := time.Now().Add(ttl)

	result := db.Model(&LockRecord{}).
		Where("lock_key = ? AND owner = ? AND expire_at > ?", l.Key(), l.Token(), now).
		Update("expire_at", newExpireAt)

	if result.Error != nil {
		return fmt.Errorf("failed to renew lock: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return bizerror.New(bizerror.LockNotHeld, "lock not held by this owner")
	}

	return nil
}

// IsLocked checks if a lock is currently held (by anyone)
func (g *GormLock) IsLocked(ctx context.Context, key string) (bool, error) {
	db := g.getDB().WithContext(ctx)

	var count int64
	err := db.Model(&LockRecord{}).
		Where("lock_key = ? AND expire_at > ?", key, time.Now()).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check lock status: %w", err)
	}

	return count > 0, nil
}

// CleanupExpiredLocks removes all expired locks from the database
// This should be called periodically as a maintenance task
func (g *GormLock) CleanupExpiredLocks(ctx context.Context) error {
	db := g.getDB().WithContext(ctx)

	result := db.Where("expire_at < ?", time.Now()).Delete(&LockRecord{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup expired locks: %w", result.Error)
	}

	return nil
}
