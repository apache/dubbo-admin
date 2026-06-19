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
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"google.golang.org/protobuf/proto"
)

// RuleIntentResource represents a pending mutation to a rule
type RuleIntentResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Mesh              string                `json:"mesh,omitempty"`
	Spec              *meshproto.RuleIntent `json:"spec,omitempty"`
}

func (r *RuleIntentResource) ResourceKind() model.ResourceKind {
	return RuleIntentKind
}

func (r *RuleIntentResource) ResourceMesh() string {
	return r.Mesh
}

func (r *RuleIntentResource) ResourceMeta() metav1.ObjectMeta {
	return r.ObjectMeta
}

func (r *RuleIntentResource) ResourceSpec() model.ResourceSpec {
	return r.Spec
}

func (r *RuleIntentResource) ResourceKey() string {
	return model.BuildResourceKey(r.Mesh, r.Name)
}

func (r *RuleIntentResource) String() string {
	jsonStr, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	return string(jsonStr)
}

func (r *RuleIntentResource) DeepCopyObject() k8sruntime.Object {
	out := &RuleIntentResource{
		TypeMeta:   r.TypeMeta,
		ObjectMeta: *r.ObjectMeta.DeepCopy(),
		Mesh:       r.Mesh,
	}
	if r.Spec != nil {
		out.Spec = proto.Clone(r.Spec).(*meshproto.RuleIntent)
	}
	return out
}

// RuleIntentResourceList contains a list of RuleIntentResource
type RuleIntentResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RuleIntentResource `json:"items"`
}

func (r *RuleIntentResourceList) DeepCopyObject() k8sruntime.Object {
	out := &RuleIntentResourceList{
		TypeMeta: r.TypeMeta,
	}
	r.ListMeta.DeepCopyInto(&out.ListMeta)
	if r.Items != nil {
		out.Items = make([]RuleIntentResource, len(r.Items))
		for i := range r.Items {
			out.Items[i] = *r.Items[i].DeepCopyObject().(*RuleIntentResource)
		}
	}
	return out
}

func (r *RuleIntentResourceList) SetItems(items []model.Resource) {
	r.Items = make([]RuleIntentResource, len(items))
	for i, res := range items {
		if typed, ok := res.(*RuleIntentResource); ok {
			r.Items[i] = *typed
		}
	}
}

// RuleMetaResource tracks the current state of a rule for version control
type RuleMetaResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Mesh              string              `json:"mesh,omitempty"`
	Spec              *meshproto.RuleMeta `json:"spec,omitempty"`
}

func (r *RuleMetaResource) ResourceKind() model.ResourceKind {
	return RuleMetaKind
}

func (r *RuleMetaResource) ResourceMesh() string {
	return r.Mesh
}

func (r *RuleMetaResource) ResourceMeta() metav1.ObjectMeta {
	return r.ObjectMeta
}

func (r *RuleMetaResource) ResourceSpec() model.ResourceSpec {
	return r.Spec
}

func (r *RuleMetaResource) ResourceKey() string {
	return model.BuildResourceKey(r.Mesh, r.Name)
}

func (r *RuleMetaResource) String() string {
	jsonStr, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	return string(jsonStr)
}

func (r *RuleMetaResource) DeepCopyObject() k8sruntime.Object {
	out := &RuleMetaResource{
		TypeMeta:   r.TypeMeta,
		ObjectMeta: *r.ObjectMeta.DeepCopy(),
		Mesh:       r.Mesh,
	}
	if r.Spec != nil {
		out.Spec = proto.Clone(r.Spec).(*meshproto.RuleMeta)
	}
	return out
}

// RuleMetaResourceList contains a list of RuleMetaResource
type RuleMetaResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RuleMetaResource `json:"items"`
}

func (r *RuleMetaResourceList) DeepCopyObject() k8sruntime.Object {
	out := &RuleMetaResourceList{
		TypeMeta: r.TypeMeta,
	}
	r.ListMeta.DeepCopyInto(&out.ListMeta)
	if r.Items != nil {
		out.Items = make([]RuleMetaResource, len(r.Items))
		for i := range r.Items {
			out.Items[i] = *r.Items[i].DeepCopyObject().(*RuleMetaResource)
		}
	}
	return out
}

func (r *RuleMetaResourceList) SetItems(items []model.Resource) {
	r.Items = make([]RuleMetaResource, len(items))
	for i, res := range items {
		if typed, ok := res.(*RuleMetaResource); ok {
			r.Items[i] = *typed
		}
	}
}

// Resource kind constants
const (
	RuleIntentKind model.ResourceKind = "RuleIntent"
	RuleMetaKind   model.ResourceKind = "RuleMeta"
)

// NewRuleIntentResource creates a new RuleIntentResource with given name and mesh
func NewRuleIntentResource() *RuleIntentResource {
	return &RuleIntentResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1alpha1",
			Kind:       string(RuleIntentKind),
		},
	}
}

func NewRuleIntentResourceWithAttributes(name, mesh string) *RuleIntentResource {
	r := NewRuleIntentResource()
	r.Name = name
	r.Mesh = mesh
	return r
}

// NewRuleMetaResource creates a new RuleMetaResource
func NewRuleMetaResource() *RuleMetaResource {
	return &RuleMetaResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1alpha1",
			Kind:       string(RuleMetaKind),
		},
	}
}

func NewRuleMetaResourceWithAttributes(name, mesh string) *RuleMetaResource {
	r := NewRuleMetaResource()
	r.Name = name
	r.Mesh = mesh
	return r
}

func init() {
	// Register RuleIntent
	model.RegisterResourceSchema(RuleIntentKind, func() model.Resource {
		return NewRuleIntentResource()
	}, func() model.ResourceList {
		return &RuleIntentResourceList{}
	})

	// Register RuleMeta
	model.RegisterResourceSchema(RuleMetaKind, func() model.Resource {
		return NewRuleMetaResource()
	}, func() model.ResourceList {
		return &RuleMetaResourceList{}
	})
}
