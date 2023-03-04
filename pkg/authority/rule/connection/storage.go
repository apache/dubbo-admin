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

package connection

import (
	"github.com/apache/dubbo-admin/pkg/authority/rule"
	"github.com/apache/dubbo-admin/pkg/logger"
	"io"
	"k8s.io/client-go/util/workqueue"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Storage struct {
	Mutex      *sync.RWMutex
	Connection []*Connection

	LatestRules map[string]rule.Origin
}

func NewStorage() *Storage {
	return &Storage{
		Mutex:       &sync.RWMutex{},
		Connection:  []*Connection{},
		LatestRules: map[string]rule.Origin{},
	}
}

func (s *Storage) Connected(endpoint *rule.Endpoint, connection EndpointConnection) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	c := &Connection{
		mutex:              &sync.RWMutex{},
		status:             Connected,
		EndpointConnection: connection,
		Endpoint:           endpoint,
		RawRuleQueue:       workqueue.NewNamed("raw-rule"),
		TypeListened:       map[string]bool{},
		ExpectedRules:      map[string]*VersionedRule{},
		ClientRules:        map[string]*ClientStatus{},
	}
	s.Connection = append(s.Connection, c)

	go s.ListenConnection(c)
	go c.ListenRule()
}

func (s *Storage) ListenConnection(c *Connection) {
	for {
		if c.status == Disconnected {
			return
		}
		req, err := c.EndpointConnection.Recv()
		if err == io.EOF {
			logger.Sugar.Infof("Observe connection closed. Connection ID: %s", c.Endpoint.ID)
			s.Disconnect(c)
			return
		}
		if err != nil {
			logger.Sugar.Warnf("Observe connection error: %v. Connection ID: %s", err, c.Endpoint.ID)
			s.Disconnect(c)
			return
		}
		s.HandleRequest(c, req)
	}
}

func (s *Storage) ObserveRule(c *Connection) {
	for {
		if c.status == Disconnected {
			return
		}

	}
}

func (s *Storage) HandleRequest(c *Connection, req *ObserveRequest) {
	if req.Type == "" {
		logger.Sugar.Errorf("Empty request type from %v", c.Endpoint.ID)
		return
	}
	if !TypeSupported(req.Type) {
		logger.Sugar.Errorf("Unsupported request type %s from %s", req.Type, c.Endpoint.ID)
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if req.Nonce != "" {
		cr := c.ClientRules[req.Type]
		if cr == nil {
			logger.Sugar.Errorf("Unexpected request type %s with nonce %s from %s", req.Type, req.Nonce, c.Endpoint.ID)
			return
		}

		if cr.PushingStatus == Pushing {
			if cr.LastPushNonce != req.Nonce {
				logger.Sugar.Errorf("Unexpected request nonce %s from %s", req.Nonce, c.Endpoint.ID)
				return
			}
			cr.ClientVersion = cr.LastPushedVersion
			cr.PushingStatus = Pushed
			logger.Sugar.Infof("Client %s pushed %s rule %s success", c.Endpoint.Ips, req.Type, cr.ClientVersion.Revision)
		}
		return
	}

	if _, ok := c.TypeListened[req.Type]; !ok {
		logger.Sugar.Infof("Client %s listen %s rule", c.Endpoint.Ips, req.Type)
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

func (c *Connection) ListenRule() {
	for {
		obj, shutdown := c.RawRuleQueue.Get()
		if shutdown {
			return
		}

		func(obj interface{}) {
			defer c.RawRuleQueue.Done(obj)

			var key rule.Origin
			var ok bool
			if key, ok = obj.(rule.Origin); !ok {
				logger.Sugar.Errorf("expected rule.Origin in workqueue but got %#v", obj)
				return
			}

			if err := c.handleRule(key); err != nil {
				logger.Sugar.Errorf("error syncing '%s': %s", key, err.Error())
				return
			}

			logger.Sugar.Infof("Successfully synced '%s'", key)
		}(obj)
	}
}

func (c *Connection) handleRule(rawRule rule.Origin) error {
	targetRule, err := rawRule.Exact(c.Endpoint)
	if err != nil {
		return err
	}

	if _, ok := c.TypeListened[targetRule.Type()]; !ok {
		return nil
	}

	cr := c.ClientRules[targetRule.Type()]

	if cr.ClientVersion.Data != nil &&
		(cr.ClientVersion.Data.Data() == targetRule.Data() || cr.ClientVersion.Data.Revision() < targetRule.Revision()) {
		logger.Sugar.Infof("Client %s %s rule is up to date", c.Endpoint.Ips, targetRule.Type())
		return nil
	}

	for cr.PushingStatus == Pushing {
		time.Sleep(1 * time.Second)
		logger.Sugar.Infof("Client %s %s rule is pushing, wait for 1 second", c.Endpoint.Ips, targetRule.Type())
	}

	newVersion := atomic.AddInt64(&cr.NonceInc, 1)
	r := &ObserveResponse{
		Nonce: strconv.FormatInt(newVersion, 10),
		Type:  targetRule.Type(),
		Data:  targetRule,
	}

	cr.LastPushedTime = time.Now().Unix()
	cr.LastPushedVersion = &VersionedRule{
		Type:     targetRule.Type(),
		Revision: targetRule.Revision(),
		Data:     targetRule,
	}
	cr.LastPushNonce = r.Nonce
	cr.PushingStatus = Pushing

	logger.Sugar.Infof("Receive new version rule. Client %s %s rule is pushing.", c.Endpoint.Ips, targetRule.Type())

	return c.EndpointConnection.Send(r)
}

func TypeSupported(t string) bool {
	return t == "authentication/v1beta1" || t == "authorization/v1beta1"
}

func (s *Storage) Disconnect(c *Connection) {
	for i, sc := range s.Connection {
		if sc == c {
			s.Connection = append(s.Connection[:i], s.Connection[i+1:]...)
			return
		}
	}
	c.EndpointConnection.Disconnect()
}

type Connection struct {
	mutex              *sync.RWMutex
	status             ConnectionStatus
	EndpointConnection EndpointConnection
	Endpoint           *rule.Endpoint

	TypeListened map[string]bool

	RawRuleQueue  workqueue.Interface
	ExpectedRules map[string]*VersionedRule
	ClientRules   map[string]*ClientStatus
}

type VersionedRule struct {
	Revision int64         `json:"revision,omitempty"`
	Type     string        `json:"type,omitempty"`
	Data     rule.ToClient `json:"data,omitempty"`
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

type ClientStatus struct {
	PushingStatus PushingStatus `json:"pushingStatus,omitempty"`

	NonceInc int64 `json:"version,omitempty"`

	ClientVersion *VersionedRule `json:"clientVersion,omitempty"`

	LastPushedTime    int64          `json:"lastPushedTime,omitempty"`
	LastPushedVersion *VersionedRule `json:"lastPushedData,omitempty"`
	LastPushNonce     string         `json:"lastPushNonce,omitempty"`
}

type ObserveResponse struct {
	Nonce string        `json:"nonce,omitempty"`
	Type  string        `json:"type,omitempty"`
	Data  rule.ToClient `json:"data,omitempty"`
}

type ObserveRequest struct {
	Nonce string `json:"nonce,omitempty"`
	Type  string `json:"type,omitempty"`
}

type EndpointConnection interface {
	Send(*ObserveResponse) error
	Recv() (*ObserveRequest, error)
	Disconnect()
}
