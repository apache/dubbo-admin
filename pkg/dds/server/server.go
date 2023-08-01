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

package server

import (
	"fmt"
	"github.com/apache/dubbo-admin/api/dds"
	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/core/cert/provider"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	endpoint2 "github.com/apache/dubbo-admin/pkg/core/tools/endpoint"
	model2 "github.com/apache/dubbo-admin/pkg/dds/kube/crdclient"
	"github.com/apache/dubbo-admin/pkg/dds/storage"
	"google.golang.org/grpc/peer"
)

type DdsServer struct {
	dds.UnimplementedRuleServiceServer

	Config      *dubbo_cp.Config
	CertStorage provider.Storage
	CertClient  provider.Client
	Storage     *storage.Storage
	CrdClient   model2.ConfigStoreCache
}

func NewRuleServer(config *dubbo_cp.Config, crdclient model2.ConfigStoreCache) *DdsServer {
	return &DdsServer{
		Config:    config,
		CrdClient: crdclient,
	}
}

func (s *DdsServer) NeedLeaderElection() bool {
	return false
}

func (s *DdsServer) Observe(stream dds.RuleService_ObserveServer) error {
	c := &GrpcEndpointConnection{
		stream:   stream,
		stopChan: make(chan struct{}),
	}

	p, ok := peer.FromContext(stream.Context())
	if !ok {
		logger.Sugar().Errorf("failed to get peer from context")

		return fmt.Errorf("failed to get peer from context")
	}

	endpoint, err := endpoint2.ExactEndpoint(stream.Context(), s.CertStorage, s.Config, s.CertClient)
	if err != nil {
		logger.Sugar().Errorf("failed to get endpoint from context: %v. RemoteAddr: %s", err, p.Addr)

		return err
	}
	logger.Sugar().Infof("New observe storage from %s", endpoint)
	s.Storage.Connected(endpoint, c)

	<-c.stopChan
	return nil
}

func (s *DdsServer) Start(stop <-chan struct{}) error {
	s.CrdClient.Start(stop)
	return nil
}

type GrpcEndpointConnection struct {
	storage.EndpointConnection

	stream   dds.RuleService_ObserveServer
	stopChan chan struct{}
}

func (c *GrpcEndpointConnection) Send(r *dds.ObserveResponse) error {
	pbr := &dds.ObserveResponse{
		Nonce:    r.Nonce,
		Type:     r.Type,
		Revision: r.Revision,
		Data:     r.Data,
	}
	return c.stream.Send(pbr)
}

func (c *GrpcEndpointConnection) Recv() (*dds.ObserveRequest, error) {
	in, err := c.stream.Recv()
	if err != nil {
		return nil, err
	}
	return &dds.ObserveRequest{
		Nonce: in.Nonce,
		Type:  in.Type,
	}, nil
}

func (c *GrpcEndpointConnection) Disconnect() {
	c.stopChan <- struct{}{}
}
