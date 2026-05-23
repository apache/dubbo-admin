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

package versioning

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

func setupGormVersionStore(t *testing.T) *GormStore {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	store := NewGormStore(db)
	require.NoError(t, store.AutoMigrate())
	require.NoError(t, store.AutoMigrate())
	return store
}

func TestGormStoreAutoMigrateSpecJSONUsesPortableText(t *testing.T) {
	store := setupGormVersionStore(t)

	columns, err := store.db.Migrator().ColumnTypes(&Version{})
	require.NoError(t, err)
	for _, column := range columns {
		if column.Name() != "spec_json" {
			continue
		}
		require.Equal(t, "text", strings.ToLower(column.DatabaseTypeName()))
		return
	}
	require.Fail(t, "spec_json column was not migrated")
}

func TestGormStoreInsertListGetAndTrim(t *testing.T) {
	store := setupGormVersionStore(t)
	key := "mesh/demo.condition-router"
	for i := 0; i < 4; i++ {
		_, err := store.InsertVersion(InsertRequest{
			RuleKind:    meshresource.ConditionRouteKind,
			Mesh:        "mesh",
			ResourceKey: key,
			RuleName:    "demo.condition-router",
			SpecJSON:    fmt.Sprintf(`{"priority":%d}`, i+1),
			ContentHash: fmt.Sprintf("hash-%d", i+1),
			Source:      SourceAdmin,
			Operation:   OperationUpdate,
			Author:      "alice",
			CreatedAt:   time.Now().Add(time.Duration(i) * time.Second),
		}, 2)
		require.NoError(t, err)
	}
	items, err := store.ListVersions(meshresource.ConditionRouteKind, key)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, int64(4), items[0].VersionNo)
	require.True(t, items[0].IsCurrent)
	_, err = store.GetVersion(meshresource.ConditionRouteKind, key, items[1].ID)
	require.NoError(t, err)
	meta, err := store.CurrentMeta(meshresource.ConditionRouteKind, key)
	require.NoError(t, err)
	require.Equal(t, int64(4), meta.LastVersionNo)

	_, err = store.InsertVersion(InsertRequest{
		RuleKind:    meshresource.ConditionRouteKind,
		Mesh:        "mesh",
		ResourceKey: key,
		RuleName:    "demo.condition-router",
		SpecJSON:    `{"priority":5}`,
		ContentHash: "hash-5",
		Source:      SourceAdmin,
		Operation:   OperationUpdate,
		Author:      "alice",
		CreatedAt:   time.Now().Add(5 * time.Second),
	}, 2)
	require.NoError(t, err)
	meta, err = store.CurrentMeta(meshresource.ConditionRouteKind, key)
	require.NoError(t, err)
	require.Equal(t, int64(5), meta.LastVersionNo)
}

func TestGormStoreDeleteIsNotDedupedAgainstEmptyCurrentSpec(t *testing.T) {
	store := setupGormVersionStore(t)
	key := "mesh/demo.condition-router"
	created, err := store.InsertVersion(InsertRequest{
		RuleKind:    meshresource.ConditionRouteKind,
		Mesh:        "mesh",
		ResourceKey: key,
		RuleName:    "demo.condition-router",
		SpecJSON:    DeleteSpecJSON,
		ContentHash: HashSpecJSON(DeleteSpecJSON),
		Source:      SourceAdmin,
		Operation:   OperationCreate,
		Author:      "alice",
		CreatedAt:   time.Now(),
	}, 5)
	require.NoError(t, err)
	deleted, err := store.InsertVersion(InsertRequest{
		RuleKind:    meshresource.ConditionRouteKind,
		Mesh:        "mesh",
		ResourceKey: key,
		RuleName:    "demo.condition-router",
		SpecJSON:    DeleteSpecJSON,
		ContentHash: HashSpecJSON(DeleteSpecJSON),
		Source:      SourceAdmin,
		Operation:   OperationDelete,
		Author:      "alice",
		CreatedAt:   time.Now().Add(time.Second),
	}, 5)
	require.NoError(t, err)

	require.NotEqual(t, created.ID, deleted.ID)
	require.Equal(t, int64(2), deleted.VersionNo)
	items, err := store.ListVersions(meshresource.ConditionRouteKind, key)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, OperationDelete, items[0].Operation)
	meta, err := store.CurrentMeta(meshresource.ConditionRouteKind, key)
	require.NoError(t, err)
	require.Nil(t, meta.CurrentVersion)
}

func TestGormStoreIntentCommit(t *testing.T) {
	store := setupGormVersionStore(t)
	key := "mesh/demo.condition-router"
	expected := int64(7)
	intent, err := store.CreateIntent(InsertRequest{
		RuleKind:    meshresource.ConditionRouteKind,
		Mesh:        "mesh",
		ResourceKey: key,
		RuleName:    "demo.condition-router",
		SpecJSON:    `{"priority":1}`,
		ContentHash: "hash-1",
		Source:      SourceAdmin,
		Operation:   OperationUpdate,
		Author:      "alice",
		Reason:      "admin edit",
		CreatedAt:   time.Now(),
	}, &expected)
	require.NoError(t, err)
	require.Equal(t, IntentStatusPending, intent.Status)

	open, err := store.FindOpenIntentByHash(meshresource.ConditionRouteKind, key, "hash-1")
	require.NoError(t, err)
	require.NotNil(t, open)
	require.Equal(t, expected, *open.ExpectedVersionID)

	require.NoError(t, store.MarkIntentApplied(intent.ID))
	version, err := store.CommitIntent(intent.ID, 5)
	require.NoError(t, err)
	require.Equal(t, SourceAdmin, version.Source)
	require.True(t, version.IsCurrent)

	open, err = store.OpenIntent(meshresource.ConditionRouteKind, key)
	require.NoError(t, err)
	require.Nil(t, open)
	committed, err := store.CommitIntent(intent.ID, 5)
	require.NoError(t, err)
	require.Equal(t, version.ID, committed.ID)
}

func TestGormStoreIntentGetListOpenAndFailWithReason(t *testing.T) {
	store := setupGormVersionStore(t)
	key := "mesh/demo.condition-router"
	intent, err := store.CreateIntent(InsertRequest{
		RuleKind:    meshresource.ConditionRouteKind,
		Mesh:        "mesh",
		ResourceKey: key,
		RuleName:    "demo.condition-router",
		SpecJSON:    `{"priority":1}`,
		ContentHash: "hash-1",
		Source:      SourceAdmin,
		Operation:   OperationUpdate,
		Author:      "alice",
		CreatedAt:   time.Now(),
	}, nil)
	require.NoError(t, err)

	got, err := store.GetIntent(intent.ID)
	require.NoError(t, err)
	require.Equal(t, IntentStatusPending, got.Status)
	open, err := store.ListOpenIntents()
	require.NoError(t, err)
	require.Len(t, open, 1)
	require.Equal(t, intent.ID, open[0].ID)

	require.NoError(t, store.MarkIntentFailedWithReason(intent.ID, "registry rejected mutation"))
	got, err = store.GetIntent(intent.ID)
	require.NoError(t, err)
	require.Equal(t, IntentStatusFailed, got.Status)
	require.Equal(t, "registry rejected mutation", got.LastError)
	open, err = store.ListOpenIntents()
	require.NoError(t, err)
	require.Empty(t, open)
	require.ErrorIs(t, store.MarkIntentFailedWithReason(intent.ID, "again"), ErrVersionIntentNotOpen)
	_, err = store.GetIntent(404)
	require.ErrorIs(t, err, ErrVersionIntentNotFound)
}

func TestGormStoreMetaCounterConcurrencyMonotonic(t *testing.T) {
	store := setupGormVersionStore(t)
	key := "mesh/concurrent.condition-router"
	var wg sync.WaitGroup
	errCh := make(chan error, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := store.InsertVersion(InsertRequest{
				RuleKind:    meshresource.ConditionRouteKind,
				Mesh:        "mesh",
				ResourceKey: key,
				RuleName:    "concurrent.condition-router",
				SpecJSON:    fmt.Sprintf(`{"priority":%d}`, i),
				ContentHash: fmt.Sprintf("hash-concurrent-%d", i),
				Source:      SourceAdmin,
				Operation:   OperationUpdate,
				Author:      "alice",
				CreatedAt:   time.Now().Add(time.Duration(i) * time.Millisecond),
			}, 10)
			errCh <- err
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}
	items, err := store.ListVersions(meshresource.ConditionRouteKind, key)
	require.NoError(t, err)
	require.Len(t, items, 6)
	seen := map[int64]bool{}
	for _, item := range items {
		require.False(t, seen[item.VersionNo])
		seen[item.VersionNo] = true
	}
	require.Len(t, seen, 6)
}
