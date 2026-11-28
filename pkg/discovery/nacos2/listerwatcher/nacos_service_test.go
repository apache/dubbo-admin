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

package listerwatcher

import (
	"errors"
	"testing"

	nacosconfigclient "github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	nacosvo "github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/stretchr/testify/require"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

type revisionConfigClient struct {
	nacosconfigclient.IConfigClient
	wantApp      string
	wantRevision string
	content      string
	err          error
}

func (c revisionConfigClient) GetConfig(param nacosvo.ConfigParam) (string, error) {
	if param.DataId != c.wantApp || param.Group != c.wantRevision {
		return "", errors.New("unexpected metadata key")
	}
	return c.content, c.err
}

func TestNacosRevisionMetadataIsNormalizedBeforeStorage(t *testing.T) {
	lw := &NacosServiceListerWatcher{configClient: revisionConfigClient{
		wantApp: "orders", wantRevision: "rev-2",
		content: `{"services":{"/com.example.OrderService::tri":{"port":50051}}}`,
	}}
	res := meshresource.NewNacosServiceResourceWithAttributes("com.example.OrderService", "mesh-1")
	res.Spec.Instances = []*meshproto.NacosInstance{
		{Metadata: map[string]string{
			constants.Application:         "orders",
			constants.MetadataRevisionKey: "rev-2",
		}},
	}
	require.NoError(t, lw.resolveRevisions(res))
	require.Len(t, res.Spec.Revisions, 1)
	require.Equal(t, "tri", res.Spec.Revisions[0].Services["/com.example.OrderService::tri"].Protocol)
	require.Equal(t, "orders", res.Spec.Revisions[0].App)

	lw.configClient = revisionConfigClient{wantApp: "orders", wantRevision: "rev-2", err: errors.New("offline")}
	res.Spec.Revisions = nil
	require.ErrorContains(t, lw.resolveRevisions(res), "offline")
	require.Empty(t, res.Spec.Revisions)
}

func TestNacosRevisionMetadataFallsBackToServiceName(t *testing.T) {
	lw := &NacosServiceListerWatcher{configClient: revisionConfigClient{
		wantApp: "orders", wantRevision: "rev-2",
		content: `{"services":{"com.example.OrderService:tri":{}}}`,
	}}
	res := meshresource.NewNacosServiceResourceWithAttributes("orders", "mesh-1")
	res.Spec.Instances = []*meshproto.NacosInstance{{
		Metadata: map[string]string{constants.MetadataRevisionKey: "rev-2"},
	}}
	require.NoError(t, lw.resolveRevisions(res))
	require.Len(t, res.Spec.Revisions, 1)
	require.Equal(t, "orders", res.Spec.Revisions[0].App)
}

func TestNacosWatchHealthKeepsPollingAndSubscriptionErrorsIndependent(t *testing.T) {
	lw := &NacosServiceListerWatcher{}
	var healthy bool
	lw.SetWatchHealthHandler(func(err error) { healthy = err == nil })
	lw.reportHealth(errors.New("poll failed"))
	require.False(t, healthy)
	lw.reportServiceHealth("orders", errors.New("subscription failed"))
	require.False(t, healthy)
	lw.reportHealth(nil)
	require.False(t, healthy, "a successful poll cannot mask a broken subscription")
	lw.reportServiceHealth("orders", nil)
	require.True(t, healthy)
	lw.reportServiceHealth("orders", errors.New("subscription failed"))
	lw.reportHealth(errors.New("poll failed"))
	lw.reportServiceHealth("orders", nil)
	require.False(t, healthy, "a successful callback cannot mask a broken poll")
	lw.reportHealth(nil)
	require.True(t, healthy)
}
