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

package security

import (
	"crypto/tls"
	cert2 "github.com/apache/dubbo-admin/pkg/authority/cert"
	"github.com/apache/dubbo-admin/pkg/authority/config"
	"github.com/apache/dubbo-admin/pkg/authority/k8s"
	"github.com/apache/dubbo-admin/pkg/authority/logger"
	"github.com/apache/dubbo-admin/pkg/authority/patch"
	v1alpha12 "github.com/apache/dubbo-admin/pkg/authority/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/authority/webhook"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"time"
)

type Server struct {
	StopChan chan os.Signal

	Options     *config.Options
	CertStorage cert2.Storage

	KubeClient k8s.Client

	CertificateServer *v1alpha12.DubboCertificateServiceServerImpl
	PlainServer       *grpc.Server
	SecureServer      *grpc.Server

	WebhookServer *webhook.Webhook
	JavaInjector  *patch.JavaSdk
}

func NewServer(options *config.Options) *Server {
	return &Server{
		Options:  options,
		StopChan: make(chan os.Signal, 1),
	}
}

func (s *Server) Init() {
	// TODO bypass k8s work
	if s.KubeClient == nil {
		s.KubeClient = k8s.NewClient()
	}
	if !s.KubeClient.Init(s.Options) {
		logger.Sugar.Warnf("Failed to connect to Kubernetes cluster. Will ignore OpenID Connect check.")
		s.Options.IsKubernetesConnected = false
	}

	if s.CertStorage == nil {
		s.CertStorage = cert2.NewStorage(s.Options)
	}
	go s.CertStorage.RefreshServerCert()

	// TODO inject pod based on Webhook

	s.LoadRootCert()
	s.LoadAuthorityCert()

	impl := &v1alpha12.DubboCertificateServiceServerImpl{
		Options:     s.Options,
		CertStorage: s.CertStorage,
		KubeClient:  s.KubeClient,
	}

	s.PlainServer = grpc.NewServer()
	v1alpha12.RegisterDubboCertificateServiceServer(s.PlainServer, impl)
	reflection.Register(s.PlainServer)

	tlsConfig := &tls.Config{
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return s.CertStorage.GetServerCert(info.ServerName), nil
		},
	}

	s.CertStorage.GetServerCert("localhost")
	s.CertStorage.GetServerCert("dubbo-ca." + s.Options.Namespace + ".svc")
	s.CertStorage.GetServerCert("dubbo-ca." + s.Options.Namespace + ".svc.cluster.local")

	s.SecureServer = grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	v1alpha12.RegisterDubboCertificateServiceServer(s.SecureServer, impl)
	reflection.Register(s.SecureServer)

	if s.Options.InPodEnv {
		s.WebhookServer = webhook.NewWebhook(
			func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return s.CertStorage.GetServerCert(info.ServerName), nil
			})
		s.WebhookServer.Init(s.Options)

		s.JavaInjector = patch.NewJavaSdk(s.Options)
		s.WebhookServer.Patches = append(s.WebhookServer.Patches, s.JavaInjector.NewPod)
	}
}

func (s *Server) LoadRootCert() {
	// todo
}

func (s *Server) LoadAuthorityCert() {
	if s.Options.IsKubernetesConnected {
		certStr, priStr := s.KubeClient.GetAuthorityCert(s.Options.Namespace)
		if certStr != "" && priStr != "" {
			s.CertStorage.GetAuthorityCert().Cert = cert2.DecodeCert(certStr)
			s.CertStorage.GetAuthorityCert().CertPem = certStr
			s.CertStorage.GetAuthorityCert().PrivateKey = cert2.DecodePrivateKey(priStr)
		}
	}

	s.RefreshAuthorityCert()
	go s.ScheduleRefreshAuthorityCert()
}

func (s *Server) ScheduleRefreshAuthorityCert() {
	interval := math.Min(math.Floor(float64(s.Options.CaValidity)/100), 10_000)
	for true {
		time.Sleep(time.Duration(interval) * time.Millisecond)
		if s.CertStorage.GetAuthorityCert().NeedRefresh() {
			logger.Sugar.Infof("Authority cert is invalid, refresh it.")
			// TODO lock if multi server
			// TODO refresh signed cert
			s.CertStorage.SetAuthorityCert(cert2.GenerateAuthorityCert(s.CertStorage.GetRootCert(), s.Options.CaValidity))

			if s.Options.IsKubernetesConnected {
				s.KubeClient.UpdateAuthorityCert(s.CertStorage.GetAuthorityCert().CertPem, cert2.EncodePrivateKey(s.CertStorage.GetAuthorityCert().PrivateKey), s.Options.Namespace)
				s.KubeClient.UpdateWebhookConfig(s.Options, s.CertStorage)
				if s.KubeClient.UpdateAuthorityPublicKey(s.CertStorage.GetAuthorityCert().CertPem) {
					logger.Sugar.Infof("Write ca to config maps success.")
				} else {
					logger.Sugar.Warnf("Write ca to config maps failed.")
				}
			}
		}

		select {
		case <-s.StopChan:
			return
		default:
			continue
		}
	}
}

func (s *Server) RefreshAuthorityCert() {
	if s.CertStorage.GetAuthorityCert().IsValid() {
		logger.Sugar.Infof("Load authority cert from kubernetes secrect success.")
	} else {
		logger.Sugar.Warnf("Load authority cert from kubernetes secrect failed.")
		s.CertStorage.SetAuthorityCert(cert2.GenerateAuthorityCert(s.CertStorage.GetRootCert(), s.Options.CaValidity))

		// TODO lock if multi server
		if s.Options.IsKubernetesConnected {
			s.KubeClient.UpdateAuthorityCert(s.CertStorage.GetAuthorityCert().CertPem, cert2.EncodePrivateKey(s.CertStorage.GetAuthorityCert().PrivateKey), s.Options.Namespace)
		}
	}

	if s.Options.IsKubernetesConnected {
		logger.Sugar.Info("Writing ca to config maps.")
		if s.KubeClient.UpdateAuthorityPublicKey(s.CertStorage.GetAuthorityCert().CertPem) {
			logger.Sugar.Info("Write ca to config maps success.")
		} else {
			logger.Sugar.Warnf("Write ca to config maps failed.")
		}
	}

	s.CertStorage.AddTrustedCert(s.CertStorage.GetAuthorityCert())
}

func (s *Server) Start() {
	lis, err := net.Listen("tcp", ":"+strconv.Itoa(s.Options.PlainServerPort))
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		err := s.PlainServer.Serve(lis)
		if err != nil {
			log.Fatal(err)
		}
	}()
	lis, err = net.Listen("tcp", ":"+strconv.Itoa(s.Options.SecureServerPort))
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		err := s.SecureServer.Serve(lis)
		if err != nil {
			log.Fatal(err)
		}
	}()

	go s.WebhookServer.Serve()

	s.KubeClient.UpdateWebhookConfig(s.Options, s.CertStorage)

	logger.Sugar.Info("Server started.")
}
