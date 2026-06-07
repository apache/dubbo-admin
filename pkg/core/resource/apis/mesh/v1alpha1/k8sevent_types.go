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

package v1alpha1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

const K8sEventKind coremodel.ResourceKind = "K8sEvent"

func init() {
	coremodel.RegisterResourceSchema(K8sEventKind, NewK8sEventResource, NewK8sEventResourceList)
}

type K8sEventResource struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Mesh is the name of the dubbo mesh this resource belongs to.
	Mesh string `json:"mesh,omitempty"`

	// Spec is the specification of the K8sEvent resource.
	Spec *meshproto.K8sEvent `json:"spec,omitempty"`

	// Status is the status of the K8sEvent resource.
	Status K8sEventResourceStatus `json:"status,omitempty"`
}

type K8sEventResourceStatus struct{}

func (r *K8sEventResource) ResourceKind() coremodel.ResourceKind {
	return K8sEventKind
}

func (r *K8sEventResource) ResourceMesh() string {
	return r.Mesh
}

func (r *K8sEventResource) ResourceKey() string {
	return coremodel.BuildResourceKey(r.Mesh, r.Name)
}

func (r *K8sEventResource) ResourceMeta() metav1.ObjectMeta {
	return r.ObjectMeta
}

func (r *K8sEventResource) ResourceSpec() coremodel.ResourceSpec {
	return r.Spec
}

func (r *K8sEventResource) DeepCopyObject() k8sruntime.Object {
	out := &K8sEventResource{
		TypeMeta: r.TypeMeta,
		Mesh:     r.Mesh,
		Status:   r.Status,
	}

	r.ObjectMeta.DeepCopyInto(&out.ObjectMeta)

	if r.Spec != nil {
		out.Spec = r.Spec.Clone()
	}

	return out
}

func (r *K8sEventResource) String() string {
	jsonStr, err := json.Marshal(r)
	if err != nil {
		logger.Errorf("failed to encode K8sEventResource: %s to json, err: %v", r.ResourceKey(), err)
		return ""
	}
	return string(jsonStr)
}

func NewK8sEventResourceWithAttributes(name string, mesh string) *K8sEventResource {
	return &K8sEventResource{
		TypeMeta: metav1.TypeMeta{
			Kind:       string(K8sEventKind),
			APIVersion: "v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{},
		},
		Mesh: mesh,
		Spec: &meshproto.K8sEvent{},
	}
}

func NewK8sEventResource() coremodel.Resource {
	return &K8sEventResource{
		TypeMeta: metav1.TypeMeta{
			Kind:       string(K8sEventKind),
			APIVersion: "v1alpha1",
		},
		Spec: &meshproto.K8sEvent{},
	}
}

type K8sEventResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []*K8sEventResource `json:"items"`
}

func (r *K8sEventResourceList) DeepCopyObject() k8sruntime.Object {
	out := &K8sEventResourceList{
		TypeMeta: r.TypeMeta,
	}
	r.ListMeta.DeepCopyInto(&out.ListMeta)

	if len(r.Items) == 0 {
		return out
	}
	out.Items = make([]*K8sEventResource, len(r.Items))
	for i := range r.Items {
		out.Items[i] = r.Items[i].DeepCopyObject().(*K8sEventResource)
	}
	return out
}

func NewK8sEventResourceList() coremodel.ResourceList {
	return &K8sEventResourceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       string(K8sEventKind),
			APIVersion: "v1alpha1",
		},
		Items: make([]*K8sEventResource, 0),
	}
}

func (r *K8sEventResourceList) SetItems(items []coremodel.Resource) {
	r.Items = make([]*K8sEventResource, len(items))
	for i := range items {
		res, ok := items[i].(*K8sEventResource)
		if !ok {
			logger.Errorf("unexpected resource type in K8sEventResourceList.SetItems: expected %T, got %T", (*K8sEventResource)(nil), items[i])
			continue
		}
		r.Items[i] = res
	}
}
