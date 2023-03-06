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
	"net"
	"testing"

	cert2 "github.com/apache/dubbo-admin/pkg/authority/cert"
	"github.com/apache/dubbo-admin/pkg/authority/config"
	"github.com/apache/dubbo-admin/pkg/authority/k8s"
	"github.com/apache/dubbo-admin/pkg/authority/rule"
	"github.com/apache/dubbo-admin/pkg/logger"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type MockKubeClient struct {
	k8s.Client
}

func (c MockKubeClient) VerifyServiceAccount(token string) (*rule.Endpoint, bool) {
	return nil, "expceted-token" == token
}

type fakeAddr struct {
	net.Addr
}

func (f *fakeAddr) String() string {
	return ""
}

func TestCSRFailed(t *testing.T) {
	t.Parallel()

	logger.Init()

	md := metadata.MD{}
	md["authorization"] = []string{"Bearer 123"}
	c := metadata.NewIncomingContext(context.TODO(), metadata.MD{})
	c = peer.NewContext(c, &peer.Peer{Addr: &fakeAddr{}})

	options := &config.Options{
		IsKubernetesConnected: false,
		CertValidity:          24 * 60 * 60 * 1000,
		CaValidity:            365 * 24 * 60 * 60 * 1000,
	}
	storage := cert2.NewStorage(options)
	storage.SetAuthorityCert(cert2.GenerateAuthorityCert(nil, options.CaValidity))

	kubeClient := &MockKubeClient{}
	impl := &DubboCertificateServiceServerImpl{
		Options:     options,
		CertStorage: storage,
		KubeClient:  kubeClient.Client,
	}

	certificate, err := impl.CreateCertificate(c, &DubboCertificateRequest{
		Csr: "",
	})
	if err != nil {
		t.Fatal(err)
		return
	}

	if certificate.Success {
		t.Fatal("Should sign failed")
		return
	}

	certificate, err = impl.CreateCertificate(c, &DubboCertificateRequest{
		Csr: "123",
	})

	if err != nil {
		t.Fatal(err)
		return
	}

	if certificate.Success {
		t.Fatal("Should sign failed")
		return
	}

	certificate, err = impl.CreateCertificate(c, &DubboCertificateRequest{
		Csr: "-----BEGIN CERTIFICATE-----\n" +
			"123\n" +
			"-----END CERTIFICATE-----",
	})

	if err != nil {
		t.Fatal(err)
		return
	}

	if certificate.Success {
		t.Fatal("Should sign failed")
		return
	}
}

func TestTokenFailed(t *testing.T) {
	t.Parallel()

	logger.Init()

	p := peer.NewContext(context.TODO(), &peer.Peer{Addr: &fakeAddr{}})

	options := &config.Options{
		IsKubernetesConnected: true,
		EnableOIDCCheck:       true,
		CertValidity:          24 * 60 * 60 * 1000,
		CaValidity:            365 * 24 * 60 * 60 * 1000,
	}
	storage := cert2.NewStorage(options)
	storage.SetAuthorityCert(cert2.GenerateAuthorityCert(nil, options.CaValidity))

	kubeClient := &MockKubeClient{}
	impl := &DubboCertificateServiceServerImpl{
		Options:     options,
		CertStorage: storage,
		KubeClient:  kubeClient,
	}

	csr, privateKey, err := cert2.GenerateCSR()
	if err != nil {
		t.Fatal(err)
		return
	}

	certificate, err := impl.CreateCertificate(p, &DubboCertificateRequest{
		Csr: csr,
	})
	if err != nil {
		t.Fatal(err)
		return
	}

	if certificate.Success {
		t.Fatal("Should sign failed")
		return
	}

	md := metadata.MD{}
	md["authorization"] = []string{"123"}
	c := metadata.NewIncomingContext(p, md)

	certificate, err = impl.CreateCertificate(c, &DubboCertificateRequest{
		Csr: csr,
	})

	if err != nil {
		t.Fatal(err)
		return
	}

	if certificate.Success {
		t.Fatal("Should sign failed")
		return
	}

	md = metadata.MD{}
	md["authorization"] = []string{"Bearer 123"}
	c = metadata.NewIncomingContext(p, md)

	certificate, err = impl.CreateCertificate(c, &DubboCertificateRequest{
		Csr: csr,
	})

	if err != nil {
		t.Fatal(err)
		return
	}

	if certificate.Success {
		t.Fatal("Should sign failed")
		return
	}

	md = metadata.MD{}
	md["authorization"] = []string{"Bearer expceted-token"}
	c = metadata.NewIncomingContext(p, md)

	certificate, err = impl.CreateCertificate(c, &DubboCertificateRequest{
		Csr: csr,
	})

	if err != nil {
		t.Fatal(err)
		return
	}

	if !certificate.Success {
		t.Fatal("Sign failed")
		return
	}

	generatedCert := cert2.DecodeCert(certificate.CertPem)
	c2 := &cert2.Cert{
		Cert:       generatedCert,
		CertPem:    certificate.CertPem,
		PrivateKey: privateKey,
	}

	if !c2.IsValid() {
		t.Fatal("Cert is not valid")
		return
	}
}

func TestSuccess(t *testing.T) {
	t.Parallel()

	md := metadata.MD{}
	md["authorization"] = []string{"Bearer 123"}
	c := metadata.NewIncomingContext(context.TODO(), metadata.MD{})
	c = peer.NewContext(c, &peer.Peer{Addr: &fakeAddr{}})

	options := &config.Options{
		IsKubernetesConnected: false,
		CertValidity:          24 * 60 * 60 * 1000,
		CaValidity:            365 * 24 * 60 * 60 * 1000,
	}
	storage := cert2.NewStorage(options)
	storage.SetAuthorityCert(cert2.GenerateAuthorityCert(nil, options.CaValidity))

	kubeClient := &MockKubeClient{}
	impl := &DubboCertificateServiceServerImpl{
		Options:     options,
		CertStorage: storage,
		KubeClient:  kubeClient,
	}

	csr, privateKey, err := cert2.GenerateCSR()
	if err != nil {
		t.Fatal(err)
		return
	}

	certificate, err := impl.CreateCertificate(c, &DubboCertificateRequest{
		Csr: csr,
	})
	if err != nil {
		t.Fatal(err)
		return
	}

	if !certificate.Success {
		t.Fatal("Sign failed")
		return
	}

	generatedCert := cert2.DecodeCert(certificate.CertPem)
	c2 := &cert2.Cert{
		Cert:       generatedCert,
		CertPem:    certificate.CertPem,
		PrivateKey: privateKey,
	}

	if !c2.IsValid() {
		t.Fatal("Cert is not valid")
		return
	}
}
