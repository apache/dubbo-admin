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
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetTraceStartTime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/traces/0123456789abcdef0123456789abcdef", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"traceID":"0123456789abcdef0123456789abcdef","spans":[{"startTime":2000000},{"startTime":1500000}]}]}`))
	}))
	defer server.Close()
	base, err := url.Parse(server.URL)
	require.NoError(t, err)
	result, err := GetTraceStartTime(context.Background(), base, "0123456789abcdef0123456789abcdef")
	require.NoError(t, err)
	require.Equal(t, int64(1500), result.StartTime)
}

func TestGetTraceStartTimeRejectsInvalidAndMissingConfig(t *testing.T) {
	_, err := GetTraceStartTime(context.Background(), nil, "bad")
	require.ErrorContains(t, err, "16 or 32 hexadecimal")
	_, err = GetTraceStartTime(context.Background(), nil, "0123456789abcdef")
	require.ErrorContains(t, err, "not configured")
}

func TestGetTraceStartTimeRejectsIDMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"traceID":"fedcba9876543210","spans":[{"startTime":1000}]}]}`))
	}))
	defer server.Close()
	base, err := url.Parse(server.URL)
	require.NoError(t, err)
	_, err = GetTraceStartTime(context.Background(), base, "0123456789abcdef")
	require.ErrorContains(t, err, "different trace id")
}
