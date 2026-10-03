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

func instanceMetricTestContext(t *testing.T) (*testContext, store.ResourceStore) {
	t.Helper()
	s := memoryst.NewMemoryResourceStore(meshresource.InstanceKind)
	require.NoError(t, s.Init(nil))
	return &testContext{rm: manager.NewResourceManager(&testRouter{stores: map[coremodel.ResourceKind]store.ResourceStore{meshresource.InstanceKind: s}}, nil)}, s
}

func TestInstanceMetricDashboardVariables(t *testing.T) {
	for _, qosPort := range []int64{0, 22222} {
		t.Run(fmt.Sprintf("qos-%d", qosPort), func(t *testing.T) {
			ctx, s := instanceMetricTestContext(t)
			instance := meshresource.NewInstanceResourceWithAttributes("instance", "mesh")
			instance.Spec.AppName = "shop-user"
			instance.Spec.Ip = "10.42.1.10"
			instance.Spec.QosPort = qosPort
			require.NoError(t, s.Add(instance))

			variables, err := GetInstanceDashboardVariables(ctx, &model.InstanceDashboardReq{Mesh: "mesh", InstanceName: "instance"})
			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"var-application": "shop-user",
				"var-instance":    "10.42.1.10",
			}, variables)
		})
	}
}

func TestInstanceMetricDashboardURLUsesBareIP(t *testing.T) {
	for _, qosPort := range []int64{0, 22222} {
		t.Run(fmt.Sprintf("qos-%d", qosPort), func(t *testing.T) {
			ctx, s := instanceMetricTestContext(t)
			instance := meshresource.NewInstanceResourceWithAttributes("instance", "mesh")
			instance.Spec.AppName = "shop-user"
			instance.Spec.Ip = "10.42.1.10"
			instance.Spec.QosPort = qosPort
			require.NoError(t, s.Add(instance))

			base, err := url.Parse("http://grafana/d/instance?kiosk=1")
			require.NoError(t, err)
			result, err := GetInstanceDashboard(ctx, base, &model.InstanceDashboardReq{Mesh: "mesh", InstanceName: "instance"})
			require.NoError(t, err)
			parsed, err := url.Parse(result)
			require.NoError(t, err)
			require.Equal(t, "shop-user", parsed.Query().Get("var-application"))
			require.Equal(t, "10.42.1.10", parsed.Query().Get("var-instance"))
			require.Equal(t, "1", parsed.Query().Get("kiosk"))
			require.NotContains(t, parsed.Query().Get("var-instance"), ":")
		})
	}
}

func TestInstanceMetricDashboardErrors(t *testing.T) {
	ctx, _ := instanceMetricTestContext(t)
	req := &model.InstanceDashboardReq{Mesh: "mesh", InstanceName: "missing"}
	result, err := GetInstanceDashboard(ctx, nil, req)
	require.ErrorContains(t, err, "grafana url is not configured")
	require.Empty(t, result)

	variables, err := GetInstanceDashboardVariables(ctx, req)
	require.ErrorContains(t, err, "instance missing not found")
	require.Nil(t, variables)

	base, err := url.Parse("http://grafana/d/instance")
	require.NoError(t, err)
	result, err = GetInstanceDashboard(ctx, base, req)
	require.ErrorContains(t, err, "instance missing not found")
	require.Empty(t, result)

	lookupFailureCtx := &testContext{rm: manager.NewResourceManager(&testRouter{stores: map[coremodel.ResourceKind]store.ResourceStore{}}, nil)}
	variables, err = GetInstanceDashboardVariables(lookupFailureCtx, req)
	require.Error(t, err)
	require.Nil(t, variables)
	result, err = GetInstanceDashboard(lookupFailureCtx, base, req)
	require.Error(t, err)
	require.Empty(t, result)
}
