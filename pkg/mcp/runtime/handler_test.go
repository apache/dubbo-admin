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

package runtime

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	jose "github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/require"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/config/app"
	configdiscovery "github.com/apache/dubbo-admin/pkg/config/discovery"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/counter"
	corediscovery "github.com/apache/dubbo-admin/pkg/core/discovery"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
	"github.com/apache/dubbo-admin/pkg/mcp/business"
	"github.com/apache/dubbo-admin/pkg/store/memory"
)

type runtimeTestRouter struct {
	stores map[coremodel.ResourceKind]store.ResourceStore
}

func (r runtimeTestRouter) ResourceRoute(resource coremodel.Resource) (store.ResourceStore, error) {
	return r.ResourceKindRoute(resource.ResourceKind())
}

func (r runtimeTestRouter) ResourceKindRoute(kind coremodel.ResourceKind) (store.ResourceStore, error) {
	resourceStore, ok := r.stores[kind]
	if !ok {
		return nil, errors.New("resource store not configured")
	}
	return resourceStore, nil
}

type runtimeTestContext struct {
	resourceManager manager.ResourceManager
	config          app.AdminConfig
	syncState       corediscovery.SyncState
}

var _ consolectx.Context = (*runtimeTestContext)(nil)

func (c *runtimeTestContext) ResourceManager() manager.ResourceManager {
	return c.resourceManager
}

func (*runtimeTestContext) CounterManager() counter.CounterManager {
	return nil
}

func (c *runtimeTestContext) Config() app.AdminConfig {
	return c.config
}

func (*runtimeTestContext) AppContext() context.Context {
	return context.Background()
}

func (*runtimeTestContext) LockManager() lock.Lock {
	return nil
}

func (*runtimeTestContext) RuleVersioning() *versioning.Service {
	return nil
}

func (c *runtimeTestContext) DiscoverySyncState(string) corediscovery.SyncState {
	return c.syncState
}

type captureInvoker struct {
	mu    sync.Mutex
	calls int
	args  []any
}

func (i *captureInvoker) Call(
	_ context.Context,
	_ string,
	_ *configdiscovery.Config,
	_ string,
	_ *meshproto.MCPOperation,
	args []any,
	_ time.Duration,
) (any, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.calls++
	i.args = append([]any(nil), args...)
	return map[string]any{"ok": true}, nil
}

func (i *captureInvoker) count() int {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.calls
}

func newRuntimeFixture(t *testing.T, syncState corediscovery.SyncState, withMetadata bool) (*runtimeTestContext, string, *meshresource.MCPCredentialResource) {
	t.Helper()
	kinds := []coremodel.ResourceKind{
		meshresource.MCPServerKind,
		meshresource.MCPCredentialKind,
		meshresource.ServiceProviderMetadataKind,
		meshresource.RPCInstanceKind,
		meshresource.RPCInstanceMetadataKind,
	}
	stores := make(map[coremodel.ResourceKind]store.ResourceStore, len(kinds))
	for _, kind := range kinds {
		resourceStore := memory.NewMemoryResourceStore(kind)
		require.NoError(t, resourceStore.Init(nil))
		stores[kind] = resourceStore
	}
	resources := manager.NewResourceManager(runtimeTestRouter{stores: stores}, nil)

	operation := &meshproto.MCPOperation{
		ServiceName:    "com.example.OrderService",
		MethodName:     "echo",
		ParameterTypes: []string{"java.lang.String"},
		Protocol:       "dubbo",
	}
	provider := meshresource.NewServiceProviderMetadataResourceWithAttributes("provider-1", "mesh-1")
	provider.Spec = &meshproto.ServiceProviderMetadata{
		ServiceName: "com.example.OrderService",
		Methods: []*meshproto.Method{{
			Name:           "echo",
			ParameterTypes: []string{"java.lang.String"},
			ReturnType:     "void",
		}},
	}
	contract, err := business.ResolveOperation("mesh-1", operation, nil, []*meshresource.ServiceProviderMetadataResource{provider})
	require.NoError(t, err)
	tool := &meshproto.MCPToolBinding{
		ToolId:            "tool-1",
		ToolName:          "echo",
		Description:       "Echo a value",
		Operation:         operation,
		Parameters:        contract.Parameters,
		InputSchema:       contract.InputSchema,
		InputFingerprint:  contract.InputFingerprint,
		OutputFingerprint: contract.OutputFingerprint,
		RelatedTypes:      contract.RelatedTypes,
		ReturnType:        contract.ReturnType,
	}
	server := meshresource.NewMCPServerResourceWithAttributes("server-1", "mesh-1")
	server.Spec = &meshproto.MCPServer{
		Published: &meshproto.MCPPublishedSnapshot{
			Revision: 1,
			Snapshot: &meshproto.MCPServerSnapshot{
				RequiredOauthScopes: []string{"orders:read"},
				Tools:               []*meshproto.MCPToolBinding{tool},
			},
		},
	}
	require.NoError(t, stores[meshresource.MCPServerKind].Add(server))

	secret := "secret-value"
	digest := sha256.Sum256([]byte(secret))
	credential := meshresource.NewMCPCredentialResourceWithAttributes("credential-1", "mesh-1")
	credential.Spec = &meshproto.MCPCredential{
		ServerId:   "server-1",
		SecretHash: "sha256:" + hex.EncodeToString(digest[:]),
		Status:     "active",
	}
	require.NoError(t, stores[meshresource.MCPCredentialKind].Add(credential))

	if withMetadata {
		require.NoError(t, stores[meshresource.ServiceProviderMetadataKind].Add(provider))
		instance := meshresource.NewRPCInstanceResourceWithAttributes("instance-1", "mesh-1")
		available := true
		instance.Spec = &meshproto.RPCInstance{
			AppName:   "orders-app",
			Revision:  "r1",
			Available: &available,
		}
		require.NoError(t, stores[meshresource.RPCInstanceKind].Add(instance))
		metadata := meshresource.NewRPCInstanceMetadataResourceWithAttributes("orders-app@r1", "mesh-1")
		metadata.Spec = &meshproto.RPCInstanceMetadata{
			App:      "orders-app",
			Revision: "r1",
			Services: map[string]*meshproto.ServiceInfo{
				"com.example.OrderService:dubbo": {
					Name:     "com.example.OrderService",
					Protocol: "dubbo",
				},
			},
		}
		require.NoError(t, stores[meshresource.RPCInstanceMetadataKind].Add(metadata))
	}

	return &runtimeTestContext{
		resourceManager: resources,
		config: app.AdminConfig{
			MCP: &app.MCPConfig{Enabled: true},
			Discovery: []*configdiscovery.Config{{
				ID:   "mesh-1",
				Type: configdiscovery.Mock,
			}},
		},
		syncState: syncState,
	}, "mcp_credential-1." + secret, credential
}

func postMCP(t *testing.T, engine http.Handler, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/mcp/mesh-1/server-1", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Mcp-Protocol-Version", "2025-06-18")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	return recorder
}

func initializeMessage() string {
	return `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`
}

func listToolsMessage() string {
	return `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`
}

func decodeJSON(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var response map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestBusinessEndpointAuthenticatesAndScopesAPIKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, token, _ := newRuntimeFixture(t, corediscovery.SyncState{
		InitialSyncComplete: true,
		Healthy:             true,
	}, true)
	handler := NewHandler(ctx, ctx.config.MCP, Options{
		Invoker: &captureInvoker{},
		OAuthVerifier: func(context.Context, string) (OAuthClaims, error) {
			return OAuthClaims{}, errors.New("OAuth must not handle mcp_ tokens")
		},
	})
	engine := gin.New()
	RegisterRoutes(engine, handler)

	initialize := postMCP(t, engine, token, initializeMessage())
	require.Equal(t, http.StatusOK, initialize.Code)
	require.Equal(t, "dubbo-admin", decodeJSON(t, initialize)["result"].(map[string]any)["serverInfo"].(map[string]any)["name"])

	list := postMCP(t, engine, token, listToolsMessage())
	require.Equal(t, http.StatusOK, list.Code)
	tools := decodeJSON(t, list)["result"].(map[string]any)["tools"].([]any)
	require.Len(t, tools, 1)
	require.Equal(t, "echo", tools[0].(map[string]any)["name"])

	wrongMesh := httptest.NewRequest(http.MethodPost, "http://example.test/mcp/mesh-2/server-1", strings.NewReader(initializeMessage()))
	wrongMesh.Header.Set("Authorization", "Bearer "+token)
	wrongMesh.Header.Set("Content-Type", "application/json")
	wrongMeshResponse := httptest.NewRecorder()
	engine.ServeHTTP(wrongMeshResponse, wrongMesh)
	require.Equal(t, http.StatusNotFound, wrongMeshResponse.Code)
}

func TestBusinessEndpointNeverFallsBackBetweenTokenVerifiers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, token, _ := newRuntimeFixture(t, corediscovery.SyncState{}, true)
	oauthCalls := 0
	handler := NewHandler(ctx, ctx.config.MCP, Options{
		Invoker: &captureInvoker{},
		OAuthVerifier: func(context.Context, string) (OAuthClaims, error) {
			oauthCalls++
			return OAuthClaims{}, errors.New("invalid token")
		},
	})
	engine := gin.New()
	RegisterRoutes(engine, handler)

	mcpToken := "mcp_credential-1.wrong"
	require.Equal(t, http.StatusUnauthorized, postMCP(t, engine, mcpToken, initializeMessage()).Code)
	require.Zero(t, oauthCalls)

	require.Equal(t, http.StatusUnauthorized, postMCP(t, engine, "oauth-token", initializeMessage()).Code)
	require.Equal(t, 1, oauthCalls)

	require.Equal(t, http.StatusOK, postMCP(t, engine, token, initializeMessage()).Code)
}

func TestUnknownAvailabilityKeepsToolListedButDoesNotExecute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, token, _ := newRuntimeFixture(t, corediscovery.SyncState{}, false)
	invoker := &captureInvoker{}
	handler := NewHandler(ctx, ctx.config.MCP, Options{Invoker: invoker})
	engine := gin.New()
	RegisterRoutes(engine, handler)

	list := postMCP(t, engine, token, listToolsMessage())
	require.Equal(t, http.StatusOK, list.Code)
	tools := decodeJSON(t, list)["result"].(map[string]any)["tools"].([]any)
	require.Len(t, tools, 1)

	call := postMCP(t, engine, token,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"arg0":"value"}}}`)
	require.Equal(t, http.StatusOK, call.Code)
	result := decodeJSON(t, call)["result"].(map[string]any)
	require.Equal(t, true, result["isError"])
	require.Contains(t, result["content"].([]any)[0].(map[string]any)["text"], "not-executed")
	require.Zero(t, invoker.count())
}

func TestToolCallUsesPublishedBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, token, _ := newRuntimeFixture(t, corediscovery.SyncState{
		InitialSyncComplete: true,
		Healthy:             true,
	}, true)
	invoker := &captureInvoker{}
	handler := NewHandler(ctx, ctx.config.MCP, Options{Invoker: invoker})
	engine := gin.New()
	RegisterRoutes(engine, handler)

	call := postMCP(t, engine, token,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"arg0":"value"}}}`)
	require.Equal(t, http.StatusOK, call.Code)
	result := decodeJSON(t, call)["result"].(map[string]any)
	require.NotEqual(t, true, result["isError"])
	require.Equal(t, `{"ok":true}`, result["content"].([]any)[0].(map[string]any)["text"])
	require.Equal(t, 1, invoker.count())
	require.Equal(t, []any{"value"}, invoker.args)
}

func TestOAuthScopesAndProtectedResourceMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _, _ := newRuntimeFixture(t, corediscovery.SyncState{
		InitialSyncComplete: true,
		Healthy:             true,
	}, true)
	handler := NewHandler(ctx, ctx.config.MCP, Options{
		OAuthVerifier: func(context.Context, string) (OAuthClaims, error) {
			return OAuthClaims{Subject: "user-1", Scopes: []string{"orders:read"}}, nil
		},
		Invoker: &captureInvoker{},
	})
	engine := gin.New()
	RegisterRoutes(engine, handler)

	require.Equal(t, http.StatusOK, postMCP(t, engine, "oauth-token", initializeMessage()).Code)

	handler.oauthVerify = func(context.Context, string) (OAuthClaims, error) {
		return OAuthClaims{Subject: "user-1", Scopes: []string{"orders:write"}}, nil
	}
	missingScope := postMCP(t, engine, "oauth-token", initializeMessage())
	require.Equal(t, http.StatusForbidden, missingScope.Code)

	metadataRequest := httptest.NewRequest(http.MethodGet,
		"http://example.test/.well-known/oauth-protected-resource/mcp/mesh-1/server-1", nil)
	metadataResponse := httptest.NewRecorder()
	engine.ServeHTTP(metadataResponse, metadataRequest)
	require.Equal(t, http.StatusOK, metadataResponse.Code)
	var metadata map[string]any
	require.NoError(t, json.Unmarshal(metadataResponse.Body.Bytes(), &metadata))
	require.Equal(t, "http://example.test/mcp/mesh-1/server-1", metadata["resource"])
	require.Equal(t, []any{"orders:read"}, metadata["scopes_supported"])
}

func TestMetadataDigestIgnoresResourceVersionAndOrdering(t *testing.T) {
	operation := &meshproto.MCPOperation{
		ServiceName: "com.example.OrderService",
		MethodName:  "echo",
		Protocol:    "dubbo",
	}
	first := meshresource.NewServiceProviderMetadataResourceWithAttributes("provider-a", "mesh-1")
	first.ResourceVersion = "1"
	first.Spec = &meshproto.ServiceProviderMetadata{
		ServiceName: "com.example.OrderService",
		Methods:     []*meshproto.Method{{Name: "echo", ReturnType: "void"}},
	}
	second := first.DeepCopyObject().(*meshresource.ServiceProviderMetadataResource)
	second.ResourceVersion = "2"
	digestA, err := metadataDigest("mesh-1", operation, []*meshresource.ServiceProviderMetadataResource{first, second})
	require.NoError(t, err)
	digestB, err := metadataDigest("mesh-1", operation, []*meshresource.ServiceProviderMetadataResource{second, first})
	require.NoError(t, err)
	require.Equal(t, digestA, digestB)

	second.Spec.Methods[0].ReturnType = "java.lang.String"
	digestC, err := metadataDigest("mesh-1", operation, []*meshresource.ServiceProviderMetadataResource{first, second})
	require.NoError(t, err)
	require.NotEqual(t, digestA, digestC)
}

func TestOIDCVerifierValidatesIssuerAudienceExpiryAndScope(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	const keyID = "mcp-key"
	var provider *httptest.Server
	provider = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer":   provider.URL,
				"jwks_uri": provider.URL + "/jwks",
			})
		case "/jwks":
			_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
				Key: &key.PublicKey, KeyID: keyID, Algorithm: string(jose.RS256), Use: "sig",
			}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()

	verifier := newOIDCVerifier(context.Background(), &app.MCPConfig{
		OAuthIssuer:   provider.URL,
		OAuthAudience: "dubbo-admin",
	})
	require.NotNil(t, verifier)

	sign := func(claims map[string]any) string {
		signer, err := jose.NewSigner(jose.SigningKey{
			Algorithm: jose.RS256,
			Key:       key,
		}, (&jose.SignerOptions{}).WithType("JWT").WithHeader(jose.HeaderKey("kid"), keyID))
		require.NoError(t, err)
		token, err := josejwt.Signed(signer).Claims(claims).Serialize()
		require.NoError(t, err)
		return token
	}
	validClaims := map[string]any{
		"iss":   provider.URL,
		"sub":   "user-1",
		"aud":   "dubbo-admin",
		"exp":   time.Now().Add(time.Minute).Unix(),
		"nbf":   time.Now().Add(-time.Minute).Unix(),
		"scope": "orders:read orders:write",
	}
	claims, err := verifier(context.Background(), sign(validClaims))
	require.NoError(t, err)
	require.Equal(t, "user-1", claims.Subject)
	require.ElementsMatch(t, []string{"orders:read", "orders:write"}, claims.Scopes)

	invalidAudience := map[string]any{}
	for key, value := range validClaims {
		invalidAudience[key] = value
	}
	invalidAudience["aud"] = "other"
	_, err = verifier(context.Background(), sign(invalidAudience))
	require.Error(t, err)

	expired := map[string]any{}
	for key, value := range validClaims {
		expired[key] = value
	}
	expired["exp"] = time.Now().Add(-time.Minute).Unix()
	_, err = verifier(context.Background(), sign(expired))
	require.Error(t, err)

	notYetValid := map[string]any{}
	for key, value := range validClaims {
		notYetValid[key] = value
	}
	notYetValid["nbf"] = time.Now().Add(10 * time.Minute).Unix()
	_, err = verifier(context.Background(), sign(notYetValid))
	require.Error(t, err)
}
