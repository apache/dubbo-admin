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

package trace

import (
	"fmt"
	"strings"
	"time"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/mcp/common"
)

const (
	// defaultSearchLimit is the number of traces requested from the provider when the
	// caller does not ask for a specific amount.
	defaultSearchLimit = 20
	// maxSearchLimit bounds the limit that is forwarded to the provider.
	maxSearchLimit = 100
	// maxSearchTraceSummaries bounds how many per-trace summaries are kept locally.
	// The provider may return more traces than the agent can consume.
	maxSearchTraceSummaries = 100
)

// SearchTraces discovers candidate traces for a service so the agent can start an
// investigation without asking the user for a trace identifier.
func SearchTraces(ctx consolectx.Context, args map[string]any) (*common.ToolResult, error) {
	query, err := buildTraceSearchQuery(args)
	if err != nil {
		return common.ErrorResult(err), nil
	}
	providerConfig, err := providerConfigFromContext(ctx, common.NewArgsHelper(args).GetString("provider", ""))
	if err != nil {
		return common.ErrorResult(err), nil
	}
	provider, err := newProvider(providerConfig)
	if err != nil {
		return common.ErrorResult(err), nil
	}
	result, err := provider.SearchTraces(traceRequestContext(ctx), query)
	if err != nil {
		return common.ErrorResult(err), nil
	}
	return common.JsonResult(result)
}

// SearchProperties returns the MCP input schema properties for trace search.
func SearchProperties() map[string]common.PropertyDef {
	return map[string]common.PropertyDef{
		"service":       {Type: "string", Description: "Service name used as the trace search anchor"},
		"operation":     {Type: "string", Description: "Optional root operation name filter"},
		"traceId":       {Type: "string", Description: "Optional Jaeger 64-bit or 128-bit hexadecimal TraceID anchor"},
		"tags":          {Type: "string", Description: "Optional tag filter formatted as key=value pairs separated by commas"},
		"startTime":     {Type: "string", Description: "Optional RFC3339 lower bound of the search window"},
		"endTime":       {Type: "string", Description: "Optional RFC3339 upper bound of the search window"},
		"minDurationMs": {Type: "integer", Description: "Optional minimum trace duration in milliseconds"},
		"limit":         {Type: "integer", Description: "Maximum number of traces to request", Default: defaultSearchLimit},
		"provider":      {Type: "string", Description: "Optional trace provider name; defaults to defaultProvider"},
	}
}

// buildTraceSearchQuery validates the tool arguments and converts them into the
// provider side query.
func buildTraceSearchQuery(args map[string]any) (TraceSearchQuery, error) {
	helper := common.NewArgsHelper(args)
	service := strings.TrimSpace(helper.GetString("service", ""))
	if service == "" {
		return TraceSearchQuery{}, fmt.Errorf("service is required for trace search")
	}
	query := TraceSearchQuery{Service: service}
	query.Operation = strings.TrimSpace(helper.GetString("operation", ""))

	if traceID := strings.TrimSpace(helper.GetString("traceId", "")); traceID != "" {
		if !traceIDPattern.MatchString(traceID) {
			return TraceSearchQuery{}, fmt.Errorf("traceId must be a 16 or 32 character hexadecimal value")
		}
		query.TraceID = traceID
	}

	tags, err := parseSearchTags(helper.GetString("tags", ""))
	if err != nil {
		return TraceSearchQuery{}, err
	}
	query.Tags = tags

	startTime, err := parseSearchTime(helper.GetString("startTime", ""), "startTime")
	if err != nil {
		return TraceSearchQuery{}, err
	}
	query.StartTime = startTime

	endTime, err := parseSearchTime(helper.GetString("endTime", ""), "endTime")
	if err != nil {
		return TraceSearchQuery{}, err
	}
	query.EndTime = endTime
	if !query.StartTime.IsZero() && !query.EndTime.IsZero() && query.EndTime.Before(query.StartTime) {
		return TraceSearchQuery{}, fmt.Errorf("endTime must not be earlier than startTime")
	}

	minDurationMs := helper.GetInt("minDurationMs", 0)
	if minDurationMs < 0 {
		return TraceSearchQuery{}, fmt.Errorf("minDurationMs must not be negative")
	}
	query.MinDuration = time.Duration(minDurationMs) * time.Millisecond

	limit := helper.GetInt("limit", defaultSearchLimit)
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}
	query.Limit = limit
	return query, nil
}

// parseSearchTags converts the "k=v,k2=v2" filter syntax into a tag map.
// A repeated key keeps the last value, which matches the object semantics of the
// provider side tag filter.
func parseSearchTags(raw string) (map[string]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	result := map[string]string{}
	for _, pair := range strings.Split(trimmed, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		key, value, found := strings.Cut(pair, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			return nil, fmt.Errorf("tags must be formatted as key=value pairs separated by commas")
		}
		result[key] = strings.TrimSpace(value)
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

// parseSearchTime converts an RFC3339 timestamp into a time value. An empty input
// yields the zero time, which the provider treats as "unbounded".
func parseSearchTime(raw string, name string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be an RFC3339 timestamp: %w", name, err)
	}
	return parsed, nil
}
