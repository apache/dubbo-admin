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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	configauth "github.com/apache/dubbo-admin/pkg/config/console/auth"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

const githubAPIBaseURL = "https://api.github.com"

type githubEndpoints struct {
	OAuth      oauth2.Endpoint
	APIBaseURL string
}

type githubProvider struct {
	id                   string
	displayName          string
	iconURL              string
	postLoginRedirectURL string
	oauth                oauth2.Config
	apiBaseURL           string
	httpClient           *http.Client
	canReadEmails        bool
}

type githubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Email string `json:"email"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func NewGitHubProvider(id string, cfg configauth.ProviderConfig) Provider {
	return newGitHubProvider(id, cfg, githubEndpoints{OAuth: github.Endpoint, APIBaseURL: githubAPIBaseURL}, http.DefaultClient)
}

func newGitHubProvider(id string, cfg configauth.ProviderConfig, endpoints githubEndpoints, client *http.Client) Provider {
	return &githubProvider{
		id:                   id,
		displayName:          cfg.DisplayName,
		iconURL:              cfg.IconURL,
		postLoginRedirectURL: cfg.PostLoginRedirectURL,
		oauth: oauth2.Config{
			ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret, RedirectURL: cfg.RedirectURL,
			Scopes: append([]string(nil), cfg.Scopes...), Endpoint: endpoints.OAuth,
		},
		apiBaseURL:    strings.TrimRight(endpoints.APIBaseURL, "/"),
		httpClient:    client,
		canReadEmails: slices.Contains(cfg.Scopes, "user:email"),
	}
}

func (p *githubProvider) ID() string                   { return p.id }
func (p *githubProvider) DisplayName() string          { return p.displayName }
func (p *githubProvider) IconURL() string              { return p.iconURL }
func (p *githubProvider) NeedsNonce() bool             { return false }
func (p *githubProvider) PostLoginRedirectURL() string { return p.postLoginRedirectURL }
func (p *githubProvider) AuthorizationURL(transaction OAuthTransaction) string {
	return p.oauth.AuthCodeURL(transaction.State,
		oauth2.SetAuthURLParam("code_challenge", PKCEChallenge(transaction.CodeVerifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"))
}

func (p *githubProvider) Authenticate(ctx context.Context, code, codeVerifier, _ string) (Principal, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.httpClient)
	token, err := p.oauth.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		return Principal{}, fmt.Errorf("exchange GitHub authorization code: %w", err)
	}
	client := p.oauth.Client(ctx, token)
	var user githubUser
	if err := getGitHubJSON(ctx, client, p.apiBaseURL+"/user", &user); err != nil {
		return Principal{}, fmt.Errorf("decode GitHub user: %w", err)
	}
	if user.ID <= 0 {
		return Principal{}, errors.New("GitHub user numeric id is missing")
	}
	email := user.Email
	if email == "" && p.canReadEmails {
		var emails []githubEmail
		if err := getGitHubJSON(ctx, client, p.apiBaseURL+"/user/emails", &emails); err != nil {
			return Principal{}, fmt.Errorf("decode GitHub emails: %w", err)
		}
		email = selectGitHubEmail(emails)
	}
	return Principal{
		Subject: fmt.Sprintf("%s:%d", p.id, user.ID), Username: user.Login, Email: email,
		Groups: []string{}, Roles: []string{}, AuthType: "oauth", Provider: p.id,
	}, nil
}

func getGitHubJSON(ctx context.Context, client *http.Client, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func selectGitHubEmail(emails []githubEmail) string {
	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email
		}
	}
	for _, email := range emails {
		if email.Verified {
			return email.Email
		}
	}
	return ""
}
