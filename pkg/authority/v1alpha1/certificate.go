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
	"time"

	"github.com/apache/dubbo-admin/pkg/authority/cert"
	"github.com/apache/dubbo-admin/pkg/authority/config"
	"github.com/apache/dubbo-admin/pkg/authority/k8s"
	"github.com/apache/dubbo-admin/pkg/logger"
	"google.golang.org/grpc/peer"
)

type DubboCertificateServiceServerImpl struct {
	UnimplementedDubboCertificateServiceServer
	Options     *config.Options
	CertStorage cert.Storage
	KubeClient  k8s.Client
}

func (s *DubboCertificateServiceServerImpl) CreateCertificate(
	c context.Context,
	req *DubboCertificateRequest,
) (*DubboCertificateResponse, error) {
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
	endpoint, err := exactEndpoint(c, s.Options, s.KubeClient)
	if err != nil {
		logger.Sugar().Warnf("Failed to exact endpoint from context: %v. RemoteAddr: %s", err, p.Addr.String())

		return &DubboCertificateResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	certPem, err := cert.SignFromCSR(csr, endpoint, s.CertStorage.GetAuthorityCert(), s.Options.CertValidity)
	if err != nil {
		logger.Sugar().Warnf("Failed to sign certificate from csr: %v. RemoteAddr: %s", err, p.Addr.String())

		return &DubboCertificateResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	logger.Sugar().Infof("Success to sign certificate from csr. RemoteAddr: %s", p.Addr.String())

	return &DubboCertificateResponse{
		Success:    true,
		Message:    "OK",
		CertPem:    certPem,
		TrustCerts: []string{s.CertStorage.GetAuthorityCert().CertPem},
		ExpireTime: time.Now().UnixMilli() + (s.Options.CertValidity / 2),
	}, nil
}
