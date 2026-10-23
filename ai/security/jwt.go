package security

import (
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken        = errors.New("invalid authentication token")
	ErrUnsupportedTokenAlg = errors.New("unsupported authentication token algorithm")
)

// JWTConfig configures the production JWT authenticator. The first production
// adapter deliberately supports an explicitly configured HMAC key; a JWKS
// adapter can be added without changing the Authenticator contract.
type JWTConfig struct {
	Secret   []byte
	Issuer   string
	Audience string
}

// JWTAuthenticator validates bearer tokens before constructing an AccessContext.
// It never trusts tenant or user headers supplied by the caller.
type JWTAuthenticator struct {
	secret   []byte
	issuer   string
	audience string
}

func NewJWTAuthenticator(cfg JWTConfig) (*JWTAuthenticator, error) {
	if len(cfg.Secret) < 32 || strings.TrimSpace(cfg.Issuer) == "" || strings.TrimSpace(cfg.Audience) == "" {
		return nil, ErrAuthenticationRequired
	}
	return &JWTAuthenticator{secret: append([]byte(nil), cfg.Secret...), issuer: cfg.Issuer, audience: cfg.Audience}, nil
}

func (a *JWTAuthenticator) Authenticate(r *http.Request) (AccessContext, error) {
	if a == nil || r == nil {
		return AccessContext{}, ErrAuthenticationRequired
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return AccessContext{}, ErrAuthenticationRequired
	}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(a.issuer),
		jwt.WithAudience(a.audience),
		jwt.WithExpirationRequired(),
	)
	token, err := parser.Parse(parts[1], func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrUnsupportedTokenAlg
		}
		return a.secret, nil
	})
	if err != nil || token == nil || !token.Valid {
		return AccessContext{}, ErrInvalidToken
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return AccessContext{}, ErrInvalidToken
	}
	userID, _ := claims["sub"].(string)
	tenantID, _ := claims["tenant_id"].(string)
	if tenantID == "" {
		tenantID, _ = claims["tenant"].(string)
	}
	if userID == "" || tenantID == "" {
		return AccessContext{}, ErrInvalidToken
	}
	roles, ok := stringClaims(claims["roles"])
	if !ok {
		return AccessContext{}, ErrInvalidToken
	}
	scopes, ok := stringClaims(claims["scopes"])
	if !ok {
		return AccessContext{}, ErrInvalidToken
	}
	return NewAccessContext(Subject{TenantID: tenantID, UserID: userID, Roles: roles, Scopes: scopes, AuthMethod: AuthMethodJWT})
}

func stringClaims(value any) ([]string, bool) {
	if value == nil {
		return nil, true
	}
	if s, ok := value.(string); ok {
		if s == "" {
			return nil, true
		}
		parts := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' })
		return parts, true
	}
	values, ok := value.([]any)
	if !ok {
		return nil, false
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		item, ok := value.(string)
		if !ok || strings.TrimSpace(item) == "" {
			return nil, false
		}
		result = append(result, item)
	}
	return result, true
}
