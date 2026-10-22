package security

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTAuthenticatorValidatesClaimsAndIgnoresSpoofedHeaders(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	auth, err := NewJWTAuthenticator(JWTConfig{Secret: secret, Issuer: "issuer", Audience: "audience"})
	if err != nil {
		t.Fatal(err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"iss": "issuer", "aud": "audience", "sub": "user-1", "tenant_id": "tenant-1", "roles": []string{"user"}, "scopes": []string{"ai:sessions:admin"}, "exp": time.Now().Add(time.Minute).Unix()})
	value, err := token.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+value)
	req.Header.Set("X-Tenant-ID", "attacker")
	access, err := auth.Authenticate(req)
	if err != nil {
		t.Fatal(err)
	}
	if access.Subject.TenantID != "tenant-1" || access.Subject.UserID != "user-1" || access.Subject.AuthMethod != AuthMethodJWT {
		t.Fatalf("unexpected subject: %+v", access.Subject)
	}
}

func TestJWTAuthenticatorRejectsWrongIssuerAudienceAndAlgorithm(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	auth, _ := NewJWTAuthenticator(JWTConfig{Secret: secret, Issuer: "issuer", Audience: "audience"})
	for name, claims := range map[string]jwt.MapClaims{
		"issuer":   {"iss": "other", "aud": "audience", "sub": "u", "tenant_id": "t", "exp": time.Now().Add(time.Minute).Unix()},
		"audience": {"iss": "issuer", "aud": "other", "sub": "u", "tenant_id": "t", "exp": time.Now().Add(time.Minute).Unix()},
	} {
		t.Run(name, func(t *testing.T) {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			value, _ := token.SignedString(secret)
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", "Bearer "+value)
			if _, err := auth.Authenticate(req); err == nil {
				t.Fatal("accepted invalid claims")
			}
		})
	}
	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"iss": "issuer", "aud": "audience", "sub": "u", "tenant_id": "t", "exp": time.Now().Add(time.Minute).Unix()})
	value, _ := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+value)
	if _, err := auth.Authenticate(req); err == nil {
		t.Fatal("accepted none algorithm")
	}
}

func TestJWTAuthenticatorRequiresExpirationAndValidatesNotBefore(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	auth, _ := NewJWTAuthenticator(JWTConfig{Secret: secret, Issuer: "issuer", Audience: "audience"})
	tests := []struct {
		name   string
		claims jwt.MapClaims
	}{
		{name: "missing exp", claims: jwt.MapClaims{"iss": "issuer", "aud": "audience", "sub": "u", "tenant_id": "t"}},
		{name: "expired", claims: jwt.MapClaims{"iss": "issuer", "aud": "audience", "sub": "u", "tenant_id": "t", "exp": time.Now().Add(-time.Minute).Unix()}},
		{name: "future nbf", claims: jwt.MapClaims{"iss": "issuer", "aud": "audience", "sub": "u", "tenant_id": "t", "exp": time.Now().Add(time.Minute).Unix(), "nbf": time.Now().Add(time.Minute).Unix()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, tt.claims)
			value, _ := token.SignedString(secret)
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", "Bearer "+value)
			if _, err := auth.Authenticate(req); err == nil {
				t.Fatal("accepted invalid time claims")
			}
		})
	}
}

func TestJWTAuthenticatorRejectsIncorrectSignature(t *testing.T) {
	auth, err := NewJWTAuthenticator(JWTConfig{Secret: []byte("01234567890123456789012345678901"), Issuer: "issuer", Audience: "audience"})
	if err != nil {
		t.Fatal(err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"iss": "issuer", "aud": "audience", "sub": "user", "tenant_id": "tenant", "exp": time.Now().Add(time.Minute).Unix()})
	value, err := token.SignedString([]byte("different-secret-0123456789012345"))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+value)
	if _, err := auth.Authenticate(req); err == nil {
		t.Fatal("accepted wrong signing key")
	}
}
