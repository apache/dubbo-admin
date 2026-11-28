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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// The logger is a process wide singleton that writes to a file next to the
// working directory. The tests therefore read the lines that their own call
// appended instead of replacing the logger.

// traceLogFile is the default output path of the shared logger.
const traceLogFile = "logs/dubbo-admin.log"

// traceLogOffset returns the current size of the log file so that a test can
// read only the lines that its own call appends.
func traceLogOffset(t *testing.T) int64 {
	t.Helper()
	info, err := os.Stat(traceLogFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatal(err)
	}
	return info.Size()
}

// traceLogLines returns the log lines that were appended after the offset.
func traceLogLines(t *testing.T, offset int64) []string {
	t.Helper()
	data, err := os.ReadFile(traceLogFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	if int64(len(data)) < offset {
		offset = 0
	}
	var lines []string
	for _, line := range strings.Split(string(data[offset:]), "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// traceLogSince waits for the log lines that were appended after the offset.
func traceLogSince(t *testing.T, offset int64) []string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		lines := traceLogLines(t, offset)
		if len(lines) > 0 || time.Now().After(deadline) {
			return lines
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// traceLogErrorFields returns the error field of every appended log line.
func traceLogErrorFields(t *testing.T, lines []string) []string {
	t.Helper()
	var values []string
	for _, line := range lines {
		fields := map[string]any{}
		if err := json.Unmarshal([]byte(line), &fields); err != nil {
			continue
		}
		if value, ok := fields["error"]; ok {
			values = append(values, fmt.Sprint(value))
		}
	}
	return values
}

func TestSanitizeTransportError(t *testing.T) {
	if sanitizeTransportError(nil) != nil {
		t.Fatal("expected no error for a nil input")
	}
	inner := errors.New("dial tcp 127.0.0.1:1: connect: connection refused")
	urlErr := &url.Error{Op: "Get", URL: "http://127.0.0.1:1/api/traces?service=secret-service", Err: inner}
	sanitized := sanitizeTransportError(urlErr)
	if sanitized != inner {
		t.Fatalf("expected the inner error, got %v", sanitized)
	}
	if strings.Contains(sanitized.Error(), "secret-service") {
		t.Fatalf("expected the request url to be removed, got %q", sanitized.Error())
	}
	plain := errors.New("boom")
	if sanitizeTransportError(plain) != plain {
		t.Fatal("expected a plain error to be kept as it is")
	}
	// A wrapped transport error must be unwrapped as well.
	wrapped := fmt.Errorf("jaeger trace query failed: %w", urlErr)
	if sanitizeTransportError(wrapped) != inner {
		t.Fatalf("expected the inner error of a wrapped failure, got %v", sanitizeTransportError(wrapped))
	}
}

func TestJaegerProviderSearchFailureDoesNotLeakQueryPayload(t *testing.T) {
	provider := newSearchProvider(t, "http://127.0.0.1:1")
	offset := traceLogOffset(t)
	if _, err := provider.SearchTraces(context.Background(), TraceSearchQuery{
		Service:   "secret-service",
		Operation: "secret-operation",
		Tags:      map[string]string{"secret.key": "secret-value"},
	}); err == nil {
		t.Fatal("expected a transport failure")
	}
	lines := traceLogSince(t, offset)
	if len(lines) == 0 {
		t.Fatal("expected the failure to be logged")
	}
	logged := strings.Join(lines, "\n")
	// The query payload must never reach the log. The service name itself is
	// logged on purpose because it is the only anchor of the failed lookup, but
	// it must not appear as part of the request URL.
	for _, forbidden := range []string{
		"http://127.0.0.1:1/api/traces",
		"service=secret-service",
		"operation=secret-operation",
		"tags=",
		"secret-operation",
		"secret-value",
		"%7B",
	} {
		if strings.Contains(logged, forbidden) {
			t.Fatalf("expected the log to omit %q, got %s", forbidden, logged)
		}
	}
	// The recorded error must not repeat the request URL either.
	recordedErrors := traceLogErrorFields(t, lines)
	if len(recordedErrors) == 0 {
		t.Fatal("expected the failure to record the error")
	}
	for _, value := range recordedErrors {
		for _, forbidden := range []string{"/api/traces", "secret-service", "secret-operation", "secret-value"} {
			if strings.Contains(value, forbidden) {
				t.Fatalf("expected the error to omit %q, got %q", forbidden, value)
			}
		}
	}
}

func TestJaegerProviderTraceFailureDoesNotLeakTraceID(t *testing.T) {
	provider := newSearchProvider(t, "http://127.0.0.1:1")
	offset := traceLogOffset(t)
	if _, err := provider.GetTraceByID(context.Background(), testTraceID); err == nil {
		t.Fatal("expected a transport failure")
	}
	lines := traceLogSince(t, offset)
	if len(lines) == 0 {
		t.Fatal("expected the failure to be logged")
	}
	logged := strings.Join(lines, "\n")
	// The trace id must be masked, the raw identifier must not be logged.
	if strings.Contains(logged, testTraceID) {
		t.Fatalf("expected the trace id to be masked, got %s", logged)
	}
	if !strings.Contains(logged, maskedTraceID(testTraceID)) {
		t.Fatalf("expected the masked trace id in the log, got %s", logged)
	}
	for _, value := range traceLogErrorFields(t, lines) {
		for _, forbidden := range []string{testTraceID, "/api/traces"} {
			if strings.Contains(value, forbidden) {
				t.Fatalf("expected the error to omit %q, got %q", forbidden, value)
			}
		}
	}
}

// The tool result reaches the model context, so it keeps the transport detail of
// the upstream failure: the model has to be able to answer which lookup failed
// and why. The request URL must not appear in it, neither in the tool result nor
// in the failure log: it repeats the configured endpoint and carries the
// retrieval conditions, and an endpoint may be configured with a credential.
func TestJaegerProviderToolResultKeepsTransportDetailButNotTheURL(t *testing.T) {
	provider := newSearchProvider(t, "http://127.0.0.1:1")
	offset := traceLogOffset(t)
	_, err := provider.GetTraceByID(context.Background(), testTraceID)
	if err == nil {
		t.Fatal("expected a transport failure")
	}
	// The trace identifier may reappear in the tool result: the model built it
	// itself and it has to be able to report the failed lookup back to the user.
	if !strings.Contains(err.Error(), testTraceID) {
		t.Fatalf("expected the tool result to keep the trace identifier: %s", err.Error())
	}
	// The request URL must not: it repeats the endpoint path and the query
	// string. The "dial tcp host:port" form of the inner error stays, it names
	// the host that refused the connection without repeating the payload.
	if strings.Contains(err.Error(), "/api/traces/") {
		t.Fatalf("expected the tool result to omit the request URL: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("expected the tool result to keep the diagnostic detail: %s", err.Error())
	}

	// The log must not contain the request URL and must not repeat the raw trace
	// identifier.
	lines := traceLogSince(t, offset)
	if len(lines) == 0 {
		t.Fatal("expected the failure to be logged")
	}
	logged := strings.Join(lines, "\n")
	if strings.Contains(logged, testTraceID) {
		t.Fatalf("expected the log to mask the trace identifier, got %s", logged)
	}
	if strings.Contains(logged, "/api/traces") {
		t.Fatalf("expected the log to omit the request URL, got %s", logged)
	}
	for _, value := range traceLogErrorFields(t, lines) {
		for _, forbidden := range []string{testTraceID, "/api/traces"} {
			if strings.Contains(value, forbidden) {
				t.Fatalf("expected the error to omit %q, got %q", forbidden, value)
			}
		}
	}
}
