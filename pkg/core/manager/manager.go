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
	"reflect"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/core/governor"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

type ReadOnlyResourceManager interface {
	// GetByKey returns the resource with the given resource key
	GetByKey(rk model.ResourceKind, key string) (r model.Resource, exist bool, err error)
	// GetByKeys returns the resources with the given resource keys
	GetByKeys(rk model.ResourceKind, keys []string) ([]model.Resource, error)
	// ListByIndexes returns the resources with the given indexes, indexes is a map of index name and index value
	ListByIndexes(rk model.ResourceKind, indexes map[string]string) ([]model.Resource, error)
	// PageListByIndexes page list the resources with the given indexes, indexes is a map of index name and index value
	PageListByIndexes(rk model.ResourceKind, indexes map[string]string, pr model.PageReq) (*model.PageData[model.Resource], error)
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
	DeleteByKey(rk model.ResourceKind, mesh string, key string) error
}

type ResourceManager interface {
	ReadOnlyResourceManager

	WriteOnlyResourceManager
}

var _ ResourceManager = &resourcesManager{}

type resourcesManager struct {
	storeRouter    store.Router
	governorRouter governor.Router
}

func NewResourceManager(router store.Router, governorRouter governor.Router) ResourceManager {
	return &resourcesManager{
		storeRouter:    router,
		governorRouter: governorRouter,
	}
}

func (rm *resourcesManager) GetByKey(rk model.ResourceKind, key string) (r model.Resource, exist bool, err error) {
	rs, err := rm.storeRouter.ResourceKindRoute(rk)
	if err != nil {
		return nil, false, err
	}
	item, exist, err := rs.GetByKey(key)
	if !exist {
		return nil, false, nil
	}
	res, ok := item.(model.Resource)
	if !ok {
		return nil, false, bizerror.NewAssertionError("Resource", reflect.TypeOf(res).Name())
	}
	return res, exist, err
}

func (rm *resourcesManager) GetByKeys(rk model.ResourceKind, keys []string) ([]model.Resource, error) {
	rs, err := rm.storeRouter.ResourceKindRoute(rk)
	if err != nil {
		return nil, err
	}
	resources, err := rs.GetByKeys(keys)
	if err != nil {
		return nil, err
	}
	return resources, nil
}

func (rm *resourcesManager) ListByIndexes(rk model.ResourceKind, indexes map[string]string) ([]model.Resource, error) {
	rs, err := rm.storeRouter.ResourceKindRoute(rk)
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
	indexes map[string]string,
	pr model.PageReq) (*model.PageData[model.Resource], error) {

	rs, err := rm.storeRouter.ResourceKindRoute(rk)
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
	if !governor.RuleResourceKinds.Contain(r.ResourceKind()) {
		return bizerror.New(bizerror.InvalidArgument, "invalid resource kind")
	}
	rs, err := rm.governorRouter.ResourceRoute(r)
	if err != nil {
		return err
	}
	return rs.CreateRule(r)
}

func (rm *resourcesManager) Update(r model.Resource) error {
	if !governor.RuleResourceKinds.Contain(r.ResourceKind()) {
		return bizerror.New(bizerror.InvalidArgument, "invalid resource kind")
	}
	rs, err := rm.governorRouter.ResourceRoute(r)
	if err != nil {
		return err
	}
	return rs.UpdateRule(r)
}

func (rm *resourcesManager) Upsert(r model.Resource) error {
	if !governor.RuleResourceKinds.Contain(r.ResourceKind()) {
		return bizerror.New(bizerror.InvalidArgument, "invalid resource kind")
	}
	if _, exists, _ := rm.GetByKey(r.ResourceKind(), r.ResourceKey()); exists {
		return rm.Update(r)
	} else {
		return rm.Add(r)
	}
}

func (rm *resourcesManager) DeleteByKey(rk model.ResourceKind, mesh string, key string) error {
	if !governor.RuleResourceKinds.Contain(rk) {
		return bizerror.New(bizerror.InvalidArgument, "invalid resource kind")
	}
	gov, err := rm.governorRouter.ResourceMeshRoute(mesh)
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
	return gov.DeleteRule(r)
}
