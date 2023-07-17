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

import (
	"encoding/json"
	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/rule/storage"
	"net/netip"
	"strings"

	"github.com/tidwall/gjson"
)

type ToClient struct {
	revision int64
	data     string
}

func (r *ToClient) Type() string {
	return storage.Authorization
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
	return storage.Authorization
}

func (o *Origin) Revision() int64 {
	return o.revision
}

func (o *Origin) Exact(endpoint *endpoint.Endpoint) (storage.ToClient, error) {
	matchedRule := make([]*PolicyToClient, 0, len(o.data))

	for _, v := range o.data {
		if v.Spec == nil {
			continue
		}

		if v.Spec.Rules != nil {
			match := true
			for _, policyRule := range v.Spec.Rules {
				if !matchSelector(policyRule.To, endpoint) {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		toClient := v.CopyToClient()
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

func matchSelector(target *Target, endpoint *endpoint.Endpoint) bool {
	if endpoint == nil {
		return true
	}

	if !matchNamespace(target, endpoint) {
		return false
	}

	if !matchNotNamespace(target, endpoint) {
		return false
	}

	if !matchIPBlocks(target, endpoint) {
		return false
	}

	if !matchNotIPBlocks(target, endpoint) {
		return false
	}

	if !matchPrincipals(target, endpoint) {
		return false
	}

	if !matchNotPrincipals(target, endpoint) {
		return false
	}

	endpointJSON, err := json.Marshal(endpoint)
	if err != nil {
		logger.Sugar().Warnf("marshal endpoint failed, %v", err)
		return false
	}

	if !matchExtends(target, endpointJSON) {
		return false
	}

	return matchNotExtends(target, endpointJSON)
}

func matchNotExtends(target *Target, endpointJSON []byte) bool {
	if len(target.NotExtends) == 0 {
		return true
	}
	for _, extend := range target.NotExtends {
		if gjson.Get(string(endpointJSON), extend.Key).String() == extend.Value {
			return false
		}
	}
	return true
}

func matchExtends(target *Target, endpointJSON []byte) bool {
	if len(target.Extends) == 0 {
		return true
	}
	for _, extend := range target.Extends {
		if gjson.Get(string(endpointJSON), extend.Key).String() == extend.Value {
			return true
		}
	}
	return false
}

func matchNotPrincipals(target *Target, endpoint *endpoint.Endpoint) bool {
	if len(target.NotPrincipals) == 0 {
		return true
	}
	for _, principal := range target.NotPrincipals {
		if principal == endpoint.SpiffeID {
			return false
		}
		if strings.ReplaceAll(endpoint.SpiffeID, "spiffe://", "") == principal {
			return false
		}
	}
	return true
}

func matchPrincipals(target *Target, endpoint *endpoint.Endpoint) bool {
	if len(target.Principals) == 0 {
		return true
	}
	for _, principal := range target.Principals {
		if principal == endpoint.SpiffeID {
			return true
		}
		if strings.ReplaceAll(endpoint.SpiffeID, "spiffe://", "") == principal {
			return true
		}
	}
	return false
}

func matchNotIPBlocks(target *Target, endpoint *endpoint.Endpoint) bool {
	if len(target.NotIpBlocks) == 0 {
		return true
	}
	for _, ipBlock := range target.NotIpBlocks {
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

func matchIPBlocks(target *Target, endpoint *endpoint.Endpoint) bool {
	if len(target.IpBlocks) == 0 {
		return true
	}
	for _, ipBlock := range target.IpBlocks {
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
	return false
}

func matchNotNamespace(target *Target, endpoint *endpoint.Endpoint) bool {
	if len(target.NotNamespaces) == 0 {
		return true
	}
	for _, namespace := range target.NotNamespaces {
		if endpoint.KubernetesEnv != nil && namespace == endpoint.KubernetesEnv.Namespace {
			return false
		}
	}
	return true
}

func matchNamespace(target *Target, endpoint *endpoint.Endpoint) bool {
	if len(target.Namespaces) == 0 {
		return true
	}
	for _, namespace := range target.Namespaces {
		if endpoint.KubernetesEnv != nil && namespace == endpoint.KubernetesEnv.Namespace {
			return true
		}
	}
	return false
}
