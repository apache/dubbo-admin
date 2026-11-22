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
	appcfg "github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProviderInstancesExposeInvocationMetadata(t *testing.T) {
	stores := map[coremodel.ResourceKind]store.ResourceStore{}
	for _, kind := range []coremodel.ResourceKind{meshresource.InstanceKind, meshresource.ServiceProviderMetadataKind} {
		memoryStore := memoryst.NewMemoryResourceStore(kind)
		require.NoError(t, memoryStore.Init(nil))
		stores[kind] = memoryStore
	}
	ctx := &testContext{cfg: &appcfg.AdminConfig{}, rm: manager.NewResourceManager(&testRouter{stores: stores}, nil)}
	meta := meshresource.NewServiceProviderMetadataResourceWithAttributes("provider", "mesh")
	meta.Spec.ServiceName = "Echo"
	meta.Spec.ProviderAppName = "app"
	meta.Spec.Parameters = map[string]string{"generic": "gson"}
	require.NoError(t, stores[meshresource.ServiceProviderMetadataKind].Add(meta))
	instance := meshresource.NewInstanceResourceWithAttributes("instance", "mesh")
	instance.Spec.Name = "instance"
	instance.Spec.AppName = "app"
	instance.Spec.Serialization = "hessian2"
	instance.Spec.PreferSerialization = "fastjson2,hessian2"
	require.NoError(t, stores[meshresource.InstanceKind].Add(instance))
	responses, err := GetServiceProviderInstances(ctx, model.BaseServiceReq{Mesh: "mesh", ServiceName: "Echo"})
	require.NoError(t, err)
	require.Len(t, responses, 1)
	require.Equal(t, "gson", responses[0].Generic)
	require.Equal(t, "hessian2", responses[0].Serialization)
	require.Equal(t, "fastjson2,hessian2", responses[0].PreferSerialization)
}
