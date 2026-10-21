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

package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	appcfg "github.com/apache/dubbo-admin/pkg/config/app"
	consolecfg "github.com/apache/dubbo-admin/pkg/config/console"
	configauth "github.com/apache/dubbo-admin/pkg/config/console/auth"
	versioningcfg "github.com/apache/dubbo-admin/pkg/config/versioning"
	consoleauth "github.com/apache/dubbo-admin/pkg/console/auth"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/counter"
	"github.com/apache/dubbo-admin/pkg/core/governor"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
)

type routerRuleAPITestContext struct {
	rm         manager.ResourceManager
	config     appcfg.AdminConfig
	versioning *versioning.Service
}

func (c *routerRuleAPITestContext) ResourceManager() manager.ResourceManager { return c.rm }
func (*routerRuleAPITestContext) CounterManager() counter.CounterManager     { return nil }
func (c *routerRuleAPITestContext) Config() appcfg.AdminConfig               { return c.config }
func (*routerRuleAPITestContext) AppContext() context.Context                { return context.Background() }
func (*routerRuleAPITestContext) LockManager() lock.Lock                     { return nil }
func (c *routerRuleAPITestContext) RuleVersioning() *versioning.Service      { return c.versioning }

var _ consolectx.Context = (*routerRuleAPITestContext)(nil)

type routerRuleAPITestStoreRouter struct {
	stores map[coremodel.ResourceKind]store.ResourceStore
}

func (r *routerRuleAPITestStoreRouter) ResourceRoute(resource coremodel.Resource) (store.ResourceStore, error) {
	return r.ResourceKindRoute(resource.ResourceKind())
}

func (r *routerRuleAPITestStoreRouter) ResourceKindRoute(kind coremodel.ResourceKind) (store.ResourceStore, error) {
	resourceStore, ok := r.stores[kind]
	if !ok {
		return nil, fmt.Errorf("resource store not found for %s", kind)
	}
	return resourceStore, nil
}

type routerRuleAPITestGovernor struct {
	stores map[coremodel.ResourceKind]store.ResourceStore
}

func (g *routerRuleAPITestGovernor) CreateRule(resource coremodel.Resource) error {
	return g.stores[resource.ResourceKind()].Add(resource)
}

func (g *routerRuleAPITestGovernor) UpdateRule(resource coremodel.Resource) error {
	return g.stores[resource.ResourceKind()].Update(resource)
}

func (g *routerRuleAPITestGovernor) DeleteRule(resource coremodel.Resource) error {
	return g.stores[resource.ResourceKind()].Delete(resource)
}

type routerRuleAPITestGovernorRouter struct {
	governor governor.RuleGovernor
}

func (r *routerRuleAPITestGovernorRouter) ResourceRoute(coremodel.Resource) (governor.RuleGovernor, error) {
	return r.governor, nil
}

func (r *routerRuleAPITestGovernorRouter) ResourceMeshRoute(string) (governor.RuleGovernor, error) {
	return r.governor, nil
}

type routerRuleAPIResponse struct {
	Code string          `json:"code"`
	Data json.RawMessage `json:"data"`
}

func TestRouterRuleAPIContractNormalizesScriptEnabled(t *testing.T) {
	router, sessionCookie := newRouterRuleAPI(t)
	const (
		mesh = "api-contract-mesh"
		name = "demo.script-router"
	)
	path := "/api/v1/script-rule/" + name + "?mesh=" + mesh

	create := mustRouterRuleJSON(t, map[string]any{
		"configVersion": "v3.0",
		"scope":         "application",
		"key":           "demo",
		"type":          "javascript",
		"script":        "return [invokers[1]];",
	})
	created := callRouterRuleAPI(t, router, sessionCookie, http.MethodPost, path, create)
	require.Contains(t, string(created.Data), `"enabled":true`)

	detail := callRouterRuleAPI(t, router, sessionCookie, http.MethodGet, path, nil)
	require.JSONEq(t, string(created.Data), string(detail.Data))
	require.Contains(t, string(detail.Data), `"enabled":true`)

	search := callRouterRuleAPI(t, router, sessionCookie, http.MethodGet,
		"/api/v1/script-rule/search?mesh="+mesh+"&keywords="+name, nil)
	var searchData struct {
		List []struct {
			RuleName string `json:"ruleName"`
		} `json:"list"`
	}
	require.NoError(t, json.Unmarshal(search.Data, &searchData))
	require.Len(t, searchData.List, 1)
	require.Equal(t, name, searchData.List[0].RuleName)

	updatedPayload := replaceJSONField(t, detail.Data, "script", "return [invokers[0]];")
	updated := callRouterRuleAPI(t, router, sessionCookie, http.MethodPut, path, updatedPayload)
	require.Contains(t, string(updated.Data), `"enabled":true`)

	versionPath := "/api/v1/script-rule/" + name + "/versions?mesh=" + mesh
	versionList := callRouterRuleAPI(t, router, sessionCookie, http.MethodGet, versionPath, nil)
	var versionData struct {
		Items []struct {
			VersionNo int64 `json:"versionNo"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(versionList.Data, &versionData))
	require.Len(t, versionData.Items, 2)
	createVersion := versionData.Items[1].VersionNo

	callRouterRuleAPI(t, router, sessionCookie, http.MethodGet,
		fmt.Sprintf("/api/v1/script-rule/%s/versions/%d?mesh=%s", name, createVersion, mesh), nil)
	callRouterRuleAPI(t, router, sessionCookie, http.MethodGet,
		fmt.Sprintf("/api/v1/script-rule/%s/versions/%d/diff?mesh=%s", name, createVersion, mesh), nil)
	callRouterRuleAPI(t, router, sessionCookie, http.MethodPost,
		fmt.Sprintf("/api/v1/script-rule/%s/versions/%d/rollback?mesh=%s", name, createVersion, mesh),
		mustRouterRuleJSON(t, map[string]string{"reason": "verify omitted enabled"}))

	restored := callRouterRuleAPI(t, router, sessionCookie, http.MethodGet, path, nil)
	require.Contains(t, string(restored.Data), `"enabled":true`)
	var restoredData map[string]any
	require.NoError(t, json.Unmarshal(restored.Data, &restoredData))
	require.Equal(t, "return [invokers[1]];", restoredData["script"])

	deleted := callRouterRuleAPI(t, router, sessionCookie, http.MethodDelete, path, nil)
	require.JSONEq(t, `""`, string(deleted.Data))
}

func TestRouterRuleAPIContractPreservesAffinityOptionalBooleans(t *testing.T) {
	router, sessionCookie := newRouterRuleAPI(t)
	const (
		mesh = "api-contract-mesh"
		name = "demo.affinity-router"
	)
	path := "/api/v1/affinity-rule/" + name + "?mesh=" + mesh

	create := mustRouterRuleJSON(t, map[string]any{
		"configVersion": "v3.1",
		"scope":         "application",
		"key":           "demo",
		"affinityAware": map[string]any{"key": "region", "ratio": 50},
	})
	created := callRouterRuleAPI(t, router, sessionCookie, http.MethodPost, path, create)
	require.NotContains(t, string(created.Data), `"runtime"`)
	require.NotContains(t, string(created.Data), `"enabled"`)

	detail := callRouterRuleAPI(t, router, sessionCookie, http.MethodGet, path, nil)
	require.JSONEq(t, string(created.Data), string(detail.Data))

	updatedPayload := replaceJSONField(t, detail.Data, "affinityAware", map[string]any{"key": "region", "ratio": 60})
	updated := callRouterRuleAPI(t, router, sessionCookie, http.MethodPut, path, updatedPayload)
	require.NotContains(t, string(updated.Data), `"runtime"`)
	require.NotContains(t, string(updated.Data), `"enabled"`)

	versionPath := "/api/v1/affinity-rule/" + name + "/versions?mesh=" + mesh
	versionList := callRouterRuleAPI(t, router, sessionCookie, http.MethodGet, versionPath, nil)
	var versionData struct {
		Items []struct {
			VersionNo int64 `json:"versionNo"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(versionList.Data, &versionData))
	require.Len(t, versionData.Items, 2)
	callRouterRuleAPI(t, router, sessionCookie, http.MethodPost,
		fmt.Sprintf("/api/v1/affinity-rule/%s/versions/%d/rollback?mesh=%s", name, versionData.Items[1].VersionNo, mesh),
		mustRouterRuleJSON(t, map[string]string{"reason": "verify optional affinity fields"}))

	restored := callRouterRuleAPI(t, router, sessionCookie, http.MethodGet, path, nil)
	require.NotContains(t, string(restored.Data), `"runtime"`)
	require.NotContains(t, string(restored.Data), `"enabled"`)
	var restoredData struct {
		AffinityAware struct {
			Ratio int `json:"ratio"`
		} `json:"affinityAware"`
	}
	require.NoError(t, json.Unmarshal(restored.Data, &restoredData))
	require.Equal(t, 50, restoredData.AffinityAware.Ratio)

	callRouterRuleAPI(t, router, sessionCookie, http.MethodDelete, path, nil)
}

func TestRouterRuleAPIContractRejectsInvalidRuleRequests(t *testing.T) {
	router, sessionCookie := newRouterRuleAPI(t)

	wrongSuffix := callRouterRuleAPIExpect(t, router, sessionCookie, http.MethodPost,
		"/api/v1/script-rule/demo.affinity-router", mustRouterRuleJSON(t, map[string]any{}), http.StatusBadRequest)
	require.Equal(t, string(bizerror.InvalidArgument), wrongSuffix.Code)

	invalidScript := callRouterRuleAPIExpect(t, router, sessionCookie, http.MethodPost,
		"/api/v1/script-rule/demo.script-router", mustRouterRuleJSON(t, map[string]any{
			"configVersion": "v3.0",
			"scope":         "application",
			"key":           "demo",
			"type":          "groovy",
			"script":        "return invokers;",
		}), http.StatusOK)
	require.Equal(t, string(bizerror.InvalidArgument), invalidScript.Code)
}

func newRouterRuleAPI(t *testing.T) (*gin.Engine, *http.Cookie) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	affinityStore := memoryst.NewMemoryResourceStore(meshresource.AffinityRouteKind)
	scriptStore := memoryst.NewMemoryResourceStore(meshresource.ScriptRouteKind)
	versionStore := memoryst.NewMemoryResourceStore(meshresource.RuleVersionKind)
	for _, resourceStore := range []store.ManagedResourceStore{affinityStore, scriptStore, versionStore} {
		require.NoError(t, resourceStore.Init(nil))
	}
	stores := map[coremodel.ResourceKind]store.ResourceStore{
		meshresource.AffinityRouteKind: affinityStore,
		meshresource.ScriptRouteKind:   scriptStore,
		meshresource.RuleVersionKind:   versionStore,
	}
	storeRouter := &routerRuleAPITestStoreRouter{stores: stores}
	resourceManager := manager.NewResourceManager(
		storeRouter,
		&routerRuleAPITestGovernorRouter{governor: &routerRuleAPITestGovernor{stores: stores}},
	)
	ctx := &routerRuleAPITestContext{
		rm: resourceManager,
		config: appcfg.AdminConfig{
			Console: &consolecfg.Config{Auth: &configauth.Config{
				Methods:        []string{configauth.MethodPassword},
				User:           "admin",
				Password:       "secret",
				ExpirationTime: 3600,
				Providers:      map[string]configauth.ProviderConfig{},
			}},
			RuleVersioning: &versioningcfg.Config{MaxVersionsPerRule: 5},
		},
		versioning: versioning.NewService(5, versioning.NewResourceStoreAdapter(versionStore)),
	}

	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("router-rule-api-test-secret"))))
	router.Use(consoleauth.SessionMiddleware())
	require.NoError(t, InitRouter(router, ctx))

	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString("user=admin&password=secret"))
	login.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRecorder := httptest.NewRecorder()
	router.ServeHTTP(loginRecorder, login)
	require.Equal(t, http.StatusOK, loginRecorder.Code, loginRecorder.Body.String())
	require.NotEmpty(t, loginRecorder.Result().Cookies())
	return router, loginRecorder.Result().Cookies()[0]
}

func callRouterRuleAPI(t *testing.T, router http.Handler, sessionCookie *http.Cookie, method, path string, body []byte) routerRuleAPIResponse {
	t.Helper()
	response := callRouterRuleAPIExpect(t, router, sessionCookie, method, path, body, http.StatusOK)
	require.Equal(t, "Success", response.Code, "%s %s returned %s", method, path, response.Code)
	return response
}

func callRouterRuleAPIExpect(t *testing.T, router http.Handler, sessionCookie *http.Cookie, method, path string, body []byte, status int) routerRuleAPIResponse {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.AddCookie(sessionCookie)
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(t, status, recorder.Code, "%s %s: %s", method, path, recorder.Body.String())

	var response routerRuleAPIResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func mustRouterRuleJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	return encoded
}

func replaceJSONField(t *testing.T, raw json.RawMessage, field string, value any) []byte {
	t.Helper()
	var object map[string]any
	require.NoError(t, json.Unmarshal(raw, &object))
	object[field] = value
	return mustRouterRuleJSON(t, object)
}
