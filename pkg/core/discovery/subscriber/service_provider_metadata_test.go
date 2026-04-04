package subscriber

import (
	"testing"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

func TestInferProviderLanguage(t *testing.T) {
	t.Run("prefers explicit language", func(t *testing.T) {
		spec := &meshproto.ServiceProviderMetadata{
			Parameters: map[string]string{
				"language": "rust",
				"release":  "dubbo-golang-3.3.0",
			},
		}
		if got := inferProviderLanguage(spec); got != "rust" {
			t.Fatalf("inferProviderLanguage() = %q, want %q", got, "rust")
		}
	})

	t.Run("infers golang from release", func(t *testing.T) {
		spec := &meshproto.ServiceProviderMetadata{
			Parameters: map[string]string{
				"release": "dubbo-golang-3.3.0",
			},
		}
		if got := inferProviderLanguage(spec); got != "golang" {
			t.Fatalf("inferProviderLanguage() = %q, want %q", got, "golang")
		}
	})

	t.Run("infers java from metadata type hints", func(t *testing.T) {
		spec := &meshproto.ServiceProviderMetadata{
			Methods: []*meshproto.Method{
				{
					Name:           "login",
					ParameterTypes: []string{"java.lang.String", "java.lang.String"},
					ReturnType:     "org.apache.dubbo.samples.User",
				},
			},
			Types: []*meshproto.Type{
				{
					Type: "org.apache.dubbo.samples.User",
					Properties: map[string]string{
						"username": "java.lang.String",
					},
				},
			},
		}
		if got := inferProviderLanguage(spec); got != "java" {
			t.Fatalf("inferProviderLanguage() = %q, want %q", got, "java")
		}
	})
}

func TestBuildServiceSpecInfersLanguage(t *testing.T) {
	spec := buildServiceSpec("org.apache.dubbo.samples.OrderService", "", "", []*meshresource.ServiceProviderMetadataResource{
		{
			Spec: &meshproto.ServiceProviderMetadata{
				Parameters: map[string]string{
					"release": "dubbo-golang-3.3.0",
				},
				Methods: []*meshproto.Method{
					{Name: "submitOrder"},
				},
			},
		},
	})

	if spec.Language != "golang" {
		t.Fatalf("buildServiceSpec().Language = %q, want %q", spec.Language, "golang")
	}
	if len(spec.Methods) != 1 || spec.Methods[0] != "submitOrder" {
		t.Fatalf("buildServiceSpec().Methods = %#v, want [submitOrder]", spec.Methods)
	}
}
