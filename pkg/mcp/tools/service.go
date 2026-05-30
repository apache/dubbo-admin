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
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/mcp/common"
)

// SearchServices 搜索服务
func SearchServices(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	helper := common.NewArgsHelper(args)
	keywords := helper.GetString("keywords", "")
	mesh := common.GetMeshArg(ctx, args)
	pageSize := helper.GetInt("pageSize", common.DefaultPageSize)
	pageNumber := helper.GetInt("pageNumber", common.DefaultPageNumber)

	req := &model.ServiceSearchReq{
		Keywords: keywords,
		Mesh:     mesh,
		PageReq:  common.BuildPageReq(pageNumber, pageSize),
	}

	result, err := service.SearchServices(ctx, req)
	if err != nil {
		return common.ErrorResult(err), nil
	}

	return buildServiceSearchResult(result, keywords, mesh, pageSize, pageNumber)
}

// GetServiceDetail 获取服务详情
func GetServiceDetail(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	helper := common.NewArgsHelper(args)
	serviceName := helper.GetString("serviceName", "")

	params := serviceDetailParams{
		serviceName: serviceName,
		group:       helper.GetString("group", ""),
		version:     helper.GetString("version", ""),
		mesh:        common.GetMeshArg(ctx, args),
	}

	side := common.ServiceSide(helper.GetString("side", string(common.ServiceSideProvider)))

	return fetchServiceDistribution(ctx, params, side)
}

// serviceDetailParams 服务详情参数
type serviceDetailParams struct {
	serviceName string
	group       string
	version     string
	mesh        string
}

// fetchServiceDistribution 获取服务分布信息
func fetchServiceDistribution(ctx consolectx.Context, params serviceDetailParams, side common.ServiceSide) (*common.ToolResult, error) {
	targetSide := string(common.ServiceSideConsumer)
	if side == common.ServiceSideConsumer {
		targetSide = string(common.ServiceSideProvider)
	}

	req := &model.ServiceTabDistributionReq{
		ServiceName: params.serviceName,
		Group:       params.group,
		Version:     params.version,
		Side:        targetSide,
		Mesh:        params.mesh,
		PageReq:     common.BuildPageReq(1, common.MaxDistributionLimit),
	}

	distribution, err := service.GetServiceTabDistribution(ctx, req)
	if err != nil {
		return common.ErrorResult(err), nil
	}

	apps := extractApplications(distribution)
	return common.JsonResult(map[string]any{
		"serviceName":  params.serviceName,
		"group":        params.group,
		"version":      params.version,
		"side":         string(side),
		"mesh":         params.mesh,
		"distribution": apps,
		"totalApps":    len(apps),
	})
}

// extractApplications 从分页结果中提取应用列表
func extractApplications(result *model.SearchPaginationResult) []any {
	if result == nil || result.List == nil {
		return []any{}
	}

	apps, ok := result.List.([]*model.ApplicationSearchResp)
	if !ok {
		return []any{}
	}

	resultSlice := make([]any, 0, len(apps))
	for _, app := range apps {
		if app != nil {
			resultSlice = append(resultSlice, map[string]any{
				"appName":          app.AppName,
				"instanceCount":    app.InstanceCount,
				"deployClusters":   app.DeployClusters,
				"registryClusters": app.RegistryClusters,
			})
		}
	}
	return resultSlice
}

// buildServiceSearchResult 构建服务搜索结果
func buildServiceSearchResult(result *model.SearchPaginationResult, keywords, mesh string, pageSize, pageNumber int) (*common.ToolResult, error) {
	services, totalCount := extractServices(result)

	return common.JsonResult(map[string]any{
		"keywords":   keywords,
		"mesh":       mesh,
		"pageSize":   pageSize,
		"pageNumber": pageNumber,
		"services":   services,
		"totalCount": totalCount,
	})
}

// extractServices 从分页结果中提取服务列表
func extractServices(result *model.SearchPaginationResult) ([]any, int) {
	if result == nil || result.List == nil {
		return []any{}, 0
	}

	services, ok := result.List.([]*model.ServiceSearchResp)
	if !ok {
		return []any{result.List}, int(result.PageInfo.Total)
	}

	resultSlice := make([]any, 0, len(services))
	for _, svc := range services {
		if svc != nil {
			resultSlice = append(resultSlice, map[string]any{
				"serviceName":     svc.ServiceName,
				"version":         svc.Version,
				"group":           svc.Group,
				"consumerAppName": svc.ConsumerAppName,
			})
		}
	}
	return resultSlice, int(result.PageInfo.Total)
}
