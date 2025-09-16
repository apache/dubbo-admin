package index

import (
	"github.com/apache/dubbo-admin/pkg/common/errors"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"k8s.io/client-go/tools/cache"
)

const (
	ByInstanceAppNameIndex = "idx_instance_app_name"
	ByInstanceIpIndex      = "idx_instance_ip"
)

func init() {
	RegisterIndexers(meshresource.InstanceKind, map[string]cache.IndexFunc{
		ByInstanceAppNameIndex: byInstanceAppName,
		ByInstanceIpIndex:      byIp,
	})
}

func byInstanceAppName(obj interface{}) ([]string, error) {
	instance, ok := obj.(meshresource.InstanceResource)
	if !ok {
		return nil, errors.NewAssertionError(string(meshresource.InstanceKind), string(instance.ResourceKind()))
	}
	if instance.Spec == nil {
		return []string{}, nil
	}
	return []string{instance.Spec.AppName}, nil
}

func byIp(obj interface{}) ([]string, error) {
	instance, ok := obj.(meshresource.InstanceResource)
	if !ok {
		return nil, errors.NewAssertionError(string(meshresource.InstanceKind), string(instance.ResourceKind()))
	}
	if instance.Spec == nil {
		return []string{}, nil
	}
	return []string{instance.Spec.Ip}, nil
}
