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
	"google.golang.org/grpc/metadata"
	"log"
	"strings"
	"time"
)

type DubboCertificateServiceServerImpl struct {
	UnimplementedDubboCertificateServiceServer
	Options     *config.Options
	CertStorage *cert.Storage
	KubeClient  *k8s.Client
}

func (s *DubboCertificateServiceServerImpl) CreateCertificate(c context.Context, req *DubboCertificateRequest) (*DubboCertificateResponse, error) {
	csr, _ := cert.LoadCSR(req.Csr)
	if s.Options.EnableKubernetes {
		md, ok := metadata.FromIncomingContext(c)
		if !ok {
			log.Printf("Failed to get metadata from context.")
			return &DubboCertificateResponse{}, nil
		}

		authorization, ok := md["authorization"]
		if !ok || len(authorization) != 1 {
			log.Printf("Failed to get Authorization header from context.")
			return &DubboCertificateResponse{}, nil
		}

		if !strings.HasPrefix(authorization[0], "Bearer ") {
			log.Printf("Failed to get Authorization header from context.")
			return &DubboCertificateResponse{}, nil
		}

		token := strings.ReplaceAll(authorization[0], "Bearer ", "")

		// TODO load principal from k8s
		if !s.KubeClient.VerifyServiceAccount(token) {
			log.Printf("Failed to verify Authorization header from kubernetes.")
			return &DubboCertificateResponse{}, nil
		}
	}

	// TODO check server token
	log.Printf("Receive csr request " + req.Csr)
	if csr == nil {
		return &DubboCertificateResponse{}, nil
	}
	publicKey, err := cert.SignFromCSR(csr, s.CertStorage.AuthorityCert, s.Options.CertValidity)
	if err != nil {
		log.Fatal(err)
	}
	return &DubboCertificateResponse{
		PublicKey:  publicKey,
		TrustCerts: []string{s.CertStorage.AuthorityCert.CertPem},
		ExpireTime: time.Now().AddDate(0, 0, 1).UnixMilli(),
	}, nil
}
