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

import (
	"encoding/json"
	"net/netip"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/apache/dubbo-admin/pkg/authority/rule"
	"github.com/apache/dubbo-admin/pkg/logger"
)

const RuleType = "authentication/v1beta1"

type ToClient struct {
	revision int64
	data     string
}

func (r *ToClient) Type() string {
	return "authentication/v1beta1"
}

func (r *ToClient) Revision() int64 {
	return r.revision
}

func (r *ToClient) Data() string {
	return r.data
}

type Origin struct {
	revision int64
	data     map[string]*Policy
}

func (o *Origin) Type() string {
	return RuleType
}

func (o *Origin) Revision() int64 {
	return o.revision
}

func (o *Origin) Exact(endpoint *rule.Endpoint) (rule.ToClient, error) {
	matchedRule := make([]*PolicyToClient, 0, len(o.data))

	// TODO match endpoint

	for _, v := range o.data {
		if v.Spec == nil {
			continue
		}

		if v.Spec.Selector != nil {
			match := true
			for _, selector := range v.Spec.Selector {
				if !matchSelector(selector, endpoint) {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		toClient := &PolicyToClient{
			Name: v.Name,
			Spec: &PolicySpecToClient{
				Action: v.Spec.Action,
			},
		}
		if v.Spec.PortLevel != nil {
			for _, p := range v.Spec.PortLevel {
				toClient.Spec.PortLevel = append(toClient.Spec.PortLevel, &PortLevelToClient{
					Port:   p.Port,
					Action: p.Action,
				})
			}
		}
		matchedRule = append(matchedRule, toClient)
	}

	allRule, err := json.Marshal(matchedRule)
	if err != nil {
		return nil, err
	}

	return &ToClient{
		revision: o.revision,
		data:     string(allRule),
	}, nil
}

func matchSelector(selector *Selector, endpoint *rule.Endpoint) bool {
	if endpoint == nil {
		return true
	}

	if !matchNamespace(selector, endpoint) {
		return false
	}

	if !matchNotNamespace(selector, endpoint) {
		return false
	}

	if !matchIPBlocks(selector, endpoint) {
		return false
	}

	if !matchNotIPBlocks(selector, endpoint) {
		return false
	}

	if !matchPrincipals(selector, endpoint) {
		return false
	}

	if !matchNotPrincipals(selector, endpoint) {
		return false
	}

	endpointJSON, err := json.Marshal(endpoint)
	if err != nil {
		logger.Sugar().Warnf("marshal endpoint failed, %v", err)
		return false
	}

	if !matchExtends(selector, endpointJSON) {
		return false
	}

	return matchNotExtends(selector, endpointJSON)
}

func matchNotExtends(selector *Selector, endpointJSON []byte) bool {
	for _, extend := range selector.NotExtends {
		if gjson.Get(string(endpointJSON), extend.Key).String() == extend.Value {
			return false
		}
	}
	return true
}

func matchExtends(selector *Selector, endpointJSON []byte) bool {
	for _, extend := range selector.Extends {
		if gjson.Get(string(endpointJSON), extend.Key).String() == extend.Value {
			return true
		}
	}
	return len(selector.Extends) == 0
}

func matchNotPrincipals(selector *Selector, endpoint *rule.Endpoint) bool {
	for _, principal := range selector.NotPrincipals {
		if principal == endpoint.SpiffeID {
			return false
		}
		cut, ok := strings.CutPrefix(endpoint.SpiffeID, "spiffe://")
		if ok && cut == principal {
			return false
		}
	}
	return true
}

func matchPrincipals(selector *Selector, endpoint *rule.Endpoint) bool {
	for _, principal := range selector.Principals {
		if principal == endpoint.SpiffeID {
			return true
		}
		cut, ok := strings.CutPrefix(endpoint.SpiffeID, "spiffe://")
		if ok && cut == principal {
			return true
		}
	}
	return len(selector.Principals) == 0
}

func matchNotIPBlocks(selector *Selector, endpoint *rule.Endpoint) bool {
	for _, ipBlock := range selector.NotIpBlocks {
		prefix, err := netip.ParsePrefix(ipBlock)
		if err != nil {
			logger.Sugar().Warnf("parse ip block %s failed, %v", ipBlock, err)
			continue
		}
		for _, ip := range endpoint.Ips {
			addr, err := netip.ParseAddr(ip)
			if err != nil {
				logger.Sugar().Warnf("parse ip %s failed, %v", ip, err)
				continue
			}
			if prefix.Contains(addr) {
				return false
			}
		}
	}
	return true
}

func matchIPBlocks(selector *Selector, endpoint *rule.Endpoint) bool {
	for _, ipBlock := range selector.IpBlocks {
		prefix, err := netip.ParsePrefix(ipBlock)
		if err != nil {
			logger.Sugar().Warnf("parse ip block %s failed, %v", ipBlock, err)
			continue
		}
		for _, ip := range endpoint.Ips {
			addr, err := netip.ParseAddr(ip)
			if err != nil {
				logger.Sugar().Warnf("parse ip %s failed, %v", ip, err)
				continue
			}
			if prefix.Contains(addr) {
				return true
			}
		}
	}
	return len(selector.IpBlocks) == 0
}

func matchNotNamespace(selector *Selector, endpoint *rule.Endpoint) bool {
	for _, namespace := range selector.NotNamespaces {
		if namespace == endpoint.KubernetesEnv.Namespace {
			return false
		}
	}
	return true
}

func matchNamespace(selector *Selector, endpoint *rule.Endpoint) bool {
	match := len(selector.Namespaces) == 0
	for _, namespace := range selector.Namespaces {
		if namespace == endpoint.KubernetesEnv.Namespace {
			match = true
		}
	}
	return match
}
