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

package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	"github.com/apache/dubbo-admin/pkg/config"
	"github.com/apache/dubbo-admin/pkg/config/versioning"
)

func TestAdminConfigSanitizeRetainsDefaultRuleVersioning(t *testing.T) {
	cfg := DefaultAdminConfig()
	cfg.RuleVersioning = nil

	cfg.Sanitize()

	require.NotNil(t, cfg.RuleVersioning)
	assert.Equal(t, versioning.DefaultMaxVersionsPerRule, cfg.RuleVersioning.MaxVersionsPerRule)
}

func TestAdminConfigRetainsMCPObservabilityAndRuleVersioning(t *testing.T) {
	cfg := DefaultAdminConfig()
	require.NoError(t, yaml.Unmarshal([]byte(`
discovery:
  - id: test-mesh
    name: test-discovery
    type: mock
mcp:
  enabled: true
  path: /api/test-mcp
  apiKey: test-api-key
observability:
  logs:
    defaultProvider: loki-test
    providers:
      - name: loki-test
        type: loki
        endpoint: http://localhost:3100
  tracing:
    defaultProvider: jaeger-test
    providers:
      - name: jaeger-test
        type: jaeger
        endpoint: http://localhost:16686
        bearerToken: test-trace-token
ruleVersioning:
  maxVersionsPerRule: 7
`), &cfg))
	require.NoError(t, cfg.PreProcess())
	require.NoError(t, cfg.PostProcess())
	require.NoError(t, cfg.Validate())
	require.NotNil(t, cfg.MCP)
	assert.True(t, cfg.MCP.Enabled)
	assert.Equal(t, "/api/test-mcp", cfg.MCP.Path)
	assert.Equal(t, "test-api-key", cfg.MCP.APIKey)
	logProvider, found := cfg.Observability.Logs.Default()
	require.True(t, found)
	assert.Equal(t, "http://localhost:3100", logProvider.Endpoint)
	traceProvider, found := cfg.Observability.Tracing.Default()
	require.True(t, found)
	assert.Equal(t, "http://localhost:16686", traceProvider.Endpoint)
	assert.Equal(t, "test-trace-token", traceProvider.BearerToken)
	assert.Equal(t, int64(7), cfg.RuleVersioning.MaxVersionsPerRule)

	cfg.Sanitize()

	traceProvider, found = cfg.Observability.Tracing.Default()
	require.True(t, found)
	assert.Equal(t, config.SanitizedValue, traceProvider.BearerToken)
	assert.Equal(t, int64(7), cfg.RuleVersioning.MaxVersionsPerRule)
}
