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
	"fmt"

	"gorm.io/gorm"

	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

func init() {
	RegisterLockFactory(&gormLockFactory{})
}

type gormLockFactory struct{}

// Support checks if a GORM Lock can be created from the context.
func (f *gormLockFactory) Support(ctx runtime.BuilderContext) bool {
	storeComp, err := ctx.GetActivatedComponent(runtime.ResourceStore)
	if err != nil {
		return false
	}

	type DataStoreProvider interface {
		GetDataStore() any
	}

	provider, ok := storeComp.(DataStoreProvider)
	if !ok {
		return false
	}

	dataStore := provider.GetDataStore()
	if dataStore == nil {
		return false
	}

	_, ok = dataStore.(*gorm.DB)
	return ok
}

// NewLock creates a GORM Lock instance
func (f *gormLockFactory) NewLock(ctx runtime.BuilderContext) (Lock, error) {
	storeComp, err := ctx.GetActivatedComponent(runtime.ResourceStore)
	if err != nil {
		return nil, fmt.Errorf("store component not found: %w", err)
	}

	type DataStoreProvider interface {
		GetDataStore() any
	}

	provider, ok := storeComp.(DataStoreProvider)
	if !ok {
		return nil, fmt.Errorf("store does not provide data store interface")
	}

	dataStore := provider.GetDataStore()
	if dataStore == nil {
		return nil, fmt.Errorf("data store is nil")
	}

	db, ok := dataStore.(*gorm.DB)
	if !ok {
		return nil, fmt.Errorf("data store is not *gorm.DB (got %T)", dataStore)
	}

	logger.Info("Creating GORM-based distributed lock")
	return NewGormLockFromDB(db), nil
}
