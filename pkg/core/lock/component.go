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

	"github.com/pkg/errors"

	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/store/dbcommon"
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
// Lock should be initialized after Store (Order 100) but before other services
func (c *Component) Order() int {
	return 90 // After Store, before Console
}

// Init initializes the distributed lock component
func (c *Component) Init(ctx runtime.BuilderContext) error {
	// Get the store component to access connection pool
	storeComp, err := ctx.GetActivatedComponent(runtime.ResourceStore)
	if err != nil {
		return err
	}

	// Try to extract connection pool from store component
	// We need to use type assertion with the proper interface
	type ConnectionPoolProvider interface {
		GetConnectionPool() *dbcommon.ConnectionPool
	}

	storeWithPool, ok := storeComp.(ConnectionPoolProvider)
	if !ok {
		// For memory store or other stores without connection pool
		logger.Warnf("Store component does not provide connection pool, distributed lock will not be available")
		return nil
	}

	pool := storeWithPool.GetConnectionPool()
	if pool == nil {
		logger.Warnf("Connection pool is nil, distributed lock will not be available")
		return nil
	}

	// Create GORM-based lock implementation using NewGormLock
	c.lock = NewGormLock(pool)

	// Initialize the lock table
	db := pool.GetDB()
	if err := db.AutoMigrate(&LockRecord{}); err != nil {
		return errors.Wrap(err, "failed to migrate lock table")
	}

	logger.Info("Distributed lock component initialized successfully")
	return nil
}

// Start starts the distributed lock component
func (c *Component) Start(rt runtime.Runtime, stop <-chan struct{}) error {
	if c.lock == nil {
		logger.Warn("Distributed lock not available, skipping")
		return nil
	}

	// Start background cleanup task
	ticker := time.NewTicker(5 * time.Minute) // Cleanup every 5 minutes
	defer ticker.Stop()

	logger.Info("Distributed lock cleanup task started")

	for {
		select {
		case <-stop:
			logger.Info("Distributed lock cleanup task stopped")
			return nil
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := c.lock.CleanupExpiredLocks(ctx); err != nil {
				logger.Errorf("Failed to cleanup expired locks:  %v", err)
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
		// 修正：使用标准错误处理
		return nil, errors.Errorf("component %s is not a valid lock component", DistributedLockComponent)
	}

	if lockComp.lock == nil {
		return nil, errors.New("distributed lock is not available (possibly using memory store)")
	}

	return lockComp.GetLock(), nil
}
