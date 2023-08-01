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
	"github.com/apache/dubbo-admin/api/dds"
	api "github.com/apache/dubbo-admin/api/resource/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/model"
	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/types/known/anypb"

	"net/netip"
	"strings"

	gvks "github.com/apache/dubbo-admin/pkg/core/schema/gvk"
	"github.com/pkg/errors"
	"io"
	"k8s.io/client-go/util/workqueue"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Storage struct {
	Mutex      *sync.RWMutex
	Connection []*Connection

	LatestRules map[string]*Origin
}

func TypeSupported(gvk string) bool {
	return gvk == gvks.Authentication ||
		gvk == gvks.Authorization ||
		gvk == gvks.ServiceMapping ||
		gvk == gvks.TagRoute ||
		gvk == gvks.DynamicConfig ||
		gvk == gvks.ConditionRoute
}

func NewStorage() *Storage {
	return &Storage{
		Mutex:       &sync.RWMutex{},
		Connection:  []*Connection{},
		LatestRules: map[string]*Origin{},
	}
}

func (s *Storage) Connected(endpoint *endpoint.Endpoint, connection EndpointConnection) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	c := &Connection{
		mutex:              &sync.RWMutex{},
		status:             Connected,
		EndpointConnection: connection,
		Endpoint:           endpoint,
		TypeListened:       map[string]bool{},
		RawRuleQueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "raw-dds"),
		ExpectedRules:      map[string]*VersionedRule{},
		ClientRules:        map[string]*ClientStatus{},
	}

	s.Connection = append(s.Connection, c)

	go s.listenConnection(c)
	go c.listenRule()
}

func (s *Storage) listenConnection(c *Connection) {
	for {
		if c.status == Disconnected {
			return
		}

		req, err := c.EndpointConnection.Recv()

		if errors.Is(err, io.EOF) {
			logger.Sugar().Infof("Observe storage closed. Connection ID: %s", c.Endpoint.ID)
			s.Disconnect(c)

			return
		}

		if err != nil {
			logger.Sugar().Warnf("Observe storage error: %v. Connection ID: %s", err, c.Endpoint.ID)
			s.Disconnect(c)

			return
		}

		s.HandleRequest(c, req)
	}
}

func (s *Storage) HandleRequest(c *Connection, req *dds.ObserveRequest) {
	if req.Type == "" {
		logger.Sugar().Errorf("Empty request type from %v", c.Endpoint.ID)

		return
	}

	if !TypeSupported(req.Type) {
		logger.Sugar().Errorf("Unsupported request type %s from %s", req.Type, c.Endpoint.ID)

		return
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if req.Nonce != "" {
		cr := c.ClientRules[req.Type]

		if cr == nil {
			logger.Sugar().Errorf("Unexpected request type %s with nonce %s from %s", req.Type, req.Nonce, c.Endpoint.ID)
			return
		}

		if cr.PushingStatus == Pushing {
			if cr.LastPushNonce != req.Nonce {
				logger.Sugar().Errorf("Unexpected request nonce %s from %s", req.Nonce, c.Endpoint.ID)

				return
			}

			cr.ClientVersion = cr.LastPushedVersion

			cr.PushingStatus = Pushed
			logger.Sugar().Infof("Client %s pushed %s dds %s success", c.Endpoint.Ips, req.Type, cr.ClientVersion.Revision)
		}
		return
	}

	if _, ok := c.TypeListened[req.Type]; !ok {
		logger.Sugar().Infof("Client %s listen %s dds", c.Endpoint.Ips, req.Type)
		c.TypeListened[req.Type] = true
		c.ClientRules[req.Type] = &ClientStatus{
			PushingStatus: Pushed,
			NonceInc:      0,
			ClientVersion: &VersionedRule{
				Revision: -1,
				Type:     req.Type,
			},
			LastPushedTime:    0,
			LastPushedVersion: nil,
			LastPushNonce:     "",
		}
		latestRule := s.LatestRules[req.Type]
		if latestRule != nil {
			c.RawRuleQueue.Add(latestRule)
		}
	}
}

func (c *Connection) listenRule() {
	for {
		obj, shutdown := c.RawRuleQueue.Get()
		if shutdown {
			return
		}

		func(obj interface{}) {
			defer c.RawRuleQueue.Done(obj)

			var key *Origin

			var ok bool

			if key, ok = obj.(*Origin); !ok {
				logger.Sugar().Errorf("expected dds.Origin in workqueue but got %#v", obj)

				return
			}

			if err := c.handleRule(key); err != nil {
				logger.Sugar().Errorf("error syncing '%s': %s", key, err.Error())

				return
			}

			logger.Sugar().Infof("Successfully synced '%s'", key)
		}(obj)
	}
}

func (c *Connection) handleRule(rawRule *Origin) error {
	targetRule, err := rawRule.Exact(rawRule.Gvk, c.Endpoint)
	if err != nil {
		return err
	}

	if _, ok := c.TypeListened[targetRule.Type]; !ok {
		return nil
	}

	cr := c.ClientRules[targetRule.Type]

	for cr.PushingStatus == Pushing {
		cr.PushQueued = true
		time.Sleep(1 * time.Second)
		logger.Sugar().Infof("Client %s %s dds is pushing, wait for 1 second", c.Endpoint.Ips, targetRule.Type)
	}
	cr.PushQueued = false

	if cr.ClientVersion.Data != nil &&
		(reflect.DeepEqual(cr.ClientVersion.Data, targetRule.Data) || cr.ClientVersion.Revision >= targetRule.Revision) {
		logger.Sugar().Infof("Client %s %s dds is up to date", c.Endpoint.Ips, targetRule.Type)
		return nil
	}
	newVersion := atomic.AddInt64(&cr.NonceInc, 1)
	r := &dds.ObserveResponse{
		Nonce:    strconv.FormatInt(newVersion, 10),
		Type:     targetRule.Type,
		Revision: targetRule.Revision,
		Data:     targetRule.Data,
	}
	cr.LastPushedTime = time.Now().Unix()
	cr.LastPushedVersion = targetRule
	cr.LastPushNonce = r.Nonce
	cr.PushingStatus = Pushing

	logger.Sugar().Infof("Receive new version dds. Client %s %s dds is pushing.", c.Endpoint.Ips, targetRule.Type)

	return c.EndpointConnection.Send(r)
}

func (s *Storage) Disconnect(c *Connection) {
	for i, sc := range s.Connection {
		if sc == c {
			s.Connection = append(s.Connection[:i], s.Connection[i+1:]...)
			break
		}
	}

	c.EndpointConnection.Disconnect()
	c.RawRuleQueue.ShutDown()
}

type PushingStatus int

const (
	Pushed PushingStatus = iota
	Pushing
)

type ConnectionStatus int

const (
	Connected ConnectionStatus = iota
	Disconnected
)

type Connection struct {
	mutex              *sync.RWMutex
	status             ConnectionStatus
	EndpointConnection EndpointConnection
	Endpoint           *endpoint.Endpoint

	TypeListened map[string]bool

	RawRuleQueue  workqueue.RateLimitingInterface
	ExpectedRules map[string]*VersionedRule
	ClientRules   map[string]*ClientStatus
}

type EndpointConnection interface {
	Send(*dds.ObserveResponse) error
	Recv() (*dds.ObserveRequest, error)
	Disconnect()
}

type VersionedRule struct {
	Revision int64
	Type     string
	Data     []*anypb.Any
}

type ClientStatus struct {
	PushQueued    bool
	PushingStatus PushingStatus

	NonceInc int64

	ClientVersion *VersionedRule

	LastPushedTime    int64
	LastPushedVersion *VersionedRule
	LastPushNonce     string
}

type Origin struct {
	Gvk      string
	Revision int64
	Data     []model.Config
}

func (o *Origin) Exact(gvk string, endpoint *endpoint.Endpoint) (*VersionedRule, error) {
	res := make([]*anypb.Any, len(o.Data))
	if gvk == gvks.Authorization {
		for _, v := range o.Data {
			deepCopy := v.DeepCopy()
			policy := deepCopy.Spec.(*api.AuthorizationPolicy)
			if policy.GetRules() != nil {
				match := true
				for _, policyRule := range policy.Rules {
					policyRule.To = nil
					if !matchSelector(policyRule.To, endpoint) {
						match = false
						break
					}
				}
				if !match {
					continue
				}
				gogo, err := model.ToProtoGogo(deepCopy)
				if err != nil {
					return nil, err
				}
				res = append(res, gogo)
			}
		}
	} else if gvk == gvks.Authentication {
		for _, v := range o.Data {
			deepCopy := v.DeepCopy()
			policy := deepCopy.Spec.(*api.AuthenticationPolicy)
			if policy.GetSelector() != nil {
				match := true
				for _, selector := range policy.Selector {
					if !matchAuthnSelector(selector, endpoint) {
						match = false
						break
					}
				}
				if !match {
					continue
				}
			}
			policy.Selector = nil
			gogo, err := model.ToProtoGogo(deepCopy)
			if err != nil {
				return nil, err
			}
			res = append(res, gogo)
		}
	} else {
		for _, data := range o.Data {
			gogo, err := model.ToProtoGogo(data.Spec)
			if err != nil {
				return nil, err
			}
			res = append(res, gogo)
		}
	}
	return &VersionedRule{
		Revision: o.Revision,
		Type:     o.Gvk,
		Data:     res,
	}, nil
}

func matchAuthnSelector(selector *api.AuthenticationPolicySelector, endpoint *endpoint.Endpoint) bool {
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

func matchSelector(target *api.AuthorizationPolicyTarget, endpoint *endpoint.Endpoint) bool {
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

func matchNotExtends(target *api.AuthorizationPolicyTarget, endpointJSON []byte) bool {
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

func matchExtends(target *api.AuthorizationPolicyTarget, endpointJSON []byte) bool {
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

func matchNotPrincipals(target *api.AuthorizationPolicyTarget, endpoint *endpoint.Endpoint) bool {
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

func matchPrincipals(target *api.AuthorizationPolicyTarget, endpoint *endpoint.Endpoint) bool {
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

func matchNotIPBlocks(target *api.AuthorizationPolicyTarget, endpoint *endpoint.Endpoint) bool {
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

func matchIPBlocks(target *api.AuthorizationPolicyTarget, endpoint *endpoint.Endpoint) bool {
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

func matchNotNamespace(target *api.AuthorizationPolicyTarget, endpoint *endpoint.Endpoint) bool {
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

func matchNamespace(target *api.AuthorizationPolicyTarget, endpoint *endpoint.Endpoint) bool {
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
