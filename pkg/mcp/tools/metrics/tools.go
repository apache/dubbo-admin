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

package metrics

import (
	"context"
	"fmt"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/mcp/common"
)

// QueryPrometheus executes an instant PromQL query against the configured Prometheus endpoint.
func QueryPrometheus(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	client, requestCtx, err := clientFromContext(ctx)
	if err != nil {
		return common.ErrorResult(err), nil
	}
	helper := common.NewArgsHelper(args)
	resp, err := client.instant(requestCtx, helper.GetString("query", ""), helper.GetString("time", ""))
	if err != nil {
		return common.ErrorResult(err), nil
	}
	return common.JsonResult(resp)
}

// QueryPrometheusRange executes a bounded PromQL range query.
func QueryPrometheusRange(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	client, requestCtx, err := clientFromContext(ctx)
	if err != nil {
		return common.ErrorResult(err), nil
	}
	helper := common.NewArgsHelper(args)
	resp, err := client.rangeQuery(
		requestCtx,
		helper.GetString("query", ""),
		helper.GetString("startTime", ""),
		helper.GetString("endTime", ""),
		helper.GetString("step", ""),
	)
	if err != nil {
		return common.ErrorResult(err), nil
	}
	return common.JsonResult(resp)
}

func clientFromContext(ctx consolectx.Context) (*prometheusClient, context.Context, error) {
	if ctx == nil || ctx.Config().Observability == nil || ctx.Config().Observability.PrometheusBaseURL == nil {
		return nil, nil, fmt.Errorf("prometheus is not configured")
	}
	requestCtx := context.Background()
	if ctx.AppContext() != nil {
		requestCtx = ctx.AppContext()
	}
	return newPrometheusClient(ctx.Config().Observability.PrometheusBaseURL), requestCtx, nil
}

// InstantProperties returns the MCP input schema properties for instant queries.
func InstantProperties() map[string]common.PropertyDef {
	return map[string]common.PropertyDef{
		"query": {Type: "string", Description: "PromQL expression"},
		"time":  {Type: "string", Description: "Optional evaluation time in RFC3339 or Unix seconds"},
	}
}

// RangeProperties returns the MCP input schema properties for range queries.
func RangeProperties() map[string]common.PropertyDef {
	return map[string]common.PropertyDef{
		"query":     {Type: "string", Description: "PromQL expression"},
		"startTime": {Type: "string", Description: "Start time in RFC3339 or Unix seconds"},
		"endTime":   {Type: "string", Description: "End time in RFC3339 or Unix seconds"},
		"step":      {Type: "string", Description: "Query step, for example 30s or 60"},
	}
}
