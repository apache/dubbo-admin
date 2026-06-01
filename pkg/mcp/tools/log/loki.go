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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	observabilitycfg "github.com/apache/dubbo-admin/pkg/config/observability"
)

const defaultQueryWindow = time.Hour

type lokiClient struct {
	config observabilitycfg.LogProviderConfig
	client *http.Client
}

type lokiQueryRangeResp struct {
	Status string `json:"status"`
	Data   struct {
		Result []lokiStream `json:"result"`
	} `json:"data"`
	Error string `json:"error,omitempty"`
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

func newLokiClient(cfg observabilitycfg.LogProviderConfig) *lokiClient {
	return &lokiClient{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *lokiClient) search(ctx context.Context, req *SearchLogsReq) (*SearchLogsResp, error) {
	if req.Limit <= 0 {
		req.Limit = defaultLogLimit
	}
	start, end, err := resolveTimeRange(req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}

	queries := buildLogQLQueries(req)
	merged := &SearchLogsResp{SourceEngine: "loki", Logs: make([]LogItem, 0, req.Limit)}
	seen := map[string]struct{}{}
	for _, query := range queries {
		logs, err := c.queryRange(ctx, query, start, end, req.Limit)
		if err != nil {
			return nil, err
		}
		// remove duplicates
		for _, item := range logs {
			key := dedupeKey(item)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged.Logs = append(merged.Logs, item)
			if len(merged.Logs) >= req.Limit {
				break
			}
		}
		if len(merged.Logs) >= req.Limit {
			break
		}
	}

	sort.SliceStable(merged.Logs, func(i, j int) bool {
		return merged.Logs[i].Timestamp > merged.Logs[j].Timestamp
	})
	return merged, nil
}

func (c *lokiClient) queryRange(ctx context.Context, query string, start, end time.Time, limit int) ([]LogItem, error) {
	queryURL, err := c.queryRangeURL(query, start, end, limit)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.InternalError, "failed to create loki query request")
	}
	if c.config.Tenant != "" {
		httpReq.Header.Set("X-Scope-OrgID", c.config.Tenant)
	}

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.NetWorkError, "failed to query loki")
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(httpResp.Body, 4096))
		return nil, bizerror.New(bizerror.NetWorkError,
			fmt.Sprintf("loki query failed with status %d: %s", httpResp.StatusCode, strings.TrimSpace(string(body))))
	}

	var lokiResp lokiQueryRangeResp
	if err := json.NewDecoder(httpResp.Body).Decode(&lokiResp); err != nil {
		return nil, bizerror.Wrap(err, bizerror.JsonError, "failed to decode loki query response")
	}
	if lokiResp.Status != "success" {
		if lokiResp.Error != "" {
			return nil, bizerror.New(bizerror.NetWorkError, fmt.Sprintf("loki query failed: %s", lokiResp.Error))
		}
		return nil, bizerror.New(bizerror.NetWorkError, fmt.Sprintf("loki query returned status %q", lokiResp.Status))
	}
	return normalizeLokiLogs(lokiResp), nil
}

// e.g: endpoint: {endpoint}/loki/api/v1/query_range?query={app="order-service"}&start=1717200000000000000&end=1717203600000000000&limit=100&direction=backward
func (c *lokiClient) queryRangeURL(logQL string, start, end time.Time, limit int) (string, error) {
	baseURL, err := url.Parse(c.config.Endpoint)
	if err != nil {
		return "", bizerror.Wrap(err, bizerror.ConfigError, "invalid loki endpoint")
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/") + "/loki/api/v1/query_range"

	query := baseURL.Query()
	query.Set("query", logQL)
	query.Set("start", strconv.FormatInt(start.UnixNano(), 10))
	query.Set("end", strconv.FormatInt(end.UnixNano(), 10))
	query.Set("limit", strconv.Itoa(limit))
	query.Set("direction", "backward")
	baseURL.RawQuery = query.Encode()
	return baseURL.String(), nil
}

func buildLogQLQueries(req *SearchLogsReq) []string {
	selectors := buildStreamSelectors(req)
	queries := make([]string, 0, len(selectors))
	// add keywords filter
	for _, selector := range selectors {
		query := selector
		if req.Keywords != "" {
			query += " |= " + strconv.Quote(req.Keywords)
		}
		queries = append(queries, query)
	}
	return queries
}

func buildStreamSelectors(req *SearchLogsReq) []string {
	labelGroups := make([][]string, 0, 4)
	if req.Mesh != "" {
		labelGroups = append(labelGroups, []string{labelMatcher("mesh", req.Mesh)})
	}
	if req.AppName != "" {
		labelGroups = append(labelGroups, labelMatchers([]string{"app", "appName"}, req.AppName))
	}
	if req.ServiceName != "" {
		labelGroups = append(labelGroups, labelMatchers([]string{"service", "serviceName", "service_name"}, req.ServiceName))
	}
	if req.InstanceName != "" {
		labelGroups = append(labelGroups, labelMatchers([]string{"instance", "instanceName", "pod"}, req.InstanceName))
	}
	if req.TraceID != "" {
		labelGroups = append(labelGroups, labelMatchers([]string{"trace_id", "traceId", "traceid"}, req.TraceID))
	}
	if len(labelGroups) == 0 {
		return []string{`{job=~".+"}`}
	}

	// Cartesian product
	selectors := []string{""}
	for _, group := range labelGroups {
		next := make([]string, 0, len(selectors)*len(group))
		for _, prefix := range selectors {
			for _, matcher := range group {
				if prefix == "" {
					next = append(next, matcher)
				} else {
					next = append(next, prefix+", "+matcher)
				}
			}
		}
		selectors = next
	}

	result := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		result = append(result, fmt.Sprintf("{%s}", selector))
	}
	return result
}

func labelMatchers(names []string, value string) []string {
	matchers := make([]string, 0, len(names))
	for _, name := range names {
		matchers = append(matchers, labelMatcher(name, value))
	}
	return matchers
}

func labelMatcher(name, value string) string {
	return fmt.Sprintf("%s=%s", name, strconv.Quote(value))
}

func resolveTimeRange(startRaw, endRaw string) (time.Time, time.Time, error) {
	end := time.Now()
	if endRaw != "" {
		parsed, err := parseLogTime("endTime", endRaw)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		end = parsed
	}
	start := end.Add(-defaultQueryWindow)
	if startRaw != "" {
		parsed, err := parseLogTime("startTime", startRaw)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		start = parsed
	}
	if start.After(end) {
		return time.Time{}, time.Time{}, bizerror.New(bizerror.InvalidArgument, "startTime must be less than or equal to endTime")
	}
	return start, end, nil
}

func parseLogTime(field, value string) (time.Time, error) {
	if ts, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return ts, nil
	}
	if ns, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(0, ns), nil
	}
	return time.Time{}, bizerror.New(bizerror.InvalidArgument,
		fmt.Sprintf("%s must be RFC3339, RFC3339Nano, or Unix nanoseconds", field))
}

func normalizeLokiLogs(resp lokiQueryRangeResp) []LogItem {
	logs := make([]LogItem, 0)
	for _, stream := range resp.Data.Result {
		for _, value := range stream.Values {
			if len(value) < 2 {
				continue
			}
			logs = append(logs, LogItem{
				Timestamp:    normalizeLokiTimestamp(value[0]),
				AppName:      firstLabel(stream.Stream, "app", "appName"),
				ServiceName:  firstLabel(stream.Stream, "service", "serviceName", "service_name"),
				InstanceName: firstLabel(stream.Stream, "instance", "instanceName", "pod"),
				Severity:     firstLabel(stream.Stream, "level", "severity", "detected_level"),
				Message:      value[1],
				TraceID:      firstLabel(stream.Stream, "trace_id", "traceId", "traceid"),
				SpanID:       firstLabel(stream.Stream, "span_id", "spanId", "spanid"),
				Attributes:   extraLabels(stream.Stream),
				Raw:          value[1],
			})
		}
	}
	return logs
}

func normalizeLokiTimestamp(value string) string {
	ns, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return value
	}
	return time.Unix(0, ns).UTC().Format(time.RFC3339Nano)
}

// firstLabel returns the first label value that matches any of the keys, or an empty string if none matches
func firstLabel(labels map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := labels[key]; value != "" {
			return value
		}
	}
	return ""
}

// extraLabels returns all labels except the ones used to filter the logs
func extraLabels(labels map[string]string) map[string]string {
	attrs := make(map[string]string)
	for key, value := range labels {
		switch key {
		case "mesh", "app", "appName", "service", "serviceName", "service_name", "instance", "instanceName",
			"pod", "level", "severity", "detected_level", "trace_id", "traceId", "traceid", "span_id", "spanId", "spanid":
			continue
		default:
			attrs[key] = value
		}
	}
	if len(attrs) == 0 {
		return nil
	}
	return attrs
}

func dedupeKey(item LogItem) string {
	return strings.Join([]string{item.Timestamp, item.Message, item.TraceID, item.SpanID}, "\x00")
}
