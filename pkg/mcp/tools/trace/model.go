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

// Trace is a backend-independent, bounded representation of a distributed trace.
type Trace struct {
	TraceID           string   `json:"traceId"`
	StartTime         string   `json:"startTime"`
	DurationMicros    int64    `json:"durationMicros"`
	SpanCount         int      `json:"spanCount"`
	ReturnedSpanCount int      `json:"returnedSpanCount"`
	ServiceCount      int      `json:"serviceCount"`
	ErrorSpanCount    int      `json:"errorSpanCount"`
	Services          []string `json:"services"`
	RootSpanIDs       []string `json:"rootSpanIds"`
	Spans             []Span   `json:"spans"`
	SourceEngine      string   `json:"sourceEngine"`
	Truncated         bool     `json:"truncated"`
}

// Span contains the diagnostic fields needed to reconstruct latency and error propagation.
type Span struct {
	SpanID         string            `json:"spanId"`
	ParentSpanID   string            `json:"parentSpanId,omitempty"`
	ServiceName    string            `json:"serviceName"`
	OperationName  string            `json:"operationName"`
	StartTime      string            `json:"startTime"`
	DurationMicros int64             `json:"durationMicros"`
	Status         string            `json:"status"`
	Error          bool              `json:"error"`
	Attributes     map[string]string `json:"attributes,omitempty"`
	Events         []Event           `json:"events,omitempty"`
	Truncated      bool              `json:"truncated"`
}

// Event represents a bounded Jaeger span log entry.
type Event struct {
	Timestamp  string            `json:"timestamp"`
	Attributes map[string]string `json:"attributes,omitempty"`
}
