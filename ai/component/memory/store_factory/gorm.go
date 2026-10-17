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

package storefactory

import (
	"context"
	"fmt"
	"io"

	conversationstore "dubbo-admin-ai/store"
	gormstore "dubbo-admin-ai/store/gorm"
	"gorm.io/gorm"
)

func openGorm(config Config) (conversationstore.Store, io.Closer, error) {
	if config.Database == nil {
		return nil, nil, fmt.Errorf("database is required for gorm backend")
	}
	db, err := gormstore.Open(config.Database.Driver, config.Database.DSN, &gorm.Config{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open conversation database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to access conversation database: %w", err)
	}
	sqlDB.SetMaxOpenConns(config.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.Database.MaxIdleConns)
	store, err := gormstore.NewGormStore(db, config.MaxTurns)
	if err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("failed to create gorm conversation store: %w", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		_ = store.Close()
		return nil, nil, fmt.Errorf("failed to migrate conversation database: %w", err)
	}
	return store, store, nil
}
