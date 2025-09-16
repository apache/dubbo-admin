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
	"fmt"

	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

type ReadOnlyResourceManager interface {
	// GetByKey returns the resource with the given resource key
	GetByKey(rk model.ResourceKind, key string) (r model.Resource, exist bool, err error)
	// ListByIndexes returns the resources with the given indexes, indexes is a map of index name and index value
	ListByIndexes(rk model.ResourceKind, indexes map[string]interface{}) ([]model.Resource, error)
	// PageListByIndexes page list the resources with the given indexes, indexes is a map of index name and index value
	PageListByIndexes(rk model.ResourceKind, indexes map[string]interface{}, pr model.PageReq) (*model.PageData[model.Resource], error)
	// PageSearchResourceByConditions page fuzzy search resource by conditions, conditions cannot be empty
	// TODO support multiple conditions
	PageSearchResourceByConditions(rk model.ResourceKind, conditions []string, pr model.PageReq) (*model.PageData[model.Resource], error)
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

func (rm *resourcesManager) ListByIndexes(rk model.ResourceKind, indexes map[string]interface{}) ([]model.Resource, error) {
	rs, err := rm.StoreRouter.ResourceKindRoute(rk)
	if err != nil {
		return nil, err
	}
	resources, err := rs.ListByIndexes(indexes)
	if err != nil {
		return nil, err
	}
	return resources, nil
}

func (rm *resourcesManager) PageListByIndexes(
	rk model.ResourceKind,
	indexes map[string]interface{},
	pr model.PageReq) (*model.PageData[model.Resource], error) {

	rs, err := rm.StoreRouter.ResourceKindRoute(rk)
	if err != nil {
		return nil, err
	}
	pageData, err := rs.PageListByIndexes(indexes, pr)
	if err != nil {
		return nil, err
	}
	return pageData, nil
}

func (rm *resourcesManager) PageSearchResourceByConditions(rk model.ResourceKind, conditions []string, pr model.PageReq) (*model.PageData[model.Resource], error) {
	//TODO implement me
	panic("implement me")
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
