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

package subscriber

import (
	"testing"

	"github.com/stretchr/testify/require"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

func TestNacosInterfaceRegistrationNormalizesHealthAndProtocol(t *testing.T) {
	service := meshresource.NewNacosServiceResourceWithAttributes("providers:com.example.OrderService:v1:", "mesh-1")
	service.Spec.Instances = []*meshproto.NacosInstance{{
		Ip: "127.0.0.1", Port: 20880, Healthy: true, Enabled: true, Weight: 1,
		Metadata: map[string]string{
			constants.Application:  "orders",
			constants.URLParamsKey: `{"protocol":"tri"}`,
		},
	}}
	resources, err := nacosInterfaceExports(service)
	require.NoError(t, err)
	require.Len(t, resources, 1)
	for _, res := range resources {
		require.Equal(t, "orders", res.Spec.App)
		require.NotEmpty(t, res.Spec.InstanceName)
		require.True(t, *res.Spec.Available)
		export := res.Spec.Services["com.example.OrderService:v1:"]
		require.Equal(t, "tri", export.Protocol)
		require.Equal(t, "v1", export.Version)
	}
	service.Spec.Instances[0].Healthy = false
	resources, err = nacosInterfaceExports(service)
	require.NoError(t, err)
	for _, res := range resources {
		require.False(t, *res.Spec.Available)
	}
}

func TestNacosInterfaceRegistrationWithoutApplicationStillHasEvidence(t *testing.T) {
	service := meshresource.NewNacosServiceResourceWithAttributes("providers:com.example.OrderService::", "mesh-1")
	service.Spec.Instances = []*meshproto.NacosInstance{{
		Ip: "127.0.0.1", Port: 20880, Healthy: true, Enabled: true, Weight: 1,
		Metadata: map[string]string{constants.URLParamsKey: `{"protocol":"tri"}`},
	}}
	resources, err := nacosInterfaceExports(service)
	require.NoError(t, err)
	require.Len(t, resources, 1)
	for _, res := range resources {
		require.Empty(t, res.Spec.App)
		require.NotEmpty(t, res.Spec.InstanceName)
		require.True(t, *res.Spec.Available)
		require.Equal(t, "tri", res.Spec.Services["com.example.OrderService::"].Protocol)
	}
}
