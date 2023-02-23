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
	"github.com/apache/dubbo-admin/ca/pkg/cert"
	"github.com/apache/dubbo-admin/ca/pkg/config"
	"github.com/apache/dubbo-admin/ca/pkg/k8s"
	"github.com/apache/dubbo-admin/ca/pkg/v1alpha1"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"log"
	"math"
	"net"
	"strconv"
	"sync"
	"time"
)

type Server struct {
	Options     *config.Options
	CertStorage *cert.Storage

	KubeClient *k8s.Client

	CertificateServer *v1alpha1.DubboCertificateServiceServerImpl
	PlainServer       *grpc.Server
	SecureServer      *grpc.Server
}

func (s *Server) Init() {
	// TODO bypass k8s work
	s.KubeClient = &k8s.Client{}
	if !s.KubeClient.Init() {
		panic("Failed to create kubernetes client.")
	}

	s.CertStorage = &cert.Storage{
		AuthorityCert: &cert.Cert{},
		TrustedCert:   []*cert.Cert{},
		Mutex:         &sync.Mutex{},
		CertValidity:  s.Options.CertValidity,
		CaValidity:    s.Options.CaValidity,
	}
	go s.CertStorage.RefreshServerCert()

	// TODO inject pod based on Webhook

	s.LoadRootCert()
	s.LoadAuthorityCert()

	impl := &v1alpha1.DubboCertificateServiceServerImpl{
		Options:     s.Options,
		CertStorage: s.CertStorage,
		KubeClient:  s.KubeClient,
	}

	logger := zap.NewExample()
	defer logger.Sync()

	// Make sure that log statements internal to gRPC library are logged using the zapLogger as well.
	grpc_zap.ReplaceGrpcLoggerV2(logger)

	s.PlainServer = grpc.NewServer()
	v1alpha1.RegisterDubboCertificateServiceServer(s.PlainServer, impl)
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
	v1alpha1.RegisterDubboCertificateServiceServer(s.SecureServer, impl)
	reflection.Register(s.SecureServer)
}

func (s *Server) LoadRootCert() {
	// todo
}

func (s *Server) LoadAuthorityCert() {
	certStr, priStr := s.KubeClient.GetAuthorityCert(s.Options.Namespace)
	if certStr != "" && priStr != "" {
		s.CertStorage.AuthorityCert.Cert = cert.DecodeCert(certStr)
		s.CertStorage.AuthorityCert.CertPem = certStr
		s.CertStorage.AuthorityCert.PrivateKey = cert.DecodePri(priStr)
	}

	s.RefreshAuthorityCert()
	go s.ScheduleRefreshAuthorityCert()
}

func (s *Server) ScheduleRefreshAuthorityCert() {
	interval := math.Min(math.Floor(float64(s.Options.CaValidity)/100), 10_000)
	for true {
		time.Sleep(time.Duration(interval) * time.Millisecond)
		if s.CertStorage.AuthorityCert.NeedRefresh() {
			log.Printf("Authority cert is invalid, refresh it.")
			// TODO lock if multi server
			// TODO refresh signed cert
			s.CertStorage.AuthorityCert = cert.CreateCA(s.CertStorage.RootCert, s.Options.CaValidity)
			s.KubeClient.UpdateAuthorityCert(s.CertStorage.AuthorityCert.CertPem, cert.EncodePri(s.CertStorage.AuthorityCert.PrivateKey), s.Options.Namespace)
			if s.KubeClient.UpdateAuthorityPublicKey(s.CertStorage.AuthorityCert.CertPem) {
				log.Printf("Write ca to config maps success.")
			} else {
				log.Printf("Write ca to config maps failed.")
			}
		}
	}
}

func (s *Server) RefreshAuthorityCert() {
	if s.CertStorage.AuthorityCert.IsValid() {
		log.Printf("Load authority cert from kubernetes secrect success.")
	} else {
		log.Printf("Load authority cert from kubernetes secrect failed.")
		s.CertStorage.AuthorityCert = cert.CreateCA(s.CertStorage.RootCert, s.Options.CaValidity)

		// TODO lock if multi server
		s.KubeClient.UpdateAuthorityCert(s.CertStorage.AuthorityCert.CertPem, cert.EncodePri(s.CertStorage.AuthorityCert.PrivateKey), s.Options.Namespace)
	}

	// TODO add task to update ca
	log.Printf("Writing ca to config maps.")
	if s.KubeClient.UpdateAuthorityPublicKey(s.CertStorage.AuthorityCert.CertPem) {
		log.Printf("Write ca to config maps success.")
	} else {
		log.Printf("Write ca to config maps failed.")
	}
	s.CertStorage.TrustedCert = append(s.CertStorage.TrustedCert, s.CertStorage.AuthorityCert)
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

	log.Printf("Server started.")
}
