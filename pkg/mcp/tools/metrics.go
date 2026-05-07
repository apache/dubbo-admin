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

package tools

import (
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/counter"
	"github.com/apache/dubbo-admin/pkg/mcp/types"
	"github.com/apache/dubbo-admin/pkg/mcp/registry"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

// MetricsRegistrar 集群工具注册器
type MetricsRegistrar struct{}

// RegisterTools 实现 ToolRegistrar 接口
func (r *MetricsRegistrar) RegisterTools(reg *registry.Registry) {
	reg.Register(types.ToolDef{
		Name:        "get_cluster_info",
		Description: "获取 Dubbo 集群基本信息，包括应用数、服务数、实例数等统计信息",
		InputSchema: types.InputSchema{
			Type: "object",
			Properties: map[string]types.PropertyDef{
				"mesh": {
					Type:        "string",
					Description: "Mesh 名称，默认使用配置中的默认 mesh",
				},
			},
		},
		Handler: GetClusterInfo,
	})
}

// GetClusterInfo 获取集群基本信息
func GetClusterInfo(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
	mesh := GetMeshArg(ctx, args)
	info := collectClusterInfo(ctx, mesh)
	return JsonResult(info)
}

// collectClusterInfo 收集集群信息
func collectClusterInfo(ctx consolectx.Context, mesh string) map[string]any {
	counterMgr := ctx.CounterManager()
	if counterMgr == nil {
		return map[string]any{
			"mesh":          mesh,
			"appCount":      0,
			"serviceCount":  0,
			"instanceCount": 0,
			"protocols":     map[string]int{},
			"releases":      map[string]int{},
			"discoveries":   map[string]int{},
			"error":         "Counter manager not available",
		}
	}

	return map[string]any{
		"mesh":          mesh,
		"appCount":      counterMgr.CountByMesh(meshresource.ApplicationKind, mesh),
		"serviceCount":  counterMgr.CountByMesh(meshresource.ServiceProviderMetadataKind, mesh),
		"instanceCount": counterMgr.CountByMesh(meshresource.InstanceKind, mesh),
		"protocols":     counterMgr.DistributionByMesh(counter.ProtocolCounter, mesh),
		"releases":      counterMgr.DistributionByMesh(counter.ReleaseCounter, mesh),
		"discoveries":   counterMgr.DistributionByMesh(counter.DiscoveryCounter, mesh),
	}
}

// Ensure MetricsRegistrar implements ToolRegistrar
var _ registry.ToolRegistrar = (*MetricsRegistrar)(nil)
