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
	"encoding/json"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
)

const (
	DefaultEnabled            = true
	DefaultMaxVersionsPerRule = int64(5)
	DefaultCoalesceWindowMs   = int64(2000)
	DefaultAdminHintTTLSec    = int64(30)
	DefaultRollbackWaitMs     = int64(5000)
)

type Config struct {
	Enabled            bool  `json:"enabled" yaml:"enabled"`
	MaxVersionsPerRule int64 `json:"maxVersionsPerRule" yaml:"maxVersionsPerRule"`
	CoalesceWindowMs   int64 `json:"coalesceWindowMs" yaml:"coalesceWindowMs"`
	AdminHintTTLSec    int64 `json:"adminHintTTLSec" yaml:"adminHintTTLSec"`
	RollbackWaitMs     int64 `json:"rollbackWaitTimeoutMs" yaml:"rollbackWaitTimeoutMs"`
}

func (c *Config) UnmarshalJSON(data []byte) error {
	type config Config
	defaults := Default()
	*c = *defaults
	return json.Unmarshal(data, (*config)(c))
}

func Default() *Config {
	return &Config{
		Enabled:            DefaultEnabled,
		MaxVersionsPerRule: DefaultMaxVersionsPerRule,
		CoalesceWindowMs:   DefaultCoalesceWindowMs,
		AdminHintTTLSec:    DefaultAdminHintTTLSec,
		RollbackWaitMs:     DefaultRollbackWaitMs,
	}
}

func (c *Config) Sanitize() {
	if c.MaxVersionsPerRule <= 0 {
		c.MaxVersionsPerRule = DefaultMaxVersionsPerRule
	}
	if c.CoalesceWindowMs < 0 {
		c.CoalesceWindowMs = DefaultCoalesceWindowMs
	}
	if c.AdminHintTTLSec <= 0 {
		c.AdminHintTTLSec = DefaultAdminHintTTLSec
	}
	if c.RollbackWaitMs < 0 {
		c.RollbackWaitMs = DefaultRollbackWaitMs
	}
}

func (c *Config) PreProcess() error {
	return nil
}

func (c *Config) PostProcess() error {
	return nil
}

func (c *Config) Validate() error {
	if c.MaxVersionsPerRule <= 0 {
		return bizerror.New(bizerror.ConfigError, "versioning.maxVersionsPerRule must be greater than 0")
	}
	if c.CoalesceWindowMs < 0 {
		return bizerror.New(bizerror.ConfigError, "versioning.coalesceWindowMs must be greater than or equal to 0")
	}
	if c.AdminHintTTLSec <= 0 {
		return bizerror.New(bizerror.ConfigError, "versioning.adminHintTTLSec must be greater than 0")
	}
	if c.RollbackWaitMs < 0 {
		return bizerror.New(bizerror.ConfigError, "versioning.rollbackWaitTimeoutMs must be greater than or equal to 0")
	}
	return nil
}
