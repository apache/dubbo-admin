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
	"time"

	configauth "github.com/apache/dubbo-admin/pkg/config/console/auth"
	jose "github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
)

var ErrAccessTokenDisabled = errors.New("AI access token signing is not configured")

type AccessTokenClaims struct {
	josejwt.Claims
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Groups   []string `json:"groups"`
	Roles    []string `json:"roles"`
	AuthType string   `json:"auth_type"`
	Provider string   `json:"provider"`
}

type TokenResponse struct {
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
	ExpiresIn   int    `json:"expiresIn"`
	ExpiresAt   int64  `json:"expiresAt"`
}

type TokenIssuer struct {
	enabled bool
	config  *configauth.AccessTokenConfig
	signer  jose.Signer
	jwks    jose.JSONWebKeySet
}

func NewTokenIssuer(cfg *configauth.AccessTokenConfig) (*TokenIssuer, error) {
	if cfg == nil || !cfg.Enabled {
		return &TokenIssuer{}, nil
	}
	if cfg.PrivateKey == nil {
		return nil, errors.New("AI access token RSA private key is missing")
	}
	options := (&jose.SignerOptions{}).WithType("JWT").WithHeader(jose.HeaderKey("kid"), cfg.KeyID)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: cfg.PrivateKey}, options)
	if err != nil {
		return nil, err
	}
	return &TokenIssuer{
		enabled: true,
		config:  cfg,
		signer:  signer,
		jwks: jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
			Key: &cfg.PrivateKey.PublicKey, KeyID: cfg.KeyID, Algorithm: string(jose.RS256), Use: "sig",
		}}},
	}, nil
}

func (i *TokenIssuer) Enabled() bool {
	return i != nil && i.enabled
}

func (i *TokenIssuer) Issue(principal Principal, now time.Time) (TokenResponse, error) {
	if !i.Enabled() {
		return TokenResponse{}, ErrAccessTokenDisabled
	}
	expiresAt := now.Add(time.Duration(i.config.TTL) * time.Second)
	claims := AccessTokenClaims{
		Claims: josejwt.Claims{
			Issuer: i.config.Issuer, Subject: principal.Subject, Audience: josejwt.Audience(append([]string(nil), i.config.Audiences...)),
			IssuedAt: josejwt.NewNumericDate(now), Expiry: josejwt.NewNumericDate(expiresAt), ID: uuid.NewString(),
		},
		Username: principal.Username, Email: principal.Email, Groups: nonNilStrings(principal.Groups), Roles: nonNilStrings(principal.Roles),
		AuthType: principal.AuthType, Provider: principal.Provider,
	}
	serialized, err := josejwt.Signed(i.signer).Claims(claims).Serialize()
	if err != nil {
		return TokenResponse{}, err
	}
	return TokenResponse{AccessToken: serialized, TokenType: "Bearer", ExpiresIn: i.config.TTL, ExpiresAt: expiresAt.Unix()}, nil
}

func (i *TokenIssuer) JWKS() jose.JSONWebKeySet {
	if !i.Enabled() {
		return jose.JSONWebKeySet{}
	}
	return i.jwks
}
