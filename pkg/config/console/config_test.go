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

package console

import (
	"strings"
	"testing"

	"github.com/apache/dubbo-admin/pkg/config/console/auth"
)

func TestReleaseProviderRequiresStrongSessionSecret(t *testing.T) {
	cfg := DefaultConsoleConfig()
	cfg.Auth.Providers = map[string]auth.ProviderConfig{
		"github": {
			Type: auth.ProviderTypeGitHub, ClientID: "id", ClientSecret: "secret",
			RedirectURL: "https://admin.example/api/v1/auth/providers/github/callback", PostLoginRedirectURL: "https://admin.example/admin/",
		},
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "sessionSecret") {
		t.Fatalf("Validate() error = %v, want sessionSecret error", err)
	}

	cfg.Auth.SessionSecret = "a-long-deployment-specific-session-secret"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() with strong secret error = %v", err)
	}
}

func TestReleaseProviderRejectsShortSessionSecret(t *testing.T) {
	cfg := DefaultConsoleConfig()
	cfg.Auth.Providers = map[string]auth.ProviderConfig{
		"github": {
			Type: auth.ProviderTypeGitHub, ClientID: "id", ClientSecret: "secret",
			RedirectURL: "https://admin.example/api/v1/auth/providers/github/callback", PostLoginRedirectURL: "https://admin.example/admin/",
		},
	}
	cfg.Auth.SessionSecret = "x"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "32 bytes") {
		t.Fatalf("Validate() error = %v, want minimum sessionSecret length error", err)
	}
}

func TestDebugProviderAllowsLegacySessionSecret(t *testing.T) {
	cfg := DefaultConsoleConfig()
	cfg.GinMode = DebugMode
	cfg.Auth.Providers = map[string]auth.ProviderConfig{
		"github": {
			Type: auth.ProviderTypeGitHub, ClientID: "id", ClientSecret: "secret",
			RedirectURL: "http://localhost:8888/api/v1/auth/providers/github/callback", PostLoginRedirectURL: "http://localhost:8881/admin/",
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
