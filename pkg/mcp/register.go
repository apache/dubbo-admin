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

package mcp

import (
	"github.com/apache/dubbo-admin/pkg/mcp/common"
	"github.com/apache/dubbo-admin/pkg/mcp/tools"
)

// RegisterTools 注册所有 MCP 工具
func RegisterTools(server *Server) {
	server.RegisterTool(&common.ToolDef{
		Name:        "get_cluster_info",
		Description: "获取集群基本信息，包括应用数量、服务数量、实例数量、协议分布、版本分布和发现方式分布",
		InputSchema: common.InputSchema{
			Type: "object",
			Properties: map[string]common.PropertyDef{
				"mesh": {
					Type:        "string",
					Description: "网格名称，默认使用第一个 discovery 配置的 id",
				},
			},
		},
		Handler: tools.GetClusterInfo,
	})

	server.RegisterTool(&common.ToolDef{
		Name:        "global_search",
		Description: "全局搜索：根据关键字搜索服务名、应用名、实例名或IP地址",
		InputSchema: common.InputSchema{
			Type: "object",
			Properties: map[string]common.PropertyDef{
				"keyword": {
					Type:        "string",
					Description: "搜索关键字",
				},
				"searchType": {
					Type:        "string",
					Description: "搜索类型: serviceName, appName, instanceName, ip",
					Enum:        []string{"serviceName", "appName", "instanceName", "ip"},
				},
				"mesh": {
					Type:        "string",
					Description: "网格名称，默认使用第一个 discovery 配置的 id",
				},
				"pageNumber": {
					Type:        "integer",
					Description: "页码，默认为1",
				},
				"pageSize": {
					Type:        "integer",
					Description: "每页大小，默认为10",
				},
			},
			Required: []string{"keyword"},
		},
		Handler: tools.GlobalSearch,
	})

	server.RegisterTool(&common.ToolDef{
		Name:        "search_services",
		Description: "搜索服务，支持关键字搜索和分页",
		InputSchema: common.InputSchema{
			Type: "object",
			Properties: map[string]common.PropertyDef{
				"keywords": {
					Type:        "string",
					Description: "搜索关键字",
				},
				"mesh": {
					Type:        "string",
					Description: "网格名称，默认使用第一个 discovery 配置的 id",
				},
				"pageNumber": {
					Type:        "integer",
					Description: "页码，默认为1",
				},
				"pageSize": {
					Type:        "integer",
					Description: "每页大小，默认为10",
				},
			},
		},
		Handler: tools.SearchServices,
	})

	server.RegisterTool(&common.ToolDef{
		Name:        "get_service_detail",
		Description: "获取服务详情，包括服务的提供者或消费者应用列表",
		InputSchema: common.InputSchema{
			Type: "object",
			Properties: map[string]common.PropertyDef{
				"serviceName": {
					Type:        "string",
					Description: "服务名称",
				},
				"group": {
					Type:        "string",
					Description: "服务分组",
				},
				"version": {
					Type:        "string",
					Description: "服务版本",
				},
				"side": {
					Type:        "string",
					Description: "服务端类型: provider, consumer",
					Enum:        []string{"provider", "consumer"},
				},
				"mesh": {
					Type:        "string",
					Description: "网格名称，默认使用第一个 discovery 配置的 id",
				},
			},
		},
		Handler: tools.GetServiceDetail,
	})

	server.RegisterTool(&common.ToolDef{
		Name:        "get_service_instances",
		Description: "获取服务的实例列表",
		InputSchema: common.InputSchema{
			Type: "object",
			Properties: map[string]common.PropertyDef{
				"serviceName": {
					Type:        "string",
					Description: "服务名称",
				},
				"mesh": {
					Type:        "string",
					Description: "网格名称，默认使用第一个 discovery 配置的 id",
				},
				"pageNumber": {
					Type:        "integer",
					Description: "页码，默认为1",
				},
				"pageSize": {
					Type:        "integer",
					Description: "每页大小，默认为10",
				},
			},
			Required: []string{"serviceName"},
		},
		Handler: tools.GetServiceInstances,
	})

	server.RegisterTool(&common.ToolDef{
		Name:        "search_instances",
		Description: "搜索实例，支持按应用名或关键字搜索",
		InputSchema: common.InputSchema{
			Type: "object",
			Properties: map[string]common.PropertyDef{
				"appName": {
					Type:        "string",
					Description: "应用名称",
				},
				"keywords": {
					Type:        "string",
					Description: "搜索关键字",
				},
				"mesh": {
					Type:        "string",
					Description: "网格名称，默认使用第一个 discovery 配置的 id",
				},
				"pageNumber": {
					Type:        "integer",
					Description: "页码，默认为1",
				},
				"pageSize": {
					Type:        "integer",
					Description: "每页大小，默认为10",
				},
			},
		},
		Handler: tools.SearchInstances,
	})

	server.RegisterTool(&common.ToolDef{
		Name:        "get_instance_detail",
		Description: "获取实例详情",
		InputSchema: common.InputSchema{
			Type: "object",
			Properties: map[string]common.PropertyDef{
				"instanceName": {
					Type:        "string",
					Description: "实例名称",
				},
				"mesh": {
					Type:        "string",
					Description: "网格名称，默认使用第一个 discovery 配置的 id",
				},
			},
			Required: []string{"instanceName"},
		},
		Handler: tools.GetInstanceDetail,
	})

	server.RegisterTool(&common.ToolDef{
		Name:        "get_instance_metrics",
		Description: "获取实例指标",
		InputSchema: common.InputSchema{
			Type: "object",
			Properties: map[string]common.PropertyDef{
				"instanceName": {
					Type:        "string",
					Description: "实例名称",
				},
				"mesh": {
					Type:        "string",
					Description: "网格名称，默认使用第一个 discovery 配置的 id",
				},
			},
			Required: []string{"instanceName"},
		},
		Handler: tools.GetInstanceMetrics,
	})

	server.RegisterTool(&common.ToolDef{
		Name:        "get_application_detail",
		Description: "获取应用详情",
		InputSchema: common.InputSchema{
			Type: "object",
			Properties: map[string]common.PropertyDef{
				"appName": {
					Type:        "string",
					Description: "应用名称",
				},
				"mesh": {
					Type:        "string",
					Description: "网格名称，默认使用第一个 discovery 配置的 id",
				},
			},
			Required: []string{"appName"},
		},
		Handler: tools.GetApplicationDetail,
	})

	server.RegisterTool(&common.ToolDef{
		Name:        "get_application_instances",
		Description: "获取应用的实例列表",
		InputSchema: common.InputSchema{
			Type: "object",
			Properties: map[string]common.PropertyDef{
				"appName": {
					Type:        "string",
					Description: "应用名称",
				},
				"mesh": {
					Type:        "string",
					Description: "网格名称，默认使用第一个 discovery 配置的 id",
				},
				"pageNumber": {
					Type:        "integer",
					Description: "页码，默认为1",
				},
				"pageSize": {
					Type:        "integer",
					Description: "每页大小，默认为10",
				},
			},
			Required: []string{"appName"},
		},
		Handler: tools.GetApplicationInstances,
	})

	server.RegisterTool(&common.ToolDef{
		Name:        "get_application_services",
		Description: "获取应用的服务列表（提供者或消费者）",
		InputSchema: common.InputSchema{
			Type: "object",
			Properties: map[string]common.PropertyDef{
				"appName": {
					Type:        "string",
					Description: "应用名称",
				},
				"side": {
					Type:        "string",
					Description: "服务端类型: provider, consumer",
					Enum:        []string{"provider", "consumer"},
				},
				"mesh": {
					Type:        "string",
					Description: "网格名称，默认使用第一个 discovery 配置的 id",
				},
				"pageNumber": {
					Type:        "integer",
					Description: "页码，默认为1",
				},
				"pageSize": {
					Type:        "integer",
					Description: "每页大小，默认为10",
				},
			},
			Required: []string{"appName"},
		},
		Handler: tools.GetApplicationServices,
	})
}
