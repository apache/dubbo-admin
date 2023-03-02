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

type PeerAuthentication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PeerAuthenticationSpec `json:"spec"`
}

type PeerAuthenticationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PeerAuthentication `json:"items"`
}

type PeerAuthenticationSpec struct {
	Action string `json:"action,omitempty"`
	Rule   Rule   `json:"rule,omitempty"`
	Order  int    `json:"order,omitempty"`
}

type Rule struct {
	From []Source `json:"from,omitempty"`
	To   []Source `json:"to,omitempty"`
}

type Source struct {
	Namespaces    []string       `json:"namespaces,omitempty"`
	NotNamespaces []string       `json:"notNamespaces,omitempty"`
	IpBlocks      []string       `json:"ipBlocks,omitempty"`
	NotIpBlocks   []string       `json:"notIpBlocks,omitempty"`
	Principals    []string       `json:"principals,omitempty"`
	NotPrincipals []string       `json:"notPrincipals,omitempty"`
	Extends       []ExtendConfig `json:"extends,omitempty"`
	NotExtends    []ExtendConfig `json:"notExtends,omitempty"`
}

type ExtendConfig struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}
