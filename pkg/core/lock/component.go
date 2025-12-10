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
	"gorm.io/gorm"
	"math"
	"time"

	"github.com/pkg/errors"

	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

const (
	// DistributedLockComponent is the component type for distributed lock
	DistributedLockComponent runtime.ComponentType = "distributed lock"
)

// Component implements the runtime.Component interface for distributed lock
type Component struct {
	lock Lock
}

// NewComponent creates a new distributed lock component
func NewComponent() *Component {
	return &Component{}
}

// Type returns the component type
func (c *Component) Type() runtime.ComponentType {
	return DistributedLockComponent
}

// Order indicates the initialization order
// Lock should be initialized after Store (Order math.MaxInt - 1)
// Higher order values are initialized first, so we use math.MaxInt - 2
func (c *Component) Order() int {
	return math.MaxInt - 2 // After Store, before other services
}

// Init initializes the distributed lock component
func (c *Component) Init(ctx runtime.BuilderContext) error {
	// Get the store component to access database connection
	storeComp, err := ctx.GetActivatedComponent(runtime.ResourceStore)
	if err != nil {
		return err
	}

	// Try to extract database connection from store component
	// We use GetDB() interface to avoid circular dependency with dbcommon package
	type DBProvider interface {
		GetDB() *gorm.DB
	}

	storeWithDB, ok := storeComp.(DBProvider)
	if !ok {
		// For memory store or other stores without database
		logger.Warnf("Store component does not provide database connection, distributed lock will not be available")
		return nil
	}

	db := storeWithDB.GetDB()
	if db == nil {
		logger.Warnf("Database connection is nil, distributed lock will not be available")
		return nil
	}

	// Create GORM-based lock implementation using NewGormLockFromDB
	c.lock = NewGormLockFromDB(db)

	// Initialize the lock table
	if err := db.AutoMigrate(&LockRecord{}); err != nil {
		return errors.Wrap(err, "failed to migrate lock table")
	}

	return nil
}

// Start starts the distributed lock component
func (c *Component) Start(rt runtime.Runtime, stop <-chan struct{}) error {
	if c.lock == nil {
		logger.Warn("Distributed lock not available, skipping")
		return nil
	}

	// Start background cleanup task
	ticker := time.NewTicker(DefaultCleanupInterval) // Cleanup every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return nil
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), DefaultCleanupTimeout)
			if err := c.lock.CleanupExpiredLocks(ctx); err != nil {
				logger.Errorf("Failed to cleanup expired locks: %v", err)
			}
			cancel()
		}
	}
}

// GetLock returns the lock instance
func (c *Component) GetLock() Lock {
	return c.lock
}

// GetLockFromRuntime extracts the lock instance from runtime
func GetLockFromRuntime(rt runtime.Runtime) (Lock, error) {
	comp, err := rt.GetComponent(DistributedLockComponent)
	if err != nil {
		return nil, err
	}

	lockComp, ok := comp.(*Component)
	if !ok {
		return nil, errors.Errorf("component %s is not a valid lock component", DistributedLockComponent)
	}

	if lockComp.lock == nil {
		return nil, errors.New("distributed lock is not available (possibly using memory store)")
	}

	return lockComp.GetLock(), nil
}
