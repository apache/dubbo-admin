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
