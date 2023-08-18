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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"

	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/core/cert/provider"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

type GrpcServer struct {
	PlainServer      *grpc.Server
	PlainServerPort  int
	SecureServer     *grpc.Server
	SecureServerPort int
}

func NewGrpcServer(s *provider.CertStorage, config *dubbo_cp.Config) GrpcServer {
	srv := GrpcServer{
		PlainServerPort:  config.GrpcServer.PlainServerPort,
		SecureServerPort: config.GrpcServer.SecureServerPort,
	}
	pool := x509.NewCertPool()
	tlsConfig := &tls.Config{
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			for _, cert := range s.GetTrustedCerts() {
				pool.AddCert(cert.Cert)
			}
			return s.GetServerCert(info.ServerName), nil
		},
		ClientCAs:  pool,
		ClientAuth: tls.VerifyClientCertIfGiven,
	}

	srv.PlainServer = grpc.NewServer()
	reflection.Register(srv.PlainServer)

	srv.SecureServer = grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	reflection.Register(srv.SecureServer)
	return srv
}

func (d *GrpcServer) NeedLeaderElection() bool {
	return false
}

func (d *GrpcServer) Start(stop <-chan struct{}) error {
	plainLis, err := net.Listen("tcp", fmt.Sprintf(":%d", d.PlainServerPort))
	if err != nil {
		return err
	}
	secureLis, err := net.Listen("tcp", fmt.Sprintf(":%d", d.SecureServerPort))
	if err != nil {
		return err
	}
	plainErrChan := make(chan error)
	secureErrChan := make(chan error)
	go func() {
		defer close(plainErrChan)
		if err = d.PlainServer.Serve(plainLis); err != nil {
			logger.Sugar().Error(err, "[cp-server] terminated with an error")
			plainErrChan <- err
		} else {
			logger.Sugar().Info("[cp-server] terminated normally")
		}
	}()
	go func() {
		defer close(secureErrChan)
		if err = d.SecureServer.Serve(secureLis); err != nil {
			logger.Sugar().Error(err, "[cp-server] terminated with an error")
			secureErrChan <- err
		} else {
			logger.Sugar().Info("[cp-server] terminated normally")
		}
	}()

	select {
	case <-stop:
		logger.Sugar().Info("[cp-server] stopping gracefully")
		d.PlainServer.GracefulStop()
		d.SecureServer.GracefulStop()
		return nil
	case err := <-secureErrChan:
		return err
	case err := <-plainErrChan:
		return err
	}
}
