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

package mysql

import (
	"sync"

	"gorm.io/driver/mysql"

	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/store/dbcommon"
)

var (
	mysqlPool *dbcommon.ConnectionPool
	mysqlOnce sync.Once
	poolMutex sync.RWMutex
)

// GetOrCreateMySQLPool returns or creates a MySQL connection pool
func GetOrCreateMySQLPool(address string, config *dbcommon.ConnectionPoolConfig) (*dbcommon.ConnectionPool, error) {
	poolMutex.Lock()
	defer poolMutex.Unlock()

	if mysqlPool != nil && mysqlPool.Address() == address {
		// Increment reference count when reusing existing pool
		mysqlPool.IncrementRef()
		logger.Infof("Reusing MySQL connection pool: address=%s, refCount=%d", address, mysqlPool.RefCount())
		return mysqlPool, nil
	}

	var initErr error
	mysqlOnce.Do(func() {
		if config == nil {
			config = dbcommon.DefaultConnectionPoolConfig()
		}

		pool, err := dbcommon.NewConnectionPool(mysql.Open(address), "mysql", address, config)
		if err != nil {
			initErr = err
			return
		}

		mysqlPool = pool
		logger.Infof("MySQL connection pool created successfully: address=%s, maxIdleConns=%d, maxOpenConns=%d",
			address, config.MaxIdleConns, config.MaxOpenConns)
	})

	if initErr != nil {
		mysqlOnce = sync.Once{} // Reset once to allow retry
		return nil, initErr
	}

	return mysqlPool, nil
}
