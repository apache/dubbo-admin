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

package server

import (
	"dubbo-admin-ai/config"
	"dubbo-admin-ai/security"
	"fmt"
	"net/url"
	"strings"
)

// ServerSpec defines server configuration
type ServerSpec struct {
	Port         int                 `yaml:"port"`
	Host         string              `yaml:"host"`
	Debug        bool                `yaml:"debug"`
	CORSOrigins  []string            `yaml:"cors_origins"`
	ReadTimeout  int                 `yaml:"read_timeout"`
	WriteTimeout int                 `yaml:"write_timeout"`
	Auth         AuthenticationSpec  `yaml:"auth"`
	Features     config.FeatureFlags `yaml:"features"`
}

// AuthenticationSpec configures the trusted identity used by the AI HTTP
// service. Enabled selects Admin RS256/JWKS authentication; disabled uses a
// fixed development identity restricted to loopback by default.
type AuthenticationSpec struct {
	Enabled                 bool     `yaml:"enabled"`
	TenantID                string   `yaml:"tenant_id"`
	UserID                  string   `yaml:"user_id"`
	Roles                   []string `yaml:"roles"`
	Scopes                  []string `yaml:"scopes"`
	AllowInsecureStaticAuth bool     `yaml:"allow_insecure_static_auth"`
	AdoptLegacySessions     bool     `yaml:"adopt_legacy_sessions"`
	JWKSURL                 string   `yaml:"jwks_url"`
	Issuer                  string   `yaml:"issuer"`
	Audience                string   `yaml:"audience"`
}

func (a AuthenticationSpec) Validate() error {
	if !a.Enabled {
		return nil
	}
	parsed, err := url.Parse(a.JWKSURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%w: auth.jwks_url must be an HTTP or HTTPS URL", security.ErrAuthMisconfigured)
	}
	for _, field := range []struct{ name, value string }{
		{"issuer", a.Issuer}, {"audience", a.Audience}, {"tenant_id", a.TenantID},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%w: auth.%s is required", security.ErrAuthMisconfigured, field.name)
		}
	}
	return nil
}

// DefaultServerSpec returns default server configuration
func DefaultServerSpec() *ServerSpec {
	return &ServerSpec{
		Port:         8888,
		Host:         "127.0.0.1",
		Debug:        false,
		CORSOrigins:  []string{},
		ReadTimeout:  30,
		WriteTimeout: 30,
		Auth: AuthenticationSpec{
			TenantID: "default",
			UserID:   "local-user",
			Roles:    []string{"user"},
		},
	}
}

func (a *AuthenticationSpec) applyDefaults() {
	defaults := DefaultServerSpec().Auth
	if a.TenantID == "" {
		a.TenantID = defaults.TenantID
	}
	if a.UserID == "" {
		a.UserID = defaults.UserID
	}
	if a.Roles == nil {
		a.Roles = append([]string(nil), defaults.Roles...)
	}
}
