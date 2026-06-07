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

// GetApplicationDetail 获取应用详情
func GetApplicationDetail(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	helper := common.NewArgsHelper(args)
	appName, ok := helper.GetRequiredString("appName")
	if !ok || appName == "" {
		return common.ErrorResult(fmt.Errorf("required parameter 'appName' is missing")), nil
	}

	req := &model.ApplicationDetailReq{
		AppName: appName,
		Mesh:    common.GetMeshArg(ctx, args),
	}

	detail, err := service.GetApplicationDetail(ctx, req)
	if err != nil {
		return common.ErrorResult(err), nil
	}

	return common.JsonResult(detail)
}

// GetApplicationInstances 获取应用的实例列表
func GetApplicationInstances(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	helper := common.NewArgsHelper(args)
	appName, ok := helper.GetRequiredString("appName")
	if !ok || appName == "" {
		return common.ErrorResult(fmt.Errorf("required parameter 'appName' is missing")), nil
	}

	req := &model.ApplicationTabInstanceInfoReq{
		AppName: appName,
		Mesh:    common.GetMeshArg(ctx, args),
		PageReq: common.BuildPageReq(
			helper.GetInt("pageNumber", common.DefaultPageNumber),
			helper.GetInt("pageSize", common.DefaultPageSize),
		),
	}

	result, err := service.GetAppInstanceInfo(ctx, req)
	if err != nil {
		return common.ErrorResult(err), nil
	}

	instances := extractAppInstances(result)
	return common.JsonResult(map[string]any{
		"appName":    appName,
		"instances":  instances,
		"totalCount": int(result.PageInfo.Total),
	})
}

// GetApplicationServices 获取应用的服务列表
func GetApplicationServices(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	helper := common.NewArgsHelper(args)
	appName, ok := helper.GetRequiredString("appName")
	if !ok || appName == "" {
		return common.ErrorResult(fmt.Errorf("required parameter 'appName' is missing")), nil
	}

	req := &model.ApplicationServiceFormReq{
		AppName: appName,
		Side:    helper.GetString("side", string(common.ServiceSideProvider)),
		Mesh:    common.GetMeshArg(ctx, args),
		PageReq: common.BuildPageReq(
			helper.GetInt("pageNumber", common.DefaultPageNumber),
			helper.GetInt("pageSize", common.DefaultPageSize),
		),
	}

	result, err := service.GetAppServiceInfo(ctx, req)
	if err != nil {
		return common.ErrorResult(err), nil
	}

	services, _ := extractServicesFromResult(result)
	return common.JsonResult(map[string]any{
		"appName":    appName,
		"side":       req.Side,
		"services":   services,
		"totalCount": int(result.PageInfo.Total),
	})
}
