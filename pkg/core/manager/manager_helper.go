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
	"reflect"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// GetByKey is a helper function of ResourceManager.GeyByKey
func GetByKey[T model.Resource](rm ReadOnlyResourceManager, rk model.ResourceKind, key string) (r T, exist bool, err error) {
	resource, exist, err := rm.GetByKey(rk, key)
	if err != nil || !exist {
		var zero T
		return zero, exist, err
	}

	typedResource, ok := resource.(T)
	if !ok {
		var zero T
		return zero, false, bizerror.NewAssertionError(rk, reflect.TypeOf(typedResource).Name())
	}
	return typedResource, true, nil
}

// ListByIndexes is a helper function of ResourceManager.ListByIndexes
func ListByIndexes[T model.Resource](rm ReadOnlyResourceManager, rk model.ResourceKind, indexes map[string]string) ([]T, error) {
	resources, err := rm.ListByIndexes(rk, indexes)
	if err != nil {
		return nil, err
	}

	typedResources := make([]T, len(resources))
	for i, resource := range resources {
		typedResource, ok := resource.(T)
		if !ok {
			return nil, bizerror.NewAssertionError(rk, reflect.TypeOf(typedResource).Name())
		}
		typedResources[i] = typedResource
	}

	return typedResources, nil
}

// PageListByIndexes is a helper function of ResourceManager.PageListByIndexes
func PageListByIndexes[T model.Resource](
	rm ReadOnlyResourceManager,
	rk model.ResourceKind,
	indexes map[string]string,
	pr model.PageReq) (*model.PageData[T], error) {

	pageData, err := rm.PageListByIndexes(rk, indexes, pr)
	if err != nil {
		return nil, err
	}

	typedResources := make([]T, len(pageData.Data))
	for i, resource := range pageData.Data {
		typedResource, ok := resource.(T)
		if !ok {
			return nil, bizerror.NewAssertionError(rk, reflect.TypeOf(typedResource).Name())
		}
		typedResources[i] = typedResource
	}
	newPageData := &model.PageData[T]{
		Pagination: model.Pagination{
			Total:      pageData.Total,
			PageOffset: pageData.PageOffset,
			PageSize:   pageData.PageSize,
		},
		Data: typedResources,
	}
	return newPageData, nil
}

// PageSearchResourceByConditions is a helper function of ResourceManager.PageSearchResourceByConditions
func PageSearchResourceByConditions[T model.Resource](
	rm ReadOnlyResourceManager,
	rk model.ResourceKind,
	conditions []string,
	pr model.PageReq) (*model.PageData[T], error) {
	pageData, err := rm.PageSearchResourceByConditions(rk, conditions, pr)
	if err != nil {
		return nil, err
	}

	typedResources := make([]T, len(pageData.Data))
	for i, resource := range pageData.Data {
		typedResource, ok := resource.(T)
		if !ok {
			return nil, bizerror.NewAssertionError(rk, reflect.TypeOf(typedResource).Name())
		}
		typedResources[i] = typedResource
	}
	newPageData := &model.PageData[T]{
		Pagination: model.Pagination{
			Total:      pageData.Total,
			PageOffset: pageData.PageOffset,
			PageSize:   pageData.PageSize,
		},
		Data: typedResources,
	}
	return newPageData, nil
}
