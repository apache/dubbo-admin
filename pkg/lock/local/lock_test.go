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

package local

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	corelock "github.com/apache/dubbo-admin/pkg/core/lock"
)

func TestLocalLockTimeoutAndOwnerSafety(t *testing.T) {
	lock1 := NewLocalLock()
	lock2 := NewLocalLock()

	lease1, err := lock1.Acquire(context.Background(), "owner-safe", time.Second)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	err = corelock.WithLock(ctx, lock2, "owner-safe", time.Second, func(context.Context) error {
		t.Fatal("second owner must not enter while the first owner holds the lock")
		return nil
	})
	require.Error(t, err)

	locked, err := lock1.IsLocked(context.Background(), "owner-safe")
	require.NoError(t, err)
	require.True(t, locked)

	require.NoError(t, lease1.Unlock(context.Background()))
}

func TestLocalLockDelayedUnlockDoesNotReleaseNewOwner(t *testing.T) {
	lock1 := NewLocalLock()
	lock2 := NewLocalLock()

	lease1, err := lock1.Acquire(context.Background(), "delayed-unlock", 20*time.Millisecond)
	require.NoError(t, err)
	time.Sleep(30 * time.Millisecond)
	lease2, acquired, err := lock2.TryAcquire(context.Background(), "delayed-unlock", time.Second)
	require.NoError(t, err)
	require.True(t, acquired)

	require.Error(t, lease1.Unlock(context.Background()))
	require.Error(t, lease1.Renew(context.Background(), time.Second))
	locked, err := lock2.IsLocked(context.Background(), "delayed-unlock")
	require.NoError(t, err)
	require.True(t, locked)

	require.NoError(t, lease2.Unlock(context.Background()))
}

func TestLocalLockSameInstanceABADelayedUnlockAndRenew(t *testing.T) {
	lock1 := NewLocalLock()

	leaseA, err := lock1.Acquire(context.Background(), "same-instance-aba", 20*time.Millisecond)
	require.NoError(t, err)
	time.Sleep(30 * time.Millisecond)

	leaseB, acquired, err := lock1.TryAcquire(context.Background(), "same-instance-aba", time.Second)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotEqual(t, leaseA.Token(), leaseB.Token())

	require.Error(t, leaseA.Unlock(context.Background()))
	require.Error(t, leaseA.Renew(context.Background(), time.Second))

	locked, err := lock1.IsLocked(context.Background(), "same-instance-aba")
	require.NoError(t, err)
	require.True(t, locked)
	require.NoError(t, leaseB.Renew(context.Background(), time.Second))
	require.NoError(t, leaseB.Unlock(context.Background()))
}

func TestLocalWithLockReportsLostLease(t *testing.T) {
	lock1 := NewLocalLock()

	err := corelock.WithLock(context.Background(), lock1, "lost-lease", 30*time.Millisecond, func(leaseCtx context.Context) error {
		local := lock1.(*LocalLock)
		local.backend.mu.Lock()
		delete(local.backend.locks, "lost-lease")
		local.backend.mu.Unlock()
		<-leaseCtx.Done()
		return nil
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, corelock.ErrLockLeaseLost))
}
