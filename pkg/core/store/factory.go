package store

import (
	"fmt"

	"github.com/apache/dubbo-admin/pkg/config/store"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
)

var factoryRegistry = newStoreFactoryRegistry()

func RegisterFactory(f Factory) {
	factoryRegistry.Register(f)
}

func FactoryRegistry() Registry {
	return factoryRegistry
}

// Factory is the interface for create a specific type of ManagedResourceStore
type Factory interface {
	// Support returns true if the factory supports the given type in config
	Support(store.Type) bool
	// New returns a new ManagedResourceStore for the model.ResourceKind using the given config
	New(model.ResourceKind, *store.Config) (ManagedResourceStore, error)
}

type Registry interface {
	GetStoreFactory(store.Type) (Factory, error)
}

type RegistryMutator interface {
	// RegisterFactory registers a resource store type and a func to new it
	Register(Factory)
}
type MutableRegistry interface {
	Registry
	RegistryMutator
}

var _ MutableRegistry = &storeFactoryRegistry{}

type storeFactoryRegistry struct {
	factories []Factory
}

func newStoreFactoryRegistry() MutableRegistry {
	return &storeFactoryRegistry{
		factories: make([]Factory, 0),
	}
}
func (s *storeFactoryRegistry) GetStoreFactory(t store.Type) (Factory, error) {
	for _, factory := range s.factories {
		if factory.Support(t) {
			return factory, nil
		}
	}
	return nil, fmt.Errorf("store type %s not supported", t)
}

func (s *storeFactoryRegistry) Register(factory Factory) {
	s.factories = append(s.factories, factory)
}
