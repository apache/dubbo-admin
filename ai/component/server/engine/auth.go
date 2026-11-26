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

	"dubbo-admin-ai/security"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const jwtClaimsContextKey = "dubbo-admin-ai.jwt-claims"

const (
	// Cache lifetime after success and retry cooldown after failure.
	jwksRefreshInterval = 30 * time.Second
	// A shared refresh must not hang forever, but it must outlive the request
	// that triggered it. Otherwise one client disconnecting would abort the
	// fetch and start the failure cooldown for every other caller.
	jwksFetchTimeout = 10 * time.Second
)

type jwksRefresh struct {
	done chan struct{}
	err  error
}

type JWTAuthConfig struct {
	JWKSURL  string
	Issuer   string
	Audience string
	// TenantID is the trusted deployment tenant for tokens without tenant_id.
	// Leave empty to require a signed tenant_id claim.
	TenantID string
}

type Claims struct {
	jwt.RegisteredClaims
	TenantID string   `json:"tenant_id"`
	Scopes   []string `json:"scopes"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Groups   []string `json:"groups"`
	Roles    []string `json:"roles"`
	AuthType string   `json:"auth_type"`
	Provider string   `json:"provider"`
}

type JWTVerifier struct {
	config      JWTAuthConfig
	client      *JWKSClient
	mu          sync.RWMutex
	keys        map[string]*rsa.PublicKey
	nextRefresh time.Time
	now         func() time.Time
	refreshing  *jwksRefresh
}

// NewJWTVerifier deliberately fails startup if Admin JWKS is unavailable.
// Subsequent requests share the bounded refresh cache and cooldown.
func NewJWTVerifier(config JWTAuthConfig, client *http.Client) (*JWTVerifier, error) {
	verifier := &JWTVerifier{config: config, client: NewJWKSClient(config.JWKSURL, client), keys: map[string]*rsa.PublicKey{}}
	if err := verifier.refresh(context.Background()); err != nil {
		return nil, fmt.Errorf("load remote JWKS: %w", err)
	}
	return verifier, nil
}

// Authenticate implements the router's trusted identity boundary.
func (v *JWTVerifier) Authenticate(r *http.Request) (security.AccessContext, error) {
	claims, err := v.verify(r)
	if err != nil {
		return security.AccessContext{}, err
	}
	return v.accessContext(claims)
}

func (v *JWTVerifier) verify(r *http.Request) (*Claims, error) {
	if v == nil || r == nil {
		return nil, security.ErrAuthenticationRequired
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, security.ErrAuthenticationRequired
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (any, error) {
		return v.keyForToken(r.Context(), token)
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithIssuer(v.config.Issuer), jwt.WithAudience(v.config.Audience),
		jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithLeeway(30*time.Second))
	if err != nil || token == nil || !token.Valid || strings.TrimSpace(claims.Subject) == "" || claims.IssuedAt == nil {
		return nil, security.ErrAuthenticationRequired
	}
	return claims, nil
}

func (v *JWTVerifier) accessContext(claims *Claims) (security.AccessContext, error) {
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = v.config.TenantID
	}
	return security.NewAccessContext(security.Subject{TenantID: tenantID, UserID: claims.Subject,
		Roles: claims.Roles, Scopes: claims.Scopes, AuthMethod: security.AuthMethodJWT})
}

// isPublicProbe reports the explicit unauthenticated health probes. Every other
// path, including future routes outside the AI prefix, requires authentication.
func isPublicProbe(path string) bool {
	switch path {
	case "/health", "/healthz", "/readyz":
		return true
	default:
		return false
	}
}

func (v *JWTVerifier) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isPublicProbe(c.Request.URL.Path) {
			c.Next()
			return
		}
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		claims, err := v.verify(c.Request)
		if err != nil {
			abortUnauthorized(c)
			return
		}
		access, err := v.accessContext(claims)
		if err != nil {
			abortUnauthorized(c)
			return
		}
		c.Set(jwtClaimsContextKey, claims)
		c.Request = c.Request.WithContext(security.WithAccessContext(c.Request.Context(), access))
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

func (v *JWTVerifier) keyForToken(ctx context.Context, token *jwt.Token) (any, error) {
	if token.Method != jwt.SigningMethodRS256 {
		return nil, errors.New("only RS256 is accepted")
	}
	kid, ok := token.Header["kid"].(string)
	if !ok || kid == "" {
		return nil, errors.New("JWT kid is missing")
	}
	v.mu.RLock()
	key := v.keys[kid]
	fresh := v.clock().Before(v.nextRefresh)
	v.mu.RUnlock()
	if fresh {
		if key == nil {
			return nil, errors.New("JWT kid is unknown")
		}
		return key, nil
	}
	if err := v.refresh(ctx); err != nil {
		return nil, err
	}
	v.mu.RLock()
	key = v.keys[kid]
	fresh = v.clock().Before(v.nextRefresh)
	v.mu.RUnlock()
	if !fresh {
		return nil, errors.New("JWKS cache expired")
	}
	if key == nil {
		return nil, fmt.Errorf("JWT kid %q is unknown", kid)
	}
	return key, nil
}

func (v *JWTVerifier) clock() time.Time {
	if v.now != nil {
		return v.now()
	}
	return time.Now()
}

func (v *JWTVerifier) refresh(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	v.mu.Lock()
	if flight := v.refreshing; flight != nil {
		v.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-flight.done:
			return flight.err
		}
	}
	if v.clock().Before(v.nextRefresh) {
		v.mu.Unlock()
		return nil // Use the cache, or reject if a failed refresh left it empty.
	}
	flight := &jwksRefresh{done: make(chan struct{})}
	v.refreshing = flight
	// Expired keys must not remain usable when refreshing fails.
	v.keys = nil
	v.mu.Unlock()

	// All concurrent misses share this attempt. Detach the fetch from the
	// initiating request: a client disconnect must not abort the shared refresh
	// or start a cooldown that rejects valid tokens for every other caller.
	fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), jwksFetchTimeout)
	defer cancel()
	err := v.fetchKeys(fetchCtx)
	v.mu.Lock()
	flight.err = err
	v.nextRefresh = v.clock().Add(jwksRefreshInterval)
	v.refreshing = nil
	close(flight.done)
	v.mu.Unlock()
	return err
}

func (v *JWTVerifier) fetchKeys(ctx context.Context) error {
	document, err := v.client.Fetch(ctx)
	if err != nil {
		return err
	}
	keys := make(map[string]*rsa.PublicKey)
	for _, item := range document.Keys {
		if item.KeyType != "RSA" || item.Algorithm != "RS256" || item.Use != "sig" || item.KeyID == "" {
			continue
		}
		if _, exists := keys[item.KeyID]; exists {
			return errors.New("JWKS contains duplicate kid")
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
		if !e.IsInt64() || e.Int64() < 3 || e.Bit(0) == 0 || e.Int64() > int64(^uint(0)>>1) {
			return fmt.Errorf("invalid JWKS exponent for %q", item.KeyID)
		}
		n := new(big.Int).SetBytes(modulus)
		if n.BitLen() < 2048 || n.Bit(0) == 0 {
			return errors.New("invalid JWKS RSA modulus")
		}
		keys[item.KeyID] = &rsa.PublicKey{N: n, E: int(e.Int64())}
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
