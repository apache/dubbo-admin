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

package authentication

type Policy struct {
	Name string `json:"name,omitempty"`

	Spec *PolicySpec `json:"spec"`
}

type PolicySpec struct {
	Action    string        `json:"action,omitempty"`
	Rules     []*PolicyRule `json:"rules,omitempty"`
	Order     int           `json:"order,omitempty"`
	MatchType string        `json:"matchType,omitempty"`
}

type PolicyRule struct {
	From *Source `json:"from,omitempty"`
	To   *Target `json:"to,omitempty"`
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

	return a.Name == b.Name && PolicySpecEquals(a.Spec, b.Spec)
}

func PolicySpecEquals(a, b *PolicySpec) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if a.Action != b.Action || a.Order != b.Order || a.MatchType != b.MatchType {
		return false
	}

	if len(a.Rules) != len(b.Rules) {
		return false
	}

	for i := range a.Rules {
		if !PolicyRuleEquals(a.Rules[i], b.Rules[i]) {
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

	return SourceEquals(a.From, b.From) && TargetEquals(a.To, b.To)
}

func SourceEquals(a, b *Source) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	return StringArrayEquals(a.Namespaces, b.Namespaces) &&
		StringArrayEquals(a.NotNamespaces, b.NotNamespaces) &&
		StringArrayEquals(a.IpBlocks, b.IpBlocks) &&
		StringArrayEquals(a.NotIpBlocks, b.NotIpBlocks) &&
		StringArrayEquals(a.Principals, b.Principals) &&
		StringArrayEquals(a.NotPrincipals, b.NotPrincipals) &&
		ExtendArrayEquals(a.Extends, b.Extends) &&
		ExtendArrayEquals(a.NotExtends, b.NotExtends)
}

func TargetEquals(a, b *Target) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	return StringArrayEquals(a.IpBlocks, b.IpBlocks) &&
		StringArrayEquals(a.NotIpBlocks, b.NotIpBlocks) &&
		StringArrayEquals(a.Principals, b.Principals) &&
		StringArrayEquals(a.NotPrincipals, b.NotPrincipals) &&
		ExtendArrayEquals(a.Extends, b.Extends) &&
		ExtendArrayEquals(a.NotExtends, b.NotExtends)
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

func ExtendArrayEquals(a, b []*Extend) bool {
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

	return a.Key == b.Key && a.Value == b.Value
}
