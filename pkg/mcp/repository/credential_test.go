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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/store/memory"
)

type credentialTestRouter struct {
	stores map[coremodel.ResourceKind]store.ResourceStore
}

func (r credentialTestRouter) ResourceRoute(resource coremodel.Resource) (store.ResourceStore, error) {
	return r.ResourceKindRoute(resource.ResourceKind())
}

func (r credentialTestRouter) ResourceKindRoute(kind coremodel.ResourceKind) (store.ResourceStore, error) {
	return r.stores[kind], nil
}

func TestCredentialLifecycle(t *testing.T) {
	serverStore := memory.NewMemoryResourceStore(meshresource.MCPServerKind)
	credentialStore := memory.NewMemoryResourceStore(meshresource.MCPCredentialKind)
	require.NoError(t, serverStore.Init(nil))
	require.NoError(t, credentialStore.Init(nil))
	resources := manager.NewResourceManager(credentialTestRouter{stores: map[coremodel.ResourceKind]store.ResourceStore{
		meshresource.MCPServerKind:     serverStore,
		meshresource.MCPCredentialKind: credentialStore,
	}}, nil)
	repo, err := NewCredentials(resources)
	require.NoError(t, err)

	_, _, err = repo.Create("mesh-1", "server-1", "agent", time.Now().Add(time.Hour))
	require.Error(t, err)
	require.NoError(t, resources.Add(meshresource.NewMCPServerResourceWithAttributes("server-1", "mesh-1")))
	_, _, err = repo.Create("mesh-1", "server-1", "agent", time.Now().Add(-time.Hour))
	require.Error(t, err)

	first, token, err := repo.Create("mesh-1", "server-1", "agent", time.Now().Add(time.Hour))
	require.NoError(t, err)
	second, _, err := repo.Create("mesh-1", "server-1", "other", time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.NotEqual(t, first.Name, second.Name)
	require.Equal(t, "1", first.ResourceVersion)
	require.NotContains(t, first.Spec.SecretHash, token)
	require.True(t, strings.HasPrefix(token, "mcp_"+first.Name+"."))
	secret := strings.TrimPrefix(token, "mcp_"+first.Name+".")
	require.True(t, VerifySecretHash(first.Spec.SecretHash, secret))
	require.False(t, VerifySecretHash(first.Spec.SecretHash, "incorrect"))
	require.False(t, VerifySecretHash(strings.TrimPrefix(first.Spec.SecretHash, "sha256:"), secret))
	permanent, permanentToken, err := repo.Create("mesh-1", "server-1", "permanent", time.Time{})
	require.NoError(t, err)
	require.Empty(t, permanent.Spec.ExpiresAt)
	permanentSecret := strings.TrimPrefix(permanentToken, "mcp_"+permanent.Name+".")
	_, err = repo.Authenticate("mesh-1", "server-1", permanent.Name, permanentSecret, time.Now().Add(100*365*24*time.Hour))
	require.NoError(t, err)
	_, err = repo.Authenticate("mesh-1", "server-1", first.Name, secret, time.Now())
	require.NoError(t, err)
	_, err = repo.Authenticate("mesh-1", "different-server", first.Name, secret, time.Now())
	require.Error(t, err)
	_, err = repo.Authenticate("mesh-1", "server-1", first.Name, secret, time.Now().Add(2*time.Hour))
	require.Error(t, err)

	listed, err := repo.List("mesh-1", "server-1")
	require.NoError(t, err)
	require.Len(t, listed, 3)
	listed, err = repo.List("mesh-2", "server-1")
	require.NoError(t, err)
	require.Empty(t, listed)

	_, err = repo.Revoke("mesh-1", "different-server", first.Name, "1")
	require.Error(t, err)
	revoked, err := repo.Revoke("mesh-1", "server-1", first.Name, "1")
	require.NoError(t, err)
	require.Equal(t, "revoked", revoked.Spec.Status)
	require.Equal(t, "2", revoked.ResourceVersion)
	_, err = repo.Authenticate("mesh-1", "server-1", first.Name, secret, time.Now())
	require.Error(t, err)
	_, err = repo.Revoke("mesh-1", "server-1", first.Name, "1")
	require.True(t, errors.Is(err, store.ErrorResourceConflict("", "", "")))
}
