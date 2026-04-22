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
	"fmt"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/mcp/types"
	"github.com/apache/dubbo-admin/pkg/mcp/registry"
)

// DetailRegistrar 详情查询工具注册器
type DetailRegistrar struct{}

// RegisterTools 实现 ToolRegistrar 接口
func (r *DetailRegistrar) RegisterTools(reg *registry.Registry) {
	// 获取服务的实例列表
	reg.Register(types.ToolDef{
		Name:        "get_service_instances",
		Description: "获取指定服务的所有实例列表",
		InputSchema: types.InputSchema{
			Type:     "object",
			Required: []string{"serviceName"},
			Properties: map[string]types.PropertyDef{
				"serviceName": {
					Type:        "string",
					Description: "服务名称（完整服务名）",
				},
				"mesh": {
					Type:        "string",
					Description: "Mesh 名称",
				},
				"pageSize": {
					Type:        "integer",
					Description: "每页数量",
					Default:     DefaultPageSize,
				},
				"pageNumber": {
					Type:        "integer",
					Description: "页码",
					Default:     DefaultPageNumber,
				},
			},
		},
		Handler: GetServiceInstances,
	})

	// 获取实例详情
	reg.Register(types.ToolDef{
		Name:        "get_instance_detail",
		Description: "获取实例的详细信息",
		InputSchema: types.InputSchema{
			Type:     "object",
			Required: []string{"instanceName"},
			Properties: map[string]types.PropertyDef{
				"instanceName": {
					Type:        "string",
					Description: "实例名称",
				},
				"mesh": {
					Type:        "string",
					Description: "Mesh 名称",
				},
			},
		},
		Handler: GetInstanceDetail,
	})

	// 获取实例指标
	reg.Register(types.ToolDef{
		Name:        "get_instance_metrics",
		Description: "获取实例的监控指标（qps、rt、成功率等）",
		InputSchema: types.InputSchema{
			Type:     "object",
			Required: []string{"instanceName"},
			Properties: map[string]types.PropertyDef{
				"instanceName": {
					Type:        "string",
					Description: "实例名称",
				},
				"mesh": {
					Type:        "string",
					Description: "Mesh 名称",
				},
			},
		},
		Handler: GetInstanceMetrics,
	})

	// 获取应用详情
	reg.Register(types.ToolDef{
		Name:        "get_application_detail",
		Description: "获取应用的详细信息（端口、版本、协议等）",
		InputSchema: types.InputSchema{
			Type:     "object",
			Required: []string{"appName"},
			Properties: map[string]types.PropertyDef{
				"appName": {
					Type:        "string",
					Description: "应用名称",
				},
				"mesh": {
					Type:        "string",
					Description: "Mesh 名称",
				},
			},
		},
		Handler: GetApplicationDetail,
	})

	// 获取应用的实例列表
	reg.Register(types.ToolDef{
		Name:        "get_application_instances",
		Description: "获取应用的所有实例",
		InputSchema: types.InputSchema{
			Type:     "object",
			Required: []string{"appName"},
			Properties: map[string]types.PropertyDef{
				"appName": {
					Type:        "string",
					Description: "应用名称",
				},
				"mesh": {
					Type:        "string",
					Description: "Mesh 名称",
				},
				"pageSize": {
					Type:        "integer",
					Description: "每页数量",
					Default:     DefaultPageSize,
				},
				"pageNumber": {
					Type:        "integer",
					Description: "页码",
					Default:     DefaultPageNumber,
				},
			},
		},
		Handler: GetApplicationInstances,
	})

	// 获取应用的服务列表
	reg.Register(types.ToolDef{
		Name:        "get_application_services",
		Description: "获取应用提供的所有服务",
		InputSchema: types.InputSchema{
			Type:     "object",
			Required: []string{"appName"},
			Properties: map[string]types.PropertyDef{
				"appName": {
					Type:        "string",
					Description: "应用名称",
				},
				"side": {
					Type:        "string",
					Description: "服务端类型: provider/consumer",
					Default:     string(ServiceSideProvider),
					Enum:        []string{string(ServiceSideProvider), string(ServiceSideConsumer)},
				},
				"mesh": {
					Type:        "string",
					Description: "Mesh 名称",
				},
				"pageSize": {
					Type:        "integer",
					Description: "每页数量",
					Default:     DefaultPageSize,
				},
				"pageNumber": {
					Type:        "integer",
					Description: "页码",
					Default:     DefaultPageNumber,
				},
			},
		},
		Handler: GetApplicationServices,
	})

	// 搜索实例
	reg.Register(types.ToolDef{
		Name:        "search_instances",
		Description: "按应用名或关键字搜索实例",
		InputSchema: types.InputSchema{
			Type: "object",
			Properties: map[string]types.PropertyDef{
				"appName": {
					Type:        "string",
					Description: "按应用名搜索",
				},
				"keywords": {
					Type:        "string",
					Description: "按关键字搜索实例名或IP",
				},
				"mesh": {
					Type:        "string",
					Description: "Mesh 名称",
				},
				"pageSize": {
					Type:        "integer",
					Description: "每页数量",
					Default:     DefaultPageSize,
				},
				"pageNumber": {
					Type:        "integer",
					Description: "页码",
					Default:     DefaultPageNumber,
				},
			},
		},
		Handler: SearchInstances,
	})
}

// GetServiceInstances 获取服务的实例列表
func GetServiceInstances(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
	helper := NewArgsHelper(args)
	serviceName, ok := helper.GetRequiredString("serviceName")
	if !ok || serviceName == "" {
		return ErrorResult(fmt.Errorf("required parameter 'serviceName' is missing")), nil
	}

	mesh := GetMeshArg(ctx, args)
	pageSize := helper.GetInt("pageSize", DefaultPageSize)
	pageNumber := helper.GetInt("pageNumber", DefaultPageNumber)

	// 使用 SearchInstances 按服务名搜索实例
	req := &model.SearchInstanceReq{
		Keywords: serviceName, // 用服务名作为关键字搜索
		Mesh:     mesh,
		PageReq:  BuildPageReq(pageNumber, pageSize),
	}

	result, err := service.SearchInstances(ctx, req)
	if err != nil {
		return ErrorResult(err), nil
	}

	instances, totalCount := extractSearchInstances(result)
	return JsonResult(map[string]any{
		"serviceName": serviceName,
		"mesh":        mesh,
		"instances":   instances,
		"totalCount":  totalCount,
	})
}

// GetInstanceDetail 获取实例详情
func GetInstanceDetail(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
	helper := NewArgsHelper(args)
	instanceName, ok := helper.GetRequiredString("instanceName")
	if !ok || instanceName == "" {
		return ErrorResult(fmt.Errorf("required parameter 'instanceName' is missing")), nil
	}

	req := &model.InstanceDetailReq{
		InstanceName: instanceName,
		Mesh:         GetMeshArg(ctx, args),
	}

	detail, err := service.GetInstanceDetail(ctx, req)
	if err != nil {
		return ErrorResult(err), nil
	}

	return JsonResult(detail)
}

// GetInstanceMetrics 获取实例指标
func GetInstanceMetrics(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
	helper := NewArgsHelper(args)
	instanceName, ok := helper.GetRequiredString("instanceName")
	if !ok || instanceName == "" {
		return ErrorResult(fmt.Errorf("required parameter 'instanceName' is missing")), nil
	}

	req := &model.MetricsReq{
		InstanceName: instanceName,
		Mesh:         GetMeshArg(ctx, args),
	}

	metrics, err := service.GetInstanceMetrics(ctx, req)
	if err != nil {
		return ErrorResult(err), nil
	}

	if len(metrics) == 0 {
		return JsonResult(map[string]any{
			"instanceName": instanceName,
			"metrics":      []any{},
			"message":      "No metrics available",
		})
	}

	return JsonResult(metrics[0])
}

// GetApplicationDetail 获取应用详情
func GetApplicationDetail(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
	helper := NewArgsHelper(args)
	appName, ok := helper.GetRequiredString("appName")
	if !ok || appName == "" {
		return ErrorResult(fmt.Errorf("required parameter 'appName' is missing")), nil
	}

	req := &model.ApplicationDetailReq{
		AppName: appName,
		Mesh:    GetMeshArg(ctx, args),
	}

	detail, err := service.GetApplicationDetail(ctx, req)
	if err != nil {
		return ErrorResult(err), nil
	}

	return JsonResult(detail)
}

// GetApplicationInstances 获取应用的实例列表
func GetApplicationInstances(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
	helper := NewArgsHelper(args)
	appName, ok := helper.GetRequiredString("appName")
	if !ok || appName == "" {
		return ErrorResult(fmt.Errorf("required parameter 'appName' is missing")), nil
	}

	req := &model.ApplicationTabInstanceInfoReq{
		AppName: appName,
		Mesh:    GetMeshArg(ctx, args),
		PageReq: BuildPageReq(
			helper.GetInt("pageNumber", DefaultPageNumber),
			helper.GetInt("pageSize", DefaultPageSize),
		),
	}

	result, err := service.GetAppInstanceInfo(ctx, req)
	if err != nil {
		return ErrorResult(err), nil
	}

	instances := extractAppInstances(result)
	return JsonResult(map[string]any{
		"appName":     appName,
		"instances":   instances,
		"totalCount":  int(result.PageInfo.Total),
	})
}

// GetApplicationServices 获取应用的服务列表
func GetApplicationServices(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
	helper := NewArgsHelper(args)
	appName, ok := helper.GetRequiredString("appName")
	if !ok || appName == "" {
		return ErrorResult(fmt.Errorf("required parameter 'appName' is missing")), nil
	}

	req := &model.ApplicationServiceFormReq{
		AppName: appName,
		Side:    helper.GetString("side", string(ServiceSideProvider)),
		Mesh:    GetMeshArg(ctx, args),
		PageReq: BuildPageReq(
			helper.GetInt("pageNumber", DefaultPageNumber),
			helper.GetInt("pageSize", DefaultPageSize),
		),
	}

	result, err := service.GetAppServiceInfo(ctx, req)
	if err != nil {
		return ErrorResult(err), nil
	}

	services, _ := extractServicesFromResult(result)
	return JsonResult(map[string]any{
		"appName":    appName,
		"side":       req.Side,
		"services":   services,
		"totalCount": int(result.PageInfo.Total),
	})
}

// SearchInstances 搜索实例
func SearchInstances(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
	helper := NewArgsHelper(args)
	appName := helper.GetString("appName", "")
	keywords := helper.GetString("keywords", "")

	if appName == "" && keywords == "" {
		return ErrorResult(fmt.Errorf("at least one of 'appName' or 'keywords' is required")), nil
	}

	req := &model.SearchInstanceReq{
		AppName:  appName,
		Keywords: keywords,
		Mesh:     GetMeshArg(ctx, args),
		PageReq: BuildPageReq(
			helper.GetInt("pageNumber", DefaultPageNumber),
			helper.GetInt("pageSize", DefaultPageSize),
		),
	}

	result, err := service.SearchInstances(ctx, req)
	if err != nil {
		return ErrorResult(err), nil
	}

	instances, totalCount := extractSearchInstances(result)
	return JsonResult(map[string]any{
		"appName":    appName,
		"keywords":   keywords,
		"instances":  instances,
		"totalCount": totalCount,
	})
}

// extractAppInstances 从应用实例结果中提取实例列表
func extractAppInstances(result *model.SearchPaginationResult) []any {
	if result == nil || result.List == nil {
		return []any{}
	}

	instances, ok := result.List.([]*model.AppInstanceInfoResp)
	if !ok {
		return []any{}
	}

	resultSlice := make([]any, 0, len(instances))
	for _, inst := range instances {
		if inst != nil {
			resultSlice = append(resultSlice, map[string]any{
				"name":             inst.Name,
				"ip":               inst.IP,
				"appName":          inst.AppName,
				"deployState":      inst.DeployState,
				"registerState":    inst.RegisterState,
				"workloadName":     inst.WorkloadName,
				"createTime":       inst.CreateTime,
				"registerTime":     inst.RegisterTime,
			})
		}
	}
	return resultSlice
}

// extractSearchInstances 从搜索结果中提取实例列表
func extractSearchInstances(result *model.SearchPaginationResult) ([]any, int) {
	if result == nil || result.List == nil {
		return []any{}, 0
	}

	instances, ok := result.List.([]*model.SearchInstanceResp)
	if !ok {
		return []any{}, 0
	}

	resultSlice := make([]any, 0, len(instances))
	for _, inst := range instances {
		if inst != nil {
			resultSlice = append(resultSlice, map[string]any{
				"name":             inst.Name,
				"appName":          inst.AppName,
				"ip":               inst.Ip,
				"workloadName":     inst.WorkloadName,
				"deployState":      inst.DeployState,
				"deployCluster":    inst.DeployCluster,
				"registerState":    inst.RegisterState,
				"registerClusters": inst.RegisterClusters,
				"createTime":       inst.CreateTime,
				"registerTime":     inst.RegisterTime,
			})
		}
	}
	return resultSlice, int(result.PageInfo.Total)
}

// Ensure DetailRegistrar implements ToolRegistrar
var _ registry.ToolRegistrar = (*DetailRegistrar)(nil)
