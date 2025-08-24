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
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
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

// ResourceKind defines the resource type
type ResourceKind string

func (rk ResourceKind) ToString() string {
	return string(rk)
}

type Resource interface {
	k8sruntime.Object
	// ResourceKind returns the resource type, e.g. Application, Service etc.
	ResourceKind() ResourceKind
	// ResourceKey returns the unique resource key
	ResourceKey() string
	// MeshName returns the mesh which the resource belongs to
	MeshName() string
	// ResourceMeta returns the resource metadata
	ResourceMeta() metav1.ObjectMeta
	// ResourceSpec returns the resource spec
	ResourceSpec() ResourceSpec
}

// BuildResourceKey build a unique identifier for a resource, usually is `mesh/kind/name`
func BuildResourceKey(mesh string, name string) string {
	return mesh + separator + name
}

func ErrorInvalidItemType(expected, actual interface{}) error {
	return fmt.Errorf("invalid argument type: expected=%q got=%q", reflect.TypeOf(expected), reflect.TypeOf(actual))
}
