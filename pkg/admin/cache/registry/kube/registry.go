package kube

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/apache/dubbo-admin/pkg/admin/cache/registry"
	"github.com/apache/dubbo-admin/pkg/core/kubeclient/client"
)

func init() {
	registry.AddRegistry("universal", func(u *common.URL) (registry.AdminRegistry, error) {
		return NewRegistry()
	})
}

type Registry struct {
	client *client.KubeClient
}

func NewRegistry() (*Registry, error) {
	return nil, nil
}

func (kr *Registry) Subscribe(listener registry.AdminNotifyListener) error {
	return nil
}

func (kr *Registry) Destroy() error {
	return nil
}
