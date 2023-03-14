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
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=ac
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AuthenticationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec AuthenticationPolicySpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AuthenticationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AuthenticationPolicy `json:"items"`
}

type AuthenticationPolicySpec struct {
	// The action to take when a rule is matched.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Enum=NONE;CLIENT_AUTH;SERVER_AUTH
	Action string `json:"action"`
	// +optional
	Rules []AuthenticationPolicyRule `json:"rules,omitempty"`
	// The order of the rule. The rule with the highest precedence is matched first.
	// +optional
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:validation:Minimum=-2147483648
	// +kubebuilder:validation:Maximum=2147483647
	// +kubebuilder:default=0
	Order int `json:"order,omitempty"`
	// The match type of the rules.
	// +optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Enum=anyMatch;allMatch
	// +kubebuilder:default=anyMatch
	MatchType string `json:"matchType,omitempty"`
}

type AuthenticationPolicyRule struct {
	// The source of the traffic to be matched.
	// +optional
	From AuthenticationPolicySource `json:"from,omitempty"`
	// The destination of the traffic to be matched.
	// +optional
	To AuthenticationPolicyTarget `json:"to,omitempty"`
}

type AuthenticationPolicySource struct {
	// The namespaces to match of the source workload.
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`
	// The namespaces not to match of the source workload.
	// +optional
	NotNamespaces []string `json:"notNamespaces,omitempty"`
	// The IP addresses to match of the source workload.
	// +optional
	IpBlocks []string `json:"ipBlocks,omitempty"`
	// The IP addresses not to match of the source workload.
	// +optional
	NotIpBlocks []string `json:"notIpBlocks,omitempty"`
	// The identities(from spiffe) to match of the source workload.
	// +optional
	Principals []string `json:"principals,omitempty"`
	// The identities(from spiffe) not to match of the source workload.
	// +optional
	NotPrincipals []string `json:"notPrincipals,omitempty"`
	// The extended identities(from Dubbo Auth) to match of the source workload.
	// +optional
	Extends []AuthenticationPolicyExtend `json:"extends,omitempty"`
	// The extended identities(from Dubbo Auth) not to match of the source workload.
	// +optional
	NotExtends []AuthenticationPolicyExtend `json:"notExtends,omitempty"`
}

type AuthenticationPolicyTarget struct {
	// The IP addresses to match of the destination workload.
	// +optional
	IpBlocks []string `json:"ipBlocks,omitempty"`
	// The IP addresses not to match of the destination workload.
	// +optional
	NotIpBlocks []string `json:"notIpBlocks,omitempty"`
	// The identities(from spiffe) to match of the destination workload.
	// +optional
	Principals []string `json:"principals,omitempty"`
	// The identities(from spiffe) not to match of the destination workload.
	// +optional
	NotPrincipals []string `json:"notPrincipals,omitempty"`
	// The extended identities(from Dubbo Auth) to match of the destination workload.
	// +optional
	Extends []AuthenticationPolicyExtend `json:"extends,omitempty"`
	// The extended identities(from Dubbo Auth) not to match of the destination workload.
	// +optional
	NotExtends []AuthenticationPolicyExtend `json:"notExtends,omitempty"`
}

type AuthenticationPolicyExtend struct {
	// The key of the extended identity.
	// +optional
	Key string `json:"key,omitempty"`
	// The value of the extended identity.
	// +optional
	Value string `json:"value,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=az
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AuthorizationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec AuthorizationPolicySpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AuthorizationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AuthorizationPolicy `json:"items"`
}

type AuthorizationPolicySpec struct {
	// The action to take when a rule is matched
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Enum=ALLOW;DENY;ADUIT
	Action string `json:"action"`
	// +optional
	Rules []AuthorizationPolicyRule `json:"rules,omitempty"`
	// The sample rate of the rule. The value is between 0 and 100.
	// +optional
	// +kubebuilder:validation:Type=number
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=100
	Samples float32 `json:"samples,omitempty"`
	// The match type of the rules.
	// +optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Enum=anyMatch;allMatch
	// +kubebuilder:default=anyMatch
	MatchType string `json:"matchType,omitempty"`
}

type AuthorizationPolicyRule struct {
	// The source of the traffic to be matched.
	// +optional
	From AuthorizationPolicySource `json:"from,omitempty"`
	// The destination of the traffic to be matched.
	// +optional
	To AuthorizationPolicyTarget `json:"to,omitempty"`
	// +optional
	When AuthorizationPolicyCondition `json:"when,omitempty"`
}

type AuthorizationPolicySource struct {
	// The namespaces to match of the source workload.
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`
	// The namespaces not to match of the source workload.
	// +optional
	NotNamespaces []string `json:"notNamespaces,omitempty"`
	// The IP addresses to match of the source workload.
	// +optional
	IpBlocks []string `json:"ipBlocks,omitempty"`
	// The IP addresses not to match of the source workload.
	// +optional
	NotIpBlocks []string `json:"notIpBlocks,omitempty"`
	// The identities(from spiffe) to match of the source workload.
	// +optional
	Principals []string `json:"principals,omitempty"`
	// The identities(from spiffe) not to match of the source workload
	// +optional
	NotPrincipals []string `json:"notPrincipals,omitempty"`
	// The extended identities(from Dubbo Auth) to match of the source workload.
	// +optional
	Extends []AuthorizationPolicyExtend `json:"extends,omitempty"`
	// The extended identities(from Dubbo Auth) not to match of the source workload.
	// +optional
	NotExtends []AuthorizationPolicyExtend `json:"notExtends,omitempty"`
}

type AuthorizationPolicyTarget struct {
	// The IP addresses to match of the destination workload.
	// +optional
	IpBlocks []string `json:"ipBlocks,omitempty"`
	// The IP addresses not to match of the destination workload.
	// +optional
	NotIpBlocks []string `json:"notIpBlocks,omitempty"`
	// The identities(from spiffe) to match of the destination workload.
	// +optional
	Principals []string `json:"principals,omitempty"`
	// The identities(from spiffe) not to match of the destination workload.
	// +optional
	NotPrincipals []string `json:"notPrincipals,omitempty"`
	// The extended identities(from Dubbo Auth) to match of the destination workload.
	// +optional
	Extends []AuthorizationPolicyExtend `json:"extends,omitempty"`
	// The extended identities(from Dubbo Auth) not to match of the destination workload.
	// +optional
	NotExtends []AuthorizationPolicyExtend `json:"notExtends,omitempty"`
}

type AuthorizationPolicyCondition struct {
	// +optional
	Key string `json:"key,omitempty"`
	// +optional
	Values []AuthorizationPolicyMatch `json:"values,omitempty"`
	// +optional
	NotValues []AuthorizationPolicyMatch `json:"notValues,omitempty"`
}

type AuthorizationPolicyMatch struct {
	// +optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Enum=equals;regex;ognl
	// +kubebuilder:default=equals
	Type string `json:"type,omitempty"`
	// +optional
	Value string `json:"value,omitempty"`
}

type AuthorizationPolicyExtend struct {
	// The key of the extended identity.
	// +optional
	Key string `json:"key,omitempty"`
	// The value of the extended identity
	// +optional
	Value string `json:"value,omitempty"`
}
