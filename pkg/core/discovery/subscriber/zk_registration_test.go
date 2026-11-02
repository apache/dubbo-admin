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
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestZKProviderRegistrationNormalizesEndpoint(t *testing.T) {
	endpoint := "tri://10.0.0.1:50051/com.example.OrderService?application=orders&enabled=true&group=payments&version=v2&weight=1"
	path := "/dubbo/com.example.OrderService/providers/" + url.PathEscape(endpoint)
	resource := zkInterfaceRegistration("mesh-1", path)
	require.NotNil(t, resource)
	registration := resource.Spec
	require.Equal(t, "orders", registration.App)
	require.Equal(t, "orders10.0.0.1:50051", registration.InstanceName)
	require.True(t, *registration.Available)
	service := registration.Services["com.example.OrderService:v2:payments"]
	require.Equal(t, "com.example.OrderService", service.Name)
	require.Equal(t, "payments", service.Group)
	require.Equal(t, "v2", service.Version)
	require.Equal(t, "tri", service.Protocol)
}

func TestZKProviderRegistrationAvailabilityUsesRegistrationFlags(t *testing.T) {
	for _, endpoint := range []string{
		"dubbo://10.0.0.1:20880/com.example.OrderService?application=orders&enabled=false",
		"dubbo://10.0.0.1:20880/com.example.OrderService?application=orders&weight=0",
	} {
		path := "/dubbo/com.example.OrderService/providers/" + url.PathEscape(endpoint)
		resource := zkInterfaceRegistration("mesh-1", path)
		require.NotNil(t, resource)
		require.False(t, *resource.Spec.Available)
	}
}

func TestZKProviderRegistrationWithoutApplicationStillHasEvidence(t *testing.T) {
	endpoint := "tri://10.0.0.1:50051/com.example.OrderService?enabled=true"
	path := "/dubbo/com.example.OrderService/providers/" + url.PathEscape(endpoint)
	resource := zkInterfaceRegistration("mesh-1", path)
	require.NotNil(t, resource)
	require.Empty(t, resource.Spec.App)
	require.NotEmpty(t, resource.Spec.InstanceName)
	require.True(t, *resource.Spec.Available)
	require.Equal(t, "tri", resource.Spec.Services["com.example.OrderService::"].Protocol)
}
