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

package server

import (
	"net"
	"testing"

	"github.com/apache/dubbo-admin/api/ca"
	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/config/kube"
	"github.com/apache/dubbo-admin/pkg/config/security"
	"github.com/apache/dubbo-admin/pkg/core/cert/provider"
	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/core/jwt"
	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/stretchr/testify/assert"

	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type fakeKubeClient struct {
	provider.Client
}

func (c fakeKubeClient) VerifyServiceAccount(token string, authorizationType string) (*endpoint.Endpoint, bool) {
	return &endpoint.Endpoint{}, "expceted-token" == token
}

type fakeAddr struct {
	net.Addr
}

func (f *fakeAddr) String() string {
	return "127.0.0.1:1234"
}

func TestCSRFailed(t *testing.T) {
	t.Parallel()

	logger.Init()

	md := metadata.MD{}
	md["authorization"] = []string{"Bearer 123"}
	c := metadata.NewIncomingContext(context.TODO(), metadata.MD{})
	c = peer.NewContext(c, &peer.Peer{Addr: &fakeAddr{}})

	options := &dubbo_cp.Config{
		KubeConfig: kube.KubeConfig{
			IsKubernetesConnected: false,
		},
		Security: security.SecurityConfig{
			CertValidity: 24 * 60 * 60 * 1000,
			CaValidity:   365 * 24 * 60 * 60 * 1000,
		},
	}
	storage := provider.NewStorage(options, &provider.ClientImpl{})
	storage.SetAuthorityCert(provider.GenerateAuthorityCert(nil, options.Security.CaValidity))

	kubeClient := &fakeKubeClient{}
	impl := &AuthorityService{
		Options:     options,
		CertStorage: storage,
		CertClient:  kubeClient.Client,
	}

	certificate, err := impl.CreateIdentity(c, &ca.IdentityRequest{
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

	certificate, err = impl.CreateIdentity(c, &ca.IdentityRequest{
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

	certificate, err = impl.CreateIdentity(c, &ca.IdentityRequest{
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

	options := &dubbo_cp.Config{
		KubeConfig: kube.KubeConfig{
			IsKubernetesConnected: true,
		},
		Security: security.SecurityConfig{
			CertValidity:    24 * 60 * 60 * 1000,
			CaValidity:      365 * 24 * 60 * 60 * 1000,
			EnableOIDCCheck: true,
		},
	}
	storage := provider.NewStorage(options, &provider.ClientImpl{})
	storage.SetAuthorityCert(provider.GenerateAuthorityCert(nil, options.Security.CaValidity))

	kubeClient := &fakeKubeClient{}
	impl := &AuthorityService{
		Options:     options,
		CertStorage: storage,
		CertClient:  kubeClient,
	}

	csr, privateKey, err := provider.GenerateCSR()
	if err != nil {
		t.Fatal(err)
		return
	}

	certificate, err := impl.CreateIdentity(p, &ca.IdentityRequest{
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

	certificate, err = impl.CreateIdentity(c, &ca.IdentityRequest{
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

	certificate, err = impl.CreateIdentity(c, &ca.IdentityRequest{
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

	certificate, err = impl.CreateIdentity(c, &ca.IdentityRequest{
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

	generatedCert := provider.DecodeCert(certificate.CertPem)
	c2 := &provider.Cert{
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

	options := &dubbo_cp.Config{
		KubeConfig: kube.KubeConfig{
			IsKubernetesConnected: false,
		},
		Security: security.SecurityConfig{
			CertValidity:  24 * 60 * 60 * 1000,
			CaValidity:    365 * 24 * 60 * 60 * 1000,
			IsTrustAnyone: true,
		},
	}

	storage := provider.NewStorage(options, &provider.ClientImpl{})
	storage.SetAuthorityCert(provider.GenerateAuthorityCert(nil, options.Security.CaValidity))
	storage.AddTrustedCert(storage.GetAuthorityCert())

	kubeClient := &fakeKubeClient{}
	impl := &AuthorityService{
		Options:     options,
		CertStorage: storage,
		CertClient:  kubeClient,
	}

	csr, privateKey, err := provider.GenerateCSR()
	if err != nil {
		t.Fatal(err)
		return
	}

	certificate, err := impl.CreateIdentity(c, &ca.IdentityRequest{
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

	generatedCert := provider.DecodeCert(certificate.CertPem)
	c2 := &provider.Cert{
		Cert:       generatedCert,
		CertPem:    certificate.CertPem,
		PrivateKey: privateKey,
	}

	if !c2.IsValid() {
		t.Fatal("Cert is not valid")
		return
	}

	claims, err := jwt.Verify(&storage.GetAuthorityCert().PrivateKey.PublicKey, certificate.Token)
	assert.Nil(t, err)
	assert.NotNil(t, claims)

	assert.Equal(t, 1, len(certificate.TrustedTokenPublicKeys))
	assert.Equal(t, provider.EncodePublicKey(&storage.GetAuthorityCert().PrivateKey.PublicKey), certificate.TrustedTokenPublicKeys[0])
}
