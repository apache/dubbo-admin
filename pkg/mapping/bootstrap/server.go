/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package bootstrap

import (
	"github.com/apache/dubbo-admin/pkg/logger"
	"github.com/apache/dubbo-admin/pkg/mapping/config"
	dubbov1alpha1 "github.com/apache/dubbo-admin/pkg/mapping/dubbo"
	informerV1alpha1 "github.com/apache/dubbo-admin/pkg/mapping/generated/informers/externalversions/dubbo.apache.org/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/mapping/kube"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"os"
)

type Server struct {
	StopChan   chan os.Signal
	Options    *config.Options
	KubeClient kube.Client
	snpServer  *dubbov1alpha1.Snp
	GrpcServer *grpc.Server
	informer   informerV1alpha1.ServiceNameMappingInformer
}

func NewServer(options *config.Options) *Server {
	return &Server{
		Options:  options,
		StopChan: make(chan os.Signal, 1),
	}
}

func (s *Server) Init() {
	if s.KubeClient == nil {
		s.KubeClient = kube.NewClient()
	}
	if s.KubeClient != nil {
		s.snpServer = dubbov1alpha1.NewSnp(s.KubeClient)
	}
	if !s.KubeClient.Init(s.Options) {
		logger.Sugar().Warnf("Failed to connect to Kubernetes cluster. Will ignore OpenID Connect check.")
		s.Options.IsKubernetesConnected = false
	} else {
		s.Options.IsKubernetesConnected = true
	}
	logger.Infof("Starting grpc Server")
	s.GrpcServer = grpc.NewServer()
	logger.Infof("Started grpc Server")
	reflection.Register(s.GrpcServer)
	logger.Infof("Service Mapping grpc Server")
	s.snpServer.Register(s.GrpcServer)
}

func (s *Server) Start() {
	s.KubeClient.InitContainer()
	logger.Sugar().Infof("Server started.")
}
