package index

import (
	"github.com/apache/dubbo-admin/pkg/common/errors"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"k8s.io/client-go/tools/cache"
)

const (
	ByServiceProviderAppName = "idx_service_provider_app_name"
)

func init() {
	RegisterIndexers(meshresource.ServiceProviderMetadataKind, map[string]cache.IndexFunc{
		ByServiceProviderAppName: byServiceProviderAppName,
	})
}

func byServiceProviderAppName(obj interface{}) ([]string, error) {
	instance, ok := obj.(meshresource.ServiceProviderMetadataResource)
	if !ok {
		return nil, errors.NewAssertionError(meshresource.ServiceProviderMetadataKind, obj)
	}
	if instance.Spec == nil {
		return []string{}, nil
	}
	return []string{instance.Spec.ProviderAppName}, nil
}
