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

package service

import (
	stdctx "context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/config/app"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/counter"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
	"github.com/apache/dubbo-admin/pkg/store/memory"
)

type mcpServiceTestRouter struct {
	stores map[coremodel.ResourceKind]store.ResourceStore
}

func (r mcpServiceTestRouter) ResourceRoute(resource coremodel.Resource) (store.ResourceStore, error) {
	return r.ResourceKindRoute(resource.ResourceKind())
}

func (r mcpServiceTestRouter) ResourceKindRoute(kind coremodel.ResourceKind) (store.ResourceStore, error) {
	resourceStore, ok := r.stores[kind]
	if !ok {
		return nil, errors.New("resource store not configured")
	}
	return resourceStore, nil
}

type mcpServiceTestContext struct {
	resourceManager manager.ResourceManager
}

var _ consolectx.Context = (*mcpServiceTestContext)(nil)

func (c *mcpServiceTestContext) ResourceManager() manager.ResourceManager {
	return c.resourceManager
}

func (*mcpServiceTestContext) CounterManager() counter.CounterManager {
	return nil
}

func (*mcpServiceTestContext) Config() app.AdminConfig {
	return app.AdminConfig{}
}

func (*mcpServiceTestContext) AppContext() stdctx.Context {
	return stdctx.Background()
}

func (*mcpServiceTestContext) LockManager() lock.Lock {
	return nil
}

func (*mcpServiceTestContext) RuleVersioning() *versioning.Service {
	return nil
}

func newMCPServiceTestContext(t *testing.T, unavailable bool) (*mcpServiceTestContext, *meshresource.MCPServerResource) {
	t.Helper()
	kinds := []coremodel.ResourceKind{
		meshresource.MCPServerKind,
		meshresource.MCPCredentialKind,
		meshresource.ServiceProviderMetadataKind,
		meshresource.RPCInstanceKind,
	}
	stores := make(map[coremodel.ResourceKind]store.ResourceStore, len(kinds))
	for _, kind := range kinds {
		resourceStore := memory.NewMemoryResourceStore(kind)
		require.NoError(t, resourceStore.Init(nil))
		stores[kind] = resourceStore
	}
	resources := manager.NewResourceManager(mcpServiceTestRouter{stores: stores}, nil)

	provider := meshresource.NewServiceProviderMetadataResourceWithAttributes("provider-1", "mesh-1")
	provider.Spec = &meshproto.ServiceProviderMetadata{
		ServiceName:     "com.example.OrderService",
		ProviderAppName: "orders-app",
		Methods:         []*meshproto.Method{{Name: "run", ReturnType: "void"}},
	}
	require.NoError(t, stores[meshresource.ServiceProviderMetadataKind].Add(provider))

	instance := meshresource.NewRPCInstanceResourceWithAttributes("instance-1", "mesh-1")
	instance.Spec = &meshproto.RPCInstance{
		AppName:   "orders-app",
		Protocol:  "tri",
		Endpoints: []*meshproto.Endpoint{{Protocol: "tri"}},
	}
	if unavailable {
		instance.Spec.UnregisterTime = "down"
	}
	require.NoError(t, stores[meshresource.RPCInstanceKind].Add(instance))

	server := meshresource.NewMCPServerResourceWithAttributes("server-1", "mesh-1")
	server.Spec = &meshproto.MCPServer{
		Draft: &meshproto.MCPServerSnapshot{
			Description: "orders tools",
			Tools: []*meshproto.MCPToolBinding{{
				ToolId:      "00000000-0000-0000-0000-000000000001",
				ToolName:    "run",
				Description: "Run the order operation",
				Operation: &meshproto.MCPOperation{
					ServiceName: "com.example.OrderService",
					MethodName:  "run",
					Protocol:    "dubbo",
				},
			}},
		},
		Published: &meshproto.MCPPublishedSnapshot{
			Revision: 7,
			Snapshot: &meshproto.MCPServerSnapshot{Description: "published"},
		},
	}
	require.NoError(t, stores[meshresource.MCPServerKind].Add(server))
	return &mcpServiceTestContext{resourceManager: resources}, server
}

func TestMCPValidationResult(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		result, err := mcpValidationResult(nil, "")
		require.NoError(t, err)
		require.True(t, result.Valid)
		require.Empty(t, result.Errors)
		require.NotNil(t, result.Errors)
		require.Empty(t, result.Warnings)
		require.NotNil(t, result.Warnings)
	})

	t.Run("invalid draft", func(t *testing.T) {
		result, err := mcpValidationResult(
			bizerror.New(bizerror.InvalidArgument, "tool description is required"),
			"tool-1",
		)
		require.NoError(t, err)
		require.False(t, result.Valid)
		require.Equal(t, "InvalidArgument", result.Errors[0].Code)
		require.Equal(t, "tool description is required", result.Errors[0].Message)
		require.Equal(t, "tool-1", result.Errors[0].ToolID)
	})

	t.Run("infrastructure failure", func(t *testing.T) {
		failure := errors.New("store unavailable")
		result, err := mcpValidationResult(failure, "tool-1")
		require.Nil(t, result)
		require.ErrorIs(t, err, failure)
	})
}

func TestMCPProtocolValidation(t *testing.T) {
	t.Run("confirmed incompatibility is an error and preserves published snapshot", func(t *testing.T) {
		ctx, server := newMCPServiceTestContext(t, false)

		_, err := PublishMCPServer(ctx, "mesh-1", "server-1", &model.MCPPublishReq{
			ExpectedResourceVersion: server.ResourceVersion,
		})
		require.Error(t, err)
		var validationError bizerror.Error
		require.ErrorAs(t, err, &validationError)
		require.Equal(t, bizerror.InvalidArgument, validationError.Code())

		stored, err := GetMCPServerInMesh(ctx, "mesh-1", "server-1")
		require.NoError(t, err)
		require.Equal(t, "1", stored.ResourceVersion)
		require.Equal(t, int64(7), stored.Spec.Published.Revision)
		require.Equal(t, "published", stored.Spec.Published.Snapshot.Description)
	})

	t.Run("unavailable provider is a warning and remains publishable", func(t *testing.T) {
		ctx, server := newMCPServiceTestContext(t, true)

		validation, err := ValidateMCPServer(ctx, "mesh-1", "server-1")
		require.NoError(t, err)
		require.True(t, validation.Valid)
		require.Empty(t, validation.Errors)
		require.Len(t, validation.Warnings, 1)
		require.Equal(t, "NoAvailableProvider", validation.Warnings[0].Code)

		published, err := PublishMCPServer(ctx, "mesh-1", "server-1", &model.MCPPublishReq{
			ExpectedResourceVersion: server.ResourceVersion,
		})
		require.NoError(t, err)
		require.Equal(t, "2", published.ResourceVersion)
		require.Equal(t, int64(8), published.Spec.Published.Revision)
	})
}

func TestMCPCredentialViewDoesNotExposeSecretHash(t *testing.T) {
	credential := meshresource.NewMCPCredentialResourceWithAttributes("credential-1", "mesh-1")
	credential.ResourceVersion = "3"
	credential.Spec = &meshproto.MCPCredential{
		ServerId: "server-1", DisplayName: "automation", SecretHash: "sha256:sensitive",
		Status: "active",
	}
	view := credentialView(credential)
	data, err := json.Marshal(MCPCredentialToken{Credential: view, Token: "mcp_one-time"})
	require.NoError(t, err)
	require.NotContains(t, string(data), "sha256:sensitive")
	require.NotContains(t, string(data), "secretHash")
	require.Contains(t, string(data), `"resourceVersion":"3"`)
}
