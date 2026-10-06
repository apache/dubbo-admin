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

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/console/model"
)

var traceIDPattern = regexp.MustCompile(`^(?:[0-9a-fA-F]{16}|[0-9a-fA-F]{32})$`)

type jaegerTraceResponse struct {
	Data []struct {
		TraceID string `json:"traceID"`
		Spans   []struct {
			StartTime int64 `json:"startTime"`
		} `json:"spans"`
	} `json:"data"`
}

func GetTraceStartTime(ctx context.Context, baseURL *url.URL, traceID string) (*model.TraceStartTimeResp, error) {
	traceID = strings.TrimSpace(traceID)
	if !traceIDPattern.MatchString(traceID) {
		return nil, bizerror.New(bizerror.InvalidArgument, "traceId must be 16 or 32 hexadecimal characters")
	}
	if baseURL == nil {
		return nil, bizerror.New(bizerror.ConfigError, "jaeger query url is not configured")
	}
	endpoint := *baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/api/traces/" + url.PathEscape(traceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.NetWorkError, "create jaeger query request failed")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.NetWorkError, "jaeger query unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, bizerror.New(bizerror.NotFoundError, "trace not found")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, bizerror.New(bizerror.NetWorkError, fmt.Sprintf("jaeger query returned status %d", resp.StatusCode))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.NetWorkError, "read jaeger query response failed")
	}
	var result jaegerTraceResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, bizerror.Wrap(err, bizerror.NetWorkError, "decode jaeger query response failed")
	}
	if len(result.Data) == 0 {
		return nil, bizerror.New(bizerror.NotFoundError, "trace not found")
	}
	if !strings.EqualFold(result.Data[0].TraceID, traceID) {
		return nil, bizerror.New(bizerror.NetWorkError, "jaeger returned a different trace id")
	}
	var earliest int64
	for _, span := range result.Data[0].Spans {
		if span.StartTime <= 0 || (earliest != 0 && span.StartTime >= earliest) {
			continue
		}
		earliest = span.StartTime
	}
	if earliest == 0 {
		return nil, bizerror.New(bizerror.NotFoundError, "trace has no valid start time")
	}
	return &model.TraceStartTimeResp{TraceID: traceID, StartTime: earliest / 1000}, nil
}
