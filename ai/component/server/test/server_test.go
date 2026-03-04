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
			comp, err := compServer.NewServerComponent(tt.port, "0.0.0.0", false, []string{"*"}, tt.readTimeout, tt.writeTimeout)
			if err != nil {
				t.Fatalf("NewServerComponent() error: %v", err)
			}
			if err := comp.Validate(); err == nil || !strings.Contains(err.Error(), tt.errContain) {
				t.Fatalf("expected %q validation error, got %v", tt.errContain, err)
			}
		})
	}
}
