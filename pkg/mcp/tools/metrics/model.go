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

// Sample is one Prometheus timestamp-value pair. Value remains a string to preserve NaN and infinities.
type Sample struct {
	Timestamp float64 `json:"timestamp"`
	Value     string  `json:"value"`
}

// Series contains labels and either an instant value or range values.
type Series struct {
	Labels map[string]string `json:"labels"`
	Value  *Sample           `json:"value,omitempty"`
	Values []Sample          `json:"values,omitempty"`
}

// QueryResponse is the stable, bounded MCP representation of a Prometheus query result.
type QueryResponse struct {
	QueryType       string   `json:"queryType"`
	ResultType      string   `json:"resultType"`
	Series          []Series `json:"series,omitempty"`
	Value           *Sample  `json:"value,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	Truncated       bool     `json:"truncated"`
	ReturnedSeries  int      `json:"returnedSeries"`
	ReturnedSamples int      `json:"returnedSamples"`
}

// QueryLimits describes server-enforced PromQL resource limits.
type QueryLimits struct {
	TimeoutSeconds   int    `json:"timeoutSeconds"`
	MaxQueryBytes    int    `json:"maxQueryBytes"`
	MaxRange         string `json:"maxRange"`
	MinStep          string `json:"minStep"`
	MaxSeries        int    `json:"maxSeries"`
	MaxSamples       int    `json:"maxSamples"`
	MaxResponseBytes int64  `json:"maxResponseBytes"`
}
