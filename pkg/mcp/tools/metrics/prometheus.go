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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/apache/dubbo-admin/pkg/core/logger"
)

const (
	queryTimeout      = 15 * time.Second
	maxQueryBytes     = 8 * 1024
	maxQueryRange     = 24 * time.Hour
	minQueryStep      = 15 * time.Second
	maxSeries         = 100
	maxSamples        = 5000
	maxResponseBytes  = int64(4 * 1024 * 1024)
	maxErrorBodyBytes = int64(4096)
)

type prometheusClient struct {
	baseURL *url.URL
	client  *http.Client
}

var prometheusTransport = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	MaxIdleConns:          20,
	MaxIdleConnsPerHost:   10,
	IdleConnTimeout:       90 * time.Second,
	ResponseHeaderTimeout: queryTimeout,
}

type prometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string          `json:"resultType"`
		Result     json.RawMessage `json:"result"`
	} `json:"data"`
	ErrorType string   `json:"errorType,omitempty"`
	Error     string   `json:"error,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

type prometheusSeries struct {
	Metric map[string]string   `json:"metric"`
	Value  []json.RawMessage   `json:"value,omitempty"`
	Values [][]json.RawMessage `json:"values,omitempty"`
}

func newPrometheusClient(baseURL *url.URL) *prometheusClient {
	return &prometheusClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout:   queryTimeout,
			Transport: prometheusTransport,
		},
	}
}

func (c *prometheusClient) instant(ctx context.Context, query, evaluationTime string) (*QueryResponse, error) {
	values := url.Values{"query": []string{query}}
	if evaluationTime != "" {
		if _, err := parsePrometheusTime("time", evaluationTime); err != nil {
			return nil, err
		}
		values.Set("time", evaluationTime)
	}
	return c.query(ctx, "instant", "/api/v1/query", values)
}

func (c *prometheusClient) rangeQuery(ctx context.Context, query, startRaw, endRaw, stepRaw string) (*QueryResponse, error) {
	start, err := parsePrometheusTime("startTime", startRaw)
	if err != nil {
		return nil, err
	}
	end, err := parsePrometheusTime("endTime", endRaw)
	if err != nil {
		return nil, err
	}
	if !end.After(start) {
		return nil, fmt.Errorf("endTime must be after startTime")
	}
	if end.Sub(start) > maxQueryRange {
		return nil, fmt.Errorf("query range must not exceed %s", maxQueryRange)
	}
	step, err := parsePrometheusStep(stepRaw)
	if err != nil {
		return nil, err
	}
	if step < minQueryStep {
		return nil, fmt.Errorf("step must be at least %s", minQueryStep)
	}

	values := url.Values{
		"query": []string{query},
		"start": []string{startRaw},
		"end":   []string{endRaw},
		"step":  []string{stepRaw},
	}
	return c.query(ctx, "range", "/api/v1/query_range", values)
}

func (c *prometheusClient) query(ctx context.Context, queryType, path string, values url.Values) (*QueryResponse, error) {
	query := values.Get("query")
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query is required")
	}
	if len(query) > maxQueryBytes {
		return nil, fmt.Errorf("query must not exceed %d bytes", maxQueryBytes)
	}

	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + path
	startedAt := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		logPrometheusFailure(queryType, query, startedAt, 0, "transport", err)
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		logPrometheusFailure(queryType, query, startedAt, resp.StatusCode, "upstream_status", nil)
		return nil, fmt.Errorf("prometheus query failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := readBoundedBody(resp.Body, maxResponseBytes)
	if err != nil {
		logPrometheusFailure(queryType, query, startedAt, resp.StatusCode, "response_limit", err)
		return nil, fmt.Errorf("failed to read prometheus response: %w", err)
	}
	var upstream prometheusResponse
	if err := json.Unmarshal(body, &upstream); err != nil {
		logPrometheusFailure(queryType, query, startedAt, resp.StatusCode, "decode", err)
		return nil, fmt.Errorf("failed to decode prometheus response: %w", err)
	}
	if upstream.Status != "success" {
		message := upstream.Error
		if message == "" {
			message = "unknown prometheus error"
		}
		errorType := upstream.ErrorType
		if errorType == "" {
			errorType = "unknown"
		}
		logPrometheusFailure(queryType, query, startedAt, resp.StatusCode, errorType, nil)
		return nil, fmt.Errorf("prometheus %s error: %s", errorType, message)
	}
	result, err := normalizePrometheusResult(queryType, &upstream)
	if err != nil {
		logPrometheusFailure(queryType, query, startedAt, resp.StatusCode, "normalize", err)
		return nil, err
	}
	logger.Sugar().Infow("prometheus query completed", "query_type", queryType, "query_hash", queryHash(query), "elapsed_ms", time.Since(startedAt).Milliseconds(), "result_type", result.ResultType, "series", result.ReturnedSeries, "samples", result.ReturnedSamples, "truncated", result.Truncated)
	return result, nil
}

func normalizePrometheusResult(queryType string, upstream *prometheusResponse) (*QueryResponse, error) {
	result := &QueryResponse{
		QueryType:  queryType,
		ResultType: upstream.Data.ResultType,
		Warnings:   upstream.Warnings,
		Series:     []Series{},
	}
	switch upstream.Data.ResultType {
	case "vector", "matrix":
		var rawSeries []prometheusSeries
		if err := json.Unmarshal(upstream.Data.Result, &rawSeries); err != nil {
			return nil, fmt.Errorf("failed to decode prometheus %s result: %w", upstream.Data.ResultType, err)
		}
		for i, raw := range rawSeries {
			if i >= maxSeries || result.ReturnedSamples >= maxSamples {
				result.Truncated = true
				break
			}
			series := Series{Labels: raw.Metric}
			if upstream.Data.ResultType == "vector" {
				sample, err := decodeSample(raw.Value)
				if err != nil {
					return nil, err
				}
				series.Value = &sample
				result.ReturnedSamples++
			} else {
				for _, rawSample := range raw.Values {
					if result.ReturnedSamples >= maxSamples {
						result.Truncated = true
						break
					}
					sample, err := decodeSample(rawSample)
					if err != nil {
						return nil, err
					}
					series.Values = append(series.Values, sample)
					result.ReturnedSamples++
				}
			}
			result.Series = append(result.Series, series)
			result.ReturnedSeries++
		}
	case "scalar", "string":
		var raw []json.RawMessage
		if err := json.Unmarshal(upstream.Data.Result, &raw); err != nil {
			return nil, fmt.Errorf("failed to decode prometheus %s result: %w", upstream.Data.ResultType, err)
		}
		sample, err := decodeSample(raw)
		if err != nil {
			return nil, err
		}
		result.Value = &sample
		result.ReturnedSamples = 1
	default:
		return nil, fmt.Errorf("unsupported prometheus result type %q", upstream.Data.ResultType)
	}
	return result, nil
}

func decodeSample(raw []json.RawMessage) (Sample, error) {
	if len(raw) != 2 {
		return Sample{}, fmt.Errorf("invalid prometheus sample: expected timestamp and value")
	}
	var timestamp float64
	if err := json.Unmarshal(raw[0], &timestamp); err != nil {
		return Sample{}, fmt.Errorf("invalid prometheus sample timestamp: %w", err)
	}
	var value string
	if err := json.Unmarshal(raw[1], &value); err != nil {
		return Sample{}, fmt.Errorf("invalid prometheus sample value: %w", err)
	}
	return Sample{Timestamp: timestamp, Value: value}, nil
}

func parsePrometheusTime(field, value string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed, nil
	}
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds > float64(math.MaxInt64) || seconds < float64(math.MinInt64) {
		return time.Time{}, fmt.Errorf("%s must be RFC3339 or Unix seconds", field)
	}
	whole, fraction := math.Modf(seconds)
	return time.Unix(int64(whole), int64(fraction*float64(time.Second))), nil
}

func parsePrometheusStep(value string) (time.Duration, error) {
	if parsed, err := time.ParseDuration(value); err == nil {
		return parsed, nil
	}
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil || seconds <= 0 {
		return 0, fmt.Errorf("step must be a positive duration or number of seconds")
	}
	return time.Duration(seconds * float64(time.Second)), nil
}

func readBoundedBody(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	return body, nil
}

func queryHash(query string) string {
	digest := sha256.Sum256([]byte(query))
	return hex.EncodeToString(digest[:6])
}

func logPrometheusFailure(queryType, query string, startedAt time.Time, status int, errorType string, err error) {
	fields := []any{"query_type", queryType, "query_hash", queryHash(query), "elapsed_ms", time.Since(startedAt).Milliseconds(), "error_type", errorType}
	if status != 0 {
		fields = append(fields, "upstream_status", status)
	}
	if err != nil {
		fields = append(fields, "error", err)
	}
	logger.Sugar().Warnw("prometheus query failed", fields...)
}

// Limits returns the resource limits enforced by the Prometheus MCP tools.
func Limits() QueryLimits {
	return QueryLimits{
		TimeoutSeconds:   int(queryTimeout / time.Second),
		MaxQueryBytes:    maxQueryBytes,
		MaxRange:         maxQueryRange.String(),
		MinStep:          minQueryStep.String(),
		MaxSeries:        maxSeries,
		MaxSamples:       maxSamples,
		MaxResponseBytes: maxResponseBytes,
	}
}
