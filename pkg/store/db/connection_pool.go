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

package db

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/apache/dubbo-admin/pkg/core/logger"
)

var (
	mysqlPool    *ConnectionPool
	postgresPool *ConnectionPool
	mysqlOnce    sync.Once
	postgresOnce sync.Once
	poolMutex    sync.RWMutex
)

// ConnectionPool manages database connections with connection pooling
type ConnectionPool struct {
	db        *gorm.DB
	sqlDB     *sql.DB
	address   string
	dbType    string
	mu        sync.RWMutex
	refCount  int       // Reference counter for the number of stores using this pool
	closeOnce sync.Once // Ensure Close is called only once
	closed    bool      // Track if the pool is closed
}

// ConnectionPoolConfig defines connection pool configuration
type ConnectionPoolConfig struct {
	MaxIdleConns    int           // Maximum number of idle connections
	MaxOpenConns    int           // Maximum number of open connections
	ConnMaxLifetime time.Duration // Maximum lifetime of a connection
	ConnMaxIdleTime time.Duration // Maximum idle time of a connection
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

// GetOrCreateMySQLPool returns or creates a MySQL connection pool
func GetOrCreateMySQLPool(address string, config *ConnectionPoolConfig) (*ConnectionPool, error) {
	poolMutex.Lock()
	defer poolMutex.Unlock()

	if mysqlPool != nil && mysqlPool.address == address {
		// Increment reference count when reusing existing pool
		mysqlPool.mu.Lock()
		mysqlPool.refCount++
		mysqlPool.mu.Unlock()
		logger.Infof("Reusing MySQL connection pool: address=%s, refCount=%d", address, mysqlPool.refCount)
		return mysqlPool, nil
	}

	var initErr error
	mysqlOnce.Do(func() {
		if config == nil {
			config = DefaultConnectionPoolConfig()
		}

		db, err := gorm.Open(mysql.Open(address), &gorm.Config{})
		if err != nil {
			initErr = fmt.Errorf("failed to connect to mysql: %w", err)
			return
		}

		sqlDB, err := db.DB()
		if err != nil {
			initErr = fmt.Errorf("failed to get underlying sql.DB: %w", err)
			return
		}

		// Configure connection pool
		sqlDB.SetMaxIdleConns(config.MaxIdleConns)
		sqlDB.SetMaxOpenConns(config.MaxOpenConns)
		sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
		sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)

		mysqlPool = &ConnectionPool{
			db:       db,
			sqlDB:    sqlDB,
			address:  address,
			dbType:   "mysql",
			refCount: 1, // Initial reference count
		}

		logger.Infof("MySQL connection pool created successfully: address=%s, maxIdleConns=%d, maxOpenConns=%d",
			address, config.MaxIdleConns, config.MaxOpenConns)
	})

	if initErr != nil {
		mysqlOnce = sync.Once{} // Reset once to allow retry
		return nil, initErr
	}

	return mysqlPool, nil
}

// GetOrCreatePostgresPool returns or creates a PostgreSQL connection pool
func GetOrCreatePostgresPool(address string, config *ConnectionPoolConfig) (*ConnectionPool, error) {
	poolMutex.Lock()
	defer poolMutex.Unlock()

	if postgresPool != nil && postgresPool.address == address {
		// Increment reference count when reusing existing pool
		postgresPool.mu.Lock()
		postgresPool.refCount++
		postgresPool.mu.Unlock()
		logger.Infof("Reusing PostgreSQL connection pool: address=%s, refCount=%d", address, postgresPool.refCount)
		return postgresPool, nil
	}

	var initErr error
	postgresOnce.Do(func() {
		if config == nil {
			config = DefaultConnectionPoolConfig()
		}

		db, err := gorm.Open(postgres.Open(address), &gorm.Config{})
		if err != nil {
			initErr = fmt.Errorf("failed to connect to postgres: %w", err)
			return
		}

		sqlDB, err := db.DB()
		if err != nil {
			initErr = fmt.Errorf("failed to get underlying sql.DB: %w", err)
			return
		}

		// Configure connection pool
		sqlDB.SetMaxIdleConns(config.MaxIdleConns)
		sqlDB.SetMaxOpenConns(config.MaxOpenConns)
		sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
		sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)

		postgresPool = &ConnectionPool{
			db:       db,
			sqlDB:    sqlDB,
			address:  address,
			dbType:   "postgres",
			refCount: 1, // Initial reference count
		}

		logger.Infof("PostgreSQL connection pool created successfully: address=%s, maxIdleConns=%d, maxOpenConns=%d",
			address, config.MaxIdleConns, config.MaxOpenConns)
	})

	if initErr != nil {
		postgresOnce = sync.Once{} // Reset once to allow retry
		return nil, initErr
	}

	return postgresPool, nil
}

// GetDB returns the gorm.DB instance
func (p *ConnectionPool) GetDB() *gorm.DB {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.db
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
	logger.Infof("Decremented %s connection pool refCount: address=%s, refCount=%d", p.dbType, p.address, p.refCount)

	// Only close the pool when no stores are using it
	if p.refCount <= 0 {
		var closeErr error
		p.closeOnce.Do(func() {
			if p.sqlDB != nil {
				logger.Infof("Closing %s connection pool: address=%s", p.dbType, p.address)
				closeErr = p.sqlDB.Close()
				p.closed = true
			}
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
