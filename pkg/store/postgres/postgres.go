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
	"gorm.io/driver/postgres"

	storecfg "github.com/apache/dubbo-admin/pkg/config/store"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/store/dbcommon"
)

func init() {
	store.RegisterFactory(&postgresStoreFactory{})
}

// postgresStoreFactory is the factory for creating PostgreSQL store instances
type postgresStoreFactory struct{}

var _ store.Factory = &postgresStoreFactory{}

// Support checks if this factory supports the given store type
func (f *postgresStoreFactory) Support(s storecfg.Type) bool {
	return s == storecfg.Postgres
}

// New creates a new PostgreSQL store instance for the specified resource kind
func (f *postgresStoreFactory) New(kind model.ResourceKind, cfg *storecfg.Config) (store.ManagedResourceStore, error) {
	// Get or create connection pool with PostgreSQL dialector
	pool, err := dbcommon.GetOrCreatePool(
		postgres.Open(cfg.Address),
		storecfg.Postgres,
		cfg.Address,
		dbcommon.DefaultConnectionPoolConfig(),
	)
	if err != nil {
		return nil, err
	}

	// Create GormStore with the pool
	return dbcommon.NewGormStore(kind, cfg.Address, pool), nil
}
