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

package engine

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const jwtClaimsContextKey = "dubbo-admin-ai.jwt-claims"

type JWTAuthConfig struct {
	JWKSURL  string
	Issuer   string
	Audience string
}

type Claims struct {
	jwt.RegisteredClaims
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Groups   []string `json:"groups"`
	Roles    []string `json:"roles"`
	AuthType string   `json:"auth_type"`
	Provider string   `json:"provider"`
}

type JWTVerifier struct {
	config JWTAuthConfig
	client *JWKSClient
	mu     sync.RWMutex
	keys   map[string]*rsa.PublicKey
}

func NewJWTVerifier(config JWTAuthConfig, client *http.Client) (*JWTVerifier, error) {
	verifier := &JWTVerifier{config: config, client: NewJWKSClient(config.JWKSURL, client), keys: map[string]*rsa.PublicKey{}}
	if err := verifier.refresh(context.Background()); err != nil {
		return nil, fmt.Errorf("load remote JWKS: %w", err)
	}
	return verifier, nil
}

func (v *JWTVerifier) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions || c.Request.URL.Path == "/health" || !strings.HasPrefix(c.Request.URL.Path, "/api/v1/ai/") {
			c.Next()
			return
		}
		header := c.GetHeader("Authorization")
		if len(header) < 8 || !strings.EqualFold(header[:7], "Bearer ") || strings.TrimSpace(header[7:]) == "" {
			abortUnauthorized(c)
			return
		}
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(strings.TrimSpace(header[7:]), claims, v.keyForToken,
			jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
			jwt.WithIssuer(v.config.Issuer), jwt.WithAudience(v.config.Audience),
			jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithLeeway(30*time.Second),
		)
		if err != nil || !token.Valid || claims.Subject == "" || claims.IssuedAt == nil {
			abortUnauthorized(c)
			return
		}
		c.Set(jwtClaimsContextKey, claims)
		c.Next()
	}
}

func ClaimsFromContext(c *gin.Context) (*Claims, bool) {
	value, ok := c.Get(jwtClaimsContextKey)
	if !ok {
		return nil, false
	}
	claims, ok := value.(*Claims)
	return claims, ok
}

func (v *JWTVerifier) keyForToken(token *jwt.Token) (any, error) {
	if token.Method != jwt.SigningMethodRS256 {
		return nil, errors.New("only RS256 is accepted")
	}
	kid, ok := token.Header["kid"].(string)
	if !ok || kid == "" {
		return nil, errors.New("JWT kid is missing")
	}
	v.mu.RLock()
	key := v.keys[kid]
	v.mu.RUnlock()
	if key != nil {
		return key, nil
	}
	if err := v.refresh(context.Background()); err != nil {
		return nil, err
	}
	v.mu.RLock()
	key = v.keys[kid]
	v.mu.RUnlock()
	if key == nil {
		return nil, fmt.Errorf("JWT kid %q is unknown", kid)
	}
	return key, nil
}

func (v *JWTVerifier) refresh(ctx context.Context) error {
	document, err := v.client.Fetch(ctx)
	if err != nil {
		return err
	}
	keys := make(map[string]*rsa.PublicKey)
	for _, item := range document.Keys {
		if item.KeyType != "RSA" || item.Algorithm != "RS256" || item.Use != "sig" || item.KeyID == "" {
			continue
		}
		modulus, err := base64.RawURLEncoding.DecodeString(item.Modulus)
		if err != nil || len(modulus) == 0 {
			return fmt.Errorf("invalid JWKS modulus for %q", item.KeyID)
		}
		exponent, err := base64.RawURLEncoding.DecodeString(item.Exponent)
		if err != nil || len(exponent) == 0 {
			return fmt.Errorf("invalid JWKS exponent for %q", item.KeyID)
		}
		e := new(big.Int).SetBytes(exponent)
		if !e.IsInt64() || e.Sign() <= 0 || e.Int64() > int64(^uint(0)>>1) {
			return fmt.Errorf("invalid JWKS exponent for %q", item.KeyID)
		}
		keys[item.KeyID] = &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: int(e.Int64())}
	}
	if len(keys) == 0 {
		return errors.New("JWKS contains no usable RS256 signing keys")
	}
	v.mu.Lock()
	v.keys = keys
	v.mu.Unlock()
	return nil
}

func abortUnauthorized(c *gin.Context) {
	c.Header("WWW-Authenticate", "Bearer")
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing bearer token"})
}
