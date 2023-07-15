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
	Action    string       `json:"action"`
	Selector  []*Selector  `json:"selector,omitempty"`
	PortLevel []*PortLevel `json:"PortLevel,omitempty"`
}

func (p *PolicySpec) CopyToClient() *PolicySpecToClient {
	toClient := &PolicySpecToClient{
		Action: p.Action,
	}

	if p.PortLevel != nil {
		toClient.PortLevel = make([]*PortLevelToClient, 0, len(p.PortLevel))
		for _, portLevel := range p.PortLevel {
			toClient.PortLevel = append(toClient.PortLevel, portLevel.CopyToClient())
		}
	}

	return toClient
}

type Selector struct {
	Namespaces    []string  `json:"namespaces,omitempty"`
	NotNamespaces []string  `json:"notNamespaces,omitempty"`
	IpBlocks      []string  `json:"ipBlocks,omitempty"`
	NotIpBlocks   []string  `json:"notIpBlocks,omitempty"`
	Principals    []string  `json:"principals,omitempty"`
	NotPrincipals []string  `json:"notPrincipals,omitempty"`
	Extends       []*Extend `json:"extends,omitempty"`
	NotExtends    []*Extend `json:"notExtends,omitempty"`
}

type PortLevel struct {
	Port   int    `json:"port,omitempty"`
	Action string `json:"action,omitempty"`
}

func (p *PortLevel) CopyToClient() *PortLevelToClient {
	return &PortLevelToClient{
		Port:   p.Port,
		Action: p.Action,
	}
}

type Extend struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// To Client Rule

type PolicyToClient struct {
	Name string              `json:"name,omitempty"`
	Spec *PolicySpecToClient `json:"spec"`
}

type PolicySpecToClient struct {
	Action    string               `json:"action"`
	PortLevel []*PortLevelToClient `json:"PortLevel,omitempty"`
}

type PortLevelToClient struct {
	Port   int    `json:"port,omitempty"`
	Action string `json:"action,omitempty"`
}
