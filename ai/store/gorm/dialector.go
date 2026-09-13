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

package gormstore

import (
	"fmt"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Open opens a Gorm database using one of the supported driver names.
// SQLite is supported for local and test stores; production configuration can
// restrict the accepted drivers separately.
func Open(driver, dsn string, config *gorm.Config) (*gorm.DB, error) {
	if strings.TrimSpace(driver) == "" {
		return nil, fmt.Errorf("database driver is required")
	}
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("database dsn is required")
	}
	if config == nil {
		config = &gorm.Config{}
	}
	config.DisableForeignKeyConstraintWhenMigrating = true

	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "mysql":
		return gorm.Open(mysql.Open(dsn), config)
	case "postgres", "postgresql":
		return gorm.Open(postgres.Open(dsn), config)
	case "sqlite":
		return gorm.Open(sqlite.Open(dsn), config)
	default:
		return nil, fmt.Errorf("unsupported database driver %q", driver)
	}
}
