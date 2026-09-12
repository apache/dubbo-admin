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

package config_test

import (
	"testing"

	"github.com/apache/dubbo-admin/pkg/config"
	"github.com/apache/dubbo-admin/pkg/config/app"
	configauth "github.com/apache/dubbo-admin/pkg/config/console/auth"
)

func TestConfigForDisplaySanitizesConsoleAuthenticationSecrets(t *testing.T) {
	cfg := app.DefaultAdminConfig()
	cfg.Console.Auth.Password = "password-secret"
	cfg.Console.Auth.SessionSecret = "session-secret"
	cfg.Console.Auth.Providers = map[string]configauth.ProviderConfig{
		"sso": {ClientID: "public-client-id", ClientSecret: "provider-secret"},
	}

	display, err := config.ConfigForDisplay(&cfg)
	if err != nil {
		t.Fatalf("ConfigForDisplay() error = %v", err)
	}
	displayCfg := display.(*app.AdminConfig)
	if displayCfg.Console.Auth.Password != config.SanitizedValue {
		t.Fatalf("display password = %q, want sanitized value", displayCfg.Console.Auth.Password)
	}
	if displayCfg.Console.Auth.SessionSecret != config.SanitizedValue {
		t.Fatalf("display sessionSecret = %q, want sanitized value", displayCfg.Console.Auth.SessionSecret)
	}
	provider := displayCfg.Console.Auth.Providers["sso"]
	if provider.ClientSecret != config.SanitizedValue {
		t.Fatalf("display provider clientSecret = %q, want sanitized value", provider.ClientSecret)
	}
	if provider.ClientID != "public-client-id" {
		t.Fatalf("display provider clientId = %q, want public value preserved", provider.ClientID)
	}
	if cfg.Console.Auth.Password != "password-secret" || cfg.Console.Auth.SessionSecret != "session-secret" || cfg.Console.Auth.Providers["sso"].ClientSecret != "provider-secret" {
		t.Fatal("ConfigForDisplay() mutated the runtime configuration")
	}
}
