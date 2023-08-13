/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package storage

import (
	"encoding/json"
	"net/netip"
	"strings"

	api "github.com/apache/dubbo-admin/api/resource/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/tidwall/gjson"
)

func MatchAuthnSelector(selector *api.AuthenticationPolicySelector, endpoint *endpoint.Endpoint) bool {
	if endpoint == nil {
		return true
	}

	if !matchAuthnNamespace(selector, endpoint) {
		return false
	}

	if !matchAuthnNotNamespace(selector, endpoint) {
		return false
	}

	if !matchAuthnIPBlocks(selector, endpoint) {
		return false
	}

	if !matchAuthnNotIPBlocks(selector, endpoint) {
		return false
	}

	if !matchAuthnPrincipals(selector, endpoint) {
		return false
	}

	if !matchAuthnNotPrincipals(selector, endpoint) {
		return false
	}

	endpointJSON, err := json.Marshal(endpoint)
	if err != nil {
		logger.Sugar().Warnf("marshal endpoint failed, %v", err)
		return false
	}

	if !matchAuthnExtends(selector, endpointJSON) {
		return false
	}

	return matchAuthnNotExtends(selector, endpointJSON)
}

func matchAuthnNotExtends(selector *api.AuthenticationPolicySelector, endpointJSON []byte) bool {
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

func matchAuthnExtends(selector *api.AuthenticationPolicySelector, endpointJSON []byte) bool {
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

func matchAuthnNotPrincipals(selector *api.AuthenticationPolicySelector, endpoint *endpoint.Endpoint) bool {
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

func matchAuthnPrincipals(selector *api.AuthenticationPolicySelector, endpoint *endpoint.Endpoint) bool {
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

func matchAuthnNotIPBlocks(selector *api.AuthenticationPolicySelector, endpoint *endpoint.Endpoint) bool {
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

func matchAuthnIPBlocks(selector *api.AuthenticationPolicySelector, endpoint *endpoint.Endpoint) bool {
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

func matchAuthnNotNamespace(selector *api.AuthenticationPolicySelector, endpoint *endpoint.Endpoint) bool {
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

func matchAuthnNamespace(selector *api.AuthenticationPolicySelector, endpoint *endpoint.Endpoint) bool {
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

func MatchAuthrSelector(target *api.AuthorizationPolicyTarget, endpoint *endpoint.Endpoint) bool {
	if endpoint == nil {
		return true
	}

	if !matchAuthrNamespace(target, endpoint) {
		return false
	}

	if !matchAuthrNotNamespace(target, endpoint) {
		return false
	}

	if !matchAuthrIPBlocks(target, endpoint) {
		return false
	}

	if !matchAuthrNotIPBlocks(target, endpoint) {
		return false
	}

	if !matchAuthrPrincipals(target, endpoint) {
		return false
	}

	if !matchAuthrNotPrincipals(target, endpoint) {
		return false
	}

	endpointJSON, err := json.Marshal(endpoint)
	if err != nil {
		logger.Sugar().Warnf("marshal endpoint failed, %v", err)
		return false
	}

	if !matchAuthrExtends(target, endpointJSON) {
		return false
	}

	return matchAuthrNotExtends(target, endpointJSON)
}

func matchAuthrNotExtends(target *api.AuthorizationPolicyTarget, endpointJSON []byte) bool {
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

func matchAuthrExtends(target *api.AuthorizationPolicyTarget, endpointJSON []byte) bool {
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

func matchAuthrNotPrincipals(target *api.AuthorizationPolicyTarget, endpoint *endpoint.Endpoint) bool {
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

func matchAuthrPrincipals(target *api.AuthorizationPolicyTarget, endpoint *endpoint.Endpoint) bool {
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

func matchAuthrNotIPBlocks(target *api.AuthorizationPolicyTarget, endpoint *endpoint.Endpoint) bool {
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

func matchAuthrIPBlocks(target *api.AuthorizationPolicyTarget, endpoint *endpoint.Endpoint) bool {
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

func matchAuthrNotNamespace(target *api.AuthorizationPolicyTarget, endpoint *endpoint.Endpoint) bool {
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

func matchAuthrNamespace(target *api.AuthorizationPolicyTarget, endpoint *endpoint.Endpoint) bool {
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
