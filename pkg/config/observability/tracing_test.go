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

package observability

import (
	"testing"

	"github.com/apache/dubbo-admin/pkg/config"
)

func TestTracingConfigValidate(t *testing.T) {
	cfg := &TracingConfig{
		DefaultProvider: "jaeger-main",
		Providers: []TraceProviderConfig{{
			Name: "jaeger-main", Type: TraceProviderJaeger, Endpoint: "http://jaeger:16686",
		}},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid tracing config, got %v", err)
	}

	cfg.Providers[0].Endpoint = "jaeger:16686"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected endpoint without scheme to be rejected")
	}
}

func TestTracingConfigRejectsDuplicateAndMissingDefault(t *testing.T) {
	cfg := &TracingConfig{
		DefaultProvider: "missing",
		Providers: []TraceProviderConfig{
			{Name: "same", Type: TraceProviderJaeger, Endpoint: "http://jaeger:16686"},
			{Name: "same", Type: TraceProviderJaeger, Endpoint: "http://jaeger-2:16686"},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected duplicate provider names to be rejected")
	}

	cfg.Providers = cfg.Providers[:1]
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing default provider to be rejected")
	}
}

func TestObservabilitySanitizeTraceToken(t *testing.T) {
	cfg := &Config{Tracing: &TracingConfig{Providers: []TraceProviderConfig{{BearerToken: "secret"}}}}
	cfg.Sanitize()
	if got := cfg.Tracing.Providers[0].BearerToken; got != config.SanitizedValue {
		t.Fatalf("expected sanitized token, got %q", got)
	}
}

func TestTracingConfigRejectsUnsupportedProvider(t *testing.T) {
	cfg := &TracingConfig{
		DefaultProvider: "tempo-main",
		Providers: []TraceProviderConfig{{
			Name: "tempo-main", Type: TraceProviderType("tempo"), Endpoint: "http://tempo:3200",
		}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected unsupported provider to be rejected")
	}
}
