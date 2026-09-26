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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apache/dubbo-admin/pkg/config"
	configauth "github.com/apache/dubbo-admin/pkg/config/console/auth"
	"github.com/apache/dubbo-admin/pkg/config/versioning"
)

func TestAdminConfigSanitizeRetainsDefaultRuleVersioning(t *testing.T) {
	cfg := DefaultAdminConfig()
	cfg.RuleVersioning = nil

	cfg.Sanitize()

	require.NotNil(t, cfg.RuleVersioning)
	assert.Equal(t, versioning.DefaultMaxVersionsPerRule, cfg.RuleVersioning.MaxVersionsPerRule)
}

func TestAdminConfigValidateRejectsMissingConsole(t *testing.T) {
	t.Setenv(configauth.SessionSecretEnvVar, "")
	cfg := DefaultAdminConfig()

	err := cfg.Validate()
	require.ErrorContains(t, err, "console config is needed")
}

func TestConfigLoadFailsClosedForMissingOrEmptyConsole(t *testing.T) {
	t.Setenv(configauth.SessionSecretEnvVar, "0123456789abcdef0123456789abcdef")
	tests := []struct {
		name      string
		contents  string
		wantError string
	}{
		{name: "missing", contents: "discovery: []\n", wantError: "console config is needed"},
		{name: "null", contents: "console: null\n", wantError: "console config is needed"},
		{name: "empty object", contents: "console: {}\n", wantError: "invalid gin mode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.WriteFile(path, []byte(tt.contents), 0600))

			cfg := DefaultAdminConfig()
			err := config.Load(path, &cfg)
			require.ErrorContains(t, err, tt.wantError)
		})
	}
}
