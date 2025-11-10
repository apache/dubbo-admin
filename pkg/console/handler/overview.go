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

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/counter"
	"github.com/apache/dubbo-admin/pkg/console/model"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

func AdminMetadata(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		res := model.NewAdminMetadata()
		// TODO
		c.JSON(http.StatusOK, model.NewSuccessResp(res))
	}
}

// ClusterOverview TODO implement
func ClusterOverview(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp := model.NewOverviewResp()
		if counterMgr := ctx.CounterManager(); counterMgr != nil {
			resp.AppCount = counterMgr.Count(meshresource.ApplicationKind)
			resp.ServiceCount = counterMgr.Count(meshresource.ServiceKind)
			resp.InsCount = counterMgr.Count(meshresource.InstanceKind)
			resp.Protocols = counterMgr.Distribution(counter.ProtocolCounter)
			resp.Releases = counterMgr.Distribution(counter.ReleaseCounter)
			resp.Discoveries = counterMgr.Distribution(counter.DiscoveryCounter)
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}
