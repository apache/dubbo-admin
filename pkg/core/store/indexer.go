package store

import (
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"k8s.io/client-go/tools/cache"
)

var indexRegistry = newIndexRegistry()

func RegisterIndexers(rk model.ResourceKind, indexers cache.Indexers) {
	indexRegistry.Register(rk, indexers)
}

func IndexersRegistry() IndexerRegistry {
	return indexRegistry
}

type IndexerRegistry interface {
	Indexers(model.ResourceKind) cache.Indexers
}

type IndexerRegistryMutator interface {
	Register(model.ResourceKind, cache.Indexers)
}

type MutableIndexerRegistry interface {
	IndexerRegistry
	IndexerRegistryMutator
}

// ResourceIndexers defines the rIndexers belong to a specific model.Resource
type ResourceIndexers struct {
	rk       model.ResourceKind
	indexers cache.Indexers
}

var _ IndexerRegistry = &indexerRegistry{}

type indexerRegistry struct {
	rIndexers []ResourceIndexers
}

func newIndexRegistry() MutableIndexerRegistry {
	return &indexerRegistry{
		rIndexers: make([]ResourceIndexers, 0),
	}
}

func (i *indexerRegistry) Indexers(k model.ResourceKind) cache.Indexers {
	for _, rIndexer := range i.rIndexers {
		if rIndexer.rk == k {
			return rIndexer.indexers
		}
	}
	return nil
}

func (i *indexerRegistry) Register(k model.ResourceKind, indexers cache.Indexers) {
	i.rIndexers = append(i.rIndexers, ResourceIndexers{rk: k, indexers: indexers})
}
