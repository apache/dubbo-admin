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

package authorization

type Policy struct {
	Name string `json:"name,omitempty"`

	Spec *PolicySpec `json:"spec"`
}

type PolicySpec struct {
	Action    string        `json:"action,omitempty"`
	Rules     []*PolicyRule `json:"rules,omitempty"`
	Samples   float32       `json:"samples,omitempty"`
	MatchType string        `json:"matchType,omitempty"`
}

type PolicyRule struct {
	From *Source    `json:"from,omitempty"`
	To   *Target    `json:"to,omitempty"`
	When *Condition `json:"when,omitempty"`
}

type Source struct {
	Namespaces    []string  `json:"namespaces,omitempty"`
	NotNamespaces []string  `json:"notNamespaces,omitempty"`
	IpBlocks      []string  `json:"ipBlocks,omitempty"`
	NotIpBlocks   []string  `json:"notIpBlocks,omitempty"`
	Principals    []string  `json:"principals,omitempty"`
	NotPrincipals []string  `json:"notPrincipals,omitempty"`
	Extends       []*Extend `json:"extends,omitempty"`
	NotExtends    []*Extend `json:"notExtends,omitempty"`
}

type Target struct {
	IpBlocks      []string  `json:"ipBlocks,omitempty"`
	NotIpBlocks   []string  `json:"notIpBlocks,omitempty"`
	Principals    []string  `json:"principals,omitempty"`
	NotPrincipals []string  `json:"notPrincipals,omitempty"`
	Extends       []*Extend `json:"extends,omitempty"`
	NotExtends    []*Extend `json:"notExtends,omitempty"`
}

type Condition struct {
	Key       string   `json:"key,omitempty"`
	Values    []*Match `json:"values,omitempty"`
	NotValues []*Match `json:"notValues,omitempty"`
}

type Match struct {
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}

type Extend struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}
