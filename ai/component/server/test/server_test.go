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
			comp, err := compServer.NewServerComponent(tt.port, "127.0.0.1", false, nil, tt.readTimeout, tt.writeTimeout)
			if err != nil {
				t.Fatalf("NewServerComponent() error: %v", err)
			}
			if err := comp.Validate(); err == nil || !strings.Contains(err.Error(), tt.errContain) {
				t.Fatalf("expected %q validation error, got %v", tt.errContain, err)
			}
		})
	}
}

func TestServerComponent_StaticAuthenticationRequiresLoopback(t *testing.T) {
	base := compServer.DefaultServerSpec().Authentication
	component, err := compServer.NewServerComponentWithAuthentication(8888, "0.0.0.0", false, nil, 30, 30, base)
	if err != nil {
		t.Fatal(err)
	}
	if err := component.Validate(); err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("Validate() error = %v, want loopback restriction", err)
	}

	base.AllowInsecureStaticAuth = true
	component, err = compServer.NewServerComponentWithAuthentication(8888, "0.0.0.0", false, nil, 30, 30, base)
	if err != nil {
		t.Fatal(err)
	}
	if err := component.Validate(); err != nil {
		t.Fatalf("Validate() with explicit insecure opt-in error = %v", err)
	}
}
