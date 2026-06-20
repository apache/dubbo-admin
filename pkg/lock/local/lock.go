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
	"fmt"
	"sync"
	"time"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	corelock "github.com/apache/dubbo-admin/pkg/core/lock"
)

var (
	defaultBackend = &backend{locks: map[string]record{}}
)

type record struct {
	token    string
	expireAt time.Time
}

type backend struct {
	mu    sync.Mutex
	locks map[string]record
}

// LocalLock implements the shared lock contract for process-local memory
// stores. It is intentionally owner-aware so tests exercise the same unlock and
// renew invariants as distributed backends.
type LocalLock struct {
	backend *backend
}

var _ corelock.Lock = (*LocalLock)(nil)

// NewLocalLock returns a process-local lock backend. It is only shared inside
// one admin process, so clustered deployments must use a distributed backend.
func NewLocalLock() corelock.Lock {
	return &LocalLock{
		backend: defaultBackend,
	}
}

type lease struct {
	*corelock.LeaseState
	lock *LocalLock
}

func (l *LocalLock) Acquire(ctx context.Context, key string, ttl time.Duration) (corelock.Lease, error) {
	ticker := time.NewTicker(constants.DefaultLockRetryInterval)
	defer ticker.Stop()

	for {
		lease, acquired, err := l.TryAcquire(ctx, key, ttl)
		if err != nil {
			return nil, err
		}
		if acquired {
			return lease, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("failed to acquire local lock %s: %w", key, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (l *LocalLock) TryAcquire(ctx context.Context, key string, ttl time.Duration) (corelock.Lease, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	token, err := corelock.NewLeaseToken()
	if err != nil {
		return nil, false, err
	}
	now := time.Now()
	l.backend.mu.Lock()
	defer l.backend.mu.Unlock()

	if current, exists := l.backend.locks[key]; exists && current.expireAt.After(now) {
		return nil, false, nil
	}
	l.backend.locks[key] = record{token: token, expireAt: now.Add(ttl)}
	return &lease{
		LeaseState: corelock.NewLeaseState(key, token),
		lock:       l,
	}, true, nil
}

func (l *lease) Unlock(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	l.lock.backend.mu.Lock()
	defer l.lock.backend.mu.Unlock()

	current, exists := l.lock.backend.locks[l.Key()]
	if !exists || current.token != l.Token() {
		return bizerror.New(bizerror.LockNotHeld, "lock not held by this owner")
	}
	delete(l.lock.backend.locks, l.Key())
	return nil
}

func (l *lease) Renew(ctx context.Context, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	l.lock.backend.mu.Lock()
	defer l.lock.backend.mu.Unlock()

	now := time.Now()
	current, exists := l.lock.backend.locks[l.Key()]
	if !exists || current.token != l.Token() {
		return bizerror.New(bizerror.LockNotHeld, "lock not held by this owner")
	}
	if !current.expireAt.After(now) {
		delete(l.lock.backend.locks, l.Key())
		return bizerror.New(bizerror.LockNotHeld, "lock lease expired")
	}
	current.expireAt = now.Add(ttl)
	l.lock.backend.locks[l.Key()] = current
	return nil
}

func (l *LocalLock) IsLocked(ctx context.Context, key string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	now := time.Now()
	l.backend.mu.Lock()
	defer l.backend.mu.Unlock()

	current, exists := l.backend.locks[key]
	if !exists {
		return false, nil
	}
	if !current.expireAt.After(now) {
		delete(l.backend.locks, key)
		return false, nil
	}
	return true, nil
}

func (l *LocalLock) CleanupExpiredLocks(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := time.Now()
	l.backend.mu.Lock()
	defer l.backend.mu.Unlock()

	for key, current := range l.backend.locks {
		if !current.expireAt.After(now) {
			delete(l.backend.locks, key)
		}
	}
	return nil
}
