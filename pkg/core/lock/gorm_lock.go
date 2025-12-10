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

package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/store/dbcommon"
)

// Ensure GormLock implements Lock interface
var _ Lock = (*GormLock)(nil)

// GormLock provides distributed locking using database as backend
// It uses GORM for database operations and supports MySQL, PostgreSQL, etc.
type GormLock struct {
	pool  *dbcommon.ConnectionPool
	owner string // Unique identifier for this lock instance
}

// NewGormLock creates a new GORM-based distributed lock instance
func NewGormLock(pool *dbcommon.ConnectionPool) Lock {
	return &GormLock{
		pool:  pool,
		owner: uuid.New().String(),
	}
}

// Lock acquires a lock with the specified key and TTL
// It blocks until the lock is acquired or context is cancelled
func (g *GormLock) Lock(ctx context.Context, key string, ttl time.Duration) error {
	ticker := time.NewTicker(100 * time.Millisecond)
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
	db := g.pool.GetDB().WithContext(ctx)
	expireAt := time.Now().Add(ttl)

	var acquired bool
	err := db.Transaction(func(tx *gorm.DB) error {
		// Clean up only this key's expired lock to improve performance
		now := time.Now()
		if err := tx.Where("lock_key = ? AND expire_at < ?", key, now).
			Delete(&LockRecord{}).Error; err != nil {
			return fmt.Errorf("failed to clean expired lock for key %s: %w", key, err)
		}

		// Try to acquire lock using INSERT ...  ON CONFLICT
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
			return fmt.Errorf("failed to insert lock record:  %w", result.Error)
		}

		// Check if we got the lock by verifying the owner
		var existingLock LockRecord
		if err := tx.Where("lock_key = ? ", key).First(&existingLock).Error; err != nil {
			return fmt.Errorf("failed to verify lock ownership:  %w", err)
		}

		// Determine if we acquired the lock
		acquired = existingLock.Owner == g.owner
		return nil
	})

	if err != nil {
		return false, err
	}

	if acquired {
		logger.Debugf("Lock acquired: key=%s, owner=%s, ttl=%v", key, g.owner, ttl)
	}

	return acquired, nil
}

// Unlock releases a lock held by this instance
func (g *GormLock) Unlock(ctx context.Context, key string) error {
	db := g.pool.GetDB().WithContext(ctx)

	result := db.Where("lock_key = ? AND owner = ?", key, g.owner).
		Delete(&LockRecord{})

	if result.Error != nil {
		return fmt.Errorf("failed to release lock: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrLockNotHeld
	}

	logger.Debugf("Lock released: key=%s, owner=%s", key, g.owner)
	return nil
}

// Renew extends the TTL of a lock held by this instance
func (g *GormLock) Renew(ctx context.Context, key string, ttl time.Duration) error {
	db := g.pool.GetDB().WithContext(ctx)
	newExpireAt := time.Now().Add(ttl)

	result := db.Model(&LockRecord{}).
		Where("lock_key = ? AND owner = ?", key, g.owner).
		Update("expire_at", newExpireAt)

	if result.Error != nil {
		return fmt.Errorf("failed to renew lock: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrLockNotHeld
	}

	logger.Debugf("Lock renewed: key=%s, owner=%s, new_expire_at=%v", key, g.owner, newExpireAt)
	return nil
}

// IsLocked checks if a lock is currently held (by anyone)
func (g *GormLock) IsLocked(ctx context.Context, key string) (bool, error) {
	db := g.pool.GetDB().WithContext(ctx)

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
// It automatically acquires the lock, executes the function, and releases the lock
// If TTL is longer than 10 seconds, it will automatically renew the lock until the function completes
func (g *GormLock) WithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
	// Acquire lock
	if err := g.Lock(ctx, key, ttl); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Ensure lock is released
	defer func() {
		// Use background context for unlock to ensure it completes even if ctx is cancelled
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := g.Unlock(unlockCtx, key); err != nil {
			logger.Errorf("Failed to release lock %s: %v", key, err)
		}
	}()

	// Start auto-renewal if TTL is long enough
	var renewDone chan struct{}
	if ttl > 10*time.Second {
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

	logger.Debugf("Auto-renewal started for lock %s (interval: %v)", key, renewInterval)

	for {
		select {
		case <-done:
			logger.Debugf("Auto-renewal stopped for lock %s (done signal)", key)
			return
		case <-ctx.Done():
			logger.Debugf("Auto-renewal stopped for lock %s (context cancelled)", key)
			return
		case <-ticker.C:
			renewCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := g.Renew(renewCtx, key, ttl); err != nil {
				logger.Warnf("Failed to renew lock %s: %v", key, err)
				cancel()
				return
			}
			cancel()
			logger.Debugf("Lock %s renewed successfully", key)
		}
	}
}

// CleanupExpiredLocks removes all expired locks from the database
// This should be called periodically as a maintenance task
func (g *GormLock) CleanupExpiredLocks(ctx context.Context) error {
	db := g.pool.GetDB().WithContext(ctx)

	result := db.Where("expire_at < ?", time.Now()).Delete(&LockRecord{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup expired locks: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		logger.Infof("Cleaned up %d expired locks", result.RowsAffected)
	}

	return nil
}
