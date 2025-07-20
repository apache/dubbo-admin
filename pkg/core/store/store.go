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
)

type ResourceStore interface {
	Store
	ListPageByKey(key string, pq model.PageQuery) (items []interface{}, p model.Pagination)
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

// AssertionError
type AssertionError struct {
	msg string
	err error
}

func ErrorResourceAssertion(msg, rt, name, mesh string) error {
	return &AssertionError{
		msg: fmt.Sprintf("%s: type=%q name=%q mesh=%q", msg, rt, name, mesh),
	}
}

func (e *AssertionError) Unwrap() error {
	return e.err
}

func (e *AssertionError) Error() string {
	msg := "store assertion failed"
	if e.msg != "" {
		msg += " " + e.msg
	}
	if e.err != nil {
		msg += fmt.Sprintf("error: %s", e.err)
	}
	return msg
}

func (e *AssertionError) Is(err error) bool {
	return reflect.TypeOf(e) == reflect.TypeOf(err)
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
