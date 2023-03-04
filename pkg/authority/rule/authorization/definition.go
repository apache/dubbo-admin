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

func PolicyEquals(a, b *Policy) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Name != b.Name {
		return false
	}
	if !PolicySpecEquals(a.Spec, b.Spec) {
		return false
	}
	return true
}

func PolicySpecEquals(a, b *PolicySpec) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Action != b.Action {
		return false
	}
	if !PolicyRuleListEquals(a.Rules, b.Rules) {
		return false
	}
	if a.MatchType != b.MatchType {
		return false
	}
	return true
}

func PolicyRuleListEquals(a, b []*PolicyRule) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !PolicyRuleEquals(a[i], b[i]) {
			return false
		}
	}
	return true
}

func PolicyRuleEquals(a, b *PolicyRule) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if !SourceEquals(a.From, b.From) {
		return false
	}
	if !TargetEquals(a.To, b.To) {
		return false
	}
	if !ConditionEquals(a.When, b.When) {
		return false
	}
	return true
}

func SourceEquals(a, b *Source) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if !StringArrayEquals(a.Namespaces, b.Namespaces) {
		return false
	}
	if !StringArrayEquals(a.NotNamespaces, b.NotNamespaces) {
		return false
	}
	if !StringArrayEquals(a.IpBlocks, b.IpBlocks) {
		return false
	}
	if !StringArrayEquals(a.NotIpBlocks, b.NotIpBlocks) {
		return false
	}
	if !StringArrayEquals(a.Principals, b.Principals) {
		return false
	}
	if !StringArrayEquals(a.NotPrincipals, b.NotPrincipals) {
		return false
	}
	if !ExtendListEquals(a.Extends, b.Extends) {
		return false
	}
	if !ExtendListEquals(a.NotExtends, b.NotExtends) {
		return false
	}
	return true
}

func TargetEquals(a, b *Target) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if !StringArrayEquals(a.IpBlocks, b.IpBlocks) {
		return false
	}
	if !StringArrayEquals(a.NotIpBlocks, b.NotIpBlocks) {
		return false
	}
	if !StringArrayEquals(a.Principals, b.Principals) {
		return false
	}
	if !StringArrayEquals(a.NotPrincipals, b.NotPrincipals) {
		return false
	}
	if !ExtendListEquals(a.Extends, b.Extends) {
		return false
	}
	if !ExtendListEquals(a.NotExtends, b.NotExtends) {
		return false
	}
	return true
}

func ConditionEquals(a, b *Condition) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Key != b.Key {
		return false
	}
	if !MatchListEquals(a.Values, b.Values) {
		return false
	}
	if !MatchListEquals(a.NotValues, b.NotValues) {
		return false
	}
	return true
}

func MatchListEquals(a, b []*Match) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !MatchEquals(a[i], b[i]) {
			return false
		}
	}
	return true
}

func MatchEquals(a, b *Match) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Type != b.Type {
		return false
	}
	if a.Value != b.Value {
		return false
	}
	return true
}

func ExtendListEquals(a, b []*Extend) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !ExtendEquals(a[i], b[i]) {
			return false
		}
	}
	return true
}

func ExtendEquals(a, b *Extend) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Key != b.Key {
		return false
	}
	if a.Value != b.Value {
		return false
	}
	return true
}

func StringArrayEquals(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
