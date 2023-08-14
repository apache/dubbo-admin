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

package provider

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"reflect"
	"sync"
	"testing"
	"time"

	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/config/security"
	"github.com/apache/dubbo-admin/pkg/core/logger"
)

func TestIsValid(t *testing.T) {
	t.Parallel()

	c := &Cert{}
	if c.IsValid() {
		t.Errorf("cert is not valid")
	}

	c.Cert = &x509.Certificate{}
	if c.IsValid() {
		t.Errorf("cert is not valid")
	}

	c.CertPem = "test"
	if c.IsValid() {
		t.Errorf("cert is not valid")
	}

	c.PrivateKey, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if c.IsValid() {
		t.Errorf("cert is not valid")
	}

	c.Cert.NotBefore = time.Now().Add(-1 * time.Hour)
	c.Cert.NotAfter = time.Now().Add(1 * time.Hour)
	if c.IsValid() {
		t.Errorf("cert is not valid")
	}

	c = GenerateAuthorityCert(nil, 2*60*60*1000)
	if !c.IsValid() {
		t.Errorf("cert is valid")
	}
}

func TestNeedRefresh(t *testing.T) {
	t.Parallel()

	c := &Cert{}
	if !c.NeedRefresh() {
		t.Errorf("cert is need refresh")
	}

	c.Cert = &x509.Certificate{}
	if !c.NeedRefresh() {
		t.Errorf("cert is need refresh")
	}

	c.CertPem = "test"
	if !c.NeedRefresh() {
		t.Errorf("cert is need refresh")
	}

	c.PrivateKey, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if !c.NeedRefresh() {
		t.Errorf("cert is need refresh")
	}

	c.Cert.NotBefore = time.Now().Add(1 * time.Hour)
	if !c.NeedRefresh() {
		t.Errorf("cert is not need refresh")
	}

	c.Cert.NotBefore = time.Now().Add(-1 * time.Hour)
	c.Cert.NotAfter = time.Now().Add(-1 * time.Hour)
	if !c.NeedRefresh() {
		t.Errorf("cert is not need refresh")
	}

	c.Cert.NotBefore = time.Now().Add(-1 * time.Hour).Add(2 * 60 * -0.3 * time.Minute)
	c.Cert.NotAfter = time.Now().Add(-1 * time.Hour).Add(2 * 60 * 0.7 * time.Minute)
	if !c.NeedRefresh() {
		t.Errorf("cert is need refresh")
	}

	c.Cert.NotAfter = time.Now().Add(1 * time.Hour)
	if !c.NeedRefresh() {
		t.Errorf("cert is need refresh")
	}

	c = GenerateAuthorityCert(nil, 2*60*60*1000)
	if c.NeedRefresh() {
		t.Errorf("cert is valid")
	}
}

func TestGetTlsCert(t *testing.T) {
	t.Parallel()

	cert := GenerateAuthorityCert(nil, 2*60*60*1000)

	tlsCert := cert.GetTlsCert()
	if !reflect.DeepEqual(tlsCert.PrivateKey, cert.PrivateKey) {
		t.Errorf("cert is not equal")
	}

	if tlsCert != cert.GetTlsCert() {
		t.Errorf("cert is not equal")
	}
}

func TestGetServerCert(t *testing.T) {
	t.Parallel()

	cert := GenerateAuthorityCert(nil, 24*60*60*1000)

	s := &CertStorage{
		authorityCert: cert,
		mutex:         &sync.Mutex{},
		config: &dubbo_cp.Config{
			Security: security.SecurityConfig{
				CaValidity:   24 * 60 * 60 * 1000,
				CertValidity: 2 * 60 * 60 * 1000,
			},
		},
	}

	c := s.GetServerCert("localhost")

	pool := x509.NewCertPool()
	pool.AddCert(cert.Cert)
	certificate, err := x509.ParseCertificate(c.Certificate[0])
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	_, err = certificate.Verify(x509.VerifyOptions{
		Roots:   pool,
		DNSName: "localhost",
	})

	if err != nil {
		t.Errorf(err.Error())
		return
	}

	if c != s.GetServerCert("localhost") {
		t.Errorf("cert is not equal")
	}

	if c != s.GetServerCert("") {
		t.Errorf("cert is not equal")
	}

	c = s.GetServerCert("newhost")

	pool = x509.NewCertPool()
	pool.AddCert(cert.Cert)
	certificate, err = x509.ParseCertificate(c.Certificate[0])
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	_, err = certificate.Verify(x509.VerifyOptions{
		Roots:   pool,
		DNSName: "localhost",
	})

	if err != nil {
		t.Errorf(err.Error())
		return
	}

	_, err = certificate.Verify(x509.VerifyOptions{
		Roots:   pool,
		DNSName: "newhost",
	})

	if err != nil {
		t.Errorf(err.Error())
		return
	}
}

func TestRefreshServerCert(t *testing.T) {
	t.Parallel()

	stop := make(chan struct{})

	logger.Init()
	s := NewStorage(&dubbo_cp.Config{
		Security: security.SecurityConfig{
			CaValidity:   24 * 60 * 60 * 1000,
			CertValidity: 10,
		},
	}, &ClientImpl{})
	s.SetAuthorityCert(GenerateAuthorityCert(nil, 24*60*60*1000))

	go s.RefreshServerCert(stop)

	c := s.GetServerCert("localhost")
	origin := s.GetServerCert("")

	for i := 0; i < 100; i++ {
		// at most 10s
		time.Sleep(100 * time.Millisecond)
		if origin != s.GetServerCert("") {
			break
		}
	}

	if c == s.GetServerCert("localhost") {
		t.Errorf("cert is not equal")
	}

	if reflect.DeepEqual(c, s.GetServerCert("localhost")) {
		t.Errorf("cert is not equal")
	}

	stop <- struct{}{}
}
