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

package repository

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/store/memory"
)

func TestServerDraftPublishIsolation(t *testing.T) {
	serverStore := memory.NewMemoryResourceStore(meshresource.MCPServerKind)
	require.NoError(t, serverStore.Init(nil))
	resources := manager.NewResourceManager(credentialTestRouter{stores: map[coremodel.ResourceKind]store.ResourceStore{
		meshresource.MCPServerKind: serverStore,
	}}, nil)
	repo, err := NewServers(resources)
	require.NoError(t, err)

	draft := &meshproto.MCPServerSnapshot{Description: "original"}
	created, err := repo.Create("mesh-1", "server-1", draft)
	require.NoError(t, err)
	require.Equal(t, "1", created.ResourceVersion)
	draft.Description = "changed after create"
	stored, err := repo.Get("mesh-1", "server-1")
	require.NoError(t, err)
	require.Equal(t, "original", stored.Spec.Draft.Description)
	require.Nil(t, stored.Spec.Published)

	published, err := repo.Publish("mesh-1", "server-1", "1", func(snapshot *meshproto.MCPServerSnapshot) error {
		snapshot.DisplayName = "compiled"
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), published.Spec.Published.Revision)
	require.Equal(t, "compiled", published.Spec.Published.Snapshot.DisplayName)
	require.Empty(t, published.Spec.Draft.DisplayName)

	updated, err := repo.UpdateDraft("mesh-1", "server-1", "2", &meshproto.MCPServerSnapshot{Description: "new draft"})
	require.NoError(t, err)
	require.Equal(t, "3", updated.ResourceVersion)
	require.Equal(t, "original", updated.Spec.Published.Snapshot.Description)
	_, err = repo.Publish("mesh-1", "server-1", "3", func(*meshproto.MCPServerSnapshot) error { return fmt.Errorf("invalid contract") })
	require.EqualError(t, err, "invalid contract")
	afterFailure, err := repo.Get("mesh-1", "server-1")
	require.NoError(t, err)
	require.Equal(t, "3", afterFailure.ResourceVersion)
	require.Equal(t, int64(1), afterFailure.Spec.Published.Revision)
	require.Equal(t, "original", afterFailure.Spec.Published.Snapshot.Description)

	_, err = repo.Publish("mesh-1", "server-1", "2", func(*meshproto.MCPServerSnapshot) error { return nil })
	require.True(t, errors.Is(err, store.ErrorResourceConflict("", "", "")))
}

func TestMemoryServerDeleteCascadesCredentials(t *testing.T) {
	serverStore := memory.NewMemoryResourceStore(meshresource.MCPServerKind)
	credentialStore := memory.NewMemoryResourceStore(meshresource.MCPCredentialKind)
	require.NoError(t, serverStore.Init(nil))
	require.NoError(t, credentialStore.Init(nil))
	resources := manager.NewResourceManager(credentialTestRouter{stores: map[coremodel.ResourceKind]store.ResourceStore{
		meshresource.MCPServerKind: serverStore, meshresource.MCPCredentialKind: credentialStore,
	}}, nil)
	servers, err := NewServers(resources)
	require.NoError(t, err)
	credentials, err := NewCredentials(resources)
	require.NoError(t, err)
	first, err := servers.Create("mesh-1", "first", &meshproto.MCPServerSnapshot{})
	require.NoError(t, err)
	_, err = servers.Create("mesh-1", "second", &meshproto.MCPServerSnapshot{})
	require.NoError(t, err)
	expiry := time.Now().Add(time.Hour)
	for i := 0; i < 2; i++ {
		_, _, err = credentials.Create("mesh-1", "first", "agent", expiry)
		require.NoError(t, err)
	}
	_, _, err = credentials.Create("mesh-1", "second", "agent", expiry)
	require.NoError(t, err)
	require.ErrorIs(t, servers.Delete("mesh-1", "first", "2"), store.ErrorResourceConflict("", "", ""))
	listed, err := credentials.List("mesh-1", "first")
	require.NoError(t, err)
	require.Len(t, listed, 2)
	require.NoError(t, servers.Delete("mesh-1", "first", first.ResourceVersion))
	listed, err = credentials.List("mesh-1", "first")
	require.NoError(t, err)
	require.Empty(t, listed)
	listed, err = credentials.List("mesh-1", "second")
	require.NoError(t, err)
	require.Len(t, listed, 1)
	_, _, err = credentials.Create("mesh-1", "first", "late", expiry)
	require.Error(t, err)
}

func TestConcurrentPublishHasOneWinner(t *testing.T) {
	serverStore := memory.NewMemoryResourceStore(meshresource.MCPServerKind)
	require.NoError(t, serverStore.Init(nil))
	resources := manager.NewResourceManager(credentialTestRouter{stores: map[coremodel.ResourceKind]store.ResourceStore{
		meshresource.MCPServerKind: serverStore,
	}}, nil)
	repo, err := NewServers(resources)
	require.NoError(t, err)
	_, err = repo.Create("mesh-1", "server-1", &meshproto.MCPServerSnapshot{Description: "ready"})
	require.NoError(t, err)

	results := make(chan error, 2)
	var started sync.WaitGroup
	started.Add(2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() {
			started.Done()
			<-start
			_, err := repo.Publish("mesh-1", "server-1", "1", func(*meshproto.MCPServerSnapshot) error { return nil })
			results <- err
		}()
	}
	started.Wait()
	close(start)
	var successes, conflicts int
	for i := 0; i < 2; i++ {
		if err := <-results; err == nil {
			successes++
		} else if errors.Is(err, store.ErrorResourceConflict("", "", "")) {
			conflicts++
		} else {
			t.Fatalf("unexpected publish error: %v", err)
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, conflicts)
}
