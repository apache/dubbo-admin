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
	"github.com/apache/dubbo-admin/pkg/mcp/types"
	"github.com/apache/dubbo-admin/pkg/mcp/registry"
)

// ResourceSearchRegistrar 搜索工具注册器
type ResourceSearchRegistrar struct{}

// searchExecutor 搜索执行器接口
type searchExecutor interface {
	execute(ctx consolectx.Context, keyword, mesh string, pageNumber, pageSize int) (*model.SearchPaginationResult, error)
	buildResult(pagedResult *model.SearchPaginationResult, keyword string, pageSize, pageNumber int) map[string]any
}

// RegisterTools 实现 ToolRegistrar 接口
func (r *ResourceSearchRegistrar) RegisterTools(reg *registry.Registry) {
	reg.Register(types.ToolDef{
		Name:        "global_search",
		Description: "全局搜索，支持搜索服务、实例、应用等资源。不传 keyword 返回所有数据",
		InputSchema: types.InputSchema{
			Type:     "object",
			Required: []string{}, // keyword 改为可选，空值返回所有数据
			Properties: map[string]types.PropertyDef{
				"keyword": {
					Type:        "string",
					Description: "搜索关键字，为空时返回所有数据",
				},
				"searchType": {
					Type:        "string",
					Description: "搜索类型: ip(按IP搜索实例), instanceName(按实例名搜索), appName(按应用名搜索), serviceName(按服务名搜索)",
					Default:     string(SearchTypeName),
					Enum:        []string{string(SearchTypeIP), string(SearchTypeInstanceName), string(SearchTypeAppName), string(SearchTypeName)},
				},
				"mesh": {
					Type:        "string",
					Description: "Mesh 名称，默认使用配置中的默认 mesh",
				},
				"pageSize": {
					Type:        "integer",
					Description: "每页数量",
					Default:     DefaultPageSize,
				},
				"pageNumber": {
					Type:        "integer",
					Description: "页码，从 1 开始",
					Default:     DefaultPageNumber,
				},
			},
		},
		Handler: GlobalSearch,
	})
}

// GlobalSearch 全局搜索
func GlobalSearch(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
	helper := NewArgsHelper(args)
	keyword := helper.GetString("keyword", "")

	searchType := SearchType(helper.GetString("searchType", string(SearchTypeName)))
	mesh := GetMeshArg(ctx, args)
	pageSize := helper.GetInt("pageSize", DefaultPageSize)
	pageNumber := helper.GetInt("pageNumber", DefaultPageNumber)

	executor := getSearchExecutor(searchType)
	result, err := executor.execute(ctx, keyword, mesh, pageNumber, pageSize)
	if err != nil {
		return ErrorResult(err), nil
	}

	searchResult := executor.buildResult(result, keyword, pageSize, pageNumber)
	searchResult["searchType"] = string(searchType)

	return JsonResult(searchResult)
}

// getSearchExecutor 根据搜索类型获取对应的执行器
func getSearchExecutor(searchType SearchType) searchExecutor {
	executors := map[SearchType]searchExecutor{
		SearchTypeIP:           &ipSearchExecutor{},
		SearchTypeInstanceName: &instanceNameSearchExecutor{},
		SearchTypeAppName:      &appNameSearchExecutor{},
		SearchTypeName:         &serviceNameSearchExecutor{},
	}

	if executor, ok := executors[searchType]; ok {
		return executor
	}
	return &serviceNameSearchExecutor{}
}

// ipSearchExecutor IP 搜索执行器
type ipSearchExecutor struct{}

func (e *ipSearchExecutor) execute(ctx consolectx.Context, keyword, mesh string, pageNumber, pageSize int) (*model.SearchPaginationResult, error) {
	req := buildSearchReq(keyword, mesh, pageNumber, pageSize)
	return service.SearchInstanceByIp(ctx, req)
}

func (e *ipSearchExecutor) buildResult(pagedResult *model.SearchPaginationResult, keyword string, pageSize, pageNumber int) map[string]any {
	instances, totalCount := extractInstances(pagedResult)
	return map[string]any{
		"keyword":    keyword,
		"pageSize":   pageSize,
		"pageNumber": pageNumber,
		"instances":  instances,
		"totalCount": totalCount,
	}
}

// instanceNameSearchExecutor 实例名搜索执行器
type instanceNameSearchExecutor struct{}

func (e *instanceNameSearchExecutor) execute(ctx consolectx.Context, keyword, mesh string, pageNumber, pageSize int) (*model.SearchPaginationResult, error) {
	req := buildSearchReq(keyword, mesh, pageNumber, pageSize)
	return service.SearchInstanceByName(ctx, req)
}

func (e *instanceNameSearchExecutor) buildResult(pagedResult *model.SearchPaginationResult, keyword string, pageSize, pageNumber int) map[string]any {
	instances, totalCount := extractInstances(pagedResult)
	return map[string]any{
		"keyword":    keyword,
		"pageSize":   pageSize,
		"pageNumber": pageNumber,
		"instances":  instances,
		"totalCount": totalCount,
	}
}

// appNameSearchExecutor 应用名搜索执行器
type appNameSearchExecutor struct{}

func (e *appNameSearchExecutor) execute(ctx consolectx.Context, keyword, mesh string, pageNumber, pageSize int) (*model.SearchPaginationResult, error) {
	// 使用 SearchApplications 而不是 SearchApplicationsByKeywords
	// SearchApplications 会正确处理空 keyword 的情况（返回所有应用的分页列表）
	req := &model.ApplicationSearchReq{
		Keywords: keyword,
		Mesh:     mesh,
		PageReq:  BuildPageReq(pageNumber, pageSize),
	}
	return service.SearchApplications(ctx, req)
}

func (e *appNameSearchExecutor) buildResult(pagedResult *model.SearchPaginationResult, keyword string, pageSize, pageNumber int) map[string]any {
	apps := extractApplicationsFromResult(pagedResult)
	return map[string]any{
		"keyword":     keyword,
		"pageSize":    pageSize,
		"pageNumber":  pageNumber,
		"applications": apps,
		"totalCount":  len(apps),
	}
}

// serviceNameSearchExecutor 服务名搜索执行器
type serviceNameSearchExecutor struct{}

func (e *serviceNameSearchExecutor) execute(ctx consolectx.Context, keyword, mesh string, pageNumber, pageSize int) (*model.SearchPaginationResult, error) {
	req := &model.ServiceSearchReq{
		ServiceName: "",
		Keywords:    keyword,
		Mesh:        mesh,
		PageReq:     BuildPageReq(pageNumber, pageSize),
	}
	// 空关键字时调用 SearchServices 获取所有服务，否则精确匹配
	if keyword == "" {
		return service.SearchServices(ctx, req)
	}
	return service.SearchServicesByKeywords(ctx, req)
}

func (e *serviceNameSearchExecutor) buildResult(pagedResult *model.SearchPaginationResult, keyword string, pageSize, pageNumber int) map[string]any {
	services, totalCount := extractServicesFromResult(pagedResult)
	return map[string]any{
		"keyword":    keyword,
		"pageSize":   pageSize,
		"pageNumber": pageNumber,
		"services":   services,
		"totalCount": totalCount,
	}
}

// buildSearchReq 构建搜索请求
func buildSearchReq(keyword, mesh string, pageNumber, pageSize int) *model.SearchReq {
	req := model.NewSearchReq()
	req.Keywords = keyword
	req.Mesh = mesh
	req.PageReq = BuildPageReq(pageNumber, pageSize)
	return req
}

// extractInstances 从分页结果中提取实例列表
func extractInstances(pagedResult *model.SearchPaginationResult) ([]any, int) {
	if pagedResult == nil || pagedResult.List == nil {
		return []any{}, 0
	}

	instances, ok := pagedResult.List.([]*model.SearchInstanceResp)
	if !ok {
		return []any{}, 0
	}

	result := make([]any, 0, len(instances))
	for _, ins := range instances {
		result = append(result, map[string]any{
			"name":             ins.Name,
			"appName":          ins.AppName,
			"ip":               ins.Ip,
			"workloadName":     ins.WorkloadName,
			"deployState":      ins.DeployState,
			"deployCluster":    ins.DeployCluster,
			"registerState":    ins.RegisterState,
			"registerClusters": ins.RegisterClusters,
			"createTime":       ins.CreateTime,
			"registerTime":     ins.RegisterTime,
			"labels":           ins.Labels,
		})
	}
	return result, int(pagedResult.PageInfo.Total)
}

// extractApplicationsFromResult 从分页结果中提取应用列表
func extractApplicationsFromResult(pagedResult *model.SearchPaginationResult) []any {
	if pagedResult == nil || pagedResult.List == nil {
		return []any{}
	}

	apps, ok := pagedResult.List.([]*model.ApplicationSearchResp)
	if !ok {
		return []any{}
	}

	result := make([]any, 0, len(apps))
	for _, app := range apps {
		result = append(result, map[string]any{
			"appName":          app.AppName,
			"instanceCount":    app.InstanceCount,
			"deployClusters":   app.DeployClusters,
			"registryClusters": app.RegistryClusters,
		})
	}
	return result
}

// extractServicesFromResult 从分页结果中提取服务列表
func extractServicesFromResult(pagedResult *model.SearchPaginationResult) ([]any, int) {
	if pagedResult == nil || pagedResult.List == nil {
		return []any{}, 0
	}

	services, ok := pagedResult.List.([]*model.ServiceSearchResp)
	if !ok {
		return []any{}, 0
	}

	result := make([]any, 0, len(services))
	for _, svc := range services {
		result = append(result, map[string]any{
			"serviceName":     svc.ServiceName,
			"version":         svc.Version,
			"group":           svc.Group,
			"providerAppName": svc.ProviderAppName,
			"consumerAppName": svc.ConsumerAppName,
		})
	}
	return result, int(pagedResult.PageInfo.Total)
}

// Ensure ResourceSearchRegistrar implements ToolRegistrar
var _ registry.ToolRegistrar = (*ResourceSearchRegistrar)(nil)
