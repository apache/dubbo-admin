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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"strconv"
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

	// TODO inject pod based on Webhook

	// TODO Load root cert
	certStr, priStr := s.KubeClient.GetCA(s.Options.Namespace)
	if certStr != "" && priStr != "" {
		s.CertStorage.AuthorityCert.Cert = cert.DecodeCert(certStr)
		s.CertStorage.AuthorityCert.CertPem = certStr
		s.CertStorage.AuthorityCert.PrivateKey = cert.DecodePri(priStr)
	}
	// TODO check cert if expired

	if s.CertStorage.AuthorityCert.IsValid() {
		s.CertStorage.AuthorityCert = cert.CreateCA(s.CertStorage.RootCert, s.Options.CaValidity)
	}

	// TODO lock if multi server
	s.KubeClient.UpdateCA(s.CertStorage.AuthorityCert.CertPem, cert.EncodePri(s.CertStorage.AuthorityCert.PrivateKey), s.Options.Namespace)

	impl := &v1alpha1.DubboCertificateServiceServerImpl{
		Options:     s.Options,
		CertStorage: s.CertStorage,
	}

	s.PlainServer = grpc.NewServer()
	v1alpha1.RegisterDubboCertificateServiceServer(s.PlainServer, impl)
	reflection.Register(s.PlainServer)

	tlsConfig := &tls.Config{
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			log.Printf("Get certificate for %s", info.ServerName)
			// TODO build cert in backend
			// TODO rotation cert
			// TODO create cert when start
			// TODO DNSName
			if s.CertStorage.ServerCerts[info.ServerName] == nil {
				serverCert, pri := cert.SignServerCert(s.CertStorage.AuthorityCert, s.Options.CertValidity)
				c, _ := tls.X509KeyPair([]byte(serverCert), []byte(pri))
				s.CertStorage.ServerCerts[info.ServerName] = &c
			}
			return s.CertStorage.ServerCerts[info.ServerName], nil
		},
	}
	s.SecureServer = grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	v1alpha1.RegisterDubboCertificateServiceServer(s.SecureServer, impl)
	reflection.Register(s.SecureServer)
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

	// TODO add task to update ca
	log.Printf("Writing ca to config maps.")
	if s.KubeClient.UpdateCAPub(s.CertStorage.AuthorityCert.CertPem) {
		log.Printf("Write ca to config maps success.")
	} else {
		log.Printf("Write ca to config maps failed.")
	}
}
