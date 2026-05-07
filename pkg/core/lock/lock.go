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
	"time"
)

// Lock defines the distributed lock interface
// This abstraction allows for multiple implementations (GORM, Redis, etcd, etc.)
type Lock interface {
	// Lock acquires a distributed lock, blocking until successful or context cancelled
	Lock(ctx context.Context, key string, ttl time.Duration) error

	// TryLock attempts to acquire a lock without blocking
	// Returns true if lock was acquired, false otherwise
	TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// Unlock releases a lock held by this instance
	Unlock(ctx context.Context, key string) error

	// Renew extends the TTL of a lock held by this instance
	Renew(ctx context.Context, key string, ttl time.Duration) error

	// IsLocked checks if a lock is currently held by anyone
	IsLocked(ctx context.Context, key string) (bool, error)

	// WithLock executes a function while holding a lock
	// Automatically acquires the lock, executes the function, and releases the lock
	WithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error

	// CleanupExpiredLocks removes expired locks (maintenance task)
	CleanupExpiredLocks(ctx context.Context) error
}
