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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"

	storecfg "github.com/apache/dubbo-admin/pkg/config/store"
)

func TestDefaultConnectionPoolConfig(t *testing.T) {
	config := DefaultConnectionPoolConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 10, config.MaxIdleConns)
	assert.Equal(t, 100, config.MaxOpenConns)
	assert.Equal(t, time.Hour, config.ConnMaxLifetime)
	assert.Equal(t, 10*time.Minute, config.ConnMaxIdleTime)
}

func TestNewConnectionPool(t *testing.T) {
	dialector := sqlite.Open("file::memory:?cache=shared")
	config := DefaultConnectionPoolConfig()

	pool, err := NewConnectionPool(dialector, storecfg.MySQL, "test-address", config)

	assert.NoError(t, err)
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.db)
	assert.NotNil(t, pool.sqlDB)
	assert.Equal(t, "test-address", pool.address)
	assert.Equal(t, storecfg.MySQL, pool.storeType)
	assert.Equal(t, 1, pool.refCount)
	assert.False(t, pool.closed)

	// Cleanup
	if err := pool.Close(); err != nil {
		return
	}
}

func TestConnectionPool_GetDB(t *testing.T) {
	dialector := sqlite.Open("file::memory:?cache=shared")
	pool, err := NewConnectionPool(dialector, storecfg.MySQL, "test-address", DefaultConnectionPoolConfig())
	require.NoError(t, err)
	defer func(pool *ConnectionPool) {
		err := pool.Close()
		if err != nil {
			return
		}
	}(pool)

	db := pool.GetDB()
	assert.NotNil(t, db)
}

func TestConnectionPool_Address(t *testing.T) {
	dialector := sqlite.Open("file::memory:?cache=shared")
	pool, err := NewConnectionPool(dialector, storecfg.MySQL, "test-address-123", DefaultConnectionPoolConfig())
	require.NoError(t, err)
	defer func(pool *ConnectionPool) {
		err := pool.Close()
		if err != nil {
			return
		}
	}(pool)

	assert.Equal(t, "test-address-123", pool.Address())
}

func TestConnectionPool_RefCount(t *testing.T) {
	dialector := sqlite.Open("file::memory:?cache=shared")
	pool, err := NewConnectionPool(dialector, storecfg.MySQL, "test-address", DefaultConnectionPoolConfig())
	require.NoError(t, err)
	defer func(pool *ConnectionPool) {
		err := pool.Close()
		if err != nil {
			return
		}
	}(pool)

	// Initial ref count should be 1
	assert.Equal(t, 1, pool.RefCount())

	// Increment ref count
	pool.IncrementRef()
	assert.Equal(t, 2, pool.RefCount())

	pool.IncrementRef()
	assert.Equal(t, 3, pool.RefCount())
}

func TestConnectionPool_IncrementRef(t *testing.T) {
	dialector := sqlite.Open("file::memory:?cache=shared")
	pool, err := NewConnectionPool(dialector, storecfg.MySQL, "test-address", DefaultConnectionPoolConfig())
	require.NoError(t, err)
	defer func(pool *ConnectionPool) {
		err := pool.Close()
		if err != nil {
			return
		}
	}(pool)

	initialRefCount := pool.RefCount()
	pool.IncrementRef()
	assert.Equal(t, initialRefCount+1, pool.RefCount())
}

func TestConnectionPool_Close(t *testing.T) {
	dialector := sqlite.Open("file::memory:?cache=shared")
	pool, err := NewConnectionPool(dialector, storecfg.MySQL, "test-address", DefaultConnectionPoolConfig())
	require.NoError(t, err)

	// First close should decrement ref count
	err = pool.Close()
	assert.NoError(t, err)
	assert.Equal(t, 0, pool.refCount)
	assert.True(t, pool.closed)

	// Second close should be a no-op
	err = pool.Close()
	assert.NoError(t, err)
	assert.Equal(t, 0, pool.refCount)
}

func TestConnectionPool_CloseWithMultipleReferences(t *testing.T) {
	dialector := sqlite.Open("file::memory:?cache=shared")
	pool, err := NewConnectionPool(dialector, storecfg.MySQL, "test-address", DefaultConnectionPoolConfig())
	require.NoError(t, err)

	// Increment ref count
	pool.IncrementRef()
	pool.IncrementRef()
	assert.Equal(t, 3, pool.RefCount())

	// First close should just decrement
	err = pool.Close()
	assert.NoError(t, err)
	assert.Equal(t, 2, pool.RefCount())
	assert.False(t, pool.closed)

	// Second close should just decrement
	err = pool.Close()
	assert.NoError(t, err)
	assert.Equal(t, 1, pool.RefCount())
	assert.False(t, pool.closed)

	// Third close should actually close
	err = pool.Close()
	assert.NoError(t, err)
	assert.Equal(t, 0, pool.RefCount())
	assert.True(t, pool.closed)
}

func TestConnectionPool_Ping(t *testing.T) {
	dialector := sqlite.Open("file::memory:?cache=shared")
	pool, err := NewConnectionPool(dialector, storecfg.MySQL, "test-address", DefaultConnectionPoolConfig())
	require.NoError(t, err)
	defer func(pool *ConnectionPool) {
		err := pool.Close()
		if err != nil {
			return
		}
	}(pool)

	err = pool.Ping()
	assert.NoError(t, err)
}

func TestConnectionPool_Stats(t *testing.T) {
	dialector := sqlite.Open("file::memory:?cache=shared")
	config := &ConnectionPoolConfig{
		MaxIdleConns:    5,
		MaxOpenConns:    10,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,
	}
	pool, err := NewConnectionPool(dialector, storecfg.MySQL, "test-address", config)
	require.NoError(t, err)
	defer func(pool *ConnectionPool) {
		err := pool.Close()
		if err != nil {
			return
		}
	}(pool)

	stats := pool.Stats()
	assert.NotNil(t, stats)
	// The max values should match configuration
	assert.Equal(t, 10, stats.MaxOpenConnections)
}

func TestGetOrCreatePool_CreateNew(t *testing.T) {
	// Clear the pools map for isolated test
	poolsMutex.Lock()
	pools = make(map[string]*ConnectionPool)
	poolsMutex.Unlock()

	dialector := sqlite.Open("file::memory:?cache=shared")
	pool, err := GetOrCreatePool(dialector, storecfg.MySQL, "test-address", DefaultConnectionPoolConfig())

	assert.NoError(t, err)
	assert.NotNil(t, pool)
	assert.Equal(t, 1, pool.RefCount())

	// Cleanup
	if err := pool.Close(); err != nil {
		return
	}
}

func TestGetOrCreatePool_ReuseExisting(t *testing.T) {
	// Clear the pools map for isolated test
	poolsMutex.Lock()
	pools = make(map[string]*ConnectionPool)
	poolsMutex.Unlock()

	dialector := sqlite.Open("file::memory:?cache=shared")

	// Create first pool
	pool1, err := GetOrCreatePool(dialector, storecfg.MySQL, "test-address", DefaultConnectionPoolConfig())
	require.NoError(t, err)
	assert.Equal(t, 1, pool1.RefCount())

	// Get same pool again
	pool2, err := GetOrCreatePool(dialector, storecfg.MySQL, "test-address", DefaultConnectionPoolConfig())
	require.NoError(t, err)

	// Should be the same pool with incremented ref count
	assert.Equal(t, pool1, pool2)
	assert.Equal(t, 2, pool1.RefCount())

	// Cleanup
	pool1.Close()
	pool2.Close()
}

func TestGetOrCreatePool_DifferentAddresses(t *testing.T) {
	// Clear the pools map for isolated test
	poolsMutex.Lock()
	pools = make(map[string]*ConnectionPool)
	poolsMutex.Unlock()

	dialector1 := sqlite.Open("file::memory:?cache=shared")
	dialector2 := sqlite.Open("file::memory:?cache=shared")

	// Create pool for address 1
	pool1, err := GetOrCreatePool(dialector1, storecfg.MySQL, "address-1", DefaultConnectionPoolConfig())
	require.NoError(t, err)

	// Create pool for address 2
	pool2, err := GetOrCreatePool(dialector2, storecfg.MySQL, "address-2", DefaultConnectionPoolConfig())
	require.NoError(t, err)

	// Should be different pools
	assert.NotEqual(t, pool1, pool2)
	assert.Equal(t, "address-1", pool1.Address())
	assert.Equal(t, "address-2", pool2.Address())

	// Cleanup
	pool1.Close()
	pool2.Close()
}

func TestGetOrCreatePool_DifferentStoreTypes(t *testing.T) {
	// Clear the pools map for isolated test
	poolsMutex.Lock()
	pools = make(map[string]*ConnectionPool)
	poolsMutex.Unlock()

	dialector1 := sqlite.Open("file::memory:?cache=shared")
	dialector2 := sqlite.Open("file::memory:?cache=shared")

	// Create MySQL pool
	pool1, err := GetOrCreatePool(dialector1, storecfg.MySQL, "test-address", DefaultConnectionPoolConfig())
	require.NoError(t, err)

	// Create Postgres pool with same address
	pool2, err := GetOrCreatePool(dialector2, storecfg.Postgres, "test-address", DefaultConnectionPoolConfig())
	require.NoError(t, err)

	// Should be different pools (different store types)
	assert.NotEqual(t, pool1, pool2)
	assert.Equal(t, storecfg.MySQL, pool1.storeType)
	assert.Equal(t, storecfg.Postgres, pool2.storeType)

	// Cleanup
	pool1.Close()
	pool2.Close()
}

func TestGetOrCreatePool_MemoryStoreError(t *testing.T) {
	dialector := sqlite.Open("file::memory:?cache=shared")

	pool, err := GetOrCreatePool(dialector, storecfg.Memory, "test-address", DefaultConnectionPoolConfig())

	assert.Error(t, err)
	assert.Nil(t, pool)
	assert.Contains(t, err.Error(), "memory pool store is no need to create connection pool")
}

func TestRemovePool(t *testing.T) {
	// Clear the pools map for isolated test
	poolsMutex.Lock()
	pools = make(map[string]*ConnectionPool)
	poolsMutex.Unlock()

	dialector := sqlite.Open("file::memory:?cache=shared")
	pool, err := GetOrCreatePool(dialector, storecfg.MySQL, "test-address", DefaultConnectionPoolConfig())
	require.NoError(t, err)

	// Verify pool exists in registry
	poolKey := "mysql:test-address"
	poolsMutex.RLock()
	_, exists := pools[poolKey]
	poolsMutex.RUnlock()
	assert.True(t, exists, "Pool should exist in registry after creation")

	// Close pool (which should call RemovePool internally)
	err = pool.Close()
	require.NoError(t, err)

	// Verify pool was removed from registry
	poolsMutex.RLock()
	_, exists = pools[poolKey]
	poolsMutex.RUnlock()
	assert.False(t, exists, "Pool should be removed from registry after close")
}

func TestConnectionPool_ConcurrentAccess(t *testing.T) {
	dialector := sqlite.Open("file::memory:?cache=shared")
	pool, err := NewConnectionPool(dialector, storecfg.MySQL, "test-address", DefaultConnectionPoolConfig())
	require.NoError(t, err)
	defer pool.Close()

	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrently access pool methods
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			// Test concurrent reads
			_ = pool.GetDB()
			_ = pool.Address()
			_ = pool.RefCount()
			_ = pool.Stats()

			// Test concurrent ping
			_ = pool.Ping()
		}()
	}

	wg.Wait()
}

func TestConnectionPool_ConcurrentIncrement(t *testing.T) {
	dialector := sqlite.Open("file::memory:?cache=shared")
	pool, err := NewConnectionPool(dialector, storecfg.MySQL, "test-address", DefaultConnectionPoolConfig())
	require.NoError(t, err)

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrently increment ref count
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			pool.IncrementRef()
		}()
	}

	wg.Wait()

	// Verify ref count (initial 1 + 100 increments)
	assert.Equal(t, 101, pool.RefCount())

	// Cleanup
	for i := 0; i <= numGoroutines; i++ {
		pool.Close()
	}
}

func TestGetOrCreatePool_ConcurrentCreation(t *testing.T) {
	// Clear the pools map for isolated test
	poolsMutex.Lock()
	pools = make(map[string]*ConnectionPool)
	poolsMutex.Unlock()

	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	createdPools := make([]*ConnectionPool, numGoroutines)

	// Concurrently try to create/get the same pool
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			dialector := sqlite.Open("file::memory:?cache=shared")
			pool, err := GetOrCreatePool(dialector, storecfg.MySQL, "concurrent-address", DefaultConnectionPoolConfig())
			assert.NoError(t, err)
			createdPools[index] = pool
		}(i)
	}

	wg.Wait()

	// All goroutines should get the same pool
	firstPool := createdPools[0]
	for i := 1; i < numGoroutines; i++ {
		assert.Equal(t, firstPool, createdPools[i])
	}

	// Ref count should be numGoroutines
	assert.Equal(t, numGoroutines, firstPool.RefCount())

	// Cleanup
	for i := 0; i < numGoroutines; i++ {
		firstPool.Close()
	}
}

func TestConnectionPool_CustomConfig(t *testing.T) {
	dialector := sqlite.Open("file::memory:?cache=shared")
	config := &ConnectionPoolConfig{
		MaxIdleConns:    5,
		MaxOpenConns:    20,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}

	pool, err := NewConnectionPool(dialector, storecfg.MySQL, "test-address", config)
	require.NoError(t, err)
	defer pool.Close()

	stats := pool.Stats()
	assert.Equal(t, 20, stats.MaxOpenConnections)
}

func TestConnectionPool_NilConfig(t *testing.T) {
	// Clear the pools map for isolated test
	poolsMutex.Lock()
	pools = make(map[string]*ConnectionPool)
	poolsMutex.Unlock()

	dialector := sqlite.Open("file::memory:?cache=shared")

	// Pass nil config - should use default
	pool, err := GetOrCreatePool(dialector, storecfg.MySQL, "test-address", nil)
	require.NoError(t, err)
	defer pool.Close()

	assert.NotNil(t, pool)
	stats := pool.Stats()
	// Should have default max open connections (100)
	assert.Equal(t, 100, stats.MaxOpenConnections)
}

func TestConnectionPool_MultipleCloseIdempotent(t *testing.T) {
	dialector := sqlite.Open("file::memory:?cache=shared")
	pool, err := NewConnectionPool(dialector, storecfg.MySQL, "test-address", DefaultConnectionPoolConfig())
	require.NoError(t, err)

	// Close multiple times
	err = pool.Close()
	assert.NoError(t, err)

	err = pool.Close()
	assert.NoError(t, err)

	err = pool.Close()
	assert.NoError(t, err)

	// Should still be closed
	assert.True(t, pool.closed)
	assert.Equal(t, 0, pool.refCount)
}
