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

	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/rule/storage"

	"github.com/tidwall/gjson"
)

type ToClient struct {
	revision int64
	data     string
}

func (r *ToClient) Type() string {
	return storage.Authentication
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
	return storage.Authentication
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

func matchSelector(selector *Selector, endpoint *endpoint.Endpoint) bool {
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
	if len(selector.NotExtends) == 0 {
		return true
	}
	for _, extend := range selector.NotExtends {
		if gjson.Get(string(endpointJSON), extend.Key).String() == extend.Value {
			return false
		}
	}
	return true
}

func matchExtends(selector *Selector, endpointJSON []byte) bool {
	if len(selector.Extends) == 0 {
		return true
	}
	for _, extend := range selector.Extends {
		if gjson.Get(string(endpointJSON), extend.Key).String() == extend.Value {
			return true
		}
	}
	return false
}

func matchNotPrincipals(selector *Selector, endpoint *endpoint.Endpoint) bool {
	if len(selector.NotPrincipals) == 0 {
		return true
	}
	for _, principal := range selector.NotPrincipals {
		if principal == endpoint.SpiffeID {
			return false
		}
		if strings.ReplaceAll(endpoint.SpiffeID, "spiffe://", "") == principal {
			return false
		}
	}
	return true
}

func matchPrincipals(selector *Selector, endpoint *endpoint.Endpoint) bool {
	if len(selector.Principals) == 0 {
		return true
	}
	for _, principal := range selector.Principals {
		if principal == endpoint.SpiffeID {
			return true
		}
		if strings.ReplaceAll(endpoint.SpiffeID, "spiffe://", "") == principal {
			return true
		}
	}
	return false
}

func matchNotIPBlocks(selector *Selector, endpoint *endpoint.Endpoint) bool {
	if len(selector.NotIpBlocks) == 0 {
		return true
	}
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

func matchIPBlocks(selector *Selector, endpoint *endpoint.Endpoint) bool {
	if len(selector.IpBlocks) == 0 {
		return true
	}
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
	return false
}

func matchNotNamespace(selector *Selector, endpoint *endpoint.Endpoint) bool {
	if len(selector.NotNamespaces) == 0 {
		return true
	}
	for _, namespace := range selector.NotNamespaces {
		if endpoint.KubernetesEnv != nil && namespace == endpoint.KubernetesEnv.Namespace {
			return false
		}
	}
	return true
}

func matchNamespace(selector *Selector, endpoint *endpoint.Endpoint) bool {
	if len(selector.Namespaces) == 0 {
		return true
	}
	for _, namespace := range selector.Namespaces {
		if endpoint.KubernetesEnv != nil && namespace == endpoint.KubernetesEnv.Namespace {
			return true
		}
	}
	return false
}
