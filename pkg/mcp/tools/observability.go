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
	"sort"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/mcp/common"
	metrictools "github.com/apache/dubbo-admin/pkg/mcp/tools/metrics"
)

type observabilityCapabilities struct {
	PrometheusConfigured bool                    `json:"prometheusConfigured"`
	TraceConfigured      bool                    `json:"traceConfigured"`
	TraceProviders       []traceCapability       `json:"traceProviders"`
	Limits               metrictools.QueryLimits `json:"limits"`
}

type traceCapability struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// GetObservabilityCapabilities reports configured providers and enforced PromQL limits without secrets.
func GetObservabilityCapabilities(ctx consolectx.Context, _ map[string]any) (*common.ToolResult, error) {
	resp := observabilityCapabilities{Limits: metrictools.Limits()}
	if ctx == nil || ctx.Config().Observability == nil {
		return common.JsonResult(resp)
	}
	cfg := ctx.Config().Observability
	resp.PrometheusConfigured = cfg.PrometheusBaseURL != nil
	resp.TraceConfigured = cfg.Tracing != nil && len(cfg.Tracing.Providers) > 0
	if cfg.Tracing != nil {
		for _, provider := range cfg.Tracing.Providers {
			resp.TraceProviders = append(resp.TraceProviders, traceCapability{Name: provider.Name, Type: string(provider.Type)})
		}
		sort.Slice(resp.TraceProviders, func(i, j int) bool { return resp.TraceProviders[i].Name < resp.TraceProviders[j].Name })
	}
	return common.JsonResult(resp)
}
