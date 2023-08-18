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

package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/apache/dubbo-admin/api/mesh"
	gvks "github.com/apache/dubbo-admin/pkg/core/schema/gvk"

	"github.com/apache/dubbo-admin/api/dds"
	api "github.com/apache/dubbo-admin/api/resource/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/cenkalti/backoff"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
)

var (
	// use plain server to test
	grpcAddr         = "127.0.0.1:30060"
	grpcUpstreamAddr = grpcAddr
)

type Config struct {
	// InitialDiscoveryRequests is a list of resources to watch at first, represented as URLs (for new DDS resource naming)
	// or type URLs.
	InitialDiscoveryRequest []*dds.ObserveRequest
	// BackoffPolicy determines the reconnect policy. Based on ddsclient.
	BackoffPolicy backoff.BackOff
	GrpcOpts      []grpc.DialOption

	Namespace string

	// It is sent by ddsclient, must match a known endpoint IP.
	IP string
}

// DDSC implements a basic ddsclient for DDS, for use in stress tests and tools
// or libraries that need to connect to Dubbo admin or other DDS servers.
// Currently only for testing!
type DDSC struct {
	// Stream is the GRPC connection stream, allowing direct GRPC send operations.
	// Set after Dial is called.
	stream dds.RuleService_ObserveClient
	// dds ddsclient used to create a stream
	ddsclient dds.RuleServiceClient
	snpclient mesh.ServiceNameMappingServiceClient
	conn      *grpc.ClientConn

	// Indicates if the DDSC ddsclient is closed
	closed bool

	// NodeID is the node identity sent to Admin
	nodeID string

	url string

	authentication []*api.AuthenticationPolicyToClient
	authorization  []*api.AuthorizationPolicyToClient
	conditionRoute []*api.ConditionRouteToClient
	tagRoute       []*api.TagRouteToClient
	dynamicConfig  []*api.DynamicConfigToClient
	serviceMapping []*api.ServiceNameMappingToClient

	// Last received message, by type
	Received map[string]*dds.ObserveResponse

	mutex sync.RWMutex

	// RecvWg is for letting goroutines know when the goroutine handling the DDS stream finishes.
	RecvWg sync.WaitGroup

	cfg *Config
}

func New(discoveryAddr string, opts *Config) (*DDSC, error) {
	if opts == nil {
		opts = &Config{}
	}
	// We want to recreate stream
	if opts.BackoffPolicy == nil {
		opts.BackoffPolicy = backoff.NewExponentialBackOff()
	}
	ddsc := &DDSC{
		url:      discoveryAddr,
		cfg:      opts,
		Received: map[string]*dds.ObserveResponse{},
		RecvWg:   sync.WaitGroup{},
	}

	if opts.IP == "" {
		opts.IP = getPrivateIPIfAvailable().String()
	}

	ddsc.nodeID = fmt.Sprintf("%s~%s", opts.IP, opts.Namespace)

	if err := ddsc.Dial(); err != nil {
		return nil, err
	}
	return ddsc, nil
}

// Dial connects to a dds server
// nolint
func (a *DDSC) Dial() error {
	opts := a.cfg
	var err error
	grpcDialOptions := opts.GrpcOpts
	if len(grpcDialOptions) == 0 {
		// Only disable transport security if the user didn't supply custom dial options
		grpcDialOptions = append(grpcDialOptions, grpc.WithInsecure())
	}

	a.conn, err = grpc.Dial(a.url, grpcDialOptions...)
	if err != nil {
		return err
	}
	return nil
}

func getPrivateIPIfAvailable() net.IP {
	addrs, _ := net.InterfaceAddrs()
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		default:
			continue
		}
		if !ip.IsLoopback() {
			return ip
		}
	}
	return net.IPv4zero
}

// reconnect will create a new stream
func (a *DDSC) reconnect() {
	a.mutex.RLock()
	if a.closed {
		a.mutex.RUnlock()
		return
	}
	a.mutex.RUnlock()

	err := a.Run()
	if err == nil {
		a.cfg.BackoffPolicy.Reset()
	} else {
		time.AfterFunc(a.cfg.BackoffPolicy.NextBackOff(), a.reconnect)
	}
}

func (a *DDSC) Run() error {
	var err error
	a.ddsclient = dds.NewRuleServiceClient(a.conn)
	a.snpclient = mesh.NewServiceNameMappingServiceClient(a.conn)
	a.stream, err = a.ddsclient.Observe(context.Background())
	if err != nil {
		return err
	}
	// Send the snp message
	a.sendSnp()
	// Send the initial requests
	for _, r := range a.cfg.InitialDiscoveryRequest {
		err := a.Send(r)
		if err != nil {
			return err
		}
	}
	// by default, we assume 1 goroutine decrements the waitgroup (go a.handleRecv()).
	// for synchronizing when the goroutine finishes reading from the gRPC stream.
	a.RecvWg.Add(1)
	go a.handleRecv()
	return nil
}

func (a *DDSC) sendSnp() {
	res, err := a.snpclient.RegisterServiceAppMapping(context.Background(), &mesh.ServiceMappingRequest{
		Namespace:       "dubbo-system",
		ApplicationName: "test-app",
		InterfaceNames: []string{
			"test-interface1",
			"test-interface2",
		},
	})
	if err != nil || !res.Success {
		a.sendSnp()
	}
}

// Send Raw send of request
func (a *DDSC) Send(req *dds.ObserveRequest) error {
	return a.stream.Send(req)
}

func (a *DDSC) handleRecv() {
	for {
		var err error
		msg, err := a.stream.Recv()
		if err != nil {
			a.RecvWg.Done()
			logger.Sugar().Infof("Connection closed for node %v with err: %v", a.nodeID, err)
			// if 'reconnect' enabled - schedule a new Run
			if a.cfg.BackoffPolicy != nil {
				time.AfterFunc(a.cfg.BackoffPolicy.NextBackOff(), a.reconnect)
			} else {
				a.Close()
			}
			return
		}
		logger.Sugar().Info("Received ", a.url, " type ", msg.Type,
			"nonce= ", msg.Nonce)

		// Process the resources
		var authentication []*api.AuthenticationPolicyToClient
		var authorization []*api.AuthorizationPolicyToClient
		var serviceMapping []*api.ServiceNameMappingToClient
		var conditionRoute []*api.ConditionRouteToClient
		var tagRoute []*api.TagRouteToClient
		var dynamicConfig []*api.DynamicConfigToClient
		switch msg.Type {
		case gvks.AuthenticationPolicy:
			for _, d := range msg.Data {
				valBytes := d.Value
				auth := &api.AuthenticationPolicyToClient{}
				err := proto.Unmarshal(valBytes, auth)
				if err != nil {
					return
				}
				authentication = append(authentication, auth)
				a.handleAuthentication(authentication)
			}
		case gvks.AuthorizationPolicy:
			for _, d := range msg.Data {
				valBytes := d.Value
				auth := &api.AuthorizationPolicyToClient{}
				err := proto.Unmarshal(valBytes, auth)
				if err != nil {
					return
				}
				authorization = append(authorization, auth)
				a.handleAuthorization(authorization)
			}
		case gvks.ServiceNameMapping:
			for _, d := range msg.Data {
				valBytes := d.Value
				auth := &api.ServiceNameMappingToClient{}
				err := proto.Unmarshal(valBytes, auth)
				if err != nil {
					return
				}
				serviceMapping = append(serviceMapping, auth)
				a.handleServiceNameMapping(serviceMapping)
			}
		case gvks.ConditionRoute:
			for _, d := range msg.Data {
				valBytes := d.Value
				auth := &api.ConditionRouteToClient{}
				err := proto.Unmarshal(valBytes, auth)
				if err != nil {
					return
				}
				conditionRoute = append(conditionRoute, auth)
				a.handleConditionRoute(conditionRoute)
			}
		case gvks.DynamicConfig:
			for _, d := range msg.Data {
				valBytes := d.Value
				auth := &api.DynamicConfigToClient{}
				err := proto.Unmarshal(valBytes, auth)
				if err != nil {
					return
				}
				dynamicConfig = append(dynamicConfig, auth)
				a.handleDynamicConfig(dynamicConfig)
			}
		case gvks.TagRoute:
			for _, d := range msg.Data {
				valBytes := d.Value
				auth := &api.TagRouteToClient{}
				err := proto.Unmarshal(valBytes, auth)
				if err != nil {
					return
				}
				tagRoute = append(tagRoute, auth)
				a.handleTagRoute(tagRoute)
			}
		}

		a.mutex.Lock()
		a.Received[msg.Type] = msg
		err = a.ack(msg)
		if err != nil {
			return
		}
		a.mutex.Unlock()
	}
}

func (a *DDSC) ack(msg *dds.ObserveResponse) error {
	return a.stream.Send(&dds.ObserveRequest{
		Nonce: msg.Nonce,
		Type:  msg.Type,
	})
}

// Close the stream
func (a *DDSC) Close() {
	a.mutex.Lock()
	err := a.conn.Close()
	if err != nil {
		return
	}
	a.closed = true
	a.mutex.Unlock()
}

func (a *DDSC) handleAuthentication(ll []*api.AuthenticationPolicyToClient) {
	a.authentication = ll
	logger.Sugar().Info(ll)
}

func (a *DDSC) handleAuthorization(ll []*api.AuthorizationPolicyToClient) {
	a.authorization = ll
	logger.Sugar().Info(ll)
}

func (a *DDSC) handleServiceNameMapping(ll []*api.ServiceNameMappingToClient) {
	a.serviceMapping = ll
	logger.Sugar().Info(ll)
}

func (a *DDSC) handleConditionRoute(ll []*api.ConditionRouteToClient) {
	a.conditionRoute = ll
	logger.Sugar().Info(ll)
}

func (a *DDSC) handleTagRoute(ll []*api.TagRouteToClient) {
	a.tagRoute = ll
	logger.Sugar().Info(ll)
}

func (a *DDSC) handleDynamicConfig(ll []*api.DynamicConfigToClient) {
	a.dynamicConfig = ll
	logger.Sugar().Info(ll)
}

func main() {
	initialWatch := []*dds.ObserveRequest{
		{
			Nonce: "",
			Type:  gvks.AuthorizationPolicy,
		},
		{
			Nonce: "",
			Type:  gvks.AuthenticationPolicy,
		},
		{
			Nonce: "",
			Type:  gvks.DynamicConfig,
		},
		{
			Nonce: "",
			Type:  gvks.TagRoute,
		},
		{
			Nonce: "",
			Type:  gvks.ConditionRoute,
		},
		{
			Nonce: "",
			Type:  gvks.ServiceNameMapping,
		},
	}
	ddscConn, err := New(grpcUpstreamAddr, &Config{
		InitialDiscoveryRequest: initialWatch,
		Namespace:               "dubbo-system",
	})
	if err != nil {
		panic(err)
	}
	err = ddscConn.Run()
	if err != nil {
		panic("DDSC: failed running")
	}
	ddscConn.RecvWg.Wait()
}
