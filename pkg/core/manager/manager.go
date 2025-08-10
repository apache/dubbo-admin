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

package manager

import (
	"errors"
	"fmt"
	"time"

	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/duke-git/lancet/v2/slice"
)

type ReadOnlyResourceManager interface {
	// GetByKey returns the resource with the given resource key
	GetByKey(rk model.ResourceKind, key string) (r model.Resource, exist bool, err error)
	// ListByIndex returns the resource with the given index name
	ListByIndex(rk model.ResourceKind, indexName string, indexKey interface{}) ([]model.Resource, error)
	// ListPageByIndex page list the resources with the given index
	ListPageByIndex(rk model.ResourceKind, indexName string,
		indexValue interface{}, pq model.PageQuery) ([]model.Resource, model.Pagination, error)
}

type WriteOnlyResourceManager interface {
	// Add adds the resource
	Add(r model.Resource) error
	// Update updates the resource
	Update(r model.Resource) error
	// Upsert upserts the resource
	Upsert(r model.Resource) error
	// DeleteByKey deletes the resource with the given resource key
	DeleteByKey(rk model.ResourceKind, key string) error
}

type ResourceManager interface {
	ReadOnlyResourceManager

	WriteOnlyResourceManager
}

func NewResourceManager(router store.Router) ResourceManager {
	return &resourcesManager{
		StoreRouter: router,
	}
}

var _ ResourceManager = &resourcesManager{}

type resourcesManager struct {
	StoreRouter store.Router
}

func (rm *resourcesManager) GetByKey(rk model.ResourceKind, key string) (r model.Resource, exist bool, err error) {
	rs, err := rm.StoreRouter.ResourceKindRoute(rk)
	if err != nil {
		return nil, false, err
	}
	item, exist, err := rs.GetByKey(key)
	return item.(model.Resource), exist, err
}

func (rm *resourcesManager) ListByIndex(rk model.ResourceKind, indexName string, indexKey interface{}) ([]model.Resource, error) {
	rs, err := rm.StoreRouter.ResourceKindRoute(rk)
	if err != nil {
		return nil, err
	}
	objList, err := rs.Index(indexName, indexKey)
	if err != nil {
		return nil, err
	}
	resources := slice.Map(objList, func(_ int, item interface{}) model.Resource {
		return item.(model.Resource)
	})
	return resources, nil
}

func (rm *resourcesManager) ListPageByIndex(
	rk model.ResourceKind,
	indexName string,
	indexValue interface{},
	pageQuery model.PageQuery) ([]model.Resource, model.Pagination, error) {
	rs, err := rm.StoreRouter.ResourceKindRoute(rk)
	if err != nil {
		return nil, model.Pagination{}, err
	}
	items, p, err := rs.ListPageByIndex(indexName, indexValue, pageQuery)
	if err != nil {
		return nil, p, err
	}
	rsList := slice.Map(items, func(_ int, item interface{}) model.Resource {
		return item.(model.Resource)
	})
	return rsList, p, nil
}

func (rm *resourcesManager) Add(r model.Resource) error {
	rs, err := rm.StoreRouter.ResourceRoute(r)
	if err != nil {
		return err
	}
	return rs.Add(r)
}

func (rm *resourcesManager) Update(r model.Resource) error {
	rs, err := rm.StoreRouter.ResourceRoute(r)
	if err != nil {
		return err
	}
	return rs.Update(r)
}

func (rm *resourcesManager) Upsert(r model.Resource) error {
	if _, exists, _ := rm.GetByKey(r.ResourceKind(), r.ResourceKey()); exists {
		return rm.Update(r)
	} else {
		return rm.Add(r)
	}
}

func (rm *resourcesManager) DeleteByKey(rk model.ResourceKind, key string) error {
	rs, err := rm.StoreRouter.ResourceKindRoute(rk)
	if err != nil {
		return err
	}
	r, exists, err := rm.GetByKey(rk, key)
	if err != nil {
		return fmt.Errorf("failed to get resource %s: %s", key, err)
	}
	if !exists {
		return fmt.Errorf("%s %s does not exist", rk, key)
	}
	return rs.Delete(r)
}

type ConflictRetry struct {
	BaseBackoff   time.Duration
	MaxTimes      uint
	JitterPercent uint
}

type UpsertOpts struct {
	ConflictRetry ConflictRetry
	Transactions  store.Transactions
}

type UpsertFunc func(opts *UpsertOpts)

func WithConflictRetry(baseBackoff time.Duration, maxTimes uint, jitterPercent uint) UpsertFunc {
	return func(opts *UpsertOpts) {
		opts.ConflictRetry.BaseBackoff = baseBackoff
		opts.ConflictRetry.MaxTimes = maxTimes
		opts.ConflictRetry.JitterPercent = jitterPercent
	}
}

func WithTransactions(transactions store.Transactions) UpsertFunc {
	return func(opts *UpsertOpts) {
		opts.Transactions = transactions
	}
}

func NewUpsertOpts(fs ...UpsertFunc) UpsertOpts {
	opts := UpsertOpts{
		Transactions: store.NoTransactions{},
	}
	for _, f := range fs {
		f(&opts)
	}
	return opts
}

type MeshNotFoundError struct {
	Mesh string
}

func (m *MeshNotFoundError) Error() string {
	return fmt.Sprintf("mesh of name %s is not found", m.Mesh)
}

func MeshNotFound(meshName string) error {
	return &MeshNotFoundError{meshName}
}

func IsMeshNotFound(err error) bool {
	var meshNotFoundError *MeshNotFoundError
	ok := errors.As(err, &meshNotFoundError)
	return ok
}
