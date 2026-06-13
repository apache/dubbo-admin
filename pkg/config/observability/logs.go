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
	"fmt"
	"net/url"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
)

type LogProviderType string

const (
	LogProviderLoki LogProviderType = "loki"
)

type LogsConfig struct {
	DefaultProvider string              `json:"defaultProvider" yaml:"defaultProvider"`
	Providers       []LogProviderConfig `json:"providers" yaml:"providers"`
}

type LogProviderConfig struct {
	Name     string          `json:"name" yaml:"name"`
	Type     LogProviderType `json:"type" yaml:"type"`
	Endpoint string          `json:"endpoint" yaml:"endpoint"`
	Tenant   string          `json:"tenant,omitempty" yaml:"tenant,omitempty"`
}

func (c *LogsConfig) Validate() error {
	if c == nil || len(c.Providers) == 0 {
		return nil
	}
	if strutil.IsBlank(c.DefaultProvider) {
		return bizerror.New(bizerror.ConfigError, "default log provider is required")
	}

	foundDefault := false
	for _, provider := range c.Providers {
		if strutil.IsBlank(provider.Name) {
			return bizerror.New(bizerror.ConfigError, "log provider name is required")
		}
		if provider.Name == c.DefaultProvider {
			foundDefault = true
		}
		if provider.Type != LogProviderLoki {
			return bizerror.New(bizerror.ConfigError, fmt.Sprintf("unsupported log provider type: %s", provider.Type))
		}
		if strutil.IsBlank(provider.Endpoint) {
			return bizerror.New(bizerror.ConfigError, "log provider endpoint is required")
		}
		parsed, err := url.Parse(provider.Endpoint)
		if err != nil {
			return bizerror.Wrap(err, bizerror.ConfigError, fmt.Sprintf("invalid log provider endpoint: %s", provider.Endpoint))
		}
		if parsed.Scheme == "" || parsed.Host == "" {
			return bizerror.New(bizerror.ConfigError, fmt.Sprintf("invalid log provider endpoint: %s", provider.Endpoint))
		}
	}
	if !foundDefault {
		return bizerror.New(bizerror.ConfigError, fmt.Sprintf("default log provider %q is not configured", c.DefaultProvider))
	}
	return nil
}

func (c *LogsConfig) Default() (LogProviderConfig, bool) {
	if c == nil {
		return LogProviderConfig{}, false
	}
	for _, provider := range c.Providers {
		if provider.Name == c.DefaultProvider {
			return provider, true
		}
	}
	return LogProviderConfig{}, false
}
