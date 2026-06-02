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

type SearchLogsReq struct {
	Mesh         string `json:"mesh,omitempty"`
	AppName      string `json:"appName,omitempty"`
	ServiceName  string `json:"serviceName,omitempty"`
	InstanceName string `json:"instanceName,omitempty"`
	TraceID      string `json:"traceId,omitempty"`
	Keywords     string `json:"keywords,omitempty"`
	StartTime    string `json:"startTime,omitempty"`
	EndTime      string `json:"endTime,omitempty"`
	Limit        int    `json:"limit,omitempty"`
}

type LogItem struct {
	Timestamp    string            `json:"timestamp"`
	AppName      string            `json:"appName,omitempty"`
	ServiceName  string            `json:"serviceName,omitempty"`
	InstanceName string            `json:"instanceName,omitempty"`
	Severity     string            `json:"severity,omitempty"`
	Message      string            `json:"message"`
	TraceID      string            `json:"traceId,omitempty"`
	SpanID       string            `json:"spanId,omitempty"`
	TraceFlags   string            `json:"traceFlags,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	Raw          string            `json:"raw,omitempty"`
}

type SearchLogsResp struct {
	Logs         []LogItem `json:"logs"`
	SourceEngine string    `json:"sourceEngine"`
}

type AnalyzeErrorLogsReq struct {
	SearchLogsReq
}

type LogCapabilitiesReq struct {
	StartTime string `json:"startTime,omitempty"`
	EndTime   string `json:"endTime,omitempty"`
}

type LogCapabilitiesResp struct {
	AvailableLabels  []string            `json:"availableLabels"`
	SupportedFilters []string            `json:"supportedFilters"`
	LabelFilters     map[string][]string `json:"labelFilters"`
	ContentFilters   []string            `json:"contentFilters"`
	FallbackLabel    string              `json:"fallbackLabel"`
	SourceEngine     string              `json:"sourceEngine"`
}

type ErrorPattern struct {
	Pattern   string    `json:"pattern"`
	Count     int       `json:"count"`
	Example   string    `json:"example,omitempty"`
	FirstSeen string    `json:"firstSeen,omitempty"`
	LastSeen  string    `json:"lastSeen,omitempty"`
	Examples  []LogItem `json:"examples,omitempty"`
}

type AnalyzeErrorLogsResp struct {
	Summary      string         `json:"summary"`
	TotalErrors  int            `json:"totalErrors"`
	Patterns     []ErrorPattern `json:"patterns"`
	SourceEngine string         `json:"sourceEngine"`
}
