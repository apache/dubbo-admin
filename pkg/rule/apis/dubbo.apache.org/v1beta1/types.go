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
	// +kubebuilder:validation:Enum=NONE;DISABLED;PERMISSIVE;STRICT
	Action string `json:"action"`
	// +optional
	Selector []AuthenticationPolicySelector `json:"selector,omitempty"`

	// +optional
	PortLevel []AuthenticationPolicyPortLevel `json:"PortLevel,omitempty"`
}

type AuthenticationPolicySelector struct {
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

type AuthenticationPolicyPortLevel struct {
	// The key of the extended identity.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=number
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=0
	Port int `json:"port,omitempty"`
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Enum=NONE;DISABLED;PERMISSIVE;STRICT
	Action string `json:"action,omitempty"`
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
	// The order of the rule.
	// +optional
	// +kubebuilder:validation:Type=number
	// +kubebuilder:validation:Minimum=-2147483648
	// +kubebuilder:validation:Maximum=2147483647
	// +kubebuilder:default=0
	Order float32 `json:"order,omitempty"`
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
	// The namespaces to match of the source workload.
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`
	// The namespaces not to match of the source workload.
	// +optional
	NotNamespaces []string `json:"notNamespaces,omitempty"`
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

// +kubetype-gen
// +kubetype-gen:groupVersion=dubbo.apache.org/v1alpha1
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ServiceNameMapping struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// +optional
	Spec ServiceNameMappingSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

type ServiceNameMappingSpec struct {
	InterfaceName    string   `json:"interfaceName,omitempty" protobuf:"bytes,1,opt,name=interfaceName,proto3"`
	ApplicationNames []string `json:"applicationNames,omitempty" protobuf:"bytes,2,rep,name=applicationNames,proto3"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ServiceNameMappingList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []*ServiceNameMapping `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// ConditionRouteSpec defines the desired state of ConditionRoute
type ConditionRouteSpec struct {
	// +optional
	Priority int `json:"priority" yaml:"priority,omitempty"`
	// Whether enable this rule or not, set enabled:false to disable this rule.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=boolean
	// +kubebuilder:default=true
	Enabled bool `json:"enabled" yaml:"enabled"`
	// The behaviour when the instance subset is empty after after routing. true means return no provider exception while false means ignore this rule.
	// +optional
	Force bool `json:"force" yaml:"force"`
	// Whether run routing rule for every rpc invocation or use routing cache if available.
	// +optional
	Runtime bool `json:"runtime" yaml:"runtime"`
	// The identifier of the target service or application that this rule is about to apply to.
	// If scope:serviceis set, then keyshould be specified as the Dubbo service key that this rule targets to control.
	// If scope:application is set, then keyshould be specified as the name of the application that this rule targets to control, application should always be a Dubbo Consumer.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	Key string `json:"key" yaml:"key"`
	// Supports service and application scope rules.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Enum=service;application
	Scope string `json:"scope" yaml:"scope"`
	// The condition routing rule definition of this configuration. Check Condition for details
	// +required
	// +kubebuilder:validation:Required
	Conditions []string `json:"conditions" yaml:"conditions"`
	// The version of the condition rule definition, currently available version is v3.0
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Enum=v3.0
	ConfigVersion string `json:"configVersion" yaml:"configVersion"`
}

// +genclient
//+kubebuilder:object:root=true
// +kubebuilder:resource:shortName=cd
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConditionRoute is the Schema for the conditionroutes API
type ConditionRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec ConditionRouteSpec `json:"spec"`
}

//+kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConditionRouteList contains a list of ConditionRoute
type ConditionRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConditionRoute `json:"items"`
}

// TagRouteSpec defines the desired state of TagRoute
type TagRouteSpec struct {
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:integer
	// +kubebuilder:validation:Minimum=-2147483648
	// +kubebuilder:validation:Maximum=2147483647
	Priority int `json:"priority,omitempty"`
	// Whether enable this rule or not, set enabled:false to disable this rule.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=boolean
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
	// The behaviour when the instance subset is empty after after routing. true means return no provider exception while false means ignore this rule.
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:boolean
	// +kubebuilder:default=true
	Force bool `json:"force,omitempty"`
	// Whether run routing rule for every rpc invocation or use routing cache if available.
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:boolean
	// +kubebuilder:default=true
	Runtime bool `json:"runtime,omitempty"`
	// The identifier of the target application that this rule is about to control
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	Key string `json:"key,omitempty"`
	// The tag definition of this rule.
	// +required
	// +kubebuilder:validation:Required
	Tags []Tag `json:"tags,omitempty"`
	// The version of the tag rule definition, currently available version is v3.0
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Enum=v3.0
	ConfigVersion string `json:"configVersion,omitempty"`
}

type Tag struct {
	// The name of the tag used to match the dubbo.tag value in the request context.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	Name string `json:"name,omitempty"`
	// A set of criterion to be met for instances to be classified as member of this tag.
	// +optional
	// +kubebuilder:validation:Optional
	Match []ParamMatch `json:"match,omitempty"`
	// +optional
	Addresses []string `json:"addresses,omitempty"`
}

type StringMatch struct {
	// +optional
	Exact string `json:"exact,omitempty"`
	// +optional
	Prefix string `json:"prefix,omitempty"`
	// +optional
	Regex string `json:"regex,omitempty"`
	// +optional
	Noempty string `json:"noempty,omitempty"`
	// +optional
	Empty string `json:"empty,omitempty"`
	// +optional
	Wildcard string `json:"wildcard,omitempty"`
}

type ParamMatch struct {
	// The name of the key in the Dubbo url address.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	Key string `json:"key,omitempty"`
	// The matching condition for the value in the Dubbo url address.
	// +required
	Value StringMatch `json:"value,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=tg
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TagRoute is the Schema for the tagroutes API
type TagRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec TagRouteSpec `json:"spec"`
}

//+kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TagRouteList contains a list of TagRoute
type TagRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TagRoute `json:"items"`
}

// DynamicConfigSpec defines the desired state of DynamicConfig
type DynamicConfigSpec struct {
	// The identifier of the target service or application that this rule is about to apply to.
	// If scope:serviceis set, then keyshould be specified as the Dubbo service key that this rule targets to control.
	// If scope:application is set, then keyshould be specified as the name of the application that this rule targets to control, application should always be a Dubbo Consumer.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	Key string `json:"key,omitempty"`
	// Supports service and application scope rules.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Enum=service;application
	Scope string `json:"scope,omitempty"`
	// The version of the tag rule definition, currently available version is v3.0
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Enum=v3.0
	ConfigVersion string `json:"configVersion,omitempty"`
	// Whether enable this rule or not, set enabled:false to disable this rule.
	// +required
	// +required
	// +kubebuilder:validation:Type=boolean
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
	// The match condition and configuration of this rule.
	// +required
	// +kubebuilder:validation:Required
	Configs []OverrideConfig `json:"configs,omitempty"`
}

type OverrideConfig struct {
	// Especially useful when scope:service is set.
	// side: providermeans this Config will only take effect on the provider instances of the service key.
	// side: consumermeans this Config will only take effect on the consumer instances of the service key
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	Side string `json:"side,omitempty"`
	// replaced with address in MatchCondition
	// +optional
	Addresses []string `json:"addresses,omitempty"`
	// not supported anymore
	// +optional
	ProviderAddresses []string `json:"providerAddresses,omitempty"`
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`
	// replaced with application in MatchCondition
	// +optional
	Applications []string `json:"applications,omitempty"`
	// replaced with service in MatchCondition
	// +optional
	Services []string `json:"services,omitempty"`
	// +optional
	Type string `json:"type,omitempty"`
	// +optional
	Enabled bool `json:"enabled,omitempty"`
	// A set of criterion to be met in order for the rule/config to be applied to the Dubbo instance.
	// +optional
	Match ConditionMatch `json:"match,omitempty"`
}

type ConditionMatch struct {
	// The instance address matching condition for this config rule to take effect.
	// xact: “value” for exact string match
	// prefix: “value” for prefix-based match
	// regex: “value” for RE2 style regex-based match (https://github.com/google/re2/wiki/Syntax)).
	// +optional
	Address AddressMatch `json:"address,omitempty"`
	// The service matching condition for this config rule to take effect. Effective when scope: application is set.
	// exact: “value” for exact string match
	// prefix: “value” for prefix-based match
	// regex: “value” for RE2 style regex-based match (https://github.com/google/re2/wiki/Syntax)).
	// +optional
	Service ListStringMatch `json:"service,omitempty"`
	// The application matching condition for this config rule to take effect. Effective when scope: service is set.
	//
	// exact: “value” for exact string match
	// prefix: “value” for prefix-based match
	// regex: “value” for RE2 style regex-based match (https://github.com/google/re2/wiki/Syntax)).
	// +optional
	Application ListStringMatch `json:"application,omitempty"`
	// The Dubbo url keys and values matching condition for this config rule to take effect.
	// +optional
	Param []ParamMatch `json:"param,omitempty"`
}

type AddressMatch struct {
	// +optional
	Wildcard string `json:"wildcard,omitempty"`
	// +optional
	Cird string `json:"cird,omitempty"`
	// +optional
	Exact string `json:"exact,omitempty"`
}

type ListStringMatch struct {
	// +optional
	Oneof []StringMatch `json:"oneof,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=dc
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DynamicConfig is the Schema for the dynamicconfigs API
type DynamicConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec DynamicConfigSpec `json:"spec"`
}

//+kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DynamicConfigList contains a list of DynamicConfig
type DynamicConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DynamicConfig `json:"items"`
}
