package store

import (
	"fmt"
	"math"

	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

type Router interface {
	ResourceRoute(coremodel.Resource) (ResourceStore, error)
	ResourceKindRoute(k coremodel.ResourceKind) (ResourceStore, error)
}

// The Component interface is composed of both functional interfaces and lifecycle interfaces
type Component interface {
	runtime.Component
	Router
}

var _ Component = &storeComponent{}

// storeComponent combine the stores and route Resource a request to an appropriate store
type storeComponent struct {
	// stores every resource corresponds to a ResourceStore
	stores map[coremodel.ResourceKind]ManagedResourceStore
}

func (sc *storeComponent) Type() runtime.ComponentType {
	return runtime.ResourceStore
}

func (sc *storeComponent) Order() int {
	return math.MaxInt
}

func (sc *storeComponent) Init(ctx runtime.BuilderContext) error {
	// 1. retrieve store config and choose the corresponding store factory
	cfg := ctx.Config().Store
	factory, err := FactoryRegistry().GetStoreFactory(cfg.Type)
	if err != nil {
		return err
	}
	// 2. create store for each resource kind
	for _, kind := range coremodel.ResourceKindRegistry().AllResourceKinds() {
		store, err := factory.New(kind, cfg)
		if err != nil {
			return err
		}
		sc.stores[kind] = store
		if err = store.Init(ctx); err != nil {
			return err
		}
	}
	// 3. add indexers for each kind of store
	for kind, store := range sc.stores {
		indexers := IndexersRegistry().Indexers(kind)
		if indexers == nil {
			continue
		}
		if err := store.AddIndexers(indexers); err != nil {
			return err
		}
	}
	return nil
}

func (sc *storeComponent) Start(rt runtime.Runtime, ch <-chan struct{}) error {
	for _, store := range sc.stores {
		if err := store.Start(rt, ch); err != nil {
			return err
		}
	}
	return nil
}

func (sc *storeComponent) ResourceRoute(r coremodel.Resource) (ResourceStore, error) {
	if store, exists := sc.stores[r.ResourceKind()]; exists {
		return store, nil
	}
	return nil, fmt.Errorf("%s is not supported by store yet", r.ResourceKind())
}

func (sc *storeComponent) ResourceKindRoute(k coremodel.ResourceKind) (ResourceStore, error) {
	if store, exists := sc.stores[k]; exists {
		return store, nil
	}
	return nil, fmt.Errorf("%s is not supported by store yet", k)

}
