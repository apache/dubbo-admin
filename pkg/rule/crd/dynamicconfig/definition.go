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

package dynamicconfig

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
	Key           string            `json:"key,omitempty"`
	Scope         string            `json:"scope,omitempty"`
	ConfigVersion string            `json:"configVersion,omitempty"`
	Enabled       bool              `json:"enabled,omitempty"`
	Configs       []*OverrideConfig `json:"configs,omitempty"`
}

func (p *PolicySpec) CopyToClient() *PolicySpecToClient {
	toClient := &PolicySpecToClient{
		Key:           p.Key,
		Scope:         p.Scope,
		ConfigVersion: p.ConfigVersion,
		Enabled:       p.Enabled,
	}

	if p.Configs != nil {
		toClient.Configs = make([]*OverrideConfigToClient, 0, len(p.Configs))
		for _, config := range p.Configs {
			toClient.Configs = append(toClient.Configs, config.CopyToClient())
		}
	}

	return toClient
}

type OverrideConfig struct {
	Side              string            `json:"side,omitempty"`
	Addresses         []string          `json:"addresses,omitempty"`
	ProviderAddresses []string          `json:"providerAddresses,omitempty"`
	Parameters        map[string]string `json:"parameters,omitempty"`
	Applications      []string          `json:"applications,omitempty"`
	Services          []string          `json:"services,omitempty"`
	Type              string            `json:"type,omitempty"`
	Enabled           bool              `json:"enabled,omitempty"`
	Match             *ConditionMatch   `json:"match,omitempty"`
}

func (p *OverrideConfig) CopyToClient() *OverrideConfigToClient {
	toClient := &OverrideConfigToClient{
		Side:              p.Side,
		Addresses:         p.Addresses,
		ProviderAddresses: p.ProviderAddresses,
		Parameters:        p.Parameters,
		Applications:      p.Applications,
		Services:          p.Services,
		Type:              p.Type,
		Enabled:           p.Enabled,
	}

	if p.Match != nil {
		toClient.Match = p.Match.CopyToClient()
	}

	return toClient
}

type ConditionMatch struct {
	Address     *AddressMatch    `json:"address,omitempty"`
	Service     *ListStringMatch `json:"service,omitempty"`
	Application *ListStringMatch `json:"application,omitempty"`
	Param       []*ParamMatch    `json:"param,omitempty"`
}

func (p *ConditionMatch) CopyToClient() *ConditionMatchToClient {
	toClient := &ConditionMatchToClient{}
	if p.Address != nil {
		toClient.Address = p.Address.CopyToClient()
	}
	if p.Service != nil {
		toClient.Service = p.Service.CopyToClient()
	}
	if p.Application != nil {
		toClient.Application = p.Application.CopyToClient()
	}
	if p.Param != nil {
		toClient.Param = make([]*ParamMatchToClient, 0, len(p.Param))
		for _, param := range p.Param {
			toClient.Param = append(toClient.Param, param.CopyToClient())
		}
	}

	return toClient
}

type AddressMatch struct {
	Wildcard string `json:"wildcard,omitempty"`
	Cird     string `json:"cird,omitempty"`
	Exact    string `json:"exact,omitempty"`
}

func (p *AddressMatch) CopyToClient() *AddressMatchToClient {
	toClient := &AddressMatchToClient{
		Wildcard: p.Wildcard,
		Cird:     p.Cird,
		Exact:    p.Exact,
	}

	return toClient
}

type ParamMatch struct {
	Key   string       `json:"key,omitempty"`
	Value *StringMatch `json:"value,omitempty"`
}

func (p *ParamMatch) CopyToClient() *ParamMatchToClient {
	toClient := &ParamMatchToClient{
		Key: p.Key,
	}

	if p.Value != nil {
		toClient.Value = p.Value.CopyToClient()
	}

	return toClient
}

type ListStringMatch struct {
	Oneof []*StringMatch `json:"oneof,omitempty"`
}

func (p *ListStringMatch) CopyToClient() *ListStringMatchToClient {
	toClient := &ListStringMatchToClient{}
	if p.Oneof != nil {
		toClient.Oneof = make([]*StringMatchToClient, 0, len(p.Oneof))
		for _, one := range p.Oneof {
			toClient.Oneof = append(toClient.Oneof, one.CopyToClient())
		}
	}
	return toClient
}

type StringMatch struct {
	Exact    string `json:"exact,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
	Regex    string `json:"regex,omitempty"`
	Noempty  string `json:"noempty,omitempty"`
	Empty    string `json:"empty,omitempty"`
	Wildcard string `json:"wildcard,omitempty"`
}

func (p *StringMatch) CopyToClient() *StringMatchToClient {
	toClient := &StringMatchToClient{
		Exact:    p.Exact,
		Prefix:   p.Prefix,
		Regex:    p.Regex,
		Noempty:  p.Noempty,
		Empty:    p.Empty,
		Wildcard: p.Wildcard,
	}
	return toClient
}

// To Client Rule

type PolicyToClient struct {
	Name string `json:"name,omitempty"`

	Spec *PolicySpecToClient `json:"spec"`
}

type PolicySpecToClient struct {
	Key           string                    `json:"key,omitempty"`
	Scope         string                    `json:"scope,omitempty"`
	ConfigVersion string                    `json:"configVersion,omitempty"`
	Enabled       bool                      `json:"enabled,omitempty"`
	Configs       []*OverrideConfigToClient `json:"configs,omitempty"`
}

type OverrideConfigToClient struct {
	Side              string                  `json:"side,omitempty"`
	Addresses         []string                `json:"addresses,omitempty"`
	ProviderAddresses []string                `json:"providerAddresses,omitempty"`
	Parameters        map[string]string       `json:"parameters,omitempty"`
	Applications      []string                `json:"applications,omitempty"`
	Services          []string                `json:"services,omitempty"`
	Type              string                  `json:"type,omitempty"`
	Enabled           bool                    `json:"enabled,omitempty"`
	Match             *ConditionMatchToClient `json:"match,omitempty"`
}

type ConditionMatchToClient struct {
	Address     *AddressMatchToClient    `json:"address,omitempty"`
	Service     *ListStringMatchToClient `json:"service,omitempty"`
	Application *ListStringMatchToClient `json:"application,omitempty"`
	Param       []*ParamMatchToClient    `json:"param,omitempty"`
}

type AddressMatchToClient struct {
	Wildcard string `json:"wildcard,omitempty"`
	Cird     string `json:"cird,omitempty"`
	Exact    string `json:"exact,omitempty"`
}

type ParamMatchToClient struct {
	Key   string               `json:"key,omitempty"`
	Value *StringMatchToClient `json:"value,omitempty"`
}

type ListStringMatchToClient struct {
	Oneof []*StringMatchToClient `json:"oneof,omitempty"`
}

type StringMatchToClient struct {
	Exact    string `json:"exact,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
	Regex    string `json:"regex,omitempty"`
	Noempty  string `json:"noempty,omitempty"`
	Empty    string `json:"empty,omitempty"`
	Wildcard string `json:"wildcard,omitempty"`
}
