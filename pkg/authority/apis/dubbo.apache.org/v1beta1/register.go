// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AuthenticationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AuthenticationPolicySpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AuthenticationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AuthenticationPolicy `json:"items"`
}

type AuthenticationPolicySpec struct {
	Action    string                     `json:"action,omitempty"`
	Rules     []AuthenticationPolicyRule `json:"rules,omitempty"`
	Order     int                        `json:"order,omitempty"`
	MatchType string                     `json:"matchType,omitempty"`
}

type AuthenticationPolicyRule struct {
	From AuthenticationPolicySource `json:"from,omitempty"`
	To   AuthenticationPolicyTarget `json:"to,omitempty"`
}

type AuthenticationPolicySource struct {
	Namespaces    []string                     `json:"namespaces,omitempty"`
	NotNamespaces []string                     `json:"notNamespaces,omitempty"`
	IpBlocks      []string                     `json:"ipBlocks,omitempty"`
	NotIpBlocks   []string                     `json:"notIpBlocks,omitempty"`
	Principals    []string                     `json:"principals,omitempty"`
	NotPrincipals []string                     `json:"notPrincipals,omitempty"`
	Extends       []AuthenticationPolicyExtend `json:"extends,omitempty"`
	NotExtends    []AuthenticationPolicyExtend `json:"notExtends,omitempty"`
}

type AuthenticationPolicyTarget struct {
	IpBlocks      []string                     `json:"ipBlocks,omitempty"`
	NotIpBlocks   []string                     `json:"notIpBlocks,omitempty"`
	Principals    []string                     `json:"principals,omitempty"`
	NotPrincipals []string                     `json:"notPrincipals,omitempty"`
	Extends       []AuthenticationPolicyExtend `json:"extends,omitempty"`
	NotExtends    []AuthenticationPolicyExtend `json:"notExtends,omitempty"`
}

type AuthenticationPolicyExtend struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AuthorizationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AuthorizationPolicySpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AuthorizationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AuthorizationPolicy `json:"items"`
}

type AuthorizationPolicySpec struct {
	Action    string                    `json:"action,omitempty"`
	Rules     []AuthorizationPolicyRule `json:"rules,omitempty"`
	Samples   float32                   `json:"samples,omitempty"`
	MatchType string                    `json:"matchType,omitempty"`
}

type AuthorizationPolicyRule struct {
	From AuthorizationPolicySource    `json:"from,omitempty"`
	To   AuthorizationPolicyTarget    `json:"to,omitempty"`
	When AuthorizationPolicyCondition `json:"when,omitempty"`
}

type AuthorizationPolicySource struct {
	Namespaces    []string                    `json:"namespaces,omitempty"`
	NotNamespaces []string                    `json:"notNamespaces,omitempty"`
	IpBlocks      []string                    `json:"ipBlocks,omitempty"`
	NotIpBlocks   []string                    `json:"notIpBlocks,omitempty"`
	Principals    []string                    `json:"principals,omitempty"`
	NotPrincipals []string                    `json:"notPrincipals,omitempty"`
	Extends       []AuthorizationPolicyExtend `json:"extends,omitempty"`
	NotExtends    []AuthorizationPolicyExtend `json:"notExtends,omitempty"`
}

type AuthorizationPolicyTarget struct {
	IpBlocks      []string                    `json:"ipBlocks,omitempty"`
	NotIpBlocks   []string                    `json:"notIpBlocks,omitempty"`
	Principals    []string                    `json:"principals,omitempty"`
	NotPrincipals []string                    `json:"notPrincipals,omitempty"`
	Extends       []AuthorizationPolicyExtend `json:"extends,omitempty"`
	NotExtends    []AuthorizationPolicyExtend `json:"notExtends,omitempty"`
}

type AuthorizationPolicyCondition struct {
	Key       string                     `json:"key,omitempty"`
	Values    []AuthorizationPolicyMatch `json:"values,omitempty"`
	NotValues []AuthorizationPolicyMatch `json:"notValues,omitempty"`
}

type AuthorizationPolicyMatch struct {
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}

type AuthorizationPolicyExtend struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}
