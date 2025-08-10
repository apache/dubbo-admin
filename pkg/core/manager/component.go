package manager

import (
	"math"

	"github.com/pkg/errors"

	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

func init() {
	runtime.RegisterComponent(&resourceManagerComponent{})
}

type ResourceManagerComponent interface {
	runtime.Component
	ResourceManager() ResourceManager
}

var _ ResourceManagerComponent = &resourceManagerComponent{}

type resourceManagerComponent struct {
	rm ResourceManager
}

func (r *resourceManagerComponent) Type() runtime.ComponentType {
	return runtime.ResourceManager
}

func (r *resourceManagerComponent) Order() int {
	return math.MaxInt
}

func (r *resourceManagerComponent) Init(ctx runtime.BuilderContext) error {
	rsc, err := ctx.GetActivatedComponent(runtime.ResourceStore)
	if err != nil {
		return errors.Wrap(err, "failed to init resource manager")
	}
	r.rm = NewResourceManager(rsc.(store.Router))
	return nil
}

func (r *resourceManagerComponent) Start(runtime.Runtime, <-chan struct{}) error {
	return nil
}

func (r *resourceManagerComponent) ResourceManager() ResourceManager {
	return r.rm
}
