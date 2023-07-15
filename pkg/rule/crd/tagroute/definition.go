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

package tagroute

type Policy struct {
	Name string `json:"name,omitempty"`

	Spec *PolicySpec `json:"spec"`
}

func (p *Policy) CopyToClient() *PolicyToClient {
	toClient := &PolicyToClient{Name: p.Name}
	if p.Spec != nil {
		toClient.Spec = p.Spec.CopyToClient()
	}
	return toClient
}

type PolicySpec struct {
	Priority      int    `json:"priority"`
	Enabled       bool   `json:"enabled"`
	Force         bool   `json:"force"`
	Runtime       bool   `json:"runtime"`
	Key           string `json:"key"`
	Tags          []*Tag `json:"tags"`
	ConfigVersion string `json:"configVersion"`
}

func (p *PolicySpec) CopyToClient() *PolicySpecToClient {
	toClient := &PolicySpecToClient{
		Priority:      p.Priority,
		Enabled:       p.Enabled,
		Force:         p.Force,
		Runtime:       p.Runtime,
		Key:           p.Key,
		ConfigVersion: p.ConfigVersion,
	}

	if p.Tags != nil {
		toClient.Tags = make([]*TagToClient, 0, len(p.Tags))
		for _, tag := range p.Tags {
			toClient.Tags = append(toClient.Tags, tag.CopyToClient())
		}
	}

	return toClient
}

type Tag struct {
	Name      string        `json:"name"`
	Match     []*ParamMatch `json:"match"`
	Addresses []string      `json:"addresses"`
}

func (p *Tag) CopyToClient() *TagToClient {
	toClient := &TagToClient{
		Name: p.Name,
	}

	if p.Addresses != nil {
		toClient.Addresses = make([]string, len(p.Addresses))
		copy(toClient.Addresses, p.Addresses)
	}

	if p.Match != nil {
		toClient.Match = make([]*ParamMatchToClient, 0, len(p.Match))
		for _, match := range p.Match {
			toClient.Match = append(toClient.Match, match.CopyToClient())
		}
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
	Priority      int            `json:"priority"`
	Enabled       bool           `json:"enabled"`
	Force         bool           `json:"force"`
	Runtime       bool           `json:"runtime"`
	Key           string         `json:"key"`
	Tags          []*TagToClient `json:"tags"`
	ConfigVersion string         `json:"configVersion"`
}

type TagToClient struct {
	Name      string                `json:"name"`
	Match     []*ParamMatchToClient `json:"match"`
	Addresses []string              `json:"addresses"`
}

type ParamMatchToClient struct {
	Key   string               `json:"key,omitempty"`
	Value *StringMatchToClient `json:"value,omitempty"`
}

type StringMatchToClient struct {
	Exact    string `json:"exact,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
	Regex    string `json:"regex,omitempty"`
	Noempty  string `json:"noempty,omitempty"`
	Empty    string `json:"empty,omitempty"`
	Wildcard string `json:"wildcard,omitempty"`
}
