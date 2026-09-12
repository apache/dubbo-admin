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

package auth

import (
	"strings"
	"testing"
)

func validConfig() *Config {
	return &Config{User: "admin", Password: "secret", ExpirationTime: 3600}
}

func TestConfigValidateDefaultsPasswordOnly(t *testing.T) {
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(cfg.Methods) != 1 || cfg.Methods[0] != MethodPassword {
		t.Fatalf("Methods = %v, want [%s]", cfg.Methods, MethodPassword)
	}
	if cfg.SessionSecret != DefaultSessionSecret {
		t.Fatalf("SessionSecret = %q, want legacy default", cfg.SessionSecret)
	}
}

func TestConfigValidatePreservesExplicitlyEmptyMethods(t *testing.T) {
	cfg := validConfig()
	cfg.Methods = []string{}
	cfg.User = ""
	cfg.Password = ""
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if cfg.Methods == nil || len(cfg.Methods) != 0 {
		t.Fatalf("Methods = %#v, want an explicitly empty slice", cfg.Methods)
	}
}

func TestConfigValidateProviders(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		provider ProviderConfig
		wantErr  string
	}{
		{name: "unsafe id", id: "../github", provider: validGitHubProvider("../github"), wantErr: "provider id"},
		{name: "unknown type", id: "github", provider: ProviderConfig{Type: "oauth", ClientID: "id", ClientSecret: "secret", RedirectURL: "https://admin.example/api/v1/auth/providers/github/callback", PostLoginRedirectURL: "https://admin.example/admin/"}, wantErr: "type"},
		{name: "bad redirect", id: "github", provider: ProviderConfig{Type: ProviderTypeGitHub, ClientID: "id", ClientSecret: "secret", RedirectURL: "://bad", PostLoginRedirectURL: "https://admin.example/admin/"}, wantErr: "redirectUrl"},
		{name: "wrong callback", id: "github", provider: ProviderConfig{Type: ProviderTypeGitHub, ClientID: "id", ClientSecret: "secret", RedirectURL: "https://admin.example/wrong", PostLoginRedirectURL: "https://admin.example/admin/"}, wantErr: "callback"},
		{name: "oidc missing issuer", id: "sso", provider: ProviderConfig{Type: ProviderTypeOIDC, ClientID: "id", ClientSecret: "secret", RedirectURL: "https://admin.example/api/v1/auth/providers/sso/callback", PostLoginRedirectURL: "https://admin.example/admin/"}, wantErr: "issuer"},
		{name: "insecure oidc issuer", id: "sso", provider: ProviderConfig{Type: ProviderTypeOIDC, Issuer: "http://sso.example", ClientID: "id", ClientSecret: "secret", RedirectURL: "https://admin.example/api/v1/auth/providers/sso/callback", PostLoginRedirectURL: "https://admin.example/admin/"}, wantErr: "HTTPS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Providers = map[string]ProviderConfig{tt.id: tt.provider}
			err := cfg.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestConfigValidateAllowsLoopbackHTTPForOIDCDevelopment(t *testing.T) {
	cfg := validConfig()
	cfg.Providers = map[string]ProviderConfig{
		"sso": {
			Type: ProviderTypeOIDC, Issuer: "http://127.0.0.1:5556", ClientID: "id", ClientSecret: "secret",
			RedirectURL: "http://localhost:8888/api/v1/auth/providers/sso/callback", PostLoginRedirectURL: "http://localhost:8881/admin/",
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestConfigValidateProviderIconURL(t *testing.T) {
	tests := []struct {
		name    string
		iconURL string
		wantErr bool
	}{
		{name: "project asset", iconURL: "/admin/auth-providers/github.svg"},
		{name: "HTTPS asset", iconURL: "https://cdn.example/github.svg"},
		{name: "loopback development asset", iconURL: "http://localhost:8881/admin/auth-providers/github.svg"},
		{name: "insecure remote asset", iconURL: "http://cdn.example/github.svg", wantErr: true},
		{name: "non-HTTP scheme", iconURL: "javascript:alert(1)", wantErr: true},
		{name: "relative path", iconURL: "github.svg", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			provider := validGitHubProvider("github")
			provider.IconURL = tt.iconURL
			cfg.Providers = map[string]ProviderConfig{"github": provider}
			err := cfg.Validate()
			if tt.wantErr && (err == nil || !strings.Contains(err.Error(), "iconUrl")) {
				t.Fatalf("Validate() error = %v, want iconUrl error", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestConfigValidateProviderScopeDefaults(t *testing.T) {
	cfg := validConfig()
	cfg.Providers = map[string]ProviderConfig{
		"github": validGitHubProvider("github"),
		"sso": {
			Type: ProviderTypeOIDC, Issuer: "https://sso.example", ClientID: "id", ClientSecret: "secret",
			RedirectURL: "https://admin.example/api/v1/auth/providers/sso/callback", PostLoginRedirectURL: "https://admin.example/admin/",
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got := strings.Join(cfg.Providers["github"].Scopes, " "); got != "read:user user:email" {
		t.Fatalf("GitHub scopes = %q", got)
	}
	if got := strings.Join(cfg.Providers["sso"].Scopes, " "); got != "openid profile email" {
		t.Fatalf("OIDC scopes = %q", got)
	}
}

func TestConfigValidateOIDCRequiresOpenIDScope(t *testing.T) {
	cfg := validConfig()
	cfg.Providers = map[string]ProviderConfig{
		"sso": {
			Type: ProviderTypeOIDC, Issuer: "https://sso.example", ClientID: "id", ClientSecret: "secret",
			RedirectURL: "https://admin.example/api/v1/auth/providers/sso/callback", PostLoginRedirectURL: "https://admin.example/admin/", Scopes: []string{"profile"},
		},
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "openid") {
		t.Fatalf("Validate() error = %v, want openid error", err)
	}
}

func validGitHubProvider(id string) ProviderConfig {
	return ProviderConfig{
		Type: ProviderTypeGitHub, ClientID: "id", ClientSecret: "secret",
		RedirectURL: "https://admin.example/api/v1/auth/providers/" + id + "/callback", PostLoginRedirectURL: "https://admin.example/admin/",
	}
}
