package servertest

import (
	"strings"
	"testing"

	compServer "dubbo-admin-ai/component/server"
)

func TestServerComponent_Validate(t *testing.T) {
	tests := []struct {
		name         string
		port         int
		readTimeout  int
		writeTimeout int
		errContain   string
	}{
		{name: "port_range", port: 70000, readTimeout: 30, writeTimeout: 30, errContain: "port"},
		{name: "read_timeout_positive", port: 8080, readTimeout: 0, writeTimeout: 30, errContain: "timeout"},
		{name: "write_timeout_positive", port: 8080, readTimeout: 30, writeTimeout: 0, errContain: "timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp, err := compServer.NewServerComponent(compServer.ServerSpec{
				Port: tt.port, Host: "0.0.0.0", CORSOrigins: []string{"*"},
				ReadTimeout: tt.readTimeout, WriteTimeout: tt.writeTimeout,
			})
			if err != nil {
				t.Fatalf("NewServerComponent() error: %v", err)
			}
			if err := comp.Validate(); err == nil || !strings.Contains(err.Error(), tt.errContain) {
				t.Fatalf("expected %q validation error, got %v", tt.errContain, err)
			}
		})
	}
}

func TestAuthSpecValidate(t *testing.T) {
	if err := (compServer.AuthSpec{}).Validate(); err != nil {
		t.Fatalf("disabled auth should be valid: %v", err)
	}
	for _, tt := range []struct {
		name string
		auth compServer.AuthSpec
		want string
	}{
		{name: "missing JWKS URL", auth: compServer.AuthSpec{Enabled: true, Issuer: "issuer", Audience: "audience"}, want: "jwks_url"},
		{name: "invalid JWKS URL", auth: compServer.AuthSpec{Enabled: true, JWKSURL: "not-a-url", Issuer: "issuer", Audience: "audience"}, want: "jwks_url"},
		{name: "missing issuer", auth: compServer.AuthSpec{Enabled: true, JWKSURL: "https://admin.example/jwks", Audience: "audience"}, want: "issuer"},
		{name: "missing audience", auth: compServer.AuthSpec{Enabled: true, JWKSURL: "https://admin.example/jwks", Issuer: "issuer"}, want: "audience"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.auth.Validate(); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() error = %v, want containing %q", err, tt.want)
			}
		})
	}
	if err := (compServer.AuthSpec{Enabled: true, JWKSURL: "https://admin.example/jwks", Issuer: "issuer", Audience: "audience"}).Validate(); err != nil {
		t.Fatalf("valid auth rejected: %v", err)
	}
}
