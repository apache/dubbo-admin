/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
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
	"fmt"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
)

func dashboardTestContext(t *testing.T) (*testContext, store.ResourceStore) {
	t.Helper()
	s := memoryst.NewMemoryResourceStore(meshresource.ServiceProviderMetadataKind)
	require.NoError(t, s.Init(nil))
	return &testContext{rm: manager.NewResourceManager(&testRouter{stores: map[coremodel.ResourceKind]store.ResourceStore{meshresource.ServiceProviderMetadataKind: s}}, nil)}, s
}

func instanceDashboardTestContext(t *testing.T) (*testContext, store.ResourceStore) {
	t.Helper()
	s := memoryst.NewMemoryResourceStore(meshresource.InstanceKind)
	require.NoError(t, s.Init(nil))
	return &testContext{rm: manager.NewResourceManager(&testRouter{stores: map[coremodel.ResourceKind]store.ResourceStore{meshresource.InstanceKind: s}}, nil)}, s
}

func TestInstanceLogDashboardURLUsesBarePodIP(t *testing.T) {
	for _, appName := range []string{"shop-user", "shop.user"} {
		t.Run(appName, func(t *testing.T) {
			ctx, s := instanceDashboardTestContext(t)
			instance := meshresource.NewInstanceResourceWithAttributes("instance", "mesh")
			instance.Spec.AppName = appName
			instance.Spec.Ip = "10.42.1.10"
			instance.Spec.QosPort = 22222
			require.NoError(t, s.Add(instance))
			base, err := url.Parse("http://grafana/d/instance-log?kiosk=1")
			require.NoError(t, err)

			result, err := GetInstanceLogDashboard(ctx, base, &model.InstanceDashboardReq{Mesh: "mesh", InstanceName: "instance"})
			require.NoError(t, err)
			parsed, err := url.Parse(result)
			require.NoError(t, err)
			require.Equal(t, appName, parsed.Query().Get("var-application"))
			require.Equal(t, appName, parsed.Query().Get("var-applicationRaw"))
			require.Equal(t, "10.42.1.10", parsed.Query().Get("var-instance"))
			require.NotEqual(t, "10.42.1.10:22222", parsed.Query().Get("var-instance"))
		})
	}
}

func TestInstanceLogDashboardErrors(t *testing.T) {
	ctx, _ := instanceDashboardTestContext(t)
	req := &model.InstanceDashboardReq{Mesh: "mesh", InstanceName: "missing"}
	result, err := GetInstanceLogDashboard(ctx, nil, req)
	require.ErrorContains(t, err, "grafana url is not configured")
	require.Empty(t, result)

	base, parseErr := url.Parse("http://grafana/d/instance-log")
	require.NoError(t, parseErr)
	result, err = GetInstanceLogDashboard(ctx, base, req)
	require.ErrorContains(t, err, "instance missing not found")
	require.Empty(t, result)

	lookupFailureCtx := &testContext{rm: manager.NewResourceManager(&testRouter{stores: map[coremodel.ResourceKind]store.ResourceStore{}}, nil)}
	result, err = GetInstanceLogDashboard(lookupFailureCtx, base, req)
	require.Error(t, err)
	require.Empty(t, result)
}

func TestInstanceMetricAndTraceDashboardVariables(t *testing.T) {
	for _, tc := range []struct {
		name           string
		qosPort        int64
		metricInstance string
	}{
		{name: "explicit QoS port", qosPort: 22222, metricInstance: "10.42.1.10:22222"},
		{name: "default QoS port", qosPort: 0, metricInstance: "10.42.1.10:22222"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, s := instanceDashboardTestContext(t)
			instance := meshresource.NewInstanceResourceWithAttributes("instance", "mesh")
			instance.Spec.AppName = "shop-user"
			instance.Spec.Ip = "10.42.1.10"
			instance.Spec.QosPort = tc.qosPort
			require.NoError(t, s.Add(instance))
			req := &model.InstanceDashboardReq{Mesh: "mesh", InstanceName: "instance"}

			metric, err := GetInstanceDashboardVariables(ctx, req)
			require.NoError(t, err)
			require.Equal(t, "shop-user", metric["var-application"])
			require.Equal(t, tc.metricInstance, metric["var-instance"])

			trace, err := GetInstanceTraceDashboardVariables(ctx, req)
			require.NoError(t, err)
			require.Equal(t, "shop-user", trace["var-application"])
			require.Equal(t, "10.42.1.10", trace["var-instance"])
			require.NotEqual(t, metric["var-instance"], trace["var-instance"])
		})
	}
}

func TestInstanceTraceDashboardURL(t *testing.T) {
	ctx, s := instanceDashboardTestContext(t)
	instance := meshresource.NewInstanceResourceWithAttributes("instance", "mesh")
	instance.Spec.AppName = "shop-user"
	instance.Spec.Ip = "10.42.1.10"
	instance.Spec.QosPort = 22222
	require.NoError(t, s.Add(instance))
	base, err := url.Parse("http://grafana/d/instance?kiosk=1")
	require.NoError(t, err)

	result, err := GetInstanceTraceDashboard(ctx, base, &model.InstanceDashboardReq{Mesh: "mesh", InstanceName: "instance"})
	require.NoError(t, err)
	parsed, err := url.Parse(result)
	require.NoError(t, err)
	require.Equal(t, url.Values{
		"kiosk":           {"1"},
		"var-application": {"shop-user"},
		"var-instance":    {"10.42.1.10"},
	}, parsed.Query())
	require.Equal(t, "kiosk=1", base.RawQuery)
}

func TestInstanceTraceDashboardErrors(t *testing.T) {
	ctx, _ := instanceDashboardTestContext(t)
	req := &model.InstanceDashboardReq{Mesh: "mesh", InstanceName: "missing"}

	result, err := GetInstanceTraceDashboard(ctx, nil, req)
	require.ErrorContains(t, err, "grafana url is not configured")
	require.Empty(t, result)

	base, parseErr := url.Parse("http://grafana/d/instance")
	require.NoError(t, parseErr)
	result, err = GetInstanceTraceDashboard(ctx, base, req)
	require.ErrorContains(t, err, "instance missing not found")
	require.Empty(t, result)

	lookupFailureCtx := &testContext{rm: manager.NewResourceManager(&testRouter{stores: map[coremodel.ResourceKind]store.ResourceStore{}}, nil)}
	result, err = GetInstanceTraceDashboard(lookupFailureCtx, base, req)
	require.Error(t, err)
	require.Empty(t, result)
}

func TestServiceTraceDashboardProviderApplications(t *testing.T) {
	for _, tc := range []struct {
		name         string
		applications []string
		wantError    string
	}{
		{"unique application", []string{"provider"}, ""},
		{"duplicate application", []string{"provider", "provider"}, ""},
		{"no providers", nil, "no provider application"},
		{"empty application", []string{"", " "}, "no provider application"},
		{"different applications", []string{"provider", "other"}, "multiple provider applications"},
		{"reversed applications", []string{"other", "provider"}, "multiple provider applications"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, s := dashboardTestContext(t)
			for i, app := range tc.applications {
				provider := meshresource.NewServiceProviderMetadataResourceWithAttributes(fmt.Sprint(i), "mesh")
				provider.Spec.ServiceName = "example.Service"
				provider.Spec.Group = "group"
				provider.Spec.Version = "version"
				provider.Spec.ProviderAppName = app
				require.NoError(t, s.Add(provider))
			}
			base, err := url.Parse("http://grafana/d/service?kiosk=1")
			require.NoError(t, err)
			req := &model.ServiceDashboardReq{Mesh: "mesh", ServiceName: "example.Service", Group: "group", Version: "version"}
			result, err := GetServiceTraceDashboard(ctx, base, req)
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				require.Empty(t, result)
			} else {
				require.NoError(t, err)
				parsed, err := url.Parse(result)
				require.NoError(t, err)
				require.Equal(t, "provider", parsed.Query().Get("var-application"))
				require.Equal(t, "example.Service", parsed.Query().Get("var-service"))
			}
			require.Equal(t, "kiosk=1", base.RawQuery)
			metric, err := GetServiceDashboard(base, req)
			require.NoError(t, err)
			parsed, err := url.Parse(metric)
			require.NoError(t, err)
			require.Equal(t, url.Values{"kiosk": {"1"}, "var-service": {"example.Service"}}, parsed.Query())
		})
	}
}

func TestServiceTraceDashboardMetadataScope(t *testing.T) {
	ctx, s := dashboardTestContext(t)
	for i, fields := range [][4]string{
		{"mesh", "example.Service", "group", "version"},
		{"other-mesh", "example.Service", "group", "version"},
		{"mesh", "other.Service", "group", "version"},
		{"mesh", "example.Service", "other-group", "version"},
		{"mesh", "example.Service", "group", "other-version"},
	} {
		provider := meshresource.NewServiceProviderMetadataResourceWithAttributes(fmt.Sprint(i), fields[0])
		provider.Spec.ServiceName, provider.Spec.Group, provider.Spec.Version = fields[1], fields[2], fields[3]
		provider.Spec.ProviderAppName = fmt.Sprintf("provider-%d", i)
		require.NoError(t, s.Add(provider))
	}
	base, err := url.Parse("http://grafana/d/service")
	require.NoError(t, err)
	result, err := GetServiceTraceDashboard(ctx, base, &model.ServiceDashboardReq{Mesh: "mesh", ServiceName: "example.Service", Group: "group", Version: "version"})
	require.NoError(t, err)
	parsed, err := url.Parse(result)
	require.NoError(t, err)
	require.Equal(t, "provider-0", parsed.Query().Get("var-application"))
}

func TestServiceTraceDashboardMetadataLookupFailure(t *testing.T) {
	ctx := &testContext{rm: manager.NewResourceManager(&testRouter{stores: map[coremodel.ResourceKind]store.ResourceStore{}}, nil)}
	base, err := url.Parse("http://grafana/d/service")
	require.NoError(t, err)
	result, err := GetServiceTraceDashboard(ctx, base, &model.ServiceDashboardReq{Mesh: "mesh", ServiceName: "example.Service"})
	require.Error(t, err)
	require.Empty(t, result)
}

func TestApplicationDashboardUnchanged(t *testing.T) {
	base, err := url.Parse("http://grafana/d/application?kiosk=1")
	require.NoError(t, err)
	result, err := GetAppDashboard(base, &model.AppDashboardReq{AppName: "provider"})
	require.NoError(t, err)
	parsed, err := url.Parse(result)
	require.NoError(t, err)
	require.Equal(t, url.Values{"kiosk": {"1"}, "var-application": {"provider"}}, parsed.Query())
}

func TestApplicationLogDashboardURL(t *testing.T) {
	base, err := url.Parse("http://grafana/d/application-log?kiosk=1")
	require.NoError(t, err)
	for _, tc := range []struct {
		appName string
		escaped string
	}{
		{appName: "shop-user", escaped: "shop-user"},
		{appName: "shop.user", escaped: `shop\\.user`},
	} {
		t.Run(tc.appName, func(t *testing.T) {
			result, err := GetApplicationLogDashboard(base, &model.AppDashboardReq{AppName: tc.appName})
			require.NoError(t, err)
			parsed, err := url.Parse(result)
			require.NoError(t, err)
			require.Equal(t, tc.escaped, parsed.Query().Get("var-application"))
			require.Equal(t, tc.appName, parsed.Query().Get("var-applicationRaw"))
		})
	}
}

func TestTraceLogDashboardURL(t *testing.T) {
	base, err := url.Parse("http://grafana/d/dubbo-application-loki-logs?kiosk=1")
	require.NoError(t, err)
	result, err := GetTraceLogDashboard(base, &model.TraceLogDashboardReq{
		TraceID: "0123456789abcdef0123456789abcdef",
		From:    "1000",
		To:      "601000",
	})
	require.NoError(t, err)
	parsed, err := url.Parse(result)
	require.NoError(t, err)
	require.Equal(t, ".*", parsed.Query().Get("var-application"))
	require.Equal(t, ".*", parsed.Query().Get("var-instance"))
	require.False(t, parsed.Query().Has("var-applicationRaw"))
	require.Equal(t, "0123456789abcdef0123456789abcdef", parsed.Query().Get("var-traceId"))
	require.Equal(t, "1000", parsed.Query().Get("from"))
	require.Equal(t, "601000", parsed.Query().Get("to"))
}

func TestTraceLogDashboardRejectsBroadOrInvalidInput(t *testing.T) {
	base, err := url.Parse("http://grafana/d/dubbo-application-loki-logs")
	require.NoError(t, err)
	_, err = GetTraceLogDashboard(base, &model.TraceLogDashboardReq{TraceID: "bad", From: "1", To: "2"})
	require.ErrorContains(t, err, "16 or 32 hexadecimal")
	_, err = GetTraceLogDashboard(base, &model.TraceLogDashboardReq{TraceID: "0123456789abcdef", From: "", To: "2"})
	require.ErrorContains(t, err, "time range is required")
	for _, tc := range []struct {
		name string
		from string
		to   string
	}{
		{"non-numeric start", "yesterday", "601000"},
		{"non-numeric end", "1000", "now"},
		{"fractional milliseconds", "1000.5", "601000"},
		{"negative start", "-1", "1000"},
		{"equal boundaries", "1000", "1000"},
		{"reversed boundaries", "2000", "1000"},
		{"over ten minutes", "1000", "601001"},
		{"integer overflow", "1000", "9223372036854775808"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := GetTraceLogDashboard(base, &model.TraceLogDashboardReq{
				TraceID: "0123456789abcdef", From: tc.from, To: tc.to,
			})
			require.ErrorContains(t, err, "valid millisecond time range")
			require.Empty(t, result)
		})
	}
}

func TestTraceLogDashboardLowercasesTraceIDWithoutChangingWidth(t *testing.T) {
	base, err := url.Parse("http://grafana/d/logs")
	require.NoError(t, err)
	for _, id := range []string{"ABCDEF0123456789", "0000000000000000ABCDEF0123456789", "ABCDEF0123456789ABCDEF0123456789"} {
		t.Run(id, func(t *testing.T) {
			result, err := GetTraceLogDashboard(base, &model.TraceLogDashboardReq{
				TraceID: " " + id + " ", From: "1000", To: "601000",
			})
			require.NoError(t, err)
			parsed, err := url.Parse(result)
			require.NoError(t, err)
			require.Equal(t, strings.ToLower(id), parsed.Query().Get("var-traceId"))
		})
	}
}

func TestDashboardURLDoesNotMutateSharedBase(t *testing.T) {
	base, err := url.Parse("http://grafana/d/application?kiosk=1&var-application=default")
	require.NoError(t, err)
	original := base.String()

	for _, app := range []string{"first", "second"} {
		result, err := GetAppDashboard(base, &model.AppDashboardReq{AppName: app})
		require.NoError(t, err)
		parsed, err := url.Parse(result)
		require.NoError(t, err)
		require.Equal(t, app, parsed.Query().Get("var-application"))
		require.Equal(t, "1", parsed.Query().Get("kiosk"))
		require.Equal(t, original, base.String())
	}
}

func TestDashboardURLConcurrentRequestsKeepOwnVariables(t *testing.T) {
	base, err := url.Parse("http://grafana/d/application?kiosk=1")
	require.NoError(t, err)
	const count = 100
	results := make([]string, count)
	errors := make([]error, count)
	var workers sync.WaitGroup
	workers.Add(count)
	for i := 0; i < count; i++ {
		go func(i int) {
			defer workers.Done()
			results[i], errors[i] = GetAppDashboard(base, &model.AppDashboardReq{AppName: fmt.Sprintf("app-%d", i)})
		}(i)
	}
	workers.Wait()
	for i := 0; i < count; i++ {
		require.NoError(t, errors[i])
		parsed, err := url.Parse(results[i])
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("app-%d", i), parsed.Query().Get("var-application"))
		require.Equal(t, "1", parsed.Query().Get("kiosk"))
	}
	require.Equal(t, "kiosk=1", base.RawQuery)
}
