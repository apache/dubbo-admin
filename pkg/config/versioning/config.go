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
	"github.com/apache/dubbo-admin/pkg/config"
)

const (
	// DefaultMaxVersionsPerRule is the retention window used when configuration
	// omits maxVersionsPerRule or provides a negative value.
	DefaultMaxVersionsPerRule = int64(20)
)

// Config controls RuleVersion audit-history retention. Versioning is always on
// for supported governance rule mutations; a zero MaxVersionsPerRule disables
// cleanup only, not history recording.
type Config struct {
	config.BaseConfig
	MaxVersionsPerRule int64 `json:"maxVersionsPerRule" yaml:"maxVersionsPerRule"`
}

func (c *Config) UnmarshalJSON(data []byte) error {
	type config Config
	defaults := Default()
	*c = *defaults
	return json.Unmarshal(data, (*config)(c))
}

// Default returns rule-history configuration with bounded retention enabled.
func Default() *Config {
	return &Config{
		MaxVersionsPerRule: DefaultMaxVersionsPerRule,
	}
}

// Sanitize normalizes invalid retention values to the default window.
func (c *Config) Sanitize() {
	if c.MaxVersionsPerRule < 0 {
		c.MaxVersionsPerRule = DefaultMaxVersionsPerRule
	}
}

// Validate rejects negative retention values before startup.
func (c *Config) Validate() error {
	if c.MaxVersionsPerRule < 0 {
		return bizerror.New(bizerror.ConfigError, "ruleVersioning.maxVersionsPerRule must be greater than or equal to 0")
	}
	return nil
}
