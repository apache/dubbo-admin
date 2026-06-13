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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	observabilitycfg "github.com/apache/dubbo-admin/pkg/config/observability"
)

const (
	defaultQueryWindow = time.Hour
	labelCacheTTL      = 5 * time.Minute
)

var fallbackSelectorPriority = []string{
	"namespace",
	"job",
	"app",
	"appName",
	"service_name",
	"serviceName",
	"service",
	"pod",
	"container",
	"instance",
	"instanceName",
	"level",
}

var lokiLabelsCache = struct {
	sync.Mutex
	items map[string]cachedLokiLabels
}{
	items: map[string]cachedLokiLabels{},
}

type cachedLokiLabels struct {
	labels    map[string]struct{}
	expiresAt time.Time
}

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

type lokiLabelsResp struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
	Error  string   `json:"error,omitempty"`
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

	labelNames, _ := c.labelNames(ctx, start, end)
	queries := buildLogQLQueriesWithLabels(req, labelNames)
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

func (c *lokiClient) capabilities(ctx context.Context, req *LogCapabilitiesReq) (*LogCapabilitiesResp, error) {
	start, end, err := resolveTimeRange(req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}
	labelNames, err := c.labelNames(ctx, start, end)
	if err != nil {
		return nil, err
	}

	return &LogCapabilitiesResp{
		AvailableLabels: supportedLabels(labelNames),
		SupportedFilters: []string{
			"mesh",
			"appName",
			"serviceName",
			"instanceName",
			"traceId",
			"keywords",
			"startTime",
			"endTime",
			"limit",
		},
		LabelFilters: map[string][]string{
			"mesh":         matchingLabels([]string{"mesh"}, labelNames),
			"appName":      matchingLabels([]string{"app", "appName"}, labelNames),
			"serviceName":  matchingLabels([]string{"service", "serviceName", "service_name"}, labelNames),
			"instanceName": matchingLabels([]string{"instance", "instanceName", "pod"}, labelNames),
		},
		ContentFilters: []string{"traceId", "keywords"},
		FallbackLabel:  fallbackSelectorLabel(labelNames),
		SourceEngine:   "loki",
	}, nil
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

func (c *lokiClient) labelNames(ctx context.Context, start, end time.Time) (map[string]struct{}, error) {
	cacheKey := c.labelCacheKey()
	now := time.Now()
	lokiLabelsCache.Lock()
	if cached, ok := lokiLabelsCache.items[cacheKey]; ok && now.Before(cached.expiresAt) {
		labels := cloneLabelSet(cached.labels)
		lokiLabelsCache.Unlock()
		return labels, nil
	}
	lokiLabelsCache.Unlock()

	labelsURL, err := c.labelsURL(start, end)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, labelsURL, nil)
	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.InternalError, "failed to create loki labels request")
	}
	if c.config.Tenant != "" {
		httpReq.Header.Set("X-Scope-OrgID", c.config.Tenant)
	}

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.NetWorkError, "failed to query loki labels")
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(httpResp.Body, 4096))
		return nil, bizerror.New(bizerror.NetWorkError,
			fmt.Sprintf("loki labels query failed with status %d: %s", httpResp.StatusCode, strings.TrimSpace(string(body))))
	}

	var labelsResp lokiLabelsResp
	if err := json.NewDecoder(httpResp.Body).Decode(&labelsResp); err != nil {
		return nil, bizerror.Wrap(err, bizerror.JsonError, "failed to decode loki labels response")
	}
	if labelsResp.Status != "success" {
		if labelsResp.Error != "" {
			return nil, bizerror.New(bizerror.NetWorkError, fmt.Sprintf("loki labels query failed: %s", labelsResp.Error))
		}
		return nil, bizerror.New(bizerror.NetWorkError, fmt.Sprintf("loki labels query returned status %q", labelsResp.Status))
	}

	labels := make(map[string]struct{}, len(labelsResp.Data))
	for _, label := range labelsResp.Data {
		labels[label] = struct{}{}
	}

	lokiLabelsCache.Lock()
	lokiLabelsCache.items[cacheKey] = cachedLokiLabels{
		labels:    cloneLabelSet(labels),
		expiresAt: now.Add(labelCacheTTL),
	}
	lokiLabelsCache.Unlock()
	return labels, nil
}

func (c *lokiClient) labelCacheKey() string {
	return c.config.Endpoint + "|" + c.config.Tenant
}

func cloneLabelSet(labels map[string]struct{}) map[string]struct{} {
	if labels == nil {
		return nil
	}
	cloned := make(map[string]struct{}, len(labels))
	for label := range labels {
		cloned[label] = struct{}{}
	}
	return cloned
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

func (c *lokiClient) labelsURL(start, end time.Time) (string, error) {
	baseURL, err := url.Parse(c.config.Endpoint)
	if err != nil {
		return "", bizerror.Wrap(err, bizerror.ConfigError, "invalid loki endpoint")
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/") + "/loki/api/v1/labels"

	query := baseURL.Query()
	query.Set("start", strconv.FormatInt(start.UnixNano(), 10))
	query.Set("end", strconv.FormatInt(end.UnixNano(), 10))
	baseURL.RawQuery = query.Encode()
	return baseURL.String(), nil
}

func buildLogQLQueries(req *SearchLogsReq) []string {
	return buildLogQLQueriesWithLabels(req, nil)
}

func buildLogQLQueriesWithLabels(req *SearchLogsReq, labelNames map[string]struct{}) []string {
	selectors := buildStreamSelectorsWithLabels(req, labelNames)
	queries := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		query := selector
		if req.Keywords != "" {
			query += " |= " + strconv.Quote(req.Keywords)
		}
		if req.TraceID != "" {
			query += " |= " + strconv.Quote(req.TraceID)
		}
		queries = append(queries, query)
	}
	return queries
}

func buildStreamSelectors(req *SearchLogsReq) []string {
	return buildStreamSelectorsWithLabels(req, nil)
}

func buildStreamSelectorsWithLabels(req *SearchLogsReq, labelNames map[string]struct{}) []string {
	labelGroups := make([][]string, 0, 4)
	if req.Mesh != "" {
		labelGroups = append(labelGroups, labelMatchersWithLabels([]string{"mesh"}, req.Mesh, labelNames))
	}
	if req.AppName != "" {
		labelGroups = append(labelGroups, labelMatchersWithLabels([]string{"app", "appName"}, req.AppName, labelNames))
	}
	if req.ServiceName != "" {
		labelGroups = append(labelGroups, labelMatchersWithLabels([]string{"service", "serviceName", "service_name"}, req.ServiceName, labelNames))
	}
	if req.InstanceName != "" {
		labelGroups = append(labelGroups, labelMatchersWithLabels([]string{"instance", "instanceName", "pod"}, req.InstanceName, labelNames))
	}
	if len(labelGroups) == 0 {
		return []string{fmt.Sprintf("{%s=~%s}", fallbackSelectorLabel(labelNames), strconv.Quote(".+"))}
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
	return labelMatchersWithLabels(names, value, nil)
}

func labelMatchersWithLabels(names []string, value string, labelNames map[string]struct{}) []string {
	selected := selectExistingLabels(names, labelNames)
	matchers := make([]string, 0, len(names))
	for _, name := range selected {
		matchers = append(matchers, labelMatcher(name, value))
	}
	return matchers
}

func selectExistingLabels(names []string, labelNames map[string]struct{}) []string {
	if len(labelNames) == 0 {
		return names
	}
	selected := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := labelNames[name]; ok {
			selected = append(selected, name)
		}
	}
	if len(selected) == 0 {
		return names
	}
	return selected
}

func matchingLabels(names []string, labelNames map[string]struct{}) []string {
	selected := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := labelNames[name]; ok {
			selected = append(selected, name)
		}
	}
	return selected
}

func fallbackSelectorLabel(labelNames map[string]struct{}) string {
	if len(labelNames) == 0 {
		return "namespace"
	}
	for _, label := range fallbackSelectorPriority {
		if _, ok := labelNames[label]; ok {
			return label
		}
	}
	return "namespace"
}

func supportedLabels(labelNames map[string]struct{}) []string {
	labels := make([]string, 0, len(labelNames))
	for label := range labelNames {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return labels
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
			raw := value[1]
			message := extractLogField(raw, "msg", "message")
			if message == "" {
				message = raw
			}
			logs = append(logs, LogItem{
				Timestamp:    normalizeLogTimestamp(value[0], extractLogField(raw, "time", "timestamp")),
				AppName:      firstLabel(stream.Stream, "app", "appName"),
				ServiceName:  firstLabel(stream.Stream, "service", "serviceName", "service_name"),
				InstanceName: firstLabel(stream.Stream, "instance", "instanceName", "pod"),
				Severity:     firstNonEmpty(extractLogField(raw, "level", "severity"), firstLabel(stream.Stream, "level", "severity", "detected_level")),
				Message:      message,
				TraceID:      firstNonEmpty(extractLogField(raw, "trace_id", "traceId", "traceid"), firstLabel(stream.Stream, "trace_id", "traceId", "traceid")),
				SpanID:       firstNonEmpty(extractLogField(raw, "span_id", "spanId", "spanid"), firstLabel(stream.Stream, "span_id", "spanId", "spanid")),
				TraceFlags:   firstNonEmpty(extractLogField(raw, "trace_flags", "traceFlags", "traceflags"), firstLabel(stream.Stream, "trace_flags", "traceFlags", "traceflags")),
				Attributes:   extraLabels(stream.Stream),
				Raw:          raw,
			})
		}
	}
	return logs
}

func normalizeLogTimestamp(lokiTimestamp, logTimestamp string) string {
	if logTimestamp != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, logTimestamp); err == nil {
			return parsed.UTC().Format(time.RFC3339Nano)
		}
	}
	return normalizeLokiTimestamp(lokiTimestamp)
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func extractLogField(message string, keys ...string) string {
	if value := extractJSONLogField(message, keys...); value != "" {
		return value
	}
	return extractTextLogField(message, keys...)
}

func extractJSONLogField(message string, keys ...string) string {
	var payload map[string]any
	if err := json.Unmarshal([]byte(message), &payload); err != nil {
		return ""
	}
	for _, key := range keys {
		if value := stringifyLogField(payload[key]); value != "" {
			return value
		}
	}
	return ""
}

func stringifyLogField(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64, bool:
		return fmt.Sprint(v)
	default:
		return ""
	}
}

// cache the compiled regex per key and reuse it.
var textLogFieldPatterns sync.Map // map[string]*regexp.Regexp
func textLogFieldPattern(key string) *regexp.Regexp {
	if re, ok := textLogFieldPatterns.Load(key); ok {
		return re.(*regexp.Regexp)
	}
	compiled := regexp.MustCompile(`(?i)(?:^|[\s{,])"?` + regexp.QuoteMeta(key) + `"?\s*[:=]\s*"?([^"\s,}]+)`)
	actual, _ := textLogFieldPatterns.LoadOrStore(key, compiled)
	return actual.(*regexp.Regexp)
}

func extractTextLogField(message string, keys ...string) string {
	for _, key := range keys {
		pattern := textLogFieldPattern(key)
		if matches := pattern.FindStringSubmatch(message); len(matches) == 2 {
			return matches[1]
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
			"pod", "level", "severity", "detected_level", "trace_id", "traceId", "traceid", "span_id", "spanId",
			"spanid", "trace_flags", "traceFlags", "traceflags":
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
