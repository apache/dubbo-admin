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
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/counter"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/util"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

func AdminMetadata(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		mesh, exists := c.GetQuery("mesh")
		if !exists {
			util.HandleServiceError(c, bizerror.New(bizerror.InvalidArgument, "mesh is required"))
			return
		}
		var registryAddr string
		var metadataAddr string
		var configAddr string
		if d := ctx.Config().FindDiscovery(mesh); d != nil {
			registryAddr = d.Address.Registry
			metadataAddr = d.Address.MetadataReport
			configAddr = d.Address.ConfigCenter
		}
		var prometheusURL string
		var grafanaURL string
		if ctx.Config().Observability.PrometheusBaseURL != nil {
			prometheusURL = ctx.Config().Observability.PrometheusBaseURL.String()
		}
		if ctx.Config().Observability.GrafanaBaseURL != nil {
			grafanaURL = ctx.Config().Observability.GrafanaBaseURL.String()
		}
		metadata := model.AdminMetadata{
			Registry:   registryAddr,
			Metadata:   metadataAddr,
			Config:     configAddr,
			Prometheus: prometheusURL,
			Grafana:    grafanaURL,
			Tracing:    "",
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(metadata))
	}
}

// ClusterOverview get cluster overview
func ClusterOverview(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := model.NewOverviewResp()
		if counterMgr := ctx.CounterManager(); counterMgr != nil {
			mesh := c.Query("mesh")

			if mesh != "" {
				resp.AppCount = counterMgr.CountByMesh(meshresource.ApplicationKind, mesh)
				resp.ServiceCount = counterMgr.CountByMesh(meshresource.ServiceProviderMetadataKind, mesh)
				resp.InsCount = counterMgr.CountByMesh(meshresource.InstanceKind, mesh)
				resp.Protocols = counterMgr.DistributionByMesh(counter.ProtocolCounter, mesh)
				resp.Releases = counterMgr.DistributionByMesh(counter.ReleaseCounter, mesh)
				resp.Discoveries = counterMgr.DistributionByMesh(counter.DiscoveryCounter, mesh)
			} else {
				resp.AppCount = counterMgr.Count(meshresource.ApplicationKind)
				resp.ServiceCount = counterMgr.Count(meshresource.ServiceProviderMetadataKind)
				resp.InsCount = counterMgr.Count(meshresource.InstanceKind)
				resp.Protocols = counterMgr.Distribution(counter.ProtocolCounter)
				resp.Releases = counterMgr.Distribution(counter.ReleaseCounter)
				resp.Discoveries = counterMgr.Distribution(counter.DiscoveryCounter)
			}
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}
