package model

import ds "github.com/duke-git/lancet/v2/datastructure/set"

var global = newRegistry()

func ResourceKindRegistry() Registry {
	return global
}

func RegisterResourceKind(kind ResourceKind) {
	global.Register(kind)
}

type Registry interface {
	// AllResourceKinds returns all registered resource kinds
	AllResourceKinds() []ResourceKind
}

type RegistryMutator interface {
	// Register registers a resource kind, if a resource kind has been registered before, it will be ignored
	Register(kind ResourceKind)
}

type MutableRegistry interface {
	Registry
	RegistryMutator
}

var _ MutableRegistry = &resourceKindRegistry{}

func newRegistry() MutableRegistry {
	return &resourceKindRegistry{
		kinds: ds.New[ResourceKind](),
	}
}

type resourceKindRegistry struct {
	kinds ds.Set[ResourceKind]
}

func (r *resourceKindRegistry) AllResourceKinds() []ResourceKind {
	return r.kinds.ToSlice()
}

func (r *resourceKindRegistry) Register(kind ResourceKind) {
	r.kinds.Add(kind)
}
