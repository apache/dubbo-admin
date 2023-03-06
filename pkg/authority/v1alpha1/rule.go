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

package v1alpha1

import (
	"fmt"

	"github.com/apache/dubbo-admin/pkg/authority/config"
	"github.com/apache/dubbo-admin/pkg/authority/k8s"
	"github.com/apache/dubbo-admin/pkg/authority/rule/connection"
	"github.com/apache/dubbo-admin/pkg/logger"
	"google.golang.org/grpc/peer"
)

type ObserveServiceServerImpl struct {
	UnimplementedObserveServiceServer

	Options    *config.Options
	KubeClient k8s.Client
	Storage    *connection.Storage
}

func (s *ObserveServiceServerImpl) Observe(stream ObserveService_ObserveServer) error {
	c := &GrpcEndpointConnection{
		stream:   stream,
		stopChan: make(chan int, 1),
	}

	p, ok := peer.FromContext(stream.Context())
	if !ok {
		logger.Sugar().Errorf("failed to get peer from context")

		return fmt.Errorf("failed to get peer from context")
	}

	endpoint, err := exactEndpoint(stream.Context(), s.Options, s.KubeClient)
	if err != nil {
		logger.Sugar().Errorf("failed to get endpoint from context: %v. RemoteAddr: %s", err, p.Addr)

		return err
	}

	logger.Sugar().Infof("New observe connection from %s", endpoint)
	s.Storage.Connected(endpoint, c)

	<-c.stopChan
	return nil
}

type GrpcEndpointConnection struct {
	connection.EndpointConnection

	stream   ObserveService_ObserveServer
	stopChan chan int
}

func (c *GrpcEndpointConnection) Send(r *connection.ObserveResponse) error {
	pbr := &ObserveResponse{
		Nonce:    r.Nonce,
		Type:     r.Type,
		Revision: r.Data.Revision(),
		Data:     r.Data.Data(),
	}

	return c.stream.Send(pbr)
}

func (c *GrpcEndpointConnection) Recv() (*connection.ObserveRequest, error) {
	in, err := c.stream.Recv()
	if err != nil {
		return nil, err
	}

	return &connection.ObserveRequest{
		Nonce: in.Nonce,
		Type:  in.Type,
	}, nil
}

func (c *GrpcEndpointConnection) Disconnect() {
	c.stopChan <- 0
}
