package engine

import (
	"math"

	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

type Component interface {
	runtime.Component
	ResourceEngine() ResourceEngine
}

var _ Component = &BaseResourceEngineComponent{}

type BaseResourceEngineComponent struct{}

func (b BaseResourceEngineComponent) Type() runtime.ComponentType {
	return runtime.ResourceEngine
}

func (b BaseResourceEngineComponent) SubType() runtime.ComponentSubType {
	panic("SubType() must be implemented by concrete BaseResourceEngineComponent")

}

func (b BaseResourceEngineComponent) Order() int {
	return math.MaxInt
}

func (b BaseResourceEngineComponent) Init(runtime.BuilderContext) error {
	panic("Init() must be implemented by concrete BaseResourceEngineComponent")

}

func (b BaseResourceEngineComponent) Start(runtime.Runtime, <-chan struct{}) error {
	panic("Start() must be implemented by concrete BaseResourceEngineComponent")
}

func (b BaseResourceEngineComponent) ResourceEngine() ResourceEngine {
	panic("Discovery() must be implemented by concrete BaseResourceEngineComponent")

}
