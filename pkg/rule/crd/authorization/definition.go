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

func (p *Policy) CopyToClient() *PolicyToClient {
	toClient := &PolicyToClient{
		Name: p.Name,
	}

	if p.Spec != nil {
		toClient.Spec = p.Spec.CopyToClient()
	}

	return toClient
}

type PolicySpec struct {
	Action    string        `json:"action,omitempty"`
	Rules     []*PolicyRule `json:"rules,omitempty"`
	Samples   float32       `json:"samples,omitempty"`
	Order     float32       `json:"order,omitempty"`
	MatchType string        `json:"matchType,omitempty"`
}

func (p *PolicySpec) CopyToClient() *PolicySpecToClient {
	toClient := &PolicySpecToClient{
		Action:    p.Action,
		Samples:   p.Samples,
		Order:     p.Order,
		MatchType: p.MatchType,
	}

	if p.Rules != nil {
		toClient.Rules = make([]*PolicyRuleToClient, 0, len(p.Rules))
		for _, rule := range p.Rules {
			toClient.Rules = append(toClient.Rules, rule.CopyToClient())
		}
	}

	return toClient
}

type PolicyRule struct {
	From *Source    `json:"from,omitempty"`
	To   *Target    `json:"to,omitempty"`
	When *Condition `json:"when,omitempty"`
}

func (p *PolicyRule) CopyToClient() *PolicyRuleToClient {
	toClient := &PolicyRuleToClient{}

	if p.From != nil {
		toClient.From = p.From.CopyToClient()
	}

	if p.When != nil {
		toClient.When = p.When.CopyToClient()
	}

	return toClient
}

type Source struct {
	Namespaces    []string  `json:"namespaces,omitempty"`
	NotNamespaces []string  `json:"notNamespaces,omitempty"`
	IpBlocks      []string  `json:"ipBlocks,omitempty"`
	NotIpBlocks   []string  `json:"notIpBlocks,omitempty"`
	Principals    []string  `json:"principals,omitempty"`
	NotPrincipals []string  `json:"notPrincipals,omitempty"`
	Extends       []*Extend `json:"sourceExtends,omitempty"`
	NotExtends    []*Extend `json:"sourceNotExtends,omitempty"`
}

func (s *Source) CopyToClient() *SourceToClient {
	toClient := &SourceToClient{}

	if s.Namespaces != nil {
		toClient.Namespaces = make([]string, len(s.Namespaces))
		copy(toClient.Namespaces, s.Namespaces)
	}

	if s.NotNamespaces != nil {
		toClient.NotNamespaces = make([]string, len(s.NotNamespaces))
		copy(toClient.NotNamespaces, s.NotNamespaces)
	}

	if s.IpBlocks != nil {
		toClient.IpBlocks = make([]string, len(s.IpBlocks))
		copy(toClient.IpBlocks, s.IpBlocks)
	}

	if s.NotIpBlocks != nil {
		toClient.NotIpBlocks = make([]string, len(s.NotIpBlocks))
		copy(toClient.NotIpBlocks, s.NotIpBlocks)
	}

	if s.Principals != nil {
		toClient.Principals = make([]string, len(s.Principals))
		copy(toClient.Principals, s.Principals)
	}

	if s.NotPrincipals != nil {
		toClient.NotPrincipals = make([]string, len(s.NotPrincipals))
		copy(toClient.NotPrincipals, s.NotPrincipals)
	}

	if s.Extends != nil {
		toClient.Extends = make([]*ExtendToClient, len(s.Extends))
		for i, v := range s.Extends {
			toClient.Extends[i] = v.CopyToClient()
		}
	}

	if s.NotExtends != nil {
		toClient.NotExtends = make([]*ExtendToClient, len(s.NotExtends))
		for i, v := range s.NotExtends {
			toClient.NotExtends[i] = v.CopyToClient()
		}
	}

	return toClient
}

type Target struct {
	Namespaces    []string  `json:"namespaces,omitempty"`
	NotNamespaces []string  `json:"notNamespaces,omitempty"`
	IpBlocks      []string  `json:"ipBlocks,omitempty"`
	NotIpBlocks   []string  `json:"notIpBlocks,omitempty"`
	Principals    []string  `json:"principals,omitempty"`
	NotPrincipals []string  `json:"notPrincipals,omitempty"`
	Extends       []*Extend `json:"targetExtends,omitempty"`
	NotExtends    []*Extend `json:"targetNotExtends,omitempty"`
}

type Condition struct {
	Key       string   `json:"key,omitempty"`
	Values    []*Match `json:"values,omitempty"`
	NotValues []*Match `json:"notValues,omitempty"`
}

func (c *Condition) CopyToClient() *ConditionToClient {
	toClient := &ConditionToClient{
		Key: c.Key,
	}

	if c.Values != nil {
		toClient.Values = make([]*MatchToClient, len(c.Values))
		for i, v := range c.Values {
			toClient.Values[i] = v.CopyToClient()
		}
	}

	if c.NotValues != nil {
		toClient.NotValues = make([]*MatchToClient, len(c.NotValues))
		for i, v := range c.NotValues {
			toClient.NotValues[i] = v.CopyToClient()
		}
	}

	return toClient
}

type Match struct {
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}

func (m *Match) CopyToClient() *MatchToClient {
	return &MatchToClient{
		Type:  m.Type,
		Value: m.Value,
	}
}

type Extend struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

func (e *Extend) CopyToClient() *ExtendToClient {
	return &ExtendToClient{
		Key:   e.Key,
		Value: e.Value,
	}
}

// To Client Rule

type PolicyToClient struct {
	Name string `json:"name,omitempty"`

	Spec *PolicySpecToClient `json:"spec"`
}

type PolicySpecToClient struct {
	Action    string                `json:"action,omitempty"`
	Rules     []*PolicyRuleToClient `json:"rules,omitempty"`
	Samples   float32               `json:"samples,omitempty"`
	Order     float32               `json:"order,omitempty"`
	MatchType string                `json:"matchType,omitempty"`
}

type PolicyRuleToClient struct {
	From *SourceToClient    `json:"from,omitempty"`
	When *ConditionToClient `json:"when,omitempty"`
}

type SourceToClient struct {
	Namespaces    []string          `json:"namespaces,omitempty"`
	NotNamespaces []string          `json:"notNamespaces,omitempty"`
	IpBlocks      []string          `json:"ipBlocks,omitempty"`
	NotIpBlocks   []string          `json:"notIpBlocks,omitempty"`
	Principals    []string          `json:"principals,omitempty"`
	NotPrincipals []string          `json:"notPrincipals,omitempty"`
	Extends       []*ExtendToClient `json:"sourceExtends,omitempty"`
	NotExtends    []*ExtendToClient `json:"sourceNotExtends,omitempty"`
}

type TargetToClient struct {
	Namespaces    []string          `json:"namespaces,omitempty"`
	NotNamespaces []string          `json:"notNamespaces,omitempty"`
	IpBlocks      []string          `json:"ipBlocks,omitempty"`
	NotIpBlocks   []string          `json:"notIpBlocks,omitempty"`
	Principals    []string          `json:"principals,omitempty"`
	NotPrincipals []string          `json:"notPrincipals,omitempty"`
	Extends       []*ExtendToClient `json:"targetExtends,omitempty"`
	NotExtends    []*ExtendToClient `json:"targetNotExtends,omitempty"`
}

type ConditionToClient struct {
	Key       string           `json:"key,omitempty"`
	Values    []*MatchToClient `json:"values,omitempty"`
	NotValues []*MatchToClient `json:"notValues,omitempty"`
}

type MatchToClient struct {
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}

type ExtendToClient struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}
