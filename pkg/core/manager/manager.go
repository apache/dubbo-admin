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

	"github.com/duke-git/lancet/v2/slice"

	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

type ReadOnlyResourceManager interface {
	// GetByKey returns the resource with the given resource key
	GetByKey(rk string) (r model.Resource, exist bool, err error)
	// ListPageByKey page list the resources with the given resource key
	ListPageByKey(rk string, pq model.PageQuery) ([]model.Resource, model.Pagination)
}

type WriteOnlyResourceManager interface {
	// Add adds the resource
	Add(r model.Resource) error
	// Update updates the resource
	Update(r model.Resource) error
	// Upsert upserts the resource
	Upsert(r model.Resource) error
	// DeleteByKey deletes the resource with the given resource key
	DeleteByKey(key string) error
}

type ResourceManager interface {
	ReadOnlyResourceManager

	WriteOnlyResourceManager
}

func NewResourceManager(store store.ResourceStore) ResourceManager {
	return &resourcesManager{
		Store: store,
	}
}

var _ ResourceManager = &resourcesManager{}

type resourcesManager struct {
	Store store.ResourceStore
}

func (rm *resourcesManager) GetByKey(key string) (r model.Resource, exist bool, err error) {
	item, exist, err := rm.Store.GetByKey(key)
	return item.(model.Resource), exist, err
}

func (rm *resourcesManager) ListPageByKey(key string, pageQuery model.PageQuery) ([]model.Resource, model.Pagination) {
	items, p := rm.Store.ListPageByKey(key, pageQuery)
	rs := slice.Map(items, func(_ int, item interface{}) model.Resource {
		return item.(model.Resource)
	})
	return rs, p
}

func (rm *resourcesManager) Add(r model.Resource) error {
	return rm.Store.Add(r)
}

func (rm *resourcesManager) Update(r model.Resource) error {
	return rm.Store.Update(r)
}

func (rm *resourcesManager) Upsert(r model.Resource) error {
	if _, exists, _ := rm.Store.GetByKey(r.GetResourceKey()); exists {
		return rm.Update(r)
	} else {
		return rm.Add(r)
	}
}

func (rm *resourcesManager) DeleteByKey(key string) error {
	r, exists, err := rm.Store.GetByKey(key)
	if exists && err == nil {
		return rm.Store.Delete(r)
	}
	if err != nil {
		return fmt.Errorf("failed to get resource %s: %s", key, err)
	}
	return fmt.Errorf("resource %s does not exist", key)
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
