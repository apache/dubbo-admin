/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package index

import (
	"github.com/duke-git/lancet/v2/maputil"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/core/resource/model"
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

var _ IndexerRegistry = &indexerRegistry{}

type indexerRegistry struct {
	rIndexers map[model.ResourceKind]cache.Indexers
}

func newIndexRegistry() MutableIndexerRegistry {
	return &indexerRegistry{
		rIndexers: make(map[model.ResourceKind]cache.Indexers),
	}
}

func (i *indexerRegistry) Indexers(k model.ResourceKind) cache.Indexers {
	return i.rIndexers[k]
}

func (i *indexerRegistry) Register(k model.ResourceKind, indexers cache.Indexers) {
	if existedIndexers, exists := i.rIndexers[k]; !exists {
		i.rIndexers[k] = indexers
	} else {
		i.rIndexers[k] = maputil.Merge(existedIndexers, indexers)
	}
}
