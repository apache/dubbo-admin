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

package dbcommon

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"

	storecfg "github.com/apache/dubbo-admin/pkg/config/store"
	"github.com/apache/dubbo-admin/pkg/core/logger"
)

var (
	// pools stores all connection pools indexed by a unique key (storeType:address)
	pools = make(map[string]*ConnectionPool)
	// poolsMutex protects concurrent access to the pools map
	poolsMutex sync.RWMutex
)

// ConnectionPoolConfig defines connection pool configuration
type ConnectionPoolConfig struct {
	MaxIdleConns    int           // Maximum number of idle connections
	MaxOpenConns    int           // Maximum number of open connections
	ConnMaxLifetime time.Duration // Maximum lifetime of a connection
	ConnMaxIdleTime time.Duration // Maximum idle time of a connection
}

// ConnectionPool manages database connections with connection pooling
type ConnectionPool struct {
	db        *gorm.DB
	sqlDB     *sql.DB
	address   string
	storeType storecfg.Type
	mu        sync.RWMutex
	refCount  int       // Reference counter for the number of stores using this pool
	closeOnce sync.Once // Ensure Close is called only once
	closed    bool      // Track if the pool is closed
}

// GetOrCreatePool returns or creates a connection pool for the given store type and address
// It implements a singleton pattern with reference counting to allow pool reuse across multiple stores
// If a pool already exists for the same storeType and address, it increments the reference count and returns the existing pool
// Otherwise, it creates a new pool with the provided dialector
func GetOrCreatePool(dialector gorm.Dialector, storeType storecfg.Type, address string, config *ConnectionPoolConfig) (*ConnectionPool, error) {
	if storeType == storecfg.Memory {
		return nil, fmt.Errorf("memory pool store is no need to create connection pool")
	}

	poolKey := fmt.Sprintf("%s:%s", storeType, address)

	poolsMutex.Lock()
	defer poolsMutex.Unlock()

	// Check if pool already exists
	if existingPool, exists := pools[poolKey]; exists {
		// Increment reference count when reusing existing pool
		existingPool.refCount++
		logger.Infof("Reusing %s connection pool: address=%s, refCount=%d", storeType, address, existingPool.refCount)
		return existingPool, nil
	}

	// Create new pool
	if config == nil {
		config = DefaultConnectionPoolConfig()
	}

	pool, err := NewConnectionPool(dialector, storeType, address, config)
	if err != nil {
		return nil, err
	}

	// Store the pool
	pools[poolKey] = pool
	logger.Infof("%s connection pool created successfully: address=%s, maxIdleConns=%d, maxOpenConns=%d",
		storeType, address, config.MaxIdleConns, config.MaxOpenConns)

	return pool, nil
}

// RemovePool removes a pool from the global registry
// This should only be called when the pool's reference count reaches zero
func RemovePool(storeType storecfg.Type, address string) {
	poolKey := fmt.Sprintf("%s:%s", storeType, address)

	poolsMutex.Lock()
	defer poolsMutex.Unlock()

	delete(pools, poolKey)
	logger.Infof("Removed %s connection pool from registry: address=%s", storeType, address)
}

// DefaultConnectionPoolConfig returns default connection pool configuration
func DefaultConnectionPoolConfig() *ConnectionPoolConfig {
	return &ConnectionPoolConfig{
		MaxIdleConns:    10,               // Default: 10 idle connections
		MaxOpenConns:    100,              // Default: 100 max open connections
		ConnMaxLifetime: time.Hour,        // Default: 1 hour max lifetime
		ConnMaxIdleTime: 10 * time.Minute, // Default: 10 minutes max idle time
	}
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(dialector gorm.Dialector, storeType storecfg.Type, address string, config *ConnectionPoolConfig) (*ConnectionPool, error) {
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", storeType, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	return &ConnectionPool{
		db:        db,
		sqlDB:     sqlDB,
		address:   address,
		storeType: storeType,
		refCount:  1, // Initial reference count
	}, nil
}

// GetDB returns the gorm.DB instance
func (p *ConnectionPool) GetDB() *gorm.DB {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.db
}

// Address returns the connection address
func (p *ConnectionPool) Address() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.address
}

// RefCount returns the current reference count
func (p *ConnectionPool) RefCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.refCount
}

// IncrementRef increments the reference count
func (p *ConnectionPool) IncrementRef() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.refCount++
}

// Close closes the connection pool gracefully with reference counting
// The pool is only actually closed when refCount reaches 0
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil // Already closed
	}

	p.refCount--
	logger.Infof("Decremented %s connection pool refCount: address=%s, refCount=%d", p.storeType, p.address, p.refCount)

	// Only close the pool when no stores are using it
	if p.refCount <= 0 {
		var closeErr error
		p.closeOnce.Do(func() {
			if p.sqlDB != nil {
				logger.Infof("Closing %s connection pool: address=%s", p.storeType, p.address)
				closeErr = p.sqlDB.Close()
				p.closed = true
			}
			RemovePool(p.storeType, p.address)
		})
		return closeErr
	}

	return nil
}

// Ping checks if the database connection is alive
func (p *ConnectionPool) Ping() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.sqlDB != nil {
		return p.sqlDB.Ping()
	}
	return fmt.Errorf("connection pool not initialized")
}

// Stats returns database connection pool statistics
func (p *ConnectionPool) Stats() sql.DBStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.sqlDB != nil {
		return p.sqlDB.Stats()
	}
	return sql.DBStats{}
}

// GetGlobalDB returns the first available gorm.DB instance from the global pool registry
// This is used by components that need database access without going through StoreComponent
// Returns nil if no database connection is available
func GetGlobalDB(storeType storecfg.Type) *gorm.DB {
	poolsMutex.RLock()
	defer poolsMutex.RUnlock()

	// Find the first pool matching the store type
	prefix := string(storeType) + ":"
	for key, pool := range pools {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			return pool.GetDB()
		}
	}

	return nil
}
