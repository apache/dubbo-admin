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

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/store/dbcommon"
)

// Ensure GormLock implements Lock interface
var _ lock.Lock = (*GormLock)(nil)

// GormLock provides distributed locking using database as backend
// It uses GORM for database operations and supports MySQL, PostgreSQL, etc.
type GormLock struct {
	pool  *dbcommon.ConnectionPool
	db    *gorm.DB // Direct DB reference to avoid circular dependency
	owner string   // Unique identifier for this lock instance
}

// NewGormLock creates a new GORM-based distributed lock instance
// Deprecated: Use NewGormLockFromDB to avoid circular dependencies
func NewGormLock(pool *dbcommon.ConnectionPool) lock.Lock {
	return &GormLock{
		pool:  pool,
		db:    pool.GetDB(),
		owner: uuid.New().String(),
	}
}

// NewGormLockFromDB creates a new GORM-based distributed lock instance from a DB connection
// This is the preferred constructor to avoid circular dependencies
func NewGormLockFromDB(db *gorm.DB) lock.Lock {
	return &GormLock{
		db:    db,
		owner: uuid.New().String(),
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

// Lock acquires a lock with the specified key and TTL
// It blocks until the lock is acquired or context is cancelled
func (g *GormLock) Lock(ctx context.Context, key string, ttl time.Duration) error {
	ticker := time.NewTicker(constants.DefaultLockRetryInterval)
	defer ticker.Stop()

	for {
		acquired, err := g.TryLock(ctx, key, ttl)
		if err != nil {
			return fmt.Errorf("failed to try lock: %w", err)
		}
		if acquired {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// TryLock attempts to acquire a lock without blocking
// Returns true if lock was acquired, false otherwise
func (g *GormLock) TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	db := g.getDB().WithContext(ctx)
	expireAt := time.Now().Add(ttl)

	var acquired bool
	err := db.Transaction(func(tx *gorm.DB) error {
		// Clean up only this key's expired lock to improve performance
		now := time.Now()
		if err := tx.Where("lock_key = ? AND expire_at < ?", key, now).
			Delete(&LockRecord{}).Error; err != nil {
			return fmt.Errorf("failed to clean expired lock for key %s: %w", key, err)
		}

		// Try to acquire lock using INSERT ... ON CONFLICT
		lock := &LockRecord{
			LockKey:  key,
			Owner:    g.owner,
			ExpireAt: expireAt,
		}

		// Try to insert the lock record
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "lock_key"}},
			DoNothing: true, // If conflict, do nothing
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
		return false, err
	}

	return acquired, nil
}

// Unlock releases a lock held by this instance
func (g *GormLock) Unlock(ctx context.Context, key string) error {
	db := g.getDB().WithContext(ctx)

	result := db.Where("lock_key = ? AND owner = ?", key, g.owner).
		Delete(&LockRecord{})

	if result.Error != nil {
		return fmt.Errorf("failed to release lock: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return bizerror.New(bizerror.LockNotHeld, "lock not held by this owner")
	}

	return nil
}

// Renew extends the TTL of a lock held by this instance
func (g *GormLock) Renew(ctx context.Context, key string, ttl time.Duration) error {
	db := g.getDB().WithContext(ctx)
	newExpireAt := time.Now().Add(ttl)

	result := db.Model(&LockRecord{}).
		Where("lock_key = ? AND owner = ?", key, g.owner).
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

// WithLock executes a function while holding a lock
func (g *GormLock) WithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
	// Acquire lock
	if err := g.Lock(ctx, key, ttl); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Ensure lock is released
	defer func() {
		// Use background context for unlock to ensure it completes even if ctx is cancelled
		unlockCtx, cancel := context.WithTimeout(context.Background(), constants.DefaultUnlockTimeout)
		defer cancel()

		if err := g.Unlock(unlockCtx, key); err != nil {
			logger.Errorf("Failed to release lock %s: %v", key, err)
		}
	}()

	// Start auto-renewal if TTL is long enough
	var renewDone chan struct{}
	if ttl > constants.DefaultAutoRenewThreshold {
		renewDone = make(chan struct{})
		go g.autoRenew(ctx, key, ttl, renewDone)
		defer close(renewDone)
	}

	// Execute the function
	return fn()
}

// autoRenew periodically renews the lock until done channel is closed
func (g *GormLock) autoRenew(ctx context.Context, key string, ttl time.Duration, done <-chan struct{}) {
	// Renew at 1/3 of TTL to ensure lock doesn't expire
	renewInterval := ttl / 3
	ticker := time.NewTicker(renewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Double-check done channel before renewing to avoid unnecessary renewal
			select {
			case <-done:
				return
			default:
			}

			renewCtx, cancel := context.WithTimeout(context.Background(), constants.DefaultRenewTimeout)
			if err := g.Renew(renewCtx, key, ttl); err != nil {
				logger.Warnf("Failed to renew lock %s: %v", key, err)
				cancel()
				return
			}
			cancel()
		}
	}
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
