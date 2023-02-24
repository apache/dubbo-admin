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
	"context"
	"github.com/apache/dubbo-admin/ca/pkg/cert"
	"github.com/apache/dubbo-admin/ca/pkg/config"
	"github.com/apache/dubbo-admin/ca/pkg/k8s"
	"github.com/apache/dubbo-admin/ca/pkg/logger"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"strings"
	"time"
)

type DubboCertificateServiceServerImpl struct {
	UnimplementedDubboCertificateServiceServer
	Options     *config.Options
	CertStorage *cert.Storage
	KubeClient  k8s.Client
}

func (s *DubboCertificateServiceServerImpl) CreateCertificate(c context.Context, req *DubboCertificateRequest) (*DubboCertificateResponse, error) {
	if req.Csr == "" {
		return &DubboCertificateResponse{
			Success: false,
			Message: "CSR is empty.",
		}, nil
	}

	csr, err := cert.LoadCSR(req.Csr)
	if csr == nil || err != nil {
		return &DubboCertificateResponse{
			Success: false,
			Message: "Decode csr failed.",
		}, nil
	}
	p, _ := peer.FromContext(c)

	if s.Options.EnableKubernetes {
		md, ok := metadata.FromIncomingContext(c)
		if !ok {
			logger.Sugar.Warnf("Failed to get metadata from context. RemoteAddr: %s", p.Addr.String())
			return &DubboCertificateResponse{
				Success: false,
				Message: "Failed to get metadata from context.",
			}, nil
		}

		authorization, ok := md["authorization"]
		if !ok || len(authorization) != 1 {
			logger.Sugar.Warnf("Failed to get Authorization header from context. RemoteAddr: %s", p.Addr.String())
			return &DubboCertificateResponse{
				Success: false,
				Message: "Failed to get Authorization header from context.",
			}, nil
		}

		if !strings.HasPrefix(authorization[0], "Bearer ") {
			logger.Sugar.Warnf("Failed to get Authorization header from context. RemoteAddr: %s", p.Addr.String())
			return &DubboCertificateResponse{
				Success: false,
				Message: "Failed to get Authorization header from context.",
			}, nil
		}

		token := strings.ReplaceAll(authorization[0], "Bearer ", "")

		// TODO load principal from k8s
		if !s.KubeClient.VerifyServiceAccount(token) {
			logger.Sugar.Warnf("Failed to verify Authorization header from kubernetes. RemoteAddr: %s", p.Addr.String())
			return &DubboCertificateResponse{
				Success: false,
				Message: "Failed to verify Authorization header from kubernetes.",
			}, nil
		}
	}

	// TODO check server token
	certPem, err := cert.SignFromCSR(csr, s.CertStorage.AuthorityCert, s.Options.CertValidity)
	if err != nil {
		logger.Sugar.Warnf("Failed to sign certificate from csr: %v. RemoteAddr: %s", err, p.Addr.String())
		return &DubboCertificateResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	logger.Sugar.Infof("Success to sign certificate from csr. RemoteAddr: %s", p.Addr.String())

	return &DubboCertificateResponse{
		Success:    true,
		Message:    "OK",
		CertPem:    certPem,
		TrustCerts: []string{s.CertStorage.AuthorityCert.CertPem},
		ExpireTime: time.Now().UnixMilli() + (s.Options.CertValidity / 2),
	}, nil
}
