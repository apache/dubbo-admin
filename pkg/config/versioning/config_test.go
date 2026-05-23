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

package versioning

import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestConfigDefaultsRollbackWaitOnYAMLUnmarshal(t *testing.T) {
	var cfg Config
	require.NoError(t, yaml.Unmarshal([]byte("enabled: false\n"), &cfg))

	require.False(t, cfg.Enabled)
	require.Equal(t, DefaultMaxVersionsPerRule, cfg.MaxVersionsPerRule)
	require.Equal(t, DefaultCoalesceWindowMs, cfg.CoalesceWindowMs)
	require.Equal(t, DefaultAdminHintTTLSec, cfg.AdminHintTTLSec)
	require.Equal(t, DefaultRollbackWaitMs, cfg.RollbackWaitMs)
}

func TestConfigValidateRollbackWait(t *testing.T) {
	cfg := Default()
	cfg.RollbackWaitMs = 0
	require.NoError(t, cfg.Validate())

	cfg.RollbackWaitMs = -1
	require.ErrorContains(t, cfg.Validate(), "versioning.rollbackWaitTimeoutMs")
}

func TestConfigSanitizePreservesExplicitZeroRollbackWait(t *testing.T) {
	cfg := Default()
	cfg.RollbackWaitMs = 0
	cfg.Sanitize()

	require.Equal(t, int64(0), cfg.RollbackWaitMs)
}
