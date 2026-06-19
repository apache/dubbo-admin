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

	"github.com/apache/dubbo-admin/pkg/config/versioning"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestAdminConfigVersioningDefaultsWhenMissing(t *testing.T) {
	cfg := DefaultAdminConfig()
	cfg.RuleVersioning = nil

	require.NotPanics(t, func() {
		cfg.Sanitize()
	})
	require.NotNil(t, cfg.RuleVersioning)
	require.True(t, cfg.RuleVersioning.Enabled)
	require.Equal(t, versioning.DefaultMaxVersionsPerRule, cfg.RuleVersioning.MaxVersionsPerRule)

	cfg.RuleVersioning = nil
	require.NoError(t, cfg.PreProcess())
	require.NotNil(t, cfg.RuleVersioning)
	require.True(t, cfg.RuleVersioning.Enabled)

	cfg.RuleVersioning = nil
	require.NoError(t, cfg.PostProcess())
	require.NotNil(t, cfg.RuleVersioning)
	require.True(t, cfg.RuleVersioning.Enabled)
}

func TestAdminConfigExplicitVersioningDisable(t *testing.T) {
	cfg := DefaultAdminConfig()
	require.True(t, cfg.RuleVersioning.Enabled)

	require.NoError(t, yaml.Unmarshal([]byte("ruleVersioning:\n  enabled: false\n"), &cfg))

	require.NotNil(t, cfg.RuleVersioning)
	require.False(t, cfg.RuleVersioning.Enabled)
	require.Equal(t, versioning.DefaultMaxVersionsPerRule, cfg.RuleVersioning.MaxVersionsPerRule)
}

func TestAdminConfigExplicitVersioningEnable(t *testing.T) {
	cfg := DefaultAdminConfig()
	require.True(t, cfg.RuleVersioning.Enabled)

	require.NoError(t, yaml.Unmarshal([]byte("ruleVersioning:\n  enabled: true\n"), &cfg))

	require.NotNil(t, cfg.RuleVersioning)
	require.True(t, cfg.RuleVersioning.Enabled)
	require.Equal(t, versioning.DefaultMaxVersionsPerRule, cfg.RuleVersioning.MaxVersionsPerRule)
}
