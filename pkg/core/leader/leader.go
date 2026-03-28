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
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/apache/dubbo-admin/pkg/core/logger"
)

const (
	// DefaultLeaseDuration is the default duration for a leader lease
	DefaultLeaseDuration = 30 * time.Second
	// DefaultRenewInterval is the default interval for renewing the lease
	DefaultRenewInterval = 10 * time.Second
	// DefaultAcquireRetryInterval is the default retry interval for acquiring leadership
	DefaultAcquireRetryInterval = 5 * time.Second
)

// LeaderElection manages leader election for distributed components
// It uses database-based optimistic locking to ensure only one replica holds the lease at any time
type LeaderElection struct {
	db             *gorm.DB
	component      string
	holderID       string
	leaseDuration  time.Duration
	renewInterval  time.Duration
	acquireRetry   time.Duration
	isLeader       atomic.Bool
	currentVersion int64
	stopCh         chan struct{}
}

// Option is a functional option for configuring LeaderElection
type Option func(*LeaderElection)

// WithLeaseDuration sets the lease duration
func WithLeaseDuration(d time.Duration) Option {
	return func(le *LeaderElection) {
		le.leaseDuration = d
	}
}

// WithRenewInterval sets the renewal interval
func WithRenewInterval(d time.Duration) Option {
	return func(le *LeaderElection) {
		le.renewInterval = d
	}
}

// WithAcquireRetryInterval sets the acquisition retry interval
func WithAcquireRetryInterval(d time.Duration) Option {
	return func(le *LeaderElection) {
		le.acquireRetry = d
	}
}

// NewLeaderElection creates a new LeaderElection instance
func NewLeaderElection(db *gorm.DB, component, holderID string, opts ...Option) *LeaderElection {
	le := &LeaderElection{
		db:            db,
		component:     component,
		holderID:      holderID,
		leaseDuration: DefaultLeaseDuration,
		renewInterval: DefaultRenewInterval,
		acquireRetry:  DefaultAcquireRetryInterval,
		stopCh:        make(chan struct{}),
	}

	for _, opt := range opts {
		opt(le)
	}

	return le
}

// EnsureTable creates the leader_leases table if it doesn't exist
// This is idempotent and can be called multiple times
func (le *LeaderElection) EnsureTable() error {
	return le.db.AutoMigrate(&LeaderLease{})
}

// TryAcquire attempts to acquire the leader lease from an expired holder.
// It only competes for leases that have already expired and does NOT renew an
// existing self-held lease — use Renew for that.
// Returns true if the current holder successfully acquired the lease.
func (le *LeaderElection) TryAcquire(ctx context.Context) bool {
	now := time.Now()
	expiresAt := now.Add(le.leaseDuration)

	// Only take over an expired lease; never pre-empt an active holder.
	result := le.db.WithContext(ctx).Model(&LeaderLease{}).
		Where("component = ? AND expires_at < ?", le.component, now).
		Updates(map[string]interface{}{
			"holder_id":   le.holderID,
			"acquired_at": now,
			"expires_at":  expiresAt,
			"version":     gorm.Expr("version + 1"),
		})

	if result.Error != nil {
		logger.Warnf("leader election: failed to update lease for component %s: %v", le.component, result.Error)
		le.isLeader.Store(false)
		return false
	}

	// If the update succeeded (found a row to update)
	if result.RowsAffected > 0 {
		// Fetch the updated version
		var lease LeaderLease
		err := le.db.WithContext(ctx).
			Where("component = ?", le.component).
			First(&lease).Error
		if err != nil {
			logger.Warnf("leader election: failed to read back updated lease for component %s: %v", le.component, err)
			le.isLeader.Store(false)
			return false
		}
		le.currentVersion = lease.Version
		le.isLeader.Store(true)
		return true
	}

	// No row was updated, try to insert a new record (lease doesn't exist)
	result = le.db.WithContext(ctx).Create(&LeaderLease{
		Component:  le.component,
		HolderID:   le.holderID,
		AcquiredAt: now,
		ExpiresAt:  expiresAt,
		Version:    1,
	})

	if result.Error != nil {
		// If insertion fails, it means another replica just created it
		// This is expected in concurrent scenarios
		logger.Debugf("leader election: failed to insert lease for component %s (probably created by another replica): %v", le.component, result.Error)
		le.isLeader.Store(false)
		return false
	}

	le.currentVersion = 1
	le.isLeader.Store(true)
	return true
}

// Renew attempts to renew the current leader lease
// Returns true if the renewal was successful
func (le *LeaderElection) Renew(ctx context.Context) bool {
	if !le.isLeader.Load() {
		return false
	}

	now := time.Now()
	expiresAt := now.Add(le.leaseDuration)

	result := le.db.WithContext(ctx).Model(&LeaderLease{}).
		Where("component = ? AND holder_id = ? AND version = ?", le.component, le.holderID, le.currentVersion).
		Updates(map[string]interface{}{
			"acquired_at": now,
			"expires_at":  expiresAt,
			"version":     gorm.Expr("version + 1"),
		})

	if result.Error != nil {
		logger.Warnf("leader election: failed to renew lease for component %s: %v", le.component, result.Error)
		le.isLeader.Store(false)
		return false
	}

	if result.RowsAffected > 0 {
		le.currentVersion++
		return true
	}

	// Lease was lost (likely held by another replica now)
	logger.Warnf("leader election: lost leader lease for component %s (renewal failed, version mismatch)", le.component)
	le.isLeader.Store(false)
	return false
}

// Release releases the leader lease for this holder
// This should be called when the holder voluntarily gives up leadership
func (le *LeaderElection) Release(ctx context.Context) {
	le.isLeader.Store(false)

	expiresAt := time.Now().Add(-1 * time.Second) // Immediately expire the lease

	result := le.db.WithContext(ctx).Model(&LeaderLease{}).
		Where("component = ? AND holder_id = ?", le.component, le.holderID).
		Update("expires_at", expiresAt)

	if result.Error != nil {
		logger.Warnf("leader election: failed to release lease for component %s: %v", le.component, result.Error)
	}
}

// IsLeader returns true if this holder currently holds the leader lease
func (le *LeaderElection) IsLeader() bool {
	return le.isLeader.Load()
}

// RunLeaderElection runs the leader election loop
// It blocks and runs onStartLeading/onStopLeading callbacks as leadership changes
// This is designed to be run in a separate goroutine
func (le *LeaderElection) RunLeaderElection(ctx context.Context, stopCh <-chan struct{},
	onStartLeading func(), onStopLeading func()) {

	ticker := time.NewTicker(le.acquireRetry)
	defer ticker.Stop()

	renewTicker := time.NewTicker(le.renewInterval)
	defer renewTicker.Stop()
	renewTicker.Stop() // Don't start renewal ticker yet

	isLeader := false

	for {
		select {
		case <-ctx.Done():
			if isLeader {
				le.Release(context.Background())
				onStopLeading()
			}
			return
		case <-stopCh:
			if isLeader {
				le.Release(context.Background())
				onStopLeading()
			}
			return
		case <-ticker.C:
			// Try to acquire leadership if not already leader
			if !isLeader {
				if le.TryAcquire(ctx) {
					logger.Infof("leader election: component %s acquired leadership (holder: %s)", le.component, le.holderID)
					isLeader = true
					renewTicker.Reset(le.renewInterval)
					onStartLeading()
				}
			}
		case <-renewTicker.C:
			// Renew leadership if currently leader
			if isLeader {
				if !le.Renew(ctx) {
					logger.Warnf("leader election: component %s lost leadership (holder: %s)", le.component, le.holderID)
					isLeader = false
					renewTicker.Stop()
					ticker.Reset(le.acquireRetry)
					onStopLeading()
				}
			}
		}
	}
}

// GenerateHolderID generates a unique holder ID combining hostname and UUID
func GenerateHolderID() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return fmt.Sprintf("%s-%s", hostname, uuid.New().String()), nil
}
