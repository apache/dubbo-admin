package v1alpha1

import (
	"fmt"

	"github.com/apache/dubbo-admin/pkg/core/store"
	"k8s.io/client-go/tools/cache"
)

func init() {
	store.RegisterIndexers(InstanceKind, map[string]cache.IndexFunc{
		"AppName": byAppName,
	})
}

func byAppName(obj interface{}) ([]string, error) {
	instance, ok := obj.(InstanceResource)
	if !ok {
		return nil, fmt.Errorf("invalid object type, required %s, got %v", InstanceKind, obj)
	}
	if instance.Spec == nil {
		return []string{}, nil
	}
	return []string{instance.Spec.AppName}, nil
}
