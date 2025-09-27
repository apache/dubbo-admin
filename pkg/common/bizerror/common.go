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

package bizerror

import (
	"errors"
	"fmt"
)

type AssertionError struct {
	msg string
}

func NewAssertionError(expected, actual interface{}) error {
	return &AssertionError{
		msg: fmt.Sprintf("type assertion error, expected:%v, actual:%v", expected, actual),
	}
}

func (e *AssertionError) Error() string {
	return e.msg
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
