package store

import (
	"math"

	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

type Component interface {
	runtime.Component
	ResourceStore() ResourceStore
}

var _ Component = &BaseResourceStoreComponent{}

type BaseResourceStoreComponent struct{}

func (b BaseResourceStoreComponent) Type() runtime.ComponentType {
	return runtime.ResourceStore
}

func (b BaseResourceStoreComponent) SubType() runtime.ComponentSubType {
	panic("SubType() must be implemented by concrete ResourceStoreComponent")
}

func (b BaseResourceStoreComponent) Order() int {
	return math.MaxInt
}

func (b BaseResourceStoreComponent) Init(_ runtime.BuilderContext) error {
	panic("Init() must be implemented by concrete ResourceStoreComponent")
}

func (b BaseResourceStoreComponent) Start(runtime.Runtime, <-chan struct{}) error {
	panic("Start() must be implemented by concrete ResourceStoreComponent")
}

func (b BaseResourceStoreComponent) ResourceStore() ResourceStore {
	panic("ResourceStore() must be implemented by concrete ResourceStoreComponent")
}
