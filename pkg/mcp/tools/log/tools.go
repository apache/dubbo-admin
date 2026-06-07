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
	"github.com/apache/dubbo-admin/pkg/mcp/common"
)

const defaultLogLimit = 100

func SearchLogs(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	client, err := lokiClientFromContext(ctx)
	if err != nil {
		return common.ErrorResult(err), nil
	}
	resp, err := client.search(requestContext(ctx), buildSearchLogsReq(args))
	if err != nil {
		return common.ErrorResult(err), nil
	}
	return common.JsonResult(resp)
}

func AnalyzeErrorLogs(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	client, err := lokiClientFromContext(ctx)
	if err != nil {
		return common.ErrorResult(err), nil
	}
	req := buildSearchLogsReq(args)
	if req.Keywords == "" {
		req.Keywords = "Error"
	}
	searchResp, err := client.search(requestContext(ctx), req)
	if err != nil {
		return common.ErrorResult(err), nil
	}
	return common.JsonResult(analyzeErrors(searchResp.Logs, searchResp.SourceEngine))
}

func GetLogCapabilities(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	client, err := lokiClientFromContext(ctx)
	if err != nil {
		return common.ErrorResult(err), nil
	}
	resp, err := client.capabilities(requestContext(ctx), buildLogCapabilitiesReq(args))
	if err != nil {
		return common.ErrorResult(err), nil
	}
	return common.JsonResult(resp)
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
	helper := common.NewArgsHelper(args)
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

func buildLogCapabilitiesReq(args map[string]any) *LogCapabilitiesReq {
	helper := common.NewArgsHelper(args)
	return &LogCapabilitiesReq{
		StartTime: helper.GetString("startTime", ""),
		EndTime:   helper.GetString("endTime", ""),
	}
}

func LogSearchProperties() map[string]common.PropertyDef {
	return map[string]common.PropertyDef{
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

func LogCapabilitiesProperties() map[string]common.PropertyDef {
	return map[string]common.PropertyDef{
		"startTime": {
			Type:        "string",
			Description: "开始时间，支持 RFC3339/RFC3339Nano 或 Unix 纳秒",
		},
		"endTime": {
			Type:        "string",
			Description: "结束时间，支持 RFC3339/RFC3339Nano 或 Unix 纳秒",
		},
	}
}
