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
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/apache/dubbo-admin/pkg/config"
)

const (
	DefaultExpirationTime = 7200
	DefaultSessionSecret  = "secret"

	MethodPassword     = "password"
	ProviderTypeGitHub = "github"
	ProviderTypeOIDC   = "oidc"
)

var providerIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

type ProviderConfig struct {
	Type                 string   `json:"type" yaml:"type"`
	DisplayName          string   `json:"displayName" yaml:"displayName"`
	IconURL              string   `json:"iconUrl,omitempty" yaml:"iconUrl,omitempty"`
	Issuer               string   `json:"issuer,omitempty" yaml:"issuer,omitempty"`
	ClientID             string   `json:"clientId" yaml:"clientId"`
	ClientSecret         string   `json:"clientSecret" yaml:"clientSecret"`
	RedirectURL          string   `json:"redirectUrl" yaml:"redirectUrl"`
	PostLoginRedirectURL string   `json:"postLoginRedirectUrl" yaml:"postLoginRedirectUrl"`
	Scopes               []string `json:"scopes,omitempty" yaml:"scopes,omitempty"`
}

// Config AuthConfig configure the valid user and password
type Config struct {
	config.BaseConfig
	Methods             []string                  `json:"methods" yaml:"methods"`
	User                string                    `json:"user" yaml:"user"`
	Password            string                    `json:"password" yaml:"password"`
	ExpirationTime      int                       `json:"expirationTime" yaml:"expirationTime"`
	SessionSecret       string                    `json:"sessionSecret" yaml:"sessionSecret"`
	SessionCookieSecure bool                      `json:"sessionCookieSecure" yaml:"sessionCookieSecure"`
	Providers           map[string]ProviderConfig `json:"providers,omitempty" yaml:"providers,omitempty"`
}

func (c *Config) Sanitize() {
	c.Password = config.SanitizedValue
	c.SessionSecret = config.SanitizedValue
	for id, provider := range c.Providers {
		provider.ClientSecret = config.SanitizedValue
		c.Providers[id] = provider
	}
}

func (c *Config) Validate() error {
	if c.Methods == nil {
		c.Methods = []string{MethodPassword}
	}
	// Methods contains built-in login methods only; OAuth and OIDC are configured through Providers.
	for _, method := range c.Methods {
		if method != MethodPassword {
			return fmt.Errorf("auth: unsupported method %q", method)
		}
	}
	if slices.Contains(c.Methods, MethodPassword) && (c.User == "" || c.Password == "") {
		return errors.New("auth: user or password is needed, but found empty")
	}
	if c.ExpirationTime <= 0 || c.ExpirationTime >= 24*60*60 {
		return errors.New("auth: expirationTime should be greater than 0 and less than 86400")
	}
	if c.SessionSecret == "" {
		c.SessionSecret = DefaultSessionSecret
	}
	for id, provider := range c.Providers {
		if err := validateProvider(id, &provider); err != nil {
			return err
		}
		c.Providers[id] = provider
	}
	return nil
}

func validateProvider(id string, provider *ProviderConfig) error {
	if !providerIDPattern.MatchString(id) {
		return fmt.Errorf("auth: invalid provider id %q", id)
	}
	if provider.Type != ProviderTypeGitHub && provider.Type != ProviderTypeOIDC {
		return fmt.Errorf("auth provider %q: unsupported type %q", id, provider.Type)
	}
	if provider.DisplayName == "" {
		provider.DisplayName = id
	}
	if err := validateIconURL(provider.IconURL); err != nil {
		return fmt.Errorf("auth provider %q: invalid iconUrl: %w", id, err)
	}
	if provider.ClientID == "" || provider.ClientSecret == "" {
		return fmt.Errorf("auth provider %q: clientId and clientSecret are required", id)
	}
	redirect, err := validateHTTPURL(provider.RedirectURL)
	if err != nil {
		return fmt.Errorf("auth provider %q: invalid redirectUrl: %w", id, err)
	}
	expectedPath := "/api/v1/auth/providers/" + id + "/callback"
	// The provider must return to the callback route registered for this provider ID.
	if redirect.Path != expectedPath {
		return fmt.Errorf("auth provider %q: redirectUrl must use callback path %q", id, expectedPath)
	}
	if _, err := validateHTTPURL(provider.PostLoginRedirectURL); err != nil {
		return fmt.Errorf("auth provider %q: invalid postLoginRedirectUrl: %w", id, err)
	}
	switch provider.Type {
	case ProviderTypeGitHub:
		if len(provider.Scopes) == 0 {
			provider.Scopes = []string{"read:user", "user:email"}
		}
	case ProviderTypeOIDC:
		if _, err := validateOIDCURL(provider.Issuer); err != nil {
			return fmt.Errorf("auth provider %q: invalid issuer: %w", id, err)
		}
		if len(provider.Scopes) == 0 {
			provider.Scopes = []string{"openid", "profile", "email"}
		}
		if !slices.Contains(provider.Scopes, "openid") {
			return fmt.Errorf("auth provider %q: OIDC scopes must include openid", id)
		}
	}
	return nil
}

func validateIconURL(raw string) error {
	if raw == "" {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if parsed.Scheme == "" {
		if parsed.Host == "" && strings.HasPrefix(parsed.Path, "/") {
			return nil
		}
		return errors.New("must be a root-relative project path or an absolute HTTPS URL")
	}
	if _, err := validateOIDCURL(raw); err != nil {
		return err
	}
	return nil
}

func validateOIDCURL(raw string) (*url.URL, error) {
	parsed, err := validateHTTPURL(raw)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "https" || parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname()) {
		return parsed, nil
	}
	return nil, errors.New("must use HTTPS, except for an HTTP loopback development endpoint")
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validateHTTPURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, errors.New("must be an absolute HTTP or HTTPS URL")
	}
	return parsed, nil
}
