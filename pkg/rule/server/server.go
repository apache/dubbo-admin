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

package server

import (
	"fmt"
	informerclient "github.com/apache/dubbo-admin/pkg/rule/clientgen/clientset/versioned"

	"github.com/apache/dubbo-admin/api/mesh"
	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"

	"github.com/apache/dubbo-admin/pkg/core/cert/provider"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/tools/endpoint"
	informFactory "github.com/apache/dubbo-admin/pkg/rule/clientgen/informers/externalversions"
	"github.com/apache/dubbo-admin/pkg/rule/crd"
	"github.com/apache/dubbo-admin/pkg/rule/storage"
	"google.golang.org/grpc/peer"
)

type RuleServer struct {
	mesh.UnimplementedRuleServiceServer

	Options         *dubbo_cp.Config
	CertStorage     provider.Storage
	CertClient      provider.Client
	InformerClient  *informerclient.Clientset
	Storage         *storage.Storage
	Controller      *crd.Controller
	InformerFactory informFactory.SharedInformerFactory
}

func NewRuleServer(options *dubbo_cp.Config) *RuleServer {
	return &RuleServer{
		Options: options,
	}
}

func (s *RuleServer) NeedLeaderElection() bool {
	return false
}

func (s *RuleServer) Start(stop <-chan struct{}) error {
	s.InformerFactory.Start(stop)
	s.Controller.WaitSynced(stop)
	s.Controller.Queue.Run(stop)
	return nil
}

func (s *RuleServer) Observe(stream mesh.RuleService_ObserveServer) error {
	c := &GrpcEndpointConnection{
		stream:   stream,
		stopChan: make(chan struct{}, 1),
	}

	p, ok := peer.FromContext(stream.Context())
	if !ok {
		logger.Sugar().Errorf("failed to get peer from context")

		return fmt.Errorf("failed to get peer from context")
	}

	endpoint, err := endpoint.ExactEndpoint(stream.Context(), s.CertStorage, s.Options, s.CertClient)
	if err != nil {
		logger.Sugar().Errorf("failed to get endpoint from context: %v. RemoteAddr: %s", err, p.Addr)

		return err
	}

	logger.Sugar().Infof("New observe storage from %s", endpoint)
	s.Storage.Connected(endpoint, c)

	<-c.stopChan
	return nil
}

type GrpcEndpointConnection struct {
	storage.EndpointConnection

	stream   mesh.RuleService_ObserveServer
	stopChan chan struct{}
}

func (c *GrpcEndpointConnection) Send(r *storage.ObserveResponse) error {
	pbr := &mesh.ObserveResponse{
		Nonce:    r.Nonce,
		Type:     r.Type,
		Revision: r.Data.Revision(),
		Data:     r.Data.Data(),
	}

	return c.stream.Send(pbr)
}

func (c *GrpcEndpointConnection) Recv() (*storage.ObserveRequest, error) {
	in, err := c.stream.Recv()
	if err != nil {
		return nil, err
	}

	return &storage.ObserveRequest{
		Nonce: in.Nonce,
		Type:  in.Type,
	}, nil
}

func (c *GrpcEndpointConnection) Disconnect() {
	c.stopChan <- struct{}{}
}
