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
	"strings"

	k8smeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/apache/dubbo-admin/pkg/common/constants"
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
	// ResourceMesh returns the mesh which the resource belongs to
	ResourceMesh() string
	// ResourceMeta returns the resource metadata
	ResourceMeta() metav1.ObjectMeta
	// ResourceSpec returns the resource spec
	ResourceSpec() ResourceSpec
	// String returns the string representation of the resource
	String() string
}

type ResourceList interface {
	k8sruntime.Object
	k8smeta.List

	SetItems(items []Resource)
}

// BuildResourceKey build a unique identifier for a resource, usually is `mesh/name`
func BuildResourceKey(mesh string, name string) string {
	return mesh + constants.PathSeparator + name
}

// ParseResourceKey is the inverse of BuildResourceKey for resource keys that
// may be stored as either "mesh/name" or just "name".
func ParseResourceKey(resourceKey string) (mesh string, name string) {
	if idx := strings.Index(resourceKey, constants.PathSeparator); idx >= 0 {
		return resourceKey[:idx], resourceKey[idx+len(constants.PathSeparator):]
	}
	return "", resourceKey
}
