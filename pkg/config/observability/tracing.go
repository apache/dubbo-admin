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
	"github.com/apache/dubbo-admin/pkg/config"
)

// TraceProviderType identifies a supported trace query backend.
type TraceProviderType string

const (
	// TraceProviderJaeger queries traces through the Jaeger query API.
	TraceProviderJaeger TraceProviderType = "jaeger"
)

// TracingConfig selects the default trace provider and contains provider definitions.
type TracingConfig struct {
	DefaultProvider string                `json:"defaultProvider" yaml:"defaultProvider"`
	Providers       []TraceProviderConfig `json:"providers" yaml:"providers"`
}

// TraceProviderConfig contains connection and optional authentication settings for one backend.
type TraceProviderConfig struct {
	Name        string            `json:"name" yaml:"name"`
	Type        TraceProviderType `json:"type" yaml:"type"`
	Endpoint    string            `json:"endpoint" yaml:"endpoint"`
	BearerToken string            `json:"bearerToken,omitempty" yaml:"bearerToken,omitempty"`
	Tenant      string            `json:"tenant,omitempty" yaml:"tenant,omitempty"`
}

// Validate verifies provider uniqueness, endpoint safety, and default-provider selection.
func (c *TracingConfig) Validate() error {
	if c == nil || len(c.Providers) == 0 {
		return nil
	}
	if strutil.IsBlank(c.DefaultProvider) {
		return bizerror.New(bizerror.ConfigError, "default trace provider is required")
	}

	seen := make(map[string]struct{}, len(c.Providers))
	foundDefault := false
	for _, provider := range c.Providers {
		if strutil.IsBlank(provider.Name) {
			return bizerror.New(bizerror.ConfigError, "trace provider name is required")
		}
		if _, ok := seen[provider.Name]; ok {
			return bizerror.New(bizerror.ConfigError, fmt.Sprintf("duplicate trace provider name: %s", provider.Name))
		}
		seen[provider.Name] = struct{}{}
		if provider.Name == c.DefaultProvider {
			foundDefault = true
		}
		if provider.Type != TraceProviderJaeger {
			return bizerror.New(bizerror.ConfigError, fmt.Sprintf("unsupported trace provider type: %s", provider.Type))
		}
		if err := validateTraceEndpoint(provider.Endpoint); err != nil {
			return err
		}
	}
	if !foundDefault {
		return bizerror.New(bizerror.ConfigError, fmt.Sprintf("default trace provider %q is not configured", c.DefaultProvider))
	}
	return nil
}

func validateTraceEndpoint(endpoint string) error {
	if strutil.IsBlank(endpoint) {
		return bizerror.New(bizerror.ConfigError, "trace provider endpoint is required")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return bizerror.Wrap(err, bizerror.ConfigError, fmt.Sprintf("invalid trace provider endpoint: %s", endpoint))
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return bizerror.New(bizerror.ConfigError, fmt.Sprintf("invalid trace provider endpoint: %s", endpoint))
	}
	return nil
}

// Default returns the configured default trace provider.
func (c *TracingConfig) Default() (TraceProviderConfig, bool) {
	if c == nil {
		return TraceProviderConfig{}, false
	}
	return c.Get(c.DefaultProvider)
}

// Get returns a named trace provider.
func (c *TracingConfig) Get(name string) (TraceProviderConfig, bool) {
	if c == nil {
		return TraceProviderConfig{}, false
	}
	for _, provider := range c.Providers {
		if provider.Name == name {
			return provider, true
		}
	}
	return TraceProviderConfig{}, false
}

// Sanitize masks provider credentials before configuration is displayed.
func (c *TracingConfig) Sanitize() {
	if c == nil {
		return
	}
	for i := range c.Providers {
		if c.Providers[i].BearerToken != "" {
			c.Providers[i].BearerToken = config.SanitizedValue
		}
	}
}
