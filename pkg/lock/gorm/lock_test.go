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
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	corelock "github.com/apache/dubbo-admin/pkg/core/lock"
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

	lease, err := lockInstance.Acquire(ctx, "test-key", 5*time.Second)
	assert.NoError(t, err, "should acquire lock successfully")

	isLocked, err := lockInstance.IsLocked(ctx, "test-key")
	assert.NoError(t, err)
	assert.True(t, isLocked, "lock should be held")

	err = lease.Unlock(ctx)
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

	lease1, acquired, err := lock1.TryAcquire(ctx, "test-key", 5*time.Second)
	assert.NoError(t, err)
	assert.True(t, acquired, "first lock should be acquired")

	_, acquired, err = lock2.TryAcquire(ctx, "test-key", 5*time.Second)
	assert.NoError(t, err)
	assert.False(t, acquired, "second lock should not be acquired")

	err = lease1.Unlock(ctx)
	assert.NoError(t, err)

	lease2, acquired, err := lock2.TryAcquire(ctx, "test-key", 5*time.Second)
	assert.NoError(t, err)
	assert.True(t, acquired, "second lock should be acquired after first is released")

	_ = lease2.Unlock(ctx)
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
			lease, acquired, err := lockInstance.TryAcquire(ctx, "concurrent-key", 1*time.Second)
			if err == nil && acquired {
				successCount.Add(1)
				time.Sleep(100 * time.Millisecond) // Hold lock briefly
				_ = lease.Unlock(ctx)
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

	_, acquired, err := lock1.TryAcquire(ctx, "expire-key", 100*time.Millisecond)
	assert.NoError(t, err)
	assert.True(t, acquired)

	_, acquired, err = lock2.TryAcquire(ctx, "expire-key", 1*time.Second)
	assert.NoError(t, err)
	assert.False(t, acquired, "lock should still be held")

	time.Sleep(200 * time.Millisecond)

	lease2, acquired, err := lock2.TryAcquire(ctx, "expire-key", 1*time.Second)
	assert.NoError(t, err)
	assert.True(t, acquired, "lock should be acquired after expiration")

	_ = lease2.Unlock(ctx)
}

func TestLockRenewal(t *testing.T) {
	db := setupTestDB(t)
	lockInstance := gormlock.NewGormLockFromDB(db)
	ctx := context.Background()

	lease, err := lockInstance.Acquire(ctx, "renew-key", 1*time.Second)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	err = lease.Renew(ctx, 2*time.Second)
	assert.NoError(t, err, "should renew lock successfully")

	isLocked, err := lockInstance.IsLocked(ctx, "renew-key")
	assert.NoError(t, err)
	assert.True(t, isLocked, "lock should still be held after renewal")

	_ = lease.Unlock(ctx)
}

func TestUnlockNotHeld(t *testing.T) {
	db := setupTestDB(t)
	lock1 := gormlock.NewGormLockFromDB(db)
	lock2 := gormlock.NewGormLockFromDB(db)
	ctx := context.Background()

	lease1, err := lock1.Acquire(ctx, "test-key", 20*time.Millisecond)
	require.NoError(t, err)
	time.Sleep(30 * time.Millisecond)

	lease2, acquired, err := lock2.TryAcquire(ctx, "test-key", 5*time.Second)
	require.NoError(t, err)
	require.True(t, acquired)
	err = lease1.Unlock(ctx)
	assert.Error(t, err, "should return error")

	// 检查错误类型和错误码
	var bizErr bizerror.Error
	if assert.ErrorAs(t, err, &bizErr) {
		assert.Equal(t, bizerror.LockNotHeld, bizErr.Code(), "should return LockNotHeld error code")
	}

	_ = lease2.Unlock(ctx)
}

func TestRenewNotHeld(t *testing.T) {
	db := setupTestDB(t)
	lock1 := gormlock.NewGormLockFromDB(db)
	lock2 := gormlock.NewGormLockFromDB(db)
	ctx := context.Background()

	lease1, err := lock1.Acquire(ctx, "test-key", 20*time.Millisecond)
	require.NoError(t, err)
	time.Sleep(30 * time.Millisecond)

	lease2, acquired, err := lock2.TryAcquire(ctx, "test-key", 5*time.Second)
	require.NoError(t, err)
	require.True(t, acquired)
	err = lease1.Renew(ctx, 10*time.Second)
	assert.Error(t, err, "should return error")

	var bizErr bizerror.Error
	if assert.ErrorAs(t, err, &bizErr) {
		assert.Equal(t, bizerror.LockNotHeld, bizErr.Code(), "should return LockNotHeld error code")
	}

	_ = lease2.Unlock(ctx)
}

func TestWithLock(t *testing.T) {
	db := setupTestDB(t)
	lockInstance := gormlock.NewGormLockFromDB(db)
	ctx := context.Background()

	executed := false
	err := corelock.WithLock(ctx, lockInstance, "with-lock-key", 2*time.Second, func(context.Context) error {
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
	err := corelock.WithLock(ctx, lockInstance, "auto-renew-key", 30*time.Millisecond, func(context.Context) error {
		time.Sleep(80 * time.Millisecond)
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
	err := corelock.WithLock(ctx, lockInstance, "cancel-key", 5*time.Second, func(context.Context) error {
		close(started)
		cancel()
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	<-started

	assert.ErrorIs(t, err, context.Canceled)

	time.Sleep(100 * time.Millisecond)
	isLocked, err := lockInstance.IsLocked(context.Background(), "cancel-key")
	assert.NoError(t, err)
	assert.False(t, isLocked, "lock should be released even after context cancellation")
}

func TestWithLockAcquisitionTimeout(t *testing.T) {
	db := setupTestDB(t)
	lock1 := gormlock.NewGormLockFromDB(db)
	lock2 := gormlock.NewGormLockFromDB(db)

	lease1, err := lock1.Acquire(context.Background(), "timeout-key", time.Second)
	require.NoError(t, err)
	defer func() { _ = lease1.Unlock(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	err = corelock.WithLock(ctx, lock2, "timeout-key", time.Second, func(context.Context) error {
		t.Fatal("second owner must not enter while lock is held")
		return nil
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestCleanupExpiredLocks(t *testing.T) {
	db := setupTestDB(t)
	lock1 := gormlock.NewGormLockFromDB(db)
	lock2 := gormlock.NewGormLockFromDB(db)
	ctx := context.Background()

	_, _, _ = lock1.TryAcquire(ctx, "cleanup-key-1", 100*time.Millisecond)
	_, _, _ = lock2.TryAcquire(ctx, "cleanup-key-2", 100*time.Millisecond)

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

	lease1, err1 := lockInstance.Acquire(ctx, "key-1", 5*time.Second)
	lease2, err2 := lockInstance.Acquire(ctx, "key-2", 5*time.Second)
	lease3, err3 := lockInstance.Acquire(ctx, "key-3", 5*time.Second)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)

	isLocked1, _ := lockInstance.IsLocked(ctx, "key-1")
	isLocked2, _ := lockInstance.IsLocked(ctx, "key-2")
	isLocked3, _ := lockInstance.IsLocked(ctx, "key-3")

	assert.True(t, isLocked1)
	assert.True(t, isLocked2)
	assert.True(t, isLocked3)

	_ = lease1.Unlock(ctx)
	_ = lease2.Unlock(ctx)
	_ = lease3.Unlock(ctx)
}

func TestLockBlockingBehavior(t *testing.T) {
	db := setupTestDB(t)
	lock1 := gormlock.NewGormLockFromDB(db)
	lock2 := gormlock.NewGormLockFromDB(db)
	ctx := context.Background()

	lease1, err := lock1.Acquire(ctx, "blocking-key", 10*time.Second)
	require.NoError(t, err)

	isLocked, err := lock1.IsLocked(ctx, "blocking-key")
	require.NoError(t, err)
	require.True(t, isLocked)

	acquiredTime := time.Now()
	done := make(chan time.Time)

	go func() {
		lease2, _ := lock2.Acquire(ctx, "blocking-key", 10*time.Second)
		defer func() {
			if lease2 != nil {
				_ = lease2.Unlock(ctx)
			}
		}()
		done <- time.Now()
	}()

	time.Sleep(500 * time.Millisecond)

	unlockErr := lease1.Unlock(ctx)
	require.NoError(t, unlockErr, "unlock should succeed")

	_, err = lock1.IsLocked(ctx, "blocking-key")
	require.NoError(t, err)

	lock2AcquiredTime := <-done

	duration := lock2AcquiredTime.Sub(acquiredTime)

	assert.GreaterOrEqual(t, duration, 500*time.Millisecond, "lock2 should acquire after lock1 releases")
	assert.Less(t, duration, 1500*time.Millisecond, "lock2 should acquire shortly after lock1 releases")
}

func TestGormLockSameInstanceABADelayedUnlockAndRenew(t *testing.T) {
	db := setupTestDB(t)
	lockInstance := gormlock.NewGormLockFromDB(db)

	leaseA, err := lockInstance.Acquire(context.Background(), "same-instance-aba", 20*time.Millisecond)
	require.NoError(t, err)
	time.Sleep(30 * time.Millisecond)

	leaseB, acquired, err := lockInstance.TryAcquire(context.Background(), "same-instance-aba", time.Second)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotEqual(t, leaseA.Token(), leaseB.Token())

	require.Error(t, leaseA.Unlock(context.Background()))
	require.Error(t, leaseA.Renew(context.Background(), time.Second))

	locked, err := lockInstance.IsLocked(context.Background(), "same-instance-aba")
	require.NoError(t, err)
	require.True(t, locked)
	require.NoError(t, leaseB.Renew(context.Background(), time.Second))
	require.NoError(t, leaseB.Unlock(context.Background()))
}

func TestGormWithLockCancelsOnLeaseLoss(t *testing.T) {
	db := setupTestDB(t)
	lockInstance := gormlock.NewGormLockFromDB(db)

	err := corelock.WithLock(context.Background(), lockInstance, "lost-lease", 30*time.Millisecond, func(leaseCtx context.Context) error {
		require.NoError(t, db.Where("lock_key = ?", "lost-lease").Delete(&gormlock.LockRecord{}).Error)
		<-leaseCtx.Done()
		return nil
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, corelock.ErrLockLeaseLost))
}
