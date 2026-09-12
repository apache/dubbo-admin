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
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	configauth "github.com/apache/dubbo-admin/pkg/config/console/auth"
	"github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"golang.org/x/oauth2"
)

const oidcHTTPTimeout = 10 * time.Second

type oidcDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
}

type oidcEndpoint struct {
	name     string
	value    string
	required bool
}

type oidcProfile struct {
	PreferredUsername string   `json:"preferred_username"`
	Name              string   `json:"name"`
	Email             string   `json:"email"`
	Groups            []string `json:"groups"`
	Roles             []string `json:"roles"`
	AuthorizedParty   string   `json:"azp"`
}

type oidcProvider struct {
	id                   string
	displayName          string
	iconURL              string
	clientID             string
	postLoginRedirectURL string
	oauth                oauth2.Config
	provider             *oidc.Provider
	verifier             *oidc.IDTokenVerifier
	httpClient           *http.Client
	jwksURI              string
}

func NewOIDCProvider(ctx context.Context, id string, cfg configauth.ProviderConfig, client *http.Client) (Provider, error) {
	if err := validateOIDCEndpoint("issuer", cfg.Issuer, true); err != nil {
		return nil, fmt.Errorf("OIDC provider %q: %w", id, err)
	}
	if client == nil {
		client = &http.Client{Timeout: oidcHTTPTimeout}
	}
	discoveryContext := context.WithValue(ctx, oauth2.HTTPClient, client)
	discoveredProvider, err := oidc.NewProvider(discoveryContext, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider %q: %w", id, err)
	}
	var discovery oidcDiscovery
	if err := discoveredProvider.Claims(&discovery); err != nil {
		return nil, fmt.Errorf("decode OIDC provider %q discovery: %w", id, err)
	}
	endpoints := []oidcEndpoint{
		{name: "authorization endpoint", value: discovery.AuthorizationEndpoint, required: true},
		{name: "token endpoint", value: discovery.TokenEndpoint, required: true},
		{name: "JWKS endpoint", value: discovery.JWKSURI, required: true},
		{name: "UserInfo endpoint", value: discovery.UserInfoEndpoint},
	}
	for _, endpoint := range endpoints {
		if err := validateOIDCEndpoint(endpoint.name, endpoint.value, endpoint.required); err != nil {
			return nil, fmt.Errorf("OIDC provider %q: %w", id, err)
		}
	}

	provider := &oidcProvider{
		id: id, displayName: cfg.DisplayName, iconURL: cfg.IconURL, clientID: cfg.ClientID,
		postLoginRedirectURL: cfg.PostLoginRedirectURL, provider: discoveredProvider, httpClient: client,
		jwksURI: discovery.JWKSURI,
	}
	provider.oauth = oauth2.Config{
		ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret, RedirectURL: cfg.RedirectURL,
		Scopes: append([]string(nil), cfg.Scopes...), Endpoint: discoveredProvider.Endpoint(),
	}
	provider.verifier = discoveredProvider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID, SupportedSigningAlgs: []string{oidc.RS256},
	})
	return provider, nil
}

func (p *oidcProvider) ID() string                   { return p.id }
func (p *oidcProvider) DisplayName() string          { return p.displayName }
func (p *oidcProvider) IconURL() string              { return p.iconURL }
func (p *oidcProvider) NeedsNonce() bool             { return true }
func (p *oidcProvider) PostLoginRedirectURL() string { return p.postLoginRedirectURL }
func (p *oidcProvider) AuthorizationURL(transaction OAuthTransaction) string {
	return p.oauth.AuthCodeURL(transaction.State,
		oauth2.SetAuthURLParam("code_challenge", PKCEChallenge(transaction.CodeVerifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("nonce", transaction.Nonce))
}

func (p *oidcProvider) Authenticate(ctx context.Context, code, codeVerifier, nonce string) (Principal, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.httpClient)
	token, err := p.oauth.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		return Principal{}, fmt.Errorf("exchange OIDC authorization code: %w", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return Principal{}, errors.New("OIDC token response is missing id_token")
	}
	if err := p.validateJWKAlgorithm(ctx, rawIDToken); err != nil {
		return Principal{}, err
	}
	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return Principal{}, fmt.Errorf("verify OIDC ID Token signature, issuer, audience, or expiration: %w", err)
	}
	var profile oidcProfile
	if err := idToken.Claims(&profile); err != nil {
		return Principal{}, fmt.Errorf("decode OIDC ID Token claims: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(idToken.Nonce), []byte(nonce)) != 1 {
		return Principal{}, errors.New("OIDC ID Token nonce does not match OAuth transaction")
	}
	if idToken.Subject == "" {
		return Principal{}, errors.New("OIDC ID Token subject is missing")
	}
	if len(idToken.Audience) > 1 && profile.AuthorizedParty == "" {
		return Principal{}, errors.New("OIDC ID Token authorized party is required for multiple audiences")
	}
	if profile.AuthorizedParty != "" && profile.AuthorizedParty != p.clientID {
		return Principal{}, errors.New("OIDC ID Token authorized party does not match client ID")
	}
	if profile.Email == "" && p.provider.UserInfoEndpoint() != "" {
		userInfo, err := p.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
		if err != nil {
			return Principal{}, fmt.Errorf("read OIDC UserInfo: %w", err)
		}
		if userInfo.Subject != idToken.Subject {
			return Principal{}, errors.New("OIDC UserInfo subject does not match ID Token subject")
		}
		var fallback oidcProfile
		if err := userInfo.Claims(&fallback); err != nil {
			return Principal{}, fmt.Errorf("decode OIDC UserInfo claims: %w", err)
		}
		mergeOIDCProfile(&profile, fallback)
	}
	return Principal{
		Subject: p.id + ":" + idToken.Subject, Username: oidcUsername(profile, idToken.Subject), Email: profile.Email,
		Groups: nonNilStrings(profile.Groups), Roles: nonNilStrings(profile.Roles), AuthType: "oidc", Provider: p.id,
	}, nil
}

func (p *oidcProvider) validateJWKAlgorithm(ctx context.Context, rawIDToken string) error {
	token, err := josejwt.ParseSigned(rawIDToken, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		return fmt.Errorf("parse OIDC ID Token header: %w", err)
	}
	if len(token.Headers) != 1 || token.Headers[0].KeyID == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.jwksURI, nil)
	if err != nil {
		return fmt.Errorf("create OIDC JWKS request: %w", err)
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("read OIDC JWKS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("read OIDC JWKS: endpoint returned HTTP %d", resp.StatusCode)
	}
	var keySet jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&keySet); err != nil {
		return fmt.Errorf("decode OIDC JWKS: %w", err)
	}
	keys := keySet.Key(token.Headers[0].KeyID)
	for _, key := range keys {
		if key.Algorithm == "" || key.Algorithm == string(jose.RS256) {
			return nil
		}
	}
	if len(keys) > 0 {
		return errors.New("OIDC ID Token signing key declares an algorithm other than RS256")
	}
	return nil
}

func validateOIDCEndpoint(name, raw string, required bool) error {
	if raw == "" {
		if required {
			return fmt.Errorf("%s is missing", name)
		}
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("%s is not an absolute URL", name)
	}
	if parsed.Scheme == "https" || parsed.Scheme == "http" && oidcLoopbackHost(parsed.Hostname()) {
		return nil
	}
	return fmt.Errorf("%s must use HTTPS, except for an HTTP loopback development endpoint", name)
}

func oidcLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func oidcUsername(profile oidcProfile, subject string) string {
	for _, candidate := range []string{profile.PreferredUsername, profile.Name, profile.Email, subject} {
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

func mergeOIDCProfile(target *oidcProfile, fallback oidcProfile) {
	if target.PreferredUsername == "" {
		target.PreferredUsername = fallback.PreferredUsername
	}
	if target.Name == "" {
		target.Name = fallback.Name
	}
	if target.Email == "" {
		target.Email = fallback.Email
	}
	if len(target.Groups) == 0 {
		target.Groups = fallback.Groups
	}
	if len(target.Roles) == 0 {
		target.Roles = fallback.Roles
	}
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
