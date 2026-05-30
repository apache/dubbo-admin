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
	"github.com/apache/dubbo-admin/pkg/mcp/common"
)

// GetServiceInstances 获取服务的实例列表
func GetServiceInstances(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	helper := common.NewArgsHelper(args)
	serviceName, ok := helper.GetRequiredString("serviceName")
	if !ok || serviceName == "" {
		return common.ErrorResult(fmt.Errorf("required parameter 'serviceName' is missing")), nil
	}

	mesh := common.GetMeshArg(ctx, args)
	pageSize := helper.GetInt("pageSize", common.DefaultPageSize)
	pageNumber := helper.GetInt("pageNumber", common.DefaultPageNumber)

	req := &model.SearchInstanceReq{
		Keywords: serviceName,
		Mesh:     mesh,
		PageReq:  common.BuildPageReq(pageNumber, pageSize),
	}

	result, err := service.SearchInstances(ctx, req)
	if err != nil {
		return common.ErrorResult(err), nil
	}

	instances, totalCount := extractSearchInstances(result)
	return common.JsonResult(map[string]any{
		"serviceName": serviceName,
		"mesh":        mesh,
		"instances":   instances,
		"totalCount":  totalCount,
	})
}

// GetInstanceDetail 获取实例详情
func GetInstanceDetail(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	helper := common.NewArgsHelper(args)
	instanceName, ok := helper.GetRequiredString("instanceName")
	if !ok || instanceName == "" {
		return common.ErrorResult(fmt.Errorf("required parameter 'instanceName' is missing")), nil
	}

	req := &model.InstanceDetailReq{
		InstanceName: instanceName,
		Mesh:         common.GetMeshArg(ctx, args),
	}

	detail, err := service.GetInstanceDetail(ctx, req)
	if err != nil {
		return common.ErrorResult(err), nil
	}

	return common.JsonResult(detail)
}

// GetInstanceMetrics 获取实例指标
func GetInstanceMetrics(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	helper := common.NewArgsHelper(args)
	instanceName, ok := helper.GetRequiredString("instanceName")
	if !ok || instanceName == "" {
		return common.ErrorResult(fmt.Errorf("required parameter 'instanceName' is missing")), nil
	}

	req := &model.MetricsReq{
		InstanceName: instanceName,
		Mesh:         common.GetMeshArg(ctx, args),
	}

	metrics, err := service.GetInstanceMetrics(ctx, req)
	if err != nil {
		return common.ErrorResult(err), nil
	}

	if len(metrics) == 0 {
		return common.JsonResult(map[string]any{
			"instanceName": instanceName,
			"metrics":      []any{},
			"message":      "No metrics available",
		})
	}

	return common.JsonResult(metrics[0])
}

// SearchInstances 搜索实例
func SearchInstances(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	helper := common.NewArgsHelper(args)
	appName := helper.GetString("appName", "")
	keywords := helper.GetString("keywords", "")

	if appName == "" && keywords == "" {
		return common.ErrorResult(fmt.Errorf("at least one of 'appName' or 'keywords' is required")), nil
	}

	req := &model.SearchInstanceReq{
		AppName:  appName,
		Keywords: keywords,
		Mesh:     common.GetMeshArg(ctx, args),
		PageReq: common.BuildPageReq(
			helper.GetInt("pageNumber", common.DefaultPageNumber),
			helper.GetInt("pageSize", common.DefaultPageSize),
		),
	}

	result, err := service.SearchInstances(ctx, req)
	if err != nil {
		return common.ErrorResult(err), nil
	}

	instances, totalCount := extractSearchInstances(result)
	return common.JsonResult(map[string]any{
		"appName":    appName,
		"keywords":   keywords,
		"instances":  instances,
		"totalCount": totalCount,
	})
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

// extractServicesFromResult 从结果中提取服务列表
func extractServicesFromResult(result *model.SearchPaginationResult) ([]any, int) {
	if result == nil || result.List == nil {
		return []any{}, 0
	}

	services, ok := result.List.([]*model.ServiceSearchResp)
	if !ok {
		return []any{}, 0
	}

	resultSlice := make([]any, 0, len(services))
	for _, svc := range services {
		if svc != nil {
			resultSlice = append(resultSlice, map[string]any{
				"serviceName":     svc.ServiceName,
				"group":           svc.Group,
				"version":         svc.Version,
				"consumerAppName": svc.ConsumerAppName,
			})
		}
	}
	return resultSlice, int(result.PageInfo.Total)
}
