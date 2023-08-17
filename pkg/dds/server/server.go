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
	"time"

	"github.com/apache/dubbo-admin/api/dds"
	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/core/cert/provider"
	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	endpoint2 "github.com/apache/dubbo-admin/pkg/core/tools/endpoint"
	model2 "github.com/apache/dubbo-admin/pkg/dds/kube/crdclient"
	"github.com/apache/dubbo-admin/pkg/dds/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type DdsServer struct {
	dds.UnimplementedRuleServiceServer

	Config      *dubbo_cp.Config
	CertStorage *provider.CertStorage
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
		stream:      stream,
		stopChan:    make(chan struct{}),
		sendTimeout: s.Config.Options.SendTimeout,
	}

	p, ok := peer.FromContext(stream.Context())
	if !ok {
		logger.Sugar().Errorf("[DDS] failed to get peer from context")

		return fmt.Errorf("failed to get peer from context")
	}

	endpoints, err := endpoint2.ExactEndpoint(stream.Context(), s.CertStorage, s.Config, s.CertClient)
	if err != nil {
		logger.Sugar().Errorf("[DDS] failed to get endpoint from context: %v. RemoteAddr: %s", err, p.Addr)

		return err
	}
	c.endpoint = endpoints
	logger.Sugar().Infof("[DDS] New observe storage from %s", endpoints)
	s.Storage.Connected(endpoints, c)

	<-c.stopChan
	return nil
}

func (s *DdsServer) Start(stop <-chan struct{}) error {
	return s.CrdClient.Start(stop)
}

type GrpcEndpointConnection struct {
	storage.EndpointConnection

	sendTimeout time.Duration
	stream      dds.RuleService_ObserveServer
	endpoint    *endpoint.Endpoint
	stopChan    chan struct{}
}

// Send with timeout
func (c *GrpcEndpointConnection) Send(targetRule *storage.VersionedRule, cr *storage.ClientStatus, r *dds.ObserveResponse) error {
	errChan := make(chan error, 1)

	// sendTimeout may be modified via environment
	t := time.NewTimer(c.sendTimeout)
	go func() {
		errChan <- c.stream.Send(&dds.ObserveResponse{
			Nonce:    r.Nonce,
			Type:     r.Type,
			Revision: r.Revision,
			Data:     r.Data,
		})
		close(errChan)
	}()

	select {
	case <-t.C:
		logger.Infof("[DDS] Timeout writing %s", c.endpoint.ID)
		return status.Errorf(codes.DeadlineExceeded, "timeout sending")
	case err := <-errChan:
		if err == nil {
			cr.Lock()
			cr.LastPushedTime = time.Now().Unix()
			cr.LastPushedVersion = targetRule
			cr.LastPushNonce = r.Nonce
			cr.PushingStatus = storage.Pushing
			cr.Unlock()
		}
		// To ensure the channel is empty after a call to Stop, check the
		// return value and drain the channel (from Stop docs).
		if !t.Stop() {
			<-t.C
		}
		return err
	}
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
