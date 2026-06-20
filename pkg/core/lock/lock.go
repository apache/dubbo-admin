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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/apache/dubbo-admin/pkg/common/constants"
)

var (
	ErrLockLeaseLost   = errors.New("lock lease lost")
	ErrLockUnavailable = errors.New("lock is required")
)

type leaseContextKey struct{}

// Lease represents one successful lock acquisition. Its token is scoped to this
// acquisition only; delayed Renew or Unlock calls from an older lease must not
// affect a newer lease for the same key.
type Lease interface {
	Key() string
	Token() string
	Lost() <-chan struct{}
	Renew(ctx context.Context, ttl time.Duration) error
	Unlock(ctx context.Context) error
}

// Lock defines the lock backend contract used by cross-instance critical
// sections. A successful Acquire returns an acquisition-scoped Lease; backends
// must reject Renew and Unlock calls made with an older token.
type Lock interface {
	// Acquire blocks until it obtains a lease or the context is cancelled.
	Acquire(ctx context.Context, key string, ttl time.Duration) (Lease, error)

	// TryAcquire attempts to acquire a lease without blocking.
	TryAcquire(ctx context.Context, key string, ttl time.Duration) (Lease, bool, error)

	// IsLocked checks if a lock is currently held by anyone.
	IsLocked(ctx context.Context, key string) (bool, error)

	// CleanupExpiredLocks removes expired locks (maintenance task).
	CleanupExpiredLocks(ctx context.Context) error
}

type statefulLease interface {
	Lease
	bindContext(context.Context)
	markLost(error)
	lostError() error
}

type LeaseState struct {
	key   string
	token string

	mu       sync.RWMutex
	ctx      context.Context
	lostErr  error
	lost     chan struct{}
	lostOnce sync.Once
}

// NewLeaseState creates the shared lease state embedded by lock backends.
func NewLeaseState(key, token string) *LeaseState {
	s := &LeaseState{
		key:   key,
		token: token,
		ctx:   context.Background(),
		lost:  make(chan struct{}),
	}
	return s
}

func (s *LeaseState) Key() string {
	return s.key
}

func (s *LeaseState) Token() string {
	return s.token
}

func (s *LeaseState) context() context.Context {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}

func (s *LeaseState) bindContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	s.ctx = ctx
	s.mu.Unlock()
}

func (s *LeaseState) Lost() <-chan struct{} {
	return s.lost
}

func (s *LeaseState) markLost(err error) {
	if err == nil {
		err = ErrLockLeaseLost
	}
	s.mu.Lock()
	s.lostErr = err
	s.mu.Unlock()
	s.lostOnce.Do(func() {
		close(s.lost)
	})
}

func (s *LeaseState) lostError() error {
	s.mu.RLock()
	err := s.lostErr
	s.mu.RUnlock()
	if err != nil {
		return err
	}
	select {
	case <-s.lost:
		return ErrLockLeaseLost
	default:
		return nil
	}
}

// NewLeaseToken returns an owner token scoped to one lock acquisition.
func NewLeaseToken() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// WithLock runs fn while holding key and returns ErrLockLeaseLost if the lease
// expires or renewal fails before the critical section is safely complete. Code
// inside fn should call CheckLease before mutating state after blocking work.
func WithLock(ctx context.Context, lockMgr Lock, key string, ttl time.Duration, fn func(context.Context) error) (err error) {
	if lockMgr == nil {
		return ErrLockUnavailable
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if fn == nil {
		return fmt.Errorf("lock callback is required")
	}

	acquireCtx := ctx
	cancelAcquire := func() {}
	if _, ok := ctx.Deadline(); !ok {
		acquireCtx, cancelAcquire = context.WithTimeout(ctx, constants.DefaultLockTimeout)
	}
	lease, acquireErr := lockMgr.Acquire(acquireCtx, key, ttl)
	cancelAcquire()
	if acquireErr != nil {
		return acquireErr
	}

	leaseCtx, cancelLease := context.WithCancel(ctx)
	leaseCtx = context.WithValue(leaseCtx, leaseContextKey{}, lease)
	if stateful, ok := lease.(statefulLease); ok {
		stateful.bindContext(leaseCtx)
	}

	stopRenew := make(chan struct{})
	renewDone := make(chan struct{})
	leaseLost := make(chan error, 1)
	if ttl > 0 {
		go autoRenewLease(leaseCtx, cancelLease, lease, ttl, stopRenew, renewDone, leaseLost)
	} else {
		close(renewDone)
	}

	defer func() {
		close(stopRenew)
		<-renewDone

		unlockCtx, unlockCancel := context.WithTimeout(context.Background(), constants.DefaultUnlockTimeout)
		unlockErr := lease.Unlock(unlockCtx)
		unlockCancel()
		cancelLease()

		if recovered := recover(); recovered != nil {
			panic(recovered)
		}
		if err != nil {
			if lostErr := leaseLostError(lease, leaseLost); lostErr != nil &&
				(errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrLockLeaseLost)) {
				err = lostErr
			}
			return
		}
		if lostErr := leaseLostError(lease, leaseLost); lostErr != nil {
			err = lostErr
			return
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			err = ctxErr
			return
		}
		if unlockErr != nil {
			err = unlockErr
		}
	}()

	err = fn(leaseCtx)
	if err != nil {
		return err
	}
	if lostErr := leaseLostError(lease, leaseLost); lostErr != nil {
		return lostErr
	}
	return CheckLease(leaseCtx)
}

// CheckLease fails closed when the context is cancelled or its bound lease has
// been lost. It is intentionally cheap so mutation code can call it between
// ResourceManager writes, intent CAS, and ledger appends.
func CheckLease(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	if lease, ok := ctx.Value(leaseContextKey{}).(Lease); ok {
		if lostErr := leaseLostError(lease, nil); lostErr != nil {
			return lostErr
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

// RequireLease returns the lease bound by WithLock or fails when a mutating
// versioning path is entered without the canonical rule lock.
func RequireLease(ctx context.Context) (Lease, error) {
	if ctx == nil {
		return nil, ErrLockUnavailable
	}
	lease, ok := ctx.Value(leaseContextKey{}).(Lease)
	if !ok || lease == nil {
		return nil, ErrLockUnavailable
	}
	if lostErr := leaseLostError(lease, nil); lostErr != nil {
		return nil, lostErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return lease, nil
}

// LeaseFromContext exposes the lease bound by WithLock for diagnostics and
// tests; callers must still use CheckLease or RequireLease before writes.
func LeaseFromContext(ctx context.Context) (Lease, bool) {
	if ctx == nil {
		return nil, false
	}
	lease, ok := ctx.Value(leaseContextKey{}).(Lease)
	return lease, ok
}

func autoRenewLease(leaseCtx context.Context, cancelLease context.CancelFunc, lease Lease, ttl time.Duration, stop <-chan struct{}, done chan<- struct{}, lost chan<- error) {
	defer close(done)
	interval := ttl / 3
	if interval <= 0 {
		interval = ttl
	}
	if interval < 10*time.Millisecond {
		interval = 10 * time.Millisecond
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-leaseCtx.Done():
			return
		case <-ticker.C:
			renewCtx, cancel := context.WithTimeout(context.Background(), constants.DefaultRenewTimeout)
			err := lease.Renew(renewCtx, ttl)
			cancel()
			if err != nil {
				lostErr := fmt.Errorf("%w: renew failed for %s: %v", ErrLockLeaseLost, lease.Key(), err)
				if stateful, ok := lease.(statefulLease); ok {
					stateful.markLost(lostErr)
				}
				select {
				case lost <- lostErr:
				default:
				}
				cancelLease()
				return
			}
		}
	}
}

func leaseLostError(lease Lease, ch <-chan error) error {
	select {
	case <-lease.Lost():
		if stateful, ok := lease.(statefulLease); ok {
			if err := stateful.lostError(); err != nil {
				if errors.Is(err, ErrLockLeaseLost) {
					return err
				}
				return fmt.Errorf("%w: %v", ErrLockLeaseLost, err)
			}
		}
		return ErrLockLeaseLost
	default:
	}
	if ch == nil {
		return nil
	}
	select {
	case err := <-ch:
		if err == nil {
			return ErrLockLeaseLost
		}
		return err
	default:
		return nil
	}
}
