package discovery

import (
	"math"

	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

type Component interface {
	runtime.Component
	ResourceDiscovery() ResourceDiscovery
}

var _ Component = &BaseResourceDiscoveryComponent{}

type BaseResourceDiscoveryComponent struct{}

func (b *BaseResourceDiscoveryComponent) Type() runtime.ComponentType {
	return runtime.ResourceDiscovery
}

func (b *BaseResourceDiscoveryComponent) SubType() runtime.ComponentSubType {
	panic("SubType() must be implemented by concrete BaseResourceDiscoveryComponent")
}

func (b *BaseResourceDiscoveryComponent) Order() int {
	return math.MaxInt
}

func (b *BaseResourceDiscoveryComponent) Init(_ runtime.BuilderContext) error {
	panic("Init() must be implemented by concrete BaseResourceDiscoveryComponent")
}

func (b *BaseResourceDiscoveryComponent) Start(_ runtime.Runtime, _ <-chan struct{}) error {
	panic("Start() must be implemented by concrete BaseResourceDiscoveryComponent")
}

func (b *BaseResourceDiscoveryComponent) ResourceDiscovery() ResourceDiscovery {
	panic("Discovery() must be implemented by concrete BaseResourceDiscoveryComponent")
}
