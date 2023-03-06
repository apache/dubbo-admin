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
	"os"
	"testing"
	"time"

	cert2 "github.com/apache/dubbo-admin/pkg/authority/cert"
	"github.com/apache/dubbo-admin/pkg/authority/config"
	"github.com/apache/dubbo-admin/pkg/authority/k8s"
	"github.com/apache/dubbo-admin/pkg/logger"
)

type mockKubeClient struct {
	k8s.Client
}

var (
	certPEM = ""
	priPEM  = ""
)

func (s *mockKubeClient) Init(options *config.Options) bool {
	return true
}

func (s *mockKubeClient) GetAuthorityCert(namespace string) (string, string) {
	return certPEM, priPEM
}

func (s *mockKubeClient) UpdateAuthorityCert(cert string, pri string, namespace string) {
}

func (s *mockKubeClient) UpdateAuthorityPublicKey(cert string) bool {
	return true
}

func (s *mockKubeClient) UpdateWebhookConfig(options *config.Options, storage cert2.Storage) {
}

type mockStorage struct {
	cert2.Storage
	origin cert2.Storage
}

func (s *mockStorage) GetServerCert(serverName string) *tls.Certificate {
	return nil
}

func (s *mockStorage) RefreshServerCert() {
}

func (s *mockStorage) SetAuthorityCert(cert *cert2.Cert) {
	s.origin.SetAuthorityCert(cert)
}

func (s *mockStorage) GetAuthorityCert() *cert2.Cert {
	return s.origin.GetAuthorityCert()
}

func (s *mockStorage) SetRootCert(cert *cert2.Cert) {
	s.origin.SetRootCert(cert)
}

func (s *mockStorage) GetRootCert() *cert2.Cert {
	return s.origin.GetRootCert()
}

func (s *mockStorage) AddTrustedCert(cert *cert2.Cert) {
	s.origin.AddTrustedCert(cert)
}

func (s *mockStorage) GetTrustedCerts() []*cert2.Cert {
	return s.origin.GetTrustedCerts()
}

func (s *mockStorage) GetStopChan() chan os.Signal {
	return s.origin.GetStopChan()
}

func TestInit(t *testing.T) {
	t.Parallel()

	logger.Init()

	options := &config.Options{
		IsKubernetesConnected: true,
		Namespace:             "dubbo-system",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		CaValidity:            30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:          1 * 60 * 60 * 1000,       // 1 hour
	}

	s := NewServer(options)
	s.KubeClient = &mockKubeClient{}

	s.Init()
	if !s.CertStorage.GetAuthorityCert().IsValid() {
		t.Fatal("Authority cert is not valid")
		return
	}

	certPEM = s.CertStorage.GetAuthorityCert().CertPem
	priPEM = cert2.EncodePrivateKey(s.CertStorage.GetAuthorityCert().PrivateKey)

	s.PlainServer.Stop()
	s.SecureServer.Stop()
	s.StopChan <- os.Kill

	s = NewServer(options)
	s.KubeClient = &mockKubeClient{}
	s.Init()

	if !s.CertStorage.GetAuthorityCert().IsValid() {
		t.Fatal("Authority cert is not valid")

		return
	}

	if s.CertStorage.GetAuthorityCert().CertPem != certPEM {
		t.Fatal("Authority cert is not equal")

		return
	}

	s.PlainServer.Stop()
	s.SecureServer.Stop()
	s.StopChan <- os.Kill
	s.CertStorage.GetStopChan() <- os.Kill
}

func TestRefresh(t *testing.T) {
	t.Parallel()

	logger.Init()

	options := &config.Options{
		IsKubernetesConnected: false,
		Namespace:             "dubbo-system",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		CaValidity:            10,
	}

	s := NewServer(options)

	s.KubeClient = &mockKubeClient{}
	storage := &mockStorage{}
	storage.origin = cert2.NewStorage(options)
	s.CertStorage = storage

	s.Init()

	origin := s.CertStorage.GetAuthorityCert()

	for i := 0; i < 1000; i++ {
		// wait at most 100s
		time.Sleep(100 * time.Millisecond)
		if s.CertStorage.GetAuthorityCert() != origin {
			break
		}
	}

	if s.CertStorage.GetAuthorityCert() == origin {
		t.Fatal("Authority cert is not refreshed")
		return
	}

	s.PlainServer.Stop()
	s.SecureServer.Stop()
	s.StopChan <- os.Kill
}
