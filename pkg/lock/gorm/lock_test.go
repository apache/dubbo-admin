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

package gorm_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	gormlock "github.com/apache/dubbo-admin/pkg/lock/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		PrepareStmt: false,
	})
	require.NoError(t, err, "failed to create test database")

	sqlDB, err := db.DB()
	require.NoError(t, err)

	sqlDB.SetMaxOpenConns(1)

	err = db.Exec("PRAGMA journal_mode=WAL;").Error
	require.NoError(t, err, "failed to set WAL mode")

	err = db.Exec("PRAGMA busy_timeout=5000;").Error
	require.NoError(t, err, "failed to set busy timeout")

	err = db.AutoMigrate(&gormlock.LockRecord{})
	require.NoError(t, err, "failed to migrate lock table")

	return db
}

func TestBasicLockUnlock(t *testing.T) {
	db := setupTestDB(t)
	lockInstance := gormlock.NewGormLockFromDB(db)
	ctx := context.Background()

	err := lockInstance.Lock(ctx, "test-key", 5*time.Second)
	assert.NoError(t, err, "should acquire lock successfully")

	isLocked, err := lockInstance.IsLocked(ctx, "test-key")
	assert.NoError(t, err)
	assert.True(t, isLocked, "lock should be held")

	err = lockInstance.Unlock(ctx, "test-key")
	assert.NoError(t, err, "should release lock successfully")

	isLocked, err = lockInstance.IsLocked(ctx, "test-key")
	assert.NoError(t, err)
	assert.False(t, isLocked, "lock should be released")
}

func TestTryLock(t *testing.T) {
	db := setupTestDB(t)
	lock1 := gormlock.NewGormLockFromDB(db)
	lock2 := gormlock.NewGormLockFromDB(db)
	ctx := context.Background()

	acquired, err := lock1.TryLock(ctx, "test-key", 5*time.Second)
	assert.NoError(t, err)
	assert.True(t, acquired, "first lock should be acquired")

	acquired, err = lock2.TryLock(ctx, "test-key", 5*time.Second)
	assert.NoError(t, err)
	assert.False(t, acquired, "second lock should not be acquired")

	err = lock1.Unlock(ctx, "test-key")
	assert.NoError(t, err)

	acquired, err = lock2.TryLock(ctx, "test-key", 5*time.Second)
	assert.NoError(t, err)
	assert.True(t, acquired, "second lock should be acquired after first is released")

	_ = lock2.Unlock(ctx, "test-key")
}

func TestConcurrentLockAttempts(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	const numGoroutines = 10
	var successCount atomic.Int32
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			lockInstance := gormlock.NewGormLockFromDB(db)
			acquired, err := lockInstance.TryLock(ctx, "concurrent-key", 1*time.Second)
			if err == nil && acquired {
				successCount.Add(1)
				time.Sleep(100 * time.Millisecond) // Hold lock briefly
				_ = lockInstance.Unlock(ctx, "concurrent-key")
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(1), successCount.Load(), "only one goroutine should acquire the lock")
}

func TestLockExpiration(t *testing.T) {
	db := setupTestDB(t)
	lock1 := gormlock.NewGormLockFromDB(db)
	lock2 := gormlock.NewGormLockFromDB(db)
	ctx := context.Background()

	acquired, err := lock1.TryLock(ctx, "expire-key", 100*time.Millisecond)
	assert.NoError(t, err)
	assert.True(t, acquired)

	acquired, err = lock2.TryLock(ctx, "expire-key", 1*time.Second)
	assert.NoError(t, err)
	assert.False(t, acquired, "lock should still be held")

	time.Sleep(200 * time.Millisecond)

	acquired, err = lock2.TryLock(ctx, "expire-key", 1*time.Second)
	assert.NoError(t, err)
	assert.True(t, acquired, "lock should be acquired after expiration")

	_ = lock2.Unlock(ctx, "expire-key")
}

func TestLockRenewal(t *testing.T) {
	db := setupTestDB(t)
	lockInstance := gormlock.NewGormLockFromDB(db)
	ctx := context.Background()

	err := lockInstance.Lock(ctx, "renew-key", 1*time.Second)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	err = lockInstance.Renew(ctx, "renew-key", 2*time.Second)
	assert.NoError(t, err, "should renew lock successfully")

	isLocked, err := lockInstance.IsLocked(ctx, "renew-key")
	assert.NoError(t, err)
	assert.True(t, isLocked, "lock should still be held after renewal")

	_ = lockInstance.Unlock(ctx, "renew-key")
}

func TestUnlockNotHeld(t *testing.T) {
	db := setupTestDB(t)
	lock1 := gormlock.NewGormLockFromDB(db)
	lock2 := gormlock.NewGormLockFromDB(db)
	ctx := context.Background()

	err := lock1.Lock(ctx, "test-key", 5*time.Second)
	require.NoError(t, err)

	err = lock2.Unlock(ctx, "test-key")
	assert.Error(t, err, "should return error")

	// 检查错误类型和错误码
	var bizErr bizerror.Error
	if assert.ErrorAs(t, err, &bizErr) {
		assert.Equal(t, bizerror.LockNotHeld, bizErr.Code(), "should return LockNotHeld error code")
	}

	_ = lock1.Unlock(ctx, "test-key")
}

func TestRenewNotHeld(t *testing.T) {
	db := setupTestDB(t)
	lock1 := gormlock.NewGormLockFromDB(db)
	lock2 := gormlock.NewGormLockFromDB(db)
	ctx := context.Background()

	err := lock1.Lock(ctx, "test-key", 5*time.Second)
	require.NoError(t, err)

	err = lock2.Renew(ctx, "test-key", 10*time.Second)
	assert.Error(t, err, "should return error")

	var bizErr bizerror.Error
	if assert.ErrorAs(t, err, &bizErr) {
		assert.Equal(t, bizerror.LockNotHeld, bizErr.Code(), "should return LockNotHeld error code")
	}

	_ = lock1.Unlock(ctx, "test-key")
}

func TestWithLock(t *testing.T) {
	db := setupTestDB(t)
	lockInstance := gormlock.NewGormLockFromDB(db)
	ctx := context.Background()

	executed := false
	err := lockInstance.WithLock(ctx, "with-lock-key", 2*time.Second, func() error {
		executed = true
		isLocked, err := lockInstance.IsLocked(ctx, "with-lock-key")
		assert.NoError(t, err)
		assert.True(t, isLocked)
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, executed, "function should be executed")

	time.Sleep(100 * time.Millisecond)
	isLocked, err := lockInstance.IsLocked(ctx, "with-lock-key")
	assert.NoError(t, err)
	assert.False(t, isLocked, "lock should be released after WithLock")
}

func TestWithLockAutoRenewal(t *testing.T) {
	db := setupTestDB(t)
	lockInstance := gormlock.NewGormLockFromDB(db)
	ctx := context.Background()

	executed := false
	err := lockInstance.WithLock(ctx, "auto-renew-key", 15*time.Second, func() error {
		time.Sleep(6 * time.Second)
		executed = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, executed, "function should be executed")

	time.Sleep(100 * time.Millisecond)
	isLocked, err := lockInstance.IsLocked(ctx, "auto-renew-key")
	assert.NoError(t, err)
	assert.False(t, isLocked, "lock should be released after WithLock")
}

func TestWithLockContextCancellation(t *testing.T) {
	db := setupTestDB(t)
	lockInstance := gormlock.NewGormLockFromDB(db)

	ctx, cancel := context.WithCancel(context.Background())

	started := make(chan struct{})
	err := lockInstance.WithLock(ctx, "cancel-key", 5*time.Second, func() error {
		close(started)
		cancel()
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	<-started

	assert.NoError(t, err, "function should complete even if context is cancelled during execution")

	time.Sleep(100 * time.Millisecond)
	isLocked, err := lockInstance.IsLocked(context.Background(), "cancel-key")
	assert.NoError(t, err)
	assert.False(t, isLocked, "lock should be released even after context cancellation")
}

func TestCleanupExpiredLocks(t *testing.T) {
	db := setupTestDB(t)
	lock1 := gormlock.NewGormLockFromDB(db)
	lock2 := gormlock.NewGormLockFromDB(db)
	ctx := context.Background()

	_, _ = lock1.TryLock(ctx, "cleanup-key-1", 100*time.Millisecond)
	_, _ = lock2.TryLock(ctx, "cleanup-key-2", 100*time.Millisecond)

	time.Sleep(200 * time.Millisecond)

	err := lock1.CleanupExpiredLocks(ctx)
	assert.NoError(t, err)

	var count int64
	db.Model(&gormlock.LockRecord{}).Count(&count)
	assert.Equal(t, int64(0), count, "all expired locks should be cleaned up")
}

func TestMultipleDifferentLocks(t *testing.T) {
	db := setupTestDB(t)
	lockInstance := gormlock.NewGormLockFromDB(db)
	ctx := context.Background()

	err1 := lockInstance.Lock(ctx, "key-1", 5*time.Second)
	err2 := lockInstance.Lock(ctx, "key-2", 5*time.Second)
	err3 := lockInstance.Lock(ctx, "key-3", 5*time.Second)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)

	isLocked1, _ := lockInstance.IsLocked(ctx, "key-1")
	isLocked2, _ := lockInstance.IsLocked(ctx, "key-2")
	isLocked3, _ := lockInstance.IsLocked(ctx, "key-3")

	assert.True(t, isLocked1)
	assert.True(t, isLocked2)
	assert.True(t, isLocked3)

	_ = lockInstance.Unlock(ctx, "key-1")
	_ = lockInstance.Unlock(ctx, "key-2")
	_ = lockInstance.Unlock(ctx, "key-3")
}

func TestLockBlockingBehavior(t *testing.T) {
	db := setupTestDB(t)
	lock1 := gormlock.NewGormLockFromDB(db)
	lock2 := gormlock.NewGormLockFromDB(db)
	ctx := context.Background()

	err := lock1.Lock(ctx, "blocking-key", 10*time.Second)
	require.NoError(t, err)

	isLocked, err := lock1.IsLocked(ctx, "blocking-key")
	require.NoError(t, err)
	require.True(t, isLocked)

	acquiredTime := time.Now()
	done := make(chan time.Time)

	go func() {
		_ = lock2.Lock(ctx, "blocking-key", 10*time.Second)
		done <- time.Now()
	}()

	time.Sleep(500 * time.Millisecond)

	unlockErr := lock1.Unlock(ctx, "blocking-key")
	require.NoError(t, unlockErr, "unlock should succeed")

	isLocked, err = lock1.IsLocked(ctx, "blocking-key")
	require.NoError(t, err)

	lock2AcquiredTime := <-done

	duration := lock2AcquiredTime.Sub(acquiredTime)

	assert.GreaterOrEqual(t, duration, 500*time.Millisecond, "lock2 should acquire after lock1 releases")
	assert.Less(t, duration, 1500*time.Millisecond, "lock2 should acquire shortly after lock1 releases")

	_ = lock2.Unlock(ctx, "blocking-key")
}
