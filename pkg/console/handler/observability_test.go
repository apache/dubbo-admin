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

package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	appcfg "github.com/apache/dubbo-admin/pkg/config/app"
	consolecfg "github.com/apache/dubbo-admin/pkg/config/console"
	observabilitycfg "github.com/apache/dubbo-admin/pkg/config/observability"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type observabilityHandlerTestContext struct {
	consolectx.Context
	cfg appcfg.AdminConfig
	rm  manager.ResourceManager
}

func (c observabilityHandlerTestContext) Config() appcfg.AdminConfig               { return c.cfg }
func (c observabilityHandlerTestContext) AppContext() context.Context              { return context.Background() }
func (c observabilityHandlerTestContext) ResourceManager() manager.ResourceManager { return c.rm }

type observabilityInstanceManager struct {
	manager.ResourceManager
	instance *meshresource.InstanceResource
}

func (m observabilityInstanceManager) GetByKey(kind coremodel.ResourceKind, key string) (coremodel.Resource, bool, error) {
	if kind == meshresource.InstanceKind && key == m.instance.ResourceKey() {
		return m.instance, true, nil
	}
	return nil, false, nil
}

func callObservabilityHandler(t *testing.T, h gin.HandlerFunc, query string) model.CommonResp {
	t.Helper()
	c, recorder := testGinContext()
	c.Request = httptest.NewRequest(http.MethodGet, "/?"+query, nil)
	require.NotPanics(t, func() { h(c) })
	require.Equal(t, http.StatusOK, recorder.Code)
	// Unmarshalling also rejects multiple JSON responses if a missing return falls through.
	return decodeCommonResp(t, recorder)
}

func TestObservabilityMissingDashboardsReturnWithoutPanic(t *testing.T) {
	ctx := observabilityHandlerTestContext{cfg: appcfg.AdminConfig{Console: &consolecfg.Config{}}}
	for _, kind := range []DashboardType{MetricDashboard, TraceDashboard, LogDashboard} {
		for _, dim := range []Dimension{AppDimension, InstanceDimension} {
			t.Run(string(kind)+"/"+string(dim), func(t *testing.T) {
				resp := callObservabilityHandler(t, GetGrafanaDashboard(ctx, dim, kind), "")
				require.Equal(t, string(bizerror.NotFoundError), resp.Code)
				require.Nil(t, resp.Data)
			})
		}
	}
}

func TestObservabilityApplicationLogDispatch(t *testing.T) {
	base, err := url.Parse("http://grafana/d/application-logs?kiosk=1")
	require.NoError(t, err)
	ctx := observabilityHandlerTestContext{cfg: appcfg.AdminConfig{
		Console: &consolecfg.Config{LogDashboards: &consolecfg.GrafanaDashboardConfig{AppDashboardURL: base}},
	}}
	resp := callObservabilityHandler(t, GetGrafanaDashboard(ctx, AppDimension, LogDashboard), "appName=shop.user")
	require.Equal(t, "Success", resp.Code)
	parsed, err := url.Parse(resp.Data.(map[string]any)["fullURL"].(string))
	require.NoError(t, err)
	require.Equal(t, `shop\\.user`, parsed.Query().Get("var-application"))
	require.Equal(t, "shop.user", parsed.Query().Get("var-applicationRaw"))
	require.Equal(t, "kiosk=1", base.RawQuery)
}

func TestObservabilityInstanceDashboardDispatch(t *testing.T) {
	instance := meshresource.NewInstanceResourceWithAttributes("instance-a", "mesh")
	instance.Spec.AppName, instance.Spec.Ip, instance.Spec.QosPort = "shop.user", "10.42.1.10", 22222
	for _, tc := range []struct {
		kind   DashboardType
		wantIP string
	}{
		{MetricDashboard, "10.42.1.10:22222"},
		{TraceDashboard, "10.42.1.10"},
		{LogDashboard, "10.42.1.10"},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			base, err := url.Parse("http://grafana/d/" + string(tc.kind) + "?kiosk=1")
			require.NoError(t, err)
			dashboard := &consolecfg.GrafanaDashboardConfig{InstanceDashboardURL: base}
			ctx := observabilityHandlerTestContext{
				cfg: appcfg.AdminConfig{Console: &consolecfg.Config{
					MetricDashboards: dashboard, TraceDashboards: dashboard, LogDashboards: dashboard,
				}},
				rm: observabilityInstanceManager{instance: instance},
			}
			resp := callObservabilityHandler(t, GetGrafanaDashboard(ctx, InstanceDimension, tc.kind), "mesh=mesh&instanceName=instance-a")
			require.Equal(t, "Success", resp.Code)
			parsed, err := url.Parse(resp.Data.(map[string]any)["fullURL"].(string))
			require.NoError(t, err)
			require.Equal(t, "shop.user", parsed.Query().Get("var-application"))
			require.Equal(t, tc.wantIP, parsed.Query().Get("var-instance"))
			if tc.kind == LogDashboard {
				require.Equal(t, "shop.user", parsed.Query().Get("var-applicationRaw"))
			} else {
				require.False(t, parsed.Query().Has("var-applicationRaw"))
			}
			require.Equal(t, "kiosk=1", base.RawQuery)
		})
	}
}

func TestObservabilityTraceLogDashboardHandler(t *testing.T) {
	base, err := url.Parse("http://grafana/d/logs?theme=light")
	require.NoError(t, err)
	for _, tc := range []struct {
		name        string
		dashboards  *consolecfg.GrafanaDashboardConfig
		query, code string
	}{
		{"missing log config", nil, "traceId=abcdef0123456789&from=1000&to=601000", string(bizerror.NotFoundError)},
		{"missing trace URL", &consolecfg.GrafanaDashboardConfig{}, "traceId=abcdef0123456789&from=1000&to=601000", string(bizerror.NotFoundError)},
		{"invalid ID", &consolecfg.GrafanaDashboardConfig{TraceDashboardURL: base}, "traceId=bad&from=1000&to=601000", string(bizerror.InvalidArgument)},
		{"invalid range", &consolecfg.GrafanaDashboardConfig{TraceDashboardURL: base}, "traceId=abcdef0123456789&from=1000&to=601001", string(bizerror.InvalidArgument)},
		{"success", &consolecfg.GrafanaDashboardConfig{TraceDashboardURL: base}, "traceId=ABCDEF0123456789&from=1000&to=601000", "Success"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := observabilityHandlerTestContext{cfg: appcfg.AdminConfig{Console: &consolecfg.Config{LogDashboards: tc.dashboards}}}
			resp := callObservabilityHandler(t, GetTraceLogDashboard(ctx), tc.query)
			require.Equal(t, tc.code, resp.Code)
			if tc.code != "Success" {
				require.Nil(t, resp.Data)
				return
			}
			parsed, err := url.Parse(resp.Data.(map[string]any)["fullURL"].(string))
			require.NoError(t, err)
			require.Equal(t, "abcdef0123456789", parsed.Query().Get("var-traceId"))
			require.Equal(t, "1000", parsed.Query().Get("from"))
			require.Equal(t, "601000", parsed.Query().Get("to"))
			require.Equal(t, ".*", parsed.Query().Get("var-instance"))
		})
	}
	require.Equal(t, "theme=light", base.RawQuery)
}

func TestObservabilityTraceStartTimeHandler(t *testing.T) {
	for _, tc := range []struct {
		name, id, body, code string
		status               int
		configured           bool
	}{
		{"success", "ABCDEF0123456789", `{"data":[{"traceID":"0000000000000000abcdef0123456789","spans":[{"startTime":1500000}]}]}`, "Success", 200, true},
		{"invalid ID", "bad", "", string(bizerror.InvalidArgument), 200, true},
		{"missing config", "abcdef0123456789", "", string(bizerror.ConfigError), 200, false},
		{"not found", "abcdef0123456789", "", string(bizerror.NotFoundError), 404, true},
		{"upstream failure", "abcdef0123456789", "", string(bizerror.NetWorkError), 500, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/traces/abcdef0123456789" {
					t.Errorf("unexpected Jaeger path: %s", r.URL.Path)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			obs := &observabilitycfg.Config{}
			if tc.configured {
				base, err := url.Parse(server.URL)
				require.NoError(t, err)
				obs.JaegerBaseURL = base
			}
			ctx := observabilityHandlerTestContext{cfg: appcfg.AdminConfig{Observability: obs}}
			resp := callObservabilityHandler(t, GetTraceStartTime(ctx), "traceId="+tc.id)
			require.Equal(t, tc.code, resp.Code)
			if tc.code == "Success" {
				data := resp.Data.(map[string]any)
				require.Equal(t, "abcdef0123456789", data["traceId"])
				require.Equal(t, float64(1500), data["startTime"])
			} else {
				require.Nil(t, resp.Data)
			}
		})
	}
}
