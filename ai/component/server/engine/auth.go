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
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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

const (
	jwksRefreshInterval = 30 * time.Second
	jwksRetryInterval   = 5 * time.Second
	// Only failed refreshes may reuse the last successful keys, for a bounded time.
	jwksStaleGracePeriod = 2 * time.Minute
	jwksFetchTimeout     = 10 * time.Second
)

var errJWKSUnavailable = errors.New("verification keys temporarily unavailable")

type jwksRefresh struct {
	done chan struct{}
	err  error
}

type JWTAuthConfig struct {
	JWKSURL  string
	Issuer   string
	Audience string
}

type Claims struct {
	// SessionHash binds the AI cookie to the Admin session that requested it.
	// This lets AI reject a stale cookie delivered after logout or an account switch.
	SessionHash string `json:"session_hash,omitempty"`
	jwt.RegisteredClaims
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Groups   []string `json:"groups"`
	Roles    []string `json:"roles"`
	AuthType string   `json:"auth_type"`
	Provider string   `json:"provider"`
}

type JWTVerifier struct {
	config         JWTAuthConfig
	client         *JWKSClient
	mu             sync.RWMutex
	keys           map[string]*rsa.PublicKey
	nextRefreshAt  time.Time
	lastRefreshErr error
	staleUntil     time.Time
	now            func() time.Time
	refreshing     *jwksRefresh
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
		if c.Request.Method == http.MethodOptions || c.Request.URL.Path == "/health" {
			c.Next()
			return
		}
		header := c.GetHeader("Authorization")
		var raw, sessionHash string
		if header != "" {
			if len(header) < 8 || !strings.EqualFold(header[:7], "Bearer ") {
				abortUnauthorized(c, "Bearer")
				return
			}
			raw = strings.TrimSpace(header[7:])
		} else {
			cookie, err := c.Request.Cookie("dubbo_ai")
			session, sessionErr := c.Request.Cookie("session")
			if err != nil || sessionErr != nil {
				abortUnauthorized(c, "Bearer")
				return
			}
			digest := sha256.Sum256([]byte(session.Value))
			sessionHash = hex.EncodeToString(digest[:])
			raw = cookie.Value
			if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
				if c.GetHeader("X-AI-Cookie") != "1" || (c.GetHeader("Sec-Fetch-Site") != "" && c.GetHeader("Sec-Fetch-Site") != "same-origin") {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
			}
		}
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
			return v.keyForToken(c.Request.Context(), token)
		},
			jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
			jwt.WithIssuer(v.config.Issuer), jwt.WithAudience(v.config.Audience),
			jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithLeeway(30*time.Second),
		)
		if errors.Is(err, errJWKSUnavailable) {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": errJWKSUnavailable.Error()})
			return
		}
		if err != nil || !token.Valid || claims.Subject == "" || claims.IssuedAt == nil || claims.SessionHash != sessionHash {
			abortUnauthorized(c, `Bearer error="invalid_token"`)
			return
		}
		access, err := security.NewAccessContext(security.Subject{TenantID: claims.Issuer, UserID: claims.Subject, AuthMethod: "jwt"})
		if err != nil {
			abortUnauthorized(c, `Bearer error="invalid_token"`)
			return
		}
		c.Request = c.Request.WithContext(security.WithAccessContext(c.Request.Context(), access))
		c.Next()
	}
}

// JWTRequestAuthenticator retrieves the identity established by JWT verification.
type JWTRequestAuthenticator struct{}

func (JWTRequestAuthenticator) Authenticate(r *http.Request) (security.AccessContext, error) {
	return security.AccessContextFromContext(r.Context())
}

func (v *JWTVerifier) keyForToken(ctx context.Context, token *jwt.Token) (any, error) {
	if token.Method != jwt.SigningMethodRS256 {
		return nil, errors.New("only RS256 is accepted")
	}
	kid, ok := token.Header["kid"].(string)
	if !ok || kid == "" {
		return nil, errors.New("JWT kid is missing")
	}
	// Refresh errors may fall back to the last successful keys, but cancellation
	// belongs to this caller and must never allow it to continue authentication.
	refreshErr := v.refresh(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	if !v.clock().Before(v.staleUntil) {
		return nil, errJWKSUnavailable
	}
	key := v.keys[kid]
	if key == nil {
		if refreshErr != nil {
			return nil, errJWKSUnavailable
		}
		return nil, errors.New("JWT kid is unknown")
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
	flight := v.refreshing
	if flight == nil {
		if v.clock().Before(v.nextRefreshAt) {
			err := v.lastRefreshErr
			v.mu.Unlock()
			return err
		}
		flight = &jwksRefresh{done: make(chan struct{})}
		v.refreshing = flight
		// A shared fetch outlives individual callers, but always has its own deadline,
		// including when an injected HTTP client has no timeout.
		go func() {
			fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), jwksFetchTimeout)
			defer cancel()
			keys, err := v.fetchKeys(fetchCtx)
			v.mu.Lock()
			defer v.mu.Unlock()
			now := v.clock()
			if err == nil {
				// Publish a complete snapshot. Keys removed by a successful refresh must
				// never be restored by the stale fallback.
				v.keys = keys
				v.nextRefreshAt = now.Add(jwksRefreshInterval)
				v.staleUntil = v.nextRefreshAt.Add(jwksStaleGracePeriod)
			} else {
				v.nextRefreshAt = now.Add(jwksRetryInterval)
				// Failure must not extend the last successful snapshot's hard expiry.
			}
			v.lastRefreshErr = err
			flight.err = err
			v.refreshing = nil
			close(flight.done)
		}()
	}
	v.mu.Unlock()
	// The initiator and every subsequent caller cancel only their own wait.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-flight.done:
		return flight.err
	}
}

func (v *JWTVerifier) fetchKeys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	document, err := v.client.Fetch(ctx)
	if err != nil {
		return nil, err
	}
	keys := make(map[string]*rsa.PublicKey)
	for _, item := range document.Keys {
		if item.KeyType != "RSA" || item.Algorithm != "RS256" || item.Use != "sig" || item.KeyID == "" {
			continue
		}
		if _, exists := keys[item.KeyID]; exists {
			return nil, errors.New("JWKS contains duplicate kid")
		}
		modulus, err := base64.RawURLEncoding.DecodeString(item.Modulus)
		if err != nil || len(modulus) == 0 {
			return nil, fmt.Errorf("invalid JWKS modulus for %q", item.KeyID)
		}
		exponent, err := base64.RawURLEncoding.DecodeString(item.Exponent)
		if err != nil || len(exponent) == 0 {
			return nil, fmt.Errorf("invalid JWKS exponent for %q", item.KeyID)
		}
		e := new(big.Int).SetBytes(exponent)
		if !e.IsInt64() || e.Int64() < 3 || e.Bit(0) == 0 || e.Int64() > int64(^uint(0)>>1) {
			return nil, fmt.Errorf("invalid JWKS exponent for %q", item.KeyID)
		}
		n := new(big.Int).SetBytes(modulus)
		if n.BitLen() < 2048 || n.Bit(0) == 0 {
			return nil, errors.New("invalid JWKS RSA modulus")
		}
		keys[item.KeyID] = &rsa.PublicKey{N: n, E: int(e.Int64())}
	}
	if len(keys) == 0 {
		return nil, errors.New("JWKS contains no usable RS256 signing keys")
	}
	return keys, nil
}

func abortUnauthorized(c *gin.Context, challenge string) {
	c.Header("WWW-Authenticate", challenge)
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing bearer token"})
}
