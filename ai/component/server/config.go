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

import "dubbo-admin-ai/config"

// ServerSpec defines server configuration
type ServerSpec struct {
	Port           int                 `yaml:"port"`
	Host           string              `yaml:"host"`
	Debug          bool                `yaml:"debug"`
	CORSOrigins    []string            `yaml:"cors_origins"`
	ReadTimeout    int                 `yaml:"read_timeout"`
	WriteTimeout   int                 `yaml:"write_timeout"`
	Authentication AuthenticationSpec  `yaml:"authentication"`
	Features       config.FeatureFlags `yaml:"features"`
}

// AuthenticationSpec configures the trusted identity used by the AI HTTP
// service. Phase 1 supports a server-configured identity; the interface allows
// JWT or signed-gateway adapters to be added without changing handlers.
type AuthenticationSpec struct {
	Mode                    string   `yaml:"mode"`
	TenantID                string   `yaml:"tenant_id"`
	UserID                  string   `yaml:"user_id"`
	Roles                   []string `yaml:"roles"`
	Scopes                  []string `yaml:"scopes"`
	AllowInsecureStaticAuth bool     `yaml:"allow_insecure_static_auth"`
	AdoptLegacySessions     bool     `yaml:"adopt_legacy_sessions"`
	JWTSecret               string   `yaml:"jwt_secret"`
	JWTIssuer               string   `yaml:"jwt_issuer"`
	JWTAudience             string   `yaml:"jwt_audience"`
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
		Authentication: AuthenticationSpec{
			Mode:     "static",
			TenantID: "default",
			UserID:   "local-user",
			Roles:    []string{"user"},
		},
	}
}

func (a *AuthenticationSpec) applyDefaults() {
	defaults := DefaultServerSpec().Authentication
	if a.Mode == "" {
		a.Mode = defaults.Mode
	}
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
