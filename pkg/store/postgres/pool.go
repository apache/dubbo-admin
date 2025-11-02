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

package postgres

import (
	"sync"

	"gorm.io/driver/postgres"

	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/store/dbcommon"
)

var (
	postgresPool *dbcommon.ConnectionPool
	postgresOnce sync.Once
	poolMutex    sync.RWMutex
)

// GetOrCreatePostgresPool returns or creates a PostgreSQL connection pool
func GetOrCreatePostgresPool(address string, config *dbcommon.ConnectionPoolConfig) (*dbcommon.ConnectionPool, error) {
	poolMutex.Lock()
	defer poolMutex.Unlock()

	if postgresPool != nil && postgresPool.Address() == address {
		// Increment reference count when reusing existing pool
		postgresPool.IncrementRef()
		logger.Infof("Reusing PostgreSQL connection pool: address=%s, refCount=%d", address, postgresPool.RefCount())
		return postgresPool, nil
	}

	var initErr error
	postgresOnce.Do(func() {
		if config == nil {
			config = dbcommon.DefaultConnectionPoolConfig()
		}

		pool, err := dbcommon.NewConnectionPool(postgres.Open(address), "postgres", address, config)
		if err != nil {
			initErr = err
			return
		}

		postgresPool = pool
		logger.Infof("PostgreSQL connection pool created successfully: address=%s, maxIdleConns=%d, maxOpenConns=%d",
			address, config.MaxIdleConns, config.MaxOpenConns)
	})

	if initErr != nil {
		postgresOnce = sync.Once{} // Reset once to allow retry
		return nil, initErr
	}

	return postgresPool, nil
}
