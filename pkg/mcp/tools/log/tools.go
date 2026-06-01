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

package log

import (
	"context"
	"fmt"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/mcp/registry"
	basetools "github.com/apache/dubbo-admin/pkg/mcp/tools"
	"github.com/apache/dubbo-admin/pkg/mcp/types"
)

const defaultLogLimit = 100

type LogRegistrar struct{}

func (r *LogRegistrar) RegisterTools(reg *registry.Registry) {
	properties := logSearchProperties()
	reg.Register(types.ToolDef{
		Name:        "search_logs",
		Description: "查询 Dubbo 服务日志，支持按应用、服务、实例、TraceID 和关键字过滤",
		InputSchema: types.InputSchema{
			Type:       "object",
			Properties: properties,
		},
		Handler: SearchLogs,
	})

	reg.Register(types.ToolDef{
		Name:        "analyze_error_logs",
		Description: "分析错误日志并按错误模式聚合",
		InputSchema: types.InputSchema{
			Type:       "object",
			Properties: properties,
		},
		Handler: AnalyzeErrorLogs,
	})
}

func SearchLogs(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
	client, err := lokiClientFromContext(ctx)
	if err != nil {
		return basetools.ErrorResult(err), nil
	}
	resp, err := client.search(requestContext(ctx), buildSearchLogsReq(args))
	if err != nil {
		return basetools.ErrorResult(err), nil
	}
	return basetools.JsonResult(resp)
}

func AnalyzeErrorLogs(ctx consolectx.Context, args map[string]any) (*types.ToolResult, error) {
	client, err := lokiClientFromContext(ctx)
	if err != nil {
		return basetools.ErrorResult(err), nil
	}
	req := buildSearchLogsReq(args)
	if req.Keywords == "" {
		req.Keywords = "Error"
	}
	searchResp, err := client.search(requestContext(ctx), req)
	if err != nil {
		return basetools.ErrorResult(err), nil
	}
	return basetools.JsonResult(analyzeErrors(searchResp.Logs, searchResp.SourceEngine))
}

func lokiClientFromContext(ctx consolectx.Context) (*lokiClient, error) {
	if ctx == nil || ctx.Config().Observability == nil || ctx.Config().Observability.Logs == nil {
		return nil, fmt.Errorf("loki log provider is not configured")
	}
	provider, ok := ctx.Config().Observability.Logs.Default()
	if !ok {
		return nil, fmt.Errorf("default loki log provider is not configured")
	}
	return newLokiClient(provider), nil
}

func requestContext(ctx consolectx.Context) context.Context {
	if ctx == nil || ctx.AppContext() == nil {
		return context.Background()
	}
	return ctx.AppContext()
}

func buildSearchLogsReq(args map[string]any) *SearchLogsReq {
	helper := basetools.NewArgsHelper(args)
	return &SearchLogsReq{
		Mesh:         helper.GetString("mesh", ""),
		AppName:      helper.GetString("appName", ""),
		ServiceName:  helper.GetString("serviceName", ""),
		InstanceName: helper.GetString("instanceName", ""),
		TraceID:      helper.GetString("traceId", ""),
		Keywords:     helper.GetString("keywords", ""),
		StartTime:    helper.GetString("startTime", ""),
		EndTime:      helper.GetString("endTime", ""),
		Limit:        helper.GetInt("limit", defaultLogLimit),
	}
}

func logSearchProperties() map[string]types.PropertyDef {
	return map[string]types.PropertyDef{
		"mesh": {
			Type:        "string",
			Description: "Mesh 名称，用于显式按 mesh label 过滤",
		},
		"appName": {
			Type:        "string",
			Description: "应用名称",
		},
		"serviceName": {
			Type:        "string",
			Description: "服务名称",
		},
		"instanceName": {
			Type:        "string",
			Description: "实例、Pod 或主机名称",
		},
		"traceId": {
			Type:        "string",
			Description: "TraceID",
		},
		"keywords": {
			Type:        "string",
			Description: "日志关键字",
		},
		"startTime": {
			Type:        "string",
			Description: "开始时间，支持 RFC3339/RFC3339Nano 或 Unix 纳秒",
		},
		"endTime": {
			Type:        "string",
			Description: "结束时间，支持 RFC3339/RFC3339Nano 或 Unix 纳秒",
		},
		"limit": {
			Type:        "integer",
			Description: "返回日志条数上限",
			Default:     defaultLogLimit,
		},
	}
}

var _ registry.ToolRegistrar = (*LogRegistrar)(nil)
