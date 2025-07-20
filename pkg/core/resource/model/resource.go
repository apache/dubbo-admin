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

package model

import (
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultMesh = "default"
	// NoMesh defines a marker that resource is not bound to a Mesh.
	// Resources not bound to a mesh (ScopeGlobal) should have an empty string in Mesh field.
	NoMesh = ""
)

const separator = "/"

const (
	ExtensionsImageKey                 = "image"
	ExtensionsPodPhaseKey              = "podPhase"
	ExtensionsPodStatusKey             = "podStatus"
	ExtensionsContainerStatusReasonKey = "containerStatus"
	ExtensionApplicationNameKey        = "applicationName" // For universial mode
	ExtensionsWorkLoadKey              = "workLoad"
	ExtensionsNodeNameKey              = "nodeName"
)



type ResourceSpec interface{}

type Resource interface {
	// GetKind returns the resource type, e.g. Application, Service etc.
	GetKind() string
	GetMesh()  string
	GetResourceKey() string
	GetMeta() metav1.ObjectMeta
	SetMeta(metav1.ObjectMeta)
	GetSpec() ResourceSpec
	SetSpec(ResourceSpec) error
}

// BuildResourceKey build a unique identifier for a resource, usually is `mesh/kind/name`
func BuildResourceKey(mesh string, kind string, name string) string {
	return mesh + separator +  kind + separator + name;
}

type PageQuery struct {
	PageSize    uint32
	CurrentPage uint32
	Page        bool
}

type Pagination struct {
	PageSize    uint32
	CurrentPage uint32
	Total       uint32
	Page        bool
	NextOffset  string
}

func ErrorInvalidItemType(expected, actual interface{}) error {
	return fmt.Errorf("invalid argument type: expected=%q got=%q", reflect.TypeOf(expected), reflect.TypeOf(actual))
}

