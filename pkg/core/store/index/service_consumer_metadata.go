package index

import (
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"k8s.io/client-go/tools/cache"
)

const (
	ByServiceConsumerAppName = "idx_service_consumer_app_name"
)

func init() {
	RegisterIndexers(meshresource.ServiceConsumerMetadataKind, map[string]cache.IndexFunc{
		ByServiceProviderAppName: byServiceConsumerAppName,
	})
}

func byServiceConsumerAppName(obj interface{}) ([]string, error) {
	serviceConsumerMetadata, ok := obj.(*meshresource.ServiceConsumerMetadataResource)
	if !ok {
		return []string{}, nil
	}
	return []string{serviceConsumerMetadata.Spec.ConsumerAppName}, nil
}
