package registry

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"
)

var registries = make(map[string]func(u *common.URL) (AdminRegistry, error))

// AddRegistry sets the registry extension with @name
func AddRegistry(name string, v func(u *common.URL) (AdminRegistry, error)) {
	registries[name] = v
}

// Registry finds the registry extension with @name
func Registry(name string, config *common.URL) (AdminRegistry, error) {
	if name != "kubernetes" && name != "kube" && name != "k8s" {
		name = "universal"
	}
	if registries[name] == nil {
		panic("registry for " + name + " does not exist. please make sure that you have imported the package dubbo.apache.org/dubbo-go/v3/registry/" + name + ".")
	}
	return registries[name](config)
}

type AdminRegistry interface {
	Subscribe(listener AdminNotifyListener) error
	Destroy() error
	Delegate() dubboRegistry.Registry
}

type AdminNotifyListener interface {
	Notify(event *dubboRegistry.ServiceEvent)
}
