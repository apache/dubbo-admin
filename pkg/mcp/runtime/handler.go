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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/proto"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/config/discovery"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	corediscovery "github.com/apache/dubbo-admin/pkg/core/discovery"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	"github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
	"github.com/apache/dubbo-admin/pkg/mcp/availability"
	"github.com/apache/dubbo-admin/pkg/mcp/business"
	"github.com/apache/dubbo-admin/pkg/mcp/invocation"
	"github.com/apache/dubbo-admin/pkg/mcp/repository"
)

const (
	defaultToolTimeout = 30 * time.Second
	oidcHTTPTimeout    = 10 * time.Second
)

type MCPPrincipal struct {
	Kind     string
	Subject  string
	Mesh     string
	ServerID string
	Scopes   []string
}

type OAuthClaims struct {
	Subject string
	Scopes  []string
}

type OAuthVerifier func(context.Context, string) (OAuthClaims, error)

type Invoker interface {
	Call(
		context.Context,
		string,
		*discovery.Config,
		string,
		*meshproto.MCPOperation,
		[]any,
		time.Duration,
	) (any, error)
}

type Options struct {
	Now           func() time.Time
	Invoker       Invoker
	OAuthVerifier OAuthVerifier
}

type Handler struct {
	ctx          consolectx.Context
	cfg          *app.MCPConfig
	invoker      Invoker
	now          func() time.Time
	oauthVerify  OAuthVerifier
	availability *availability.LiveServiceIndex

	cacheMu sync.Mutex
	cache   map[cacheKey]*compiledTool
}

type cacheKey struct {
	mesh           string
	serverID       string
	revision       int64
	toolID         string
	metadataDigest string
}

type compiledTool struct {
	published *meshproto.MCPToolBinding
	contract  *business.ResolvedContract
	callable  bool
}

func NewHandler(ctx consolectx.Context, cfg *app.MCPConfig, options Options) *Handler {
	if cfg == nil {
		cfg = &app.MCPConfig{}
	}
	invoker := options.Invoker
	if invoker == nil {
		invoker = invocation.NewClient()
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	oauthVerify := options.OAuthVerifier
	if oauthVerify == nil && ctx != nil {
		oauthVerify = newOIDCVerifier(ctx.AppContext(), cfg)
	}
	return &Handler{
		ctx:          ctx,
		cfg:          cfg,
		invoker:      invoker,
		now:          now,
		oauthVerify:  oauthVerify,
		availability: availability.NewLiveServiceIndex(60 * time.Second),
		cache:        make(map[cacheKey]*compiledTool),
	}
}

// RegisterRoutes adds the business MCP endpoint without changing the legacy
// /api/mcp administration endpoint.
func RegisterRoutes(engine *gin.Engine, handler *Handler) {
	if engine == nil || handler == nil {
		return
	}
	business := engine.Group("/mcp")
	business.POST("/:mesh/:serverId", func(c *gin.Context) {
		handler.ServeHTTP(c.Param("mesh"), c.Param("serverId"), c.Writer, c.Request)
	})
	metadata := engine.Group("/.well-known/oauth-protected-resource/mcp")
	metadata.GET("/:mesh/:serverId", func(c *gin.Context) {
		handler.ServeProtectedResourceMetadata(c.Param("mesh"), c.Param("serverId"), c.Writer, c.Request)
	})
}

func (h *Handler) ServeHTTP(mesh, serverID string, w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(mesh) == "" || strings.TrimSpace(serverID) == "" {
		http.Error(w, "mesh and serverId are required", http.StatusBadRequest)
		return
	}
	server, err := h.publishedServer(mesh, serverID)
	if err != nil {
		http.Error(w, err.Error(), statusForLookupError(err))
		return
	}
	principal, status, err := h.authenticate(r, mesh, serverID, server.Spec.Published.Snapshot.RequiredOauthScopes)
	if err != nil {
		if status == http.StatusUnauthorized {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer resource_metadata=%q", h.metadataURL(r, mesh, serverID)))
		}
		http.Error(w, err.Error(), status)
		return
	}
	r = r.WithContext(context.WithValue(r.Context(), principalKey{}, principal))

	mcpServer, err := h.buildServer(r.Context(), mesh, serverID, server.Spec.Published)
	if err != nil {
		http.Error(w, "MCP server is unavailable", http.StatusServiceUnavailable)
		return
	}
	logger.Sugar().Infow("MCP business request",
		"principal_type", principal.Kind,
		"principal", principal.Subject,
		"mesh", mesh,
		"server_id", serverID,
	)
	stream := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})
	stream.ServeHTTP(w, r)
}

func (h *Handler) ServeProtectedResourceMetadata(mesh, serverID string, w http.ResponseWriter, r *http.Request) {
	server, err := h.publishedServer(mesh, serverID)
	if err != nil {
		http.Error(w, err.Error(), statusForLookupError(err))
		return
	}
	metadata := &oauthex.ProtectedResourceMetadata{
		Resource:               h.endpointURL(r, mesh, serverID),
		ScopesSupported:        append([]string(nil), server.Spec.Published.Snapshot.RequiredOauthScopes...),
		BearerMethodsSupported: []string{"header"},
	}
	if strings.TrimSpace(h.cfg.OAuthIssuer) != "" {
		metadata.AuthorizationServers = []string{h.cfg.OAuthIssuer}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metadata)
}

func (h *Handler) authenticate(
	r *http.Request,
	mesh, serverID string,
	requiredScopes []string,
) (MCPPrincipal, int, error) {
	fields := strings.Fields(r.Header.Get("Authorization"))
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return MCPPrincipal{}, http.StatusUnauthorized, errors.New("bearer token is required")
	}
	token := fields[1]
	if strings.HasPrefix(token, "mcp_") {
		principal, err := h.authenticateAPIKey(mesh, serverID, token)
		if err != nil {
			return MCPPrincipal{}, http.StatusUnauthorized, err
		}
		return principal, 0, nil
	}
	if h.oauthVerify == nil {
		return MCPPrincipal{}, http.StatusUnauthorized, errors.New("OAuth token verification is unavailable")
	}
	claims, err := h.oauthVerify(r.Context(), token)
	if err != nil {
		return MCPPrincipal{}, http.StatusUnauthorized, errors.New("invalid OAuth access token")
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return MCPPrincipal{}, http.StatusUnauthorized, errors.New("OAuth access token subject is missing")
	}
	for _, required := range requiredScopes {
		if !slices.Contains(claims.Scopes, required) {
			return MCPPrincipal{}, http.StatusForbidden, fmt.Errorf("missing required scope %q", required)
		}
	}
	return MCPPrincipal{
		Kind: "oauth", Subject: claims.Subject, Mesh: mesh, ServerID: serverID,
		Scopes: append([]string(nil), claims.Scopes...),
	}, 0, nil
}

func (h *Handler) authenticateAPIKey(mesh, serverID, token string) (MCPPrincipal, error) {
	encoded := strings.TrimPrefix(token, "mcp_")
	credentialID, secret, ok := strings.Cut(encoded, ".")
	if !ok || strings.TrimSpace(credentialID) == "" || secret == "" {
		return MCPPrincipal{}, errors.New("invalid MCP API key")
	}
	credential, exists, err := manager.GetByKey[*v1alpha1.MCPCredentialResource](
		h.ctx.ResourceManager(),
		v1alpha1.MCPCredentialKind,
		coremodel.BuildResourceKey(mesh, credentialID),
	)
	if err != nil {
		return MCPPrincipal{}, err
	}
	if !exists || credential.Spec == nil ||
		credential.Spec.ServerId != serverID ||
		credential.Spec.Status != "active" ||
		!repository.VerifySecretHash(credential.Spec.SecretHash, secret) {
		return MCPPrincipal{}, errors.New("invalid MCP API key")
	}
	if strings.TrimSpace(credential.Spec.ExpiresAt) != "" {
		expiresAt, err := time.Parse(time.RFC3339, credential.Spec.ExpiresAt)
		if err != nil || !h.now().Before(expiresAt) {
			return MCPPrincipal{}, errors.New("invalid MCP API key")
		}
	}
	return MCPPrincipal{
		Kind: "api-key", Subject: credentialID, Mesh: mesh, ServerID: serverID,
	}, nil
}

func (h *Handler) buildServer(ctx context.Context, mesh, serverID string, published *meshproto.MCPPublishedSnapshot) (*mcp.Server, error) {
	if published == nil || published.Snapshot == nil {
		return nil, errors.New("published MCP snapshot is missing")
	}
	metadata, err := manager.ListByIndexes[*v1alpha1.ServiceProviderMetadataResource](
		h.ctx.ResourceManager(),
		v1alpha1.ServiceProviderMetadataKind,
		[]index.IndexCondition{{IndexName: index.ByMeshIndex, Value: mesh, Operator: index.Equals}},
	)
	if err != nil {
		return nil, err
	}
	snapshot, err := h.readAvailabilitySnapshot(mesh)
	if err != nil {
		return nil, err
	}
	snapshot.Now = h.now()
	visible := h.availability.FilterTools(mesh, published.Snapshot.Tools, snapshot)
	visibleByID := make(map[string]struct{}, len(visible))
	for _, tool := range visible {
		if tool != nil {
			visibleByID[tool.ToolId] = struct{}{}
		}
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "dubbo-admin", Version: "1.0.0"}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{
			Tools: &mcp.ToolCapabilities{ListChanged: false},
		},
	})
	for _, publishedTool := range published.Snapshot.Tools {
		if publishedTool == nil || publishedTool.Operation == nil {
			continue
		}
		if _, ok := visibleByID[publishedTool.ToolId]; !ok {
			continue
		}
		compiled, err := h.compileTool(mesh, serverID, published.Revision, publishedTool, metadata)
		if err != nil {
			identity := availability.Identity{
				Mesh:        mesh,
				ServiceName: publishedTool.Operation.ServiceName,
				Group:       publishedTool.Operation.Group,
				Version:     publishedTool.Operation.Version,
			}
			if result, ok := h.availability.Get(identity, publishedTool.Operation.Protocol); ok &&
				(result.State == availability.Unknown ||
					(result.State == availability.Inactive && h.availability.Visible(result, snapshot.Now))) {
				compiled = &compiledTool{
					published: proto.Clone(publishedTool).(*meshproto.MCPToolBinding),
					contract:  publishedContract(publishedTool),
					callable:  false,
				}
				logger.Sugar().Warnw("MCP tool is visible but unavailable while discovery evidence is unknown",
					"mesh", mesh, "server_id", serverID, "tool_id", publishedTool.ToolId, "error", err)
			} else {
				logger.Sugar().Warnw("MCP tool disabled after contract validation",
					"mesh", mesh, "server_id", serverID, "tool_id", publishedTool.ToolId, "error", err)
				continue
			}
		}
		h.addTool(ctx, server, mesh, compiled)
	}
	return server, nil
}

func (h *Handler) addTool(ctx context.Context, server *mcp.Server, mesh string, compiled *compiledTool) {
	published := compiled.published
	inputSchema := map[string]any{"type": "object"}
	if published.InputSchema != nil {
		inputSchema = published.InputSchema.AsMap()
	}
	tool := &mcp.Tool{
		Name:        published.ToolName,
		Description: published.Description,
		InputSchema: inputSchema,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    published.ReadOnlyHint != nil && *published.ReadOnlyHint,
			DestructiveHint: published.DestructiveHint,
			IdempotentHint:  published.IdempotentHint != nil && *published.IdempotentHint,
		},
	}
	server.AddTool(tool, func(callCtx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return h.callTool(callCtx, mesh, compiled, req)
	})
}

func (h *Handler) callTool(ctx context.Context, mesh string, compiled *compiledTool, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !compiled.callable {
		return toolError(&invocation.CallError{
			Outcome:  invocation.NotExecuted,
			Protocol: compiled.contract.Operation.Protocol,
			Cause:    errors.New("discovery evidence is unknown"),
		}), nil
	}
	arguments := make(map[string]any)
	if req != nil && req.Params != nil && len(req.Params.Arguments) > 0 {
		if err := json.Unmarshal(req.Params.Arguments, &arguments); err != nil {
			return toolError(fmt.Errorf("invalid tool arguments: %w", err)), nil
		}
	}
	args, err := business.RestoreArguments(compiled.contract, arguments)
	if err != nil {
		return toolError(err), nil
	}
	adminConfig := h.ctx.Config()
	config := adminConfig.FindDiscovery(mesh)
	if config == nil {
		return toolError(&invocation.CallError{
			Outcome:  invocation.NotExecuted,
			Protocol: compiled.contract.Operation.Protocol,
			Cause:    fmt.Errorf("discovery %q is not configured", mesh),
		}), nil
	}
	timeout := defaultToolTimeout
	if compiled.published.TimeoutEnabled {
		timeout = time.Duration(compiled.published.TimeoutMillis) * time.Millisecond
	}
	value, err := h.invoker.Call(ctx, mesh, config, constant.RegistryTypeService, compiled.contract.Operation, args, timeout)
	if err != nil {
		return toolError(err), nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return toolError(fmt.Errorf("encode invocation result: %w", err)), nil
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(data)}}}, nil
}

func toolError(err error) *mcp.CallToolResult {
	result := &mcp.CallToolResult{}
	result.SetError(err)
	return result
}

func (h *Handler) compileTool(mesh, serverID string, revision int64, published *meshproto.MCPToolBinding, metadata []*v1alpha1.ServiceProviderMetadataResource) (*compiledTool, error) {
	digest, err := metadataDigest(mesh, published.Operation, metadata)
	if err != nil {
		return nil, err
	}
	key := cacheKey{
		mesh: mesh, serverID: serverID, revision: revision,
		toolID: published.ToolId, metadataDigest: digest,
	}
	h.cacheMu.Lock()
	if cached := h.cache[key]; cached != nil {
		h.cacheMu.Unlock()
		return cached, nil
	}
	h.cacheMu.Unlock()

	contract, err := business.ResolveOperation(mesh, published.Operation, published.Parameters, metadata)
	if err != nil {
		return nil, err
	}
	if contract.InputFingerprint != published.InputFingerprint {
		return nil, fmt.Errorf("published input contract drifted")
	}
	if contract.OutputFingerprint != published.OutputFingerprint {
		logger.Sugar().Warnw("MCP tool output contract changed without republish",
			"mesh", mesh, "server_id", serverID, "tool_id", published.ToolId)
	}
	contract.Operation = proto.Clone(published.Operation).(*meshproto.MCPOperation)
	contract.Parameters = make([]*meshproto.MCPParameterBinding, len(published.Parameters))
	for i, parameter := range published.Parameters {
		if parameter != nil {
			contract.Parameters[i] = proto.Clone(parameter).(*meshproto.MCPParameterBinding)
		}
	}
	contract.InputSchema = published.InputSchema
	contract.RelatedTypes = make([]*meshproto.MCPTypeDefinition, len(published.RelatedTypes))
	for i, relatedType := range published.RelatedTypes {
		if relatedType != nil {
			contract.RelatedTypes[i] = proto.Clone(relatedType).(*meshproto.MCPTypeDefinition)
		}
	}
	compiled := &compiledTool{
		published: proto.Clone(published).(*meshproto.MCPToolBinding),
		contract:  contract,
		callable:  true,
	}
	h.cacheMu.Lock()
	h.cache[key] = compiled
	h.cacheMu.Unlock()
	return compiled, nil
}

func publishedContract(published *meshproto.MCPToolBinding) *business.ResolvedContract {
	contract := &business.ResolvedContract{
		ReturnType:        published.ReturnType,
		InputSchema:       published.InputSchema,
		InputFingerprint:  published.InputFingerprint,
		OutputFingerprint: published.OutputFingerprint,
	}
	if published.Operation != nil {
		contract.Operation = proto.Clone(published.Operation).(*meshproto.MCPOperation)
	}
	contract.Parameters = make([]*meshproto.MCPParameterBinding, len(published.Parameters))
	for i, parameter := range published.Parameters {
		if parameter != nil {
			contract.Parameters[i] = proto.Clone(parameter).(*meshproto.MCPParameterBinding)
		}
	}
	contract.RelatedTypes = make([]*meshproto.MCPTypeDefinition, len(published.RelatedTypes))
	for i, relatedType := range published.RelatedTypes {
		if relatedType != nil {
			contract.RelatedTypes[i] = proto.Clone(relatedType).(*meshproto.MCPTypeDefinition)
		}
	}
	return contract
}

func metadataDigest(mesh string, operation *meshproto.MCPOperation, metadata []*v1alpha1.ServiceProviderMetadataResource) (string, error) {
	if operation == nil {
		return "", errors.New("operation is required")
	}
	type item struct {
		key  string
		data []byte
	}
	items := make([]item, 0)
	for _, resource := range metadata {
		if resource == nil || resource.Spec == nil || resource.Mesh != mesh ||
			resource.Spec.ServiceName != operation.ServiceName ||
			resource.Spec.Group != operation.Group ||
			resource.Spec.Version != operation.Version {
			continue
		}
		data, err := (proto.MarshalOptions{Deterministic: true}).Marshal(resource.Spec)
		if err != nil {
			return "", err
		}
		items = append(items, item{key: resource.ResourceKey(), data: data})
	}
	slices.SortFunc(items, func(left, right item) int {
		return strings.Compare(left.key, right.key)
	})
	hash := sha256.New()
	for _, item := range items {
		_, _ = hash.Write([]byte(item.key))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(item.data)
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (h *Handler) publishedServer(mesh, serverID string) (*v1alpha1.MCPServerResource, error) {
	server, exists, err := manager.GetByKey[*v1alpha1.MCPServerResource](
		h.ctx.ResourceManager(),
		v1alpha1.MCPServerKind,
		coremodel.BuildResourceKey(mesh, serverID),
	)
	if err != nil {
		return nil, err
	}
	if !exists || server.Spec == nil || server.Spec.Published == nil || server.Spec.Published.Snapshot == nil {
		return nil, fmt.Errorf("published MCP server %q not found", serverID)
	}
	return server, nil
}

func statusForLookupError(err error) int {
	if strings.Contains(err.Error(), "not found") {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

type principalKey struct{}

func PrincipalFromContext(ctx context.Context) (MCPPrincipal, bool) {
	principal, ok := ctx.Value(principalKey{}).(MCPPrincipal)
	return principal, ok
}

func (h *Handler) readAvailabilitySnapshot(mesh string) (availability.Snapshot, error) {
	var source availability.SyncStateSource
	if candidate, ok := h.ctx.(interface {
		DiscoverySyncState(string) corediscovery.SyncState
	}); ok {
		source = candidate
	}
	return availability.ReadSnapshot(h.ctx.ResourceManager(), source, mesh)
}

func (h *Handler) endpointURL(r *http.Request, mesh, serverID string) string {
	return h.baseURL(r) + "/mcp/" + url.PathEscape(mesh) + "/" + url.PathEscape(serverID)
}

func (h *Handler) metadataURL(r *http.Request, mesh, serverID string) string {
	return h.baseURL(r) + "/.well-known/oauth-protected-resource/mcp/" +
		url.PathEscape(mesh) + "/" + url.PathEscape(serverID)
}

func (h *Handler) baseURL(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
		if r.TLS != nil {
			scheme = "https"
		}
	}
	return scheme + "://" + r.Host
}

func newOIDCVerifier(ctx context.Context, cfg *app.MCPConfig) OAuthVerifier {
	if cfg == nil || strings.TrimSpace(cfg.OAuthIssuer) == "" || strings.TrimSpace(cfg.OAuthAudience) == "" {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	httpClient := &http.Client{Timeout: oidcHTTPTimeout}
	discoveryContext := context.WithValue(ctx, oauth2.HTTPClient, httpClient)
	provider, err := oidc.NewProvider(discoveryContext, cfg.OAuthIssuer)
	if err != nil {
		logger.Sugar().Warnw("MCP OAuth verifier unavailable", "issuer", cfg.OAuthIssuer, "error", err)
		return nil
	}
	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.OAuthAudience,
	})
	return func(ctx context.Context, token string) (OAuthClaims, error) {
		idToken, err := verifier.Verify(ctx, token)
		if err != nil {
			return OAuthClaims{}, err
		}
		var claims struct {
			Scope string   `json:"scope"`
			Scp   []string `json:"scp"`
		}
		if err := idToken.Claims(&claims); err != nil {
			return OAuthClaims{}, err
		}
		scopes := append([]string(nil), claims.Scp...)
		scopes = append(scopes, strings.Fields(claims.Scope)...)
		slices.Sort(scopes)
		return OAuthClaims{Subject: idToken.Subject, Scopes: slices.Compact(scopes)}, nil
	}
}
