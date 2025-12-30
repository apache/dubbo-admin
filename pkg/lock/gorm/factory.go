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

package gorm

import (
	"fmt"

	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/store/dbcommon"
)

func init() {
	lock.RegisterLockFactory(&gormLockFactory{})
}

type gormLockFactory struct{}

// Support checks if GORM-based lock is supported based on store configuration
func (f *gormLockFactory) Support(ctx runtime.BuilderContext) bool {
	cfg := ctx.Config().Store
	// GORM lock is supported for database-backed stores (mysql, postgres)
	return cfg.Type == "mysql" || cfg.Type == "postgres"
}

// NewLock creates a GORM Lock instance by obtaining DB from dbcommon package
func (f *gormLockFactory) NewLock(ctx runtime.BuilderContext) (lock.Lock, error) {
	cfg := ctx.Config().Store

	// Get the database connection from dbcommon's global connection pool
	// This reuses the existing connection pool created by the store
	// but accesses it through the dbcommon package instead of StoreComponent
	db := dbcommon.GetGlobalDB(cfg.Type)
	if db == nil {
		return nil, fmt.Errorf("no database connection found for store type: %s", cfg.Type)
	}

	// Auto-migrate lock table
	if err := db.AutoMigrate(&LockRecord{}); err != nil {
		return nil, fmt.Errorf("failed to migrate lock table: %w", err)
	}

	logger.Info("Creating GORM-based distributed lock using existing database connection")
	return NewGormLockFromDB(db), nil
}
