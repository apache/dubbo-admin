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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validConfig() *Config {
	return &Config{
		User:           "admin",
		Password:       "secret",
		ExpirationTime: 3600,
		SessionSecret:  strings.Repeat("s", MinimumSessionSecretLength),
	}
}

func TestConfigValidatePasswordOnly(t *testing.T) {
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(cfg.Methods) != 1 || cfg.Methods[0] != MethodPassword {
		t.Fatalf("Methods = %v, want [%s]", cfg.Methods, MethodPassword)
	}
	if cfg.SessionSecret != strings.Repeat("s", MinimumSessionSecretLength) {
		t.Fatalf("SessionSecret = %q, want configured secret", cfg.SessionSecret)
	}
}

func TestConfigValidateRequiresSessionSecret(t *testing.T) {
	t.Setenv(SessionSecretEnvVar, "")
	cfg := validConfig()
	cfg.SessionSecret = ""
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "sessionSecret") {
		t.Fatalf("Validate() error = %v, want missing sessionSecret error", err)
	}
}

func TestConfigValidateLoadsSessionSecretFromEnvironment(t *testing.T) {
	t.Setenv(SessionSecretEnvVar, strings.Repeat("e", MinimumSessionSecretLength))
	cfg := validConfig()
	cfg.SessionSecret = ""
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if cfg.SessionSecret != strings.Repeat("e", MinimumSessionSecretLength) {
		t.Fatalf("SessionSecret = %q, want environment value", cfg.SessionSecret)
	}
}

func TestConfigValidateRejectsShortSessionSecret(t *testing.T) {
	cfg := validConfig()
	cfg.SessionSecret = "short"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "32 bytes") {
		t.Fatalf("Validate() error = %v, want minimum sessionSecret length error", err)
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

func TestAccessTokenConfigValidation(t *testing.T) {
	t.Run("disabled does not load key", func(t *testing.T) {
		cfg := validConfig()
		cfg.AccessToken = &AccessTokenConfig{Keys: []AccessTokenKey{{PrivateKeyFile: "/does/not/exist"}}}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("enabled requires keys", func(t *testing.T) {
		cfg := validConfig()
		cfg.AccessToken = &AccessTokenConfig{Enabled: true}
		err := cfg.Validate()
		if err == nil || !strings.Contains(err.Error(), "keys") {
			t.Fatalf("Validate() error = %v, want keys error", err)
		}
	})

	t.Run("rejects undersized RSA key", func(t *testing.T) {
		cfg := validConfig()
		cfg.AccessToken = validAccessTokenConfig(writeRSAKey(t, 1024, false))
		err := cfg.Validate()
		if err == nil || !strings.Contains(err.Error(), "2048") {
			t.Fatalf("Validate() error = %v, want 2048-bit error", err)
		}
	})

	for _, pkcs8 := range []bool{false, true} {
		t.Run(map[bool]string{false: "PKCS1", true: "PKCS8"}[pkcs8], func(t *testing.T) {
			cfg := validConfig()
			cfg.AccessToken = validAccessTokenConfig(writeRSAKey(t, 2048, pkcs8))
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if cfg.AccessToken.Keys[0].PrivateKey == nil || cfg.AccessToken.TTL != DefaultAccessTokenTTL {
				t.Fatalf("validated access token = %+v", cfg.AccessToken)
			}
		})
	}
}

func validGitHubProvider(id string) ProviderConfig {
	return ProviderConfig{
		Type: ProviderTypeGitHub, ClientID: "id", ClientSecret: "secret",
		RedirectURL: "https://admin.example/api/v1/auth/providers/" + id + "/callback", PostLoginRedirectURL: "https://admin.example/admin/",
	}
}

func TestAccessTokenMultipleKeys(t *testing.T) {
	oldPath := writeRSAKey(t, 2048, false)
	newPath := writeRSAKey(t, 2048, true)
	for _, tt := range []struct {
		name   string
		change func(*AccessTokenConfig)
		bad    bool
	}{
		{"valid", func(c *AccessTokenConfig) {}, false},
		{"duplicate", func(c *AccessTokenConfig) { c.Keys[1].KeyID = "old" }, true},
		{"empty ID", func(c *AccessTokenConfig) { c.Keys[0].KeyID = " " }, true},
		{"missing active", func(c *AccessTokenConfig) { c.ActiveKeyID = "missing" }, true},
		{"empty keys", func(c *AccessTokenConfig) { c.Keys = []AccessTokenKey{} }, true},
		{"empty active ID", func(c *AccessTokenConfig) { c.ActiveKeyID = "" }, true},
		{"missing file", func(c *AccessTokenConfig) { c.Keys[0].PrivateKeyFile = "/missing" }, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := &AccessTokenConfig{Enabled: true, Issuer: "admin", Audiences: []string{"ai"}, ActiveKeyID: "new", Keys: []AccessTokenKey{{KeyID: "old", PrivateKeyFile: oldPath}, {KeyID: "new", PrivateKeyFile: newPath}}}
			tt.change(c)
			if err := c.validate(); (err != nil) != tt.bad {
				t.Fatalf("validate = %v", err)
			}
			if !tt.bad {
				if c.Keys[0].PrivateKey == nil || c.Keys[1].PrivateKey == nil {
					t.Fatal("keys not loaded")
				}
				if err := c.validate(); err != nil {
					t.Fatalf("repeated validation: %v", err)
				}
			}
		})
	}
}

func validAccessTokenConfig(path string) *AccessTokenConfig {
	return &AccessTokenConfig{
		Enabled: true, Issuer: "dubbo-admin", ActiveKeyID: "key-1", Keys: []AccessTokenKey{{KeyID: "key-1", PrivateKeyFile: path}},
		Audiences: []string{"dubbo-admin-ai"},
	}
}

func writeRSAKey(t *testing.T, bits int, pkcs8 bool) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatal(err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	blockType := "RSA PRIVATE KEY"
	if pkcs8 {
		der, err = x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			t.Fatal(err)
		}
		blockType = "PRIVATE KEY"
	}
	path := filepath.Join(t.TempDir(), "key.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
