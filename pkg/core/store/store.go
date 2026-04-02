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

package store

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	. "k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

// ResourceStore defines the interface for the persistance of a resource
// ResourceStore expanded the interface of cache.Indexer and cache.Store
type ResourceStore interface {
	Indexer
	// GetByKeys get resources by keys, return list of resource.
	// if a resource of specified key doesn't exist in the store, resource list will not include it
	GetByKeys(keys []string) ([]model.Resource, error)
	// ListByIndexes list resources by index conditions
	ListByIndexes(indexes []index.IndexCondition) ([]model.Resource, error)
	// PageListByIndexes list resources by index conditions pageable
	PageListByIndexes(indexes []index.IndexCondition, pq model.PageReq) (*model.PageData[model.Resource], error)
}

// ManagedResourceStore includes both functional interfaces and lifecycle interfaces
// If there is a new type of ResourceStore, it should implement this interface
type ManagedResourceStore interface {
	runtime.Lifecycle
	ResourceStore
}

type ResourceConflictError struct {
	rType string
	name  string
	mesh  string
	msg   string
}

func (e *ResourceConflictError) Error() string {
	return fmt.Sprintf("%s: type=%q name=%q mesh=%q", e.msg, e.rType, e.name, e.mesh)
}

func (e *ResourceConflictError) Is(err error) bool {
	return reflect.TypeOf(e) == reflect.TypeOf(err)
}

func ErrorResourceAlreadyExists(rt, name, mesh string) error {
	return &ResourceConflictError{msg: "resource already exists", rType: rt, name: name, mesh: mesh}
}

func ErrorResourceConflict(rt, name, mesh string) error {
	return &ResourceConflictError{msg: "resource conflict", rType: rt, name: name, mesh: mesh}
}

func ErrorResourceNotFound(rt, name, mesh string) error {
	return fmt.Errorf("resource not found: type=%q name=%q mesh=%q", rt, name, mesh)
}

var ErrorInvalidOffset = errors.New("invalid offset")

func IsResourceNotFound(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "Resource not found")
}

type PreconditionError struct {
	Reason string
}

func (a *PreconditionError) Error() string {
	return a.Reason
}

func (a *PreconditionError) Is(err error) bool {
	return reflect.TypeOf(a) == reflect.TypeOf(err)
}

func PreconditionFormatError(reason string) *PreconditionError {
	return &PreconditionError{Reason: "invalid format: " + reason}
}
