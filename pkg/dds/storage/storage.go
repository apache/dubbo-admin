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
	"io"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/apache/dubbo-admin/api/dds"
	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/model"
	gvks "github.com/apache/dubbo-admin/pkg/core/schema/gvk"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/anypb"

	"k8s.io/client-go/util/workqueue"
)

type Storage struct {
	Mutex      *sync.RWMutex
	Connection []*Connection
	Config     *dubbo_cp.Config
	Generators map[string]DdsResourceGenerator

	LatestRules map[string]Origin
}

func TypeSupported(gvk string) bool {
	return gvk == gvks.AuthenticationPolicy ||
		gvk == gvks.AuthorizationPolicy ||
		gvk == gvks.ServiceNameMapping ||
		gvk == gvks.TagRoute ||
		gvk == gvks.DynamicConfig ||
		gvk == gvks.ConditionRoute
}

func NewStorage(cfg *dubbo_cp.Config) *Storage {
	s := &Storage{
		Mutex:       &sync.RWMutex{},
		Connection:  []*Connection{},
		LatestRules: map[string]Origin{},
		Config:      cfg,
		Generators:  map[string]DdsResourceGenerator{},
	}
	s.Generators[gvks.AuthenticationPolicy] = &AuthenticationGenerator{}
	s.Generators[gvks.AuthorizationPolicy] = &AuthorizationGenerator{}
	s.Generators[gvks.ServiceNameMapping] = &ServiceMappingGenerator{}
	s.Generators[gvks.ConditionRoute] = &ConditionRoutesGenerator{}
	s.Generators[gvks.TagRoute] = &TagRoutesGenerator{}
	s.Generators[gvks.DynamicConfig] = &DynamicConfigsGenerator{}
	return s
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
		blockedPushedMutex: &sync.RWMutex{},
		Generator:          s.Generators,
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
		logger.Sugar().Errorf("[DDS] Empty request type from %v", c.Endpoint.ID)

		return
	}

	if !TypeSupported(req.Type) {
		logger.Sugar().Errorf("[DDS] Unsupported request type %s from %s", req.Type, c.Endpoint.ID)

		return
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if req.Nonce != "" {
		// It is an ACK
		cr := c.ClientRules[req.Type]

		if cr == nil {
			logger.Sugar().Errorf("[DDS] Unexpected request type %s with nonce %s from %s", req.Type, req.Nonce, c.Endpoint.ID)
			return
		}

		if cr.PushingStatus == Pushing {
			if cr.LastPushNonce != req.Nonce {
				logger.Sugar().Errorf("[DDS] Unexpected request nonce %s from %s", req.Nonce, c.Endpoint.ID)

				return
			}

			cr.ClientVersion = cr.LastPushedVersion

			cr.PushingStatus = Pushed
			logger.Sugar().Infof("[DDS] Client %s pushed %s dds %d success", c.Endpoint.Ips, req.Type, cr.ClientVersion.Revision)
		}
		return
	}

	if _, ok := c.TypeListened[req.Type]; !ok {
		logger.Sugar().Infof("[DDS] Client %s listen %s dds", c.Endpoint.Ips, req.Type)
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

			var key Origin

			var ok bool

			if key, ok = obj.(Origin); !ok {
				logger.Sugar().Errorf("[DDS] expected dds.Origin in workqueue but got %#v", obj)

				return
			}

			if err := c.handleRule(key); err != nil {
				logger.Sugar().Errorf("[DDS] error syncing '%s': %s", key, err.Error())

				return
			}

			logger.Sugar().Infof("[DDS] Successfully synced '%s'", key)
		}(obj)
	}
}

func (c *Connection) handleRule(rawRule Origin) error {
	targetRule, err := rawRule.Exact(c.Generator, c.Endpoint)
	if err != nil {
		return err
	}

	if _, ok := c.TypeListened[targetRule.Type]; !ok {
		return nil
	}

	cr := c.ClientRules[targetRule.Type]

	// TODO how to improve this one
	for cr.PushingStatus == Pushing {
		cr.PushQueued = true
		time.Sleep(1 * time.Second)
		logger.Sugar().Infof("[DDS] Client %s %s rule is pushing, wait for 1 second", c.Endpoint.Ips, targetRule.Type)
	}
	cr.PushQueued = false

	if cr.ClientVersion.Data != nil &&
		(reflect.DeepEqual(cr.ClientVersion.Data, targetRule.Data) || cr.ClientVersion.Revision >= targetRule.Revision) {
		logger.Sugar().Infof("[DDS] Client %s %s dds is up to date", c.Endpoint.Ips, targetRule.Type)
		return nil
	}
	newVersion := atomic.AddInt64(&cr.NonceInc, 1)
	r := &dds.ObserveResponse{
		Nonce:    strconv.FormatInt(newVersion, 10),
		Type:     targetRule.Type,
		Revision: targetRule.Revision,
		Data:     targetRule.Data,
	}

	logger.Sugar().Infof("[DDS] Receive new version dds. Client %s %s dds is pushing.", c.Endpoint.Ips, targetRule.Type)

	return c.EndpointConnection.Send(targetRule, cr, r)
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
	Generator          map[string]DdsResourceGenerator
	mutex              *sync.RWMutex
	status             ConnectionStatus
	EndpointConnection EndpointConnection
	Endpoint           *endpoint.Endpoint

	TypeListened map[string]bool

	RawRuleQueue  workqueue.RateLimitingInterface
	ExpectedRules map[string]*VersionedRule
	ClientRules   map[string]*ClientStatus

	blockedPushedMutex *sync.RWMutex
}

type EndpointConnection interface {
	Send(*VersionedRule, *ClientStatus, *dds.ObserveResponse) error
	Recv() (*dds.ObserveRequest, error)
	Disconnect()
}

type VersionedRule struct {
	Revision int64
	Type     string
	Data     []*anypb.Any
}

type ClientStatus struct {
	sync.RWMutex
	PushQueued    bool
	PushingStatus PushingStatus

	NonceInc int64

	ClientVersion *VersionedRule

	LastPushedTime    int64
	LastPushedVersion *VersionedRule
	LastPushNonce     string
}

type Origin interface {
	Type() string
	Exact(gen map[string]DdsResourceGenerator, endpoint *endpoint.Endpoint) (*VersionedRule, error)
	Revision() int64
}

type OriginImpl struct {
	Gvk  string
	Rev  int64
	Data []model.Config
}

func (o *OriginImpl) Revision() int64 {
	return o.Rev
}

func (o *OriginImpl) Type() string {
	return o.Gvk
}

func (o *OriginImpl) Exact(gen map[string]DdsResourceGenerator, endpoint *endpoint.Endpoint) (*VersionedRule, error) {
	gvk := o.Type()
	g := gen[gvk]
	res, err := g.Generate(o.Data, endpoint)
	if err != nil {
		return nil, err
	}
	return &VersionedRule{
		Revision: o.Rev,
		Type:     o.Gvk,
		Data:     res,
	}, nil
}
