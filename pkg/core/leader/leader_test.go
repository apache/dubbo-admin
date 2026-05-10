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

package leader

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestLeaderElection_EnsureTable(t *testing.T) {
	db := setupTestDB(t)
	le := NewLeaderElection(db, "test-component", "holder-1")

	err := le.EnsureTable()
	assert.NoError(t, err)

	// Verify the table exists by creating another instance and checking again
	err = le.EnsureTable()
	assert.NoError(t, err)
}

func TestLeaderElection_TryAcquire(t *testing.T) {
	db := setupTestDB(t)
	le := NewLeaderElection(db, "test-component", "holder-1")
	err := le.EnsureTable()
	require.NoError(t, err)

	ctx := context.Background()

	// First attempt should succeed (no lease exists)
	acquired := le.TryAcquire(ctx)
	assert.True(t, acquired)
	assert.True(t, le.IsLeader())

	// Verify the lease was created
	var lease LeaderLease
	result := db.Where("component = ?", "test-component").First(&lease)
	assert.NoError(t, result.Error)
	assert.Equal(t, "holder-1", lease.HolderID)
}

func TestLeaderElection_TryAcquire_AlreadyHeld(t *testing.T) {
	db := setupTestDB(t)
	le1 := NewLeaderElection(db, "test-component", "holder-1")
	le2 := NewLeaderElection(db, "test-component", "holder-2")

	err := le1.EnsureTable()
	require.NoError(t, err)

	ctx := context.Background()

	// First holder acquires
	acquired := le1.TryAcquire(ctx)
	assert.True(t, acquired)

	// Second holder tries to acquire (should fail because lease is not expired)
	acquired = le2.TryAcquire(ctx)
	assert.False(t, acquired)
	assert.False(t, le2.IsLeader())
}

func TestLeaderElection_Renew(t *testing.T) {
	db := setupTestDB(t)
	le := NewLeaderElection(db, "test-component", "holder-1",
		WithLeaseDuration(1*time.Second),
		WithRenewInterval(500*time.Millisecond))

	err := le.EnsureTable()
	require.NoError(t, err)

	ctx := context.Background()

	// Acquire the lease first
	acquired := le.TryAcquire(ctx)
	assert.True(t, acquired)

	oldVersion := le.currentVersion

	// Renew should succeed
	renewed := le.Renew(ctx)
	assert.True(t, renewed)
	assert.Greater(t, le.currentVersion, oldVersion)

	// Verify the lease was updated
	var lease LeaderLease
	result := db.Where("component = ?", "test-component").First(&lease)
	assert.NoError(t, result.Error)
	assert.Greater(t, lease.Version, int64(1))
}

func TestLeaderElection_Release(t *testing.T) {
	db := setupTestDB(t)
	le := NewLeaderElection(db, "test-component", "holder-1")
	err := le.EnsureTable()
	require.NoError(t, err)

	ctx := context.Background()

	// Acquire the lease
	acquired := le.TryAcquire(ctx)
	assert.True(t, acquired)

	// Release it
	le.Release(ctx)
	assert.False(t, le.IsLeader())

	// Verify the lease has expired
	var lease LeaderLease
	result := db.Where("component = ?", "test-component").First(&lease)
	assert.NoError(t, result.Error)
	assert.True(t, lease.ExpiresAt.Before(time.Now()))
}

func TestLeaderElection_Failover(t *testing.T) {
	db := setupTestDB(t)
	le1 := NewLeaderElection(db, "test-component", "holder-1",
		WithLeaseDuration(100*time.Millisecond))
	le2 := NewLeaderElection(db, "test-component", "holder-2",
		WithLeaseDuration(100*time.Millisecond))

	err := le1.EnsureTable()
	require.NoError(t, err)

	ctx := context.Background()

	// First holder acquires
	acquired := le1.TryAcquire(ctx)
	assert.True(t, acquired)

	// Second holder cannot acquire yet
	acquired = le2.TryAcquire(ctx)
	assert.False(t, acquired)

	// Wait for lease to expire
	time.Sleep(150 * time.Millisecond)

	// Now second holder should be able to acquire
	acquired = le2.TryAcquire(ctx)
	assert.True(t, acquired)
	assert.True(t, le2.IsLeader())
}

func TestGenerateHolderID(t *testing.T) {
	id1, err := GenerateHolderID()
	assert.NoError(t, err)
	assert.NotEmpty(t, id1)

	id2, err := GenerateHolderID()
	assert.NoError(t, err)
	assert.NotEmpty(t, id2)

	// IDs should be different (due to UUID)
	assert.NotEqual(t, id1, id2)
}

func TestLeaderElection_IsLeader(t *testing.T) {
	db := setupTestDB(t)
	le := NewLeaderElection(db, "test-component", "holder-1")
	err := le.EnsureTable()
	require.NoError(t, err)

	ctx := context.Background()

	// Initially not leader
	assert.False(t, le.IsLeader())

	// After acquiring, should be leader
	le.TryAcquire(ctx)
	assert.True(t, le.IsLeader())

	// After releasing, should not be leader
	le.Release(ctx)
	assert.False(t, le.IsLeader())
}
