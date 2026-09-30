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
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	storecfg "github.com/apache/dubbo-admin/pkg/config/store"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

func setupConditionalStore(t *testing.T) *GormStore {
	t.Helper()
	file, err := os.CreateTemp("", fmt.Sprintf("mcp-cas-%s-*.db", t.Name()))
	require.NoError(t, err)
	require.NoError(t, file.Close())

	pool, err := NewConnectionPool(
		sqlite.Open(file.Name()),
		storecfg.MySQL,
		t.Name(),
		DefaultConnectionPoolConfig(),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, pool.Close())
		require.NoError(t, os.Remove(file.Name()))
	})

	result := NewGormStore(meshresource.MCPServerKind, t.Name(), pool)
	require.NoError(t, result.Init(nil))
	return result
}

func TestGormConditionalMutations(t *testing.T) {
	managed := setupConditionalStore(t)
	conditional := any(managed).(store.ConditionalResourceStore)

	resource := meshresource.NewMCPServerResourceWithAttributes("server-1", "mesh-1")
	require.NoError(t, managed.Add(resource))
	require.Equal(t, "1", resource.ResourceVersion)

	updated := resource.DeepCopyObject().(*meshresource.MCPServerResource)
	updated.Spec.Draft = &meshproto.MCPServerSnapshot{DisplayName: "Orders"}
	require.NoError(t, conditional.CompareAndSwap(updated, "1"))
	require.Equal(t, "2", updated.ResourceVersion)

	stale := resource.DeepCopyObject().(*meshresource.MCPServerResource)
	err := conditional.CompareAndSwap(stale, "1")
	require.True(t, errors.Is(err, store.ErrorResourceConflict("", "", "")))

	item, exists, err := managed.GetByKey(updated.ResourceKey())
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "2", item.(*meshresource.MCPServerResource).ResourceVersion)
	require.Equal(t, "Orders", item.(*meshresource.MCPServerResource).Spec.Draft.DisplayName)
	listed := managed.List()
	require.Len(t, listed, 1)
	require.Equal(t, "2", listed[0].(*meshresource.MCPServerResource).ResourceVersion)

	require.NoError(t, conditional.CompareAndDelete(updated, "2"))
	_, exists, err = managed.GetByKey(updated.ResourceKey())
	require.NoError(t, err)
	require.False(t, exists)
}

func TestGormVersionColumnOnlyForVersionedResources(t *testing.T) {
	managed := setupConditionalStore(t)
	require.True(t, hasSQLiteColumn(t, managed, TableNameForKind(meshresource.MCPServerKind.ToString()), "version"))
	require.Error(t, managed.Replace(nil, ""))
	credentialStore := NewGormStore(meshresource.MCPCredentialKind, t.Name()+"-credential", managed.pool)
	require.NoError(t, credentialStore.Init(nil))
	require.True(t, hasSQLiteColumn(t, managed, TableNameForKind(meshresource.MCPCredentialKind.ToString()), "version"))

	ordinary := NewGormStore(meshresource.ApplicationKind, t.Name()+"-application", managed.pool)
	require.NoError(t, ordinary.Init(nil))
	require.False(t, hasSQLiteColumn(t, managed, TableNameForKind(meshresource.ApplicationKind.ToString()), "version"))
}

func TestGormCredentialRevokeDoesNotAffectOtherCredentials(t *testing.T) {
	managed := setupConditionalStore(t)
	credentials := NewGormStore(meshresource.MCPCredentialKind, t.Name()+"-credential", managed.pool)
	require.NoError(t, credentials.Init(nil))
	first := meshresource.NewMCPCredentialResourceWithAttributes("first", "mesh-1")
	first.Spec.ServerId = "server-1"
	second := meshresource.NewMCPCredentialResourceWithAttributes("second", "mesh-1")
	second.Spec.ServerId = "server-1"
	require.NoError(t, credentials.Add(first))
	require.NoError(t, credentials.Add(second))
	revoked := first.DeepCopyObject().(*meshresource.MCPCredentialResource)
	revoked.Spec.Status = "revoked"
	conditional := any(credentials).(store.ConditionalResourceStore)
	require.NoError(t, conditional.CompareAndSwap(revoked, "1"))
	require.Equal(t, "2", revoked.ResourceVersion)
	require.ErrorIs(t, conditional.CompareAndSwap(first, "1"), store.ErrorResourceConflict("", "", ""))
	item, exists, err := credentials.GetByKey(second.ResourceKey())
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "1", item.(*meshresource.MCPCredentialResource).ResourceVersion)
	require.Empty(t, item.(*meshresource.MCPCredentialResource).Spec.Status)
}

func hasSQLiteColumn(t *testing.T, managed *GormStore, table, name string) bool {
	t.Helper()
	var columns []struct {
		Name string `gorm:"column:name"`
	}
	require.NoError(t, managed.pool.GetDB().Raw("PRAGMA table_info("+table+")").Scan(&columns).Error)
	for _, column := range columns {
		if column.Name == name {
			return true
		}
	}
	return false
}
