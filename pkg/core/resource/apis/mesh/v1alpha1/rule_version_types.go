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
	"google.golang.org/protobuf/proto"
)

const (
	RuleVersionKind coremodel.ResourceKind = "RuleVersion"
)

func init() {
	coremodel.RegisterResourceSchema(RuleVersionKind, NewRuleVersionResource, NewRuleVersionResourceList)
}

// RuleVersionResource stores one immutable history entry for a parent traffic
// rule. The live rule state is always read from ResourceManager.
type RuleVersionResource struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Mesh is the name of the dubbo mesh this resource belongs to.
	Mesh string `json:"mesh,omitempty"`

	// Spec is the specification of the Dubbo RuleVersion resource.
	Spec *meshproto.RuleVersion `json:"spec,omitempty"`
}

func (r *RuleVersionResource) ResourceKind() coremodel.ResourceKind {
	return RuleVersionKind
}

func (r *RuleVersionResource) ResourceMesh() string {
	return r.Mesh
}

func (r *RuleVersionResource) ResourceKey() string {
	return coremodel.BuildResourceKey(r.Mesh, r.Name)
}

func (r *RuleVersionResource) ResourceMeta() metav1.ObjectMeta {
	return r.ObjectMeta
}

func (r *RuleVersionResource) ResourceSpec() coremodel.ResourceSpec {
	return r.Spec
}

func (r *RuleVersionResource) String() string {
	jsonStr, err := json.Marshal(r)
	if err != nil {
		logger.Errorf("failed to encode RuleVersionResource: %s to json, err: %v", r.ResourceKey(), err)
		return ""
	}
	return string(jsonStr)
}

func (r *RuleVersionResource) DeepCopyObject() k8sruntime.Object {
	out := &RuleVersionResource{
		TypeMeta:   r.TypeMeta,
		ObjectMeta: *r.ObjectMeta.DeepCopy(),
		Mesh:       r.Mesh,
	}
	if r.Spec != nil {
		out.Spec = proto.Clone(r.Spec).(*meshproto.RuleVersion)
	}
	return out
}

func NewRuleVersionResource() coremodel.Resource {
	return &RuleVersionResource{}
}

func NewRuleVersionResourceWithAttributes(name, mesh string) *RuleVersionResource {
	return &RuleVersionResource{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Mesh:       mesh,
		Spec:       &meshproto.RuleVersion{},
	}
}

type RuleVersionResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []*RuleVersionResource `json:"items"`
}

func (r *RuleVersionResourceList) DeepCopyObject() k8sruntime.Object {
	out := &RuleVersionResourceList{
		TypeMeta: r.TypeMeta,
	}
	r.ListMeta.DeepCopyInto(&out.ListMeta)

	if len(r.Items) == 0 {
		return out
	}
	out.Items = make([]*RuleVersionResource, len(r.Items))
	for i := range r.Items {
		out.Items[i] = r.Items[i].DeepCopyObject().(*RuleVersionResource)
	}
	return out
}

func NewRuleVersionResourceList() coremodel.ResourceList {
	return &RuleVersionResourceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       string(RuleVersionKind),
			APIVersion: "v1alpha1",
		},
		Items: make([]*RuleVersionResource, 0),
	}
}

func (r *RuleVersionResourceList) GetItems() []coremodel.Resource {
	res := make([]coremodel.Resource, len(r.Items))
	for i := range r.Items {
		res[i] = r.Items[i]
	}
	return res
}

func (r *RuleVersionResourceList) SetItems(items []coremodel.Resource) {
	r.Items = make([]*RuleVersionResource, len(items))
	for i, res := range items {
		if typed, ok := res.(*RuleVersionResource); ok {
			r.Items[i] = typed
		}
	}
}
