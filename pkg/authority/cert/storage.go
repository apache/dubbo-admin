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

package cert

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"math"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/apache/dubbo-admin/pkg/authority/config"
	"github.com/apache/dubbo-admin/pkg/logger"
)

type storageImpl struct {
	Storage

	mutex    *sync.Mutex
	stopChan chan os.Signal

	caValidity   int64
	certValidity int64

	rootCert      *Cert
	authorityCert *Cert

	trustedCerts []*Cert
	serverNames  []string
	serverCerts  *Cert
}

type Storage interface {
	GetServerCert(serverName string) *tls.Certificate
	RefreshServerCert()

	SetAuthorityCert(*Cert)
	GetAuthorityCert() *Cert

	SetRootCert(*Cert)
	GetRootCert() *Cert

	AddTrustedCert(*Cert)
	GetTrustedCerts() []*Cert

	GetStopChan() chan os.Signal
}

type Cert struct {
	Cert       *x509.Certificate
	CertPem    string
	PrivateKey *rsa.PrivateKey

	tlsCert *tls.Certificate
}

func NewStorage(options *config.Options) *storageImpl {
	return &storageImpl{
		mutex:    &sync.Mutex{},
		stopChan: make(chan os.Signal, 1),

		authorityCert: &Cert{},
		trustedCerts:  []*Cert{},
		certValidity:  options.CertValidity,
		caValidity:    options.CaValidity,
	}
}

func (c *Cert) IsValid() bool {
	if c.Cert == nil || c.CertPem == "" || c.PrivateKey == nil {
		return false
	}
	if time.Now().Before(c.Cert.NotBefore) || time.Now().After(c.Cert.NotAfter) {
		return false
	}

	if c.tlsCert == nil || !reflect.DeepEqual(c.tlsCert.PrivateKey, c.PrivateKey) {
		tlsCert, err := tls.X509KeyPair([]byte(c.CertPem), []byte(EncodePrivateKey(c.PrivateKey)))
		if err != nil {
			return false
		}

		c.tlsCert = &tlsCert
	}

	return true
}

func (c *Cert) NeedRefresh() bool {
	if c.Cert == nil || c.CertPem == "" || c.PrivateKey == nil {
		return true
	}
	if time.Now().Before(c.Cert.NotBefore) || time.Now().After(c.Cert.NotAfter) {
		return true
	}
	validity := c.Cert.NotAfter.UnixMilli() - c.Cert.NotBefore.UnixMilli()
	if time.Now().Add(time.Duration(math.Floor(float64(validity)*0.2)) * time.Millisecond).After(c.Cert.NotAfter) {
		return true
	}
	if !reflect.DeepEqual(c.Cert.PublicKey, c.PrivateKey.Public()) {
		return true
	}
	return false
}

func (c *Cert) GetTlsCert() *tls.Certificate {
	if c.tlsCert != nil && reflect.DeepEqual(c.tlsCert.PrivateKey, c.PrivateKey) {
		return c.tlsCert
	}
	tlsCert, err := tls.X509KeyPair([]byte(c.CertPem), []byte(EncodePrivateKey(c.PrivateKey)))
	if err != nil {
		logger.Sugar().Warnf("Failed to load x509 cert. %v", err)
	}
	c.tlsCert = &tlsCert
	return c.tlsCert
}

func (s *storageImpl) GetServerCert(serverName string) *tls.Certificate {
	nameSigned := serverName == ""
	for _, name := range s.serverNames {
		if name == serverName {
			nameSigned = true
			break
		}
	}
	if nameSigned && s.serverCerts != nil && s.serverCerts.IsValid() {
		return s.serverCerts.GetTlsCert()
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !nameSigned {
		s.serverNames = append(s.serverNames, serverName)
	}

	s.serverCerts = SignServerCert(s.authorityCert, s.serverNames, s.certValidity)
	return s.serverCerts.GetTlsCert()
}

func (s *storageImpl) RefreshServerCert() {
	interval := math.Min(math.Floor(float64(s.certValidity)/100), 10_000)
	for true {
		select {
		case <-s.stopChan:
			return
		default:
		}

		time.Sleep(time.Duration(interval) * time.Millisecond)
		s.mutex.Lock()
		if s.authorityCert == nil || !s.authorityCert.IsValid() {
			// ignore if authority cert is invalid
			continue
		}
		if s.serverCerts == nil || !s.serverCerts.IsValid() {
			logger.Sugar().Infof("Server cert is invalid, refresh it.")
			s.serverCerts = SignServerCert(s.authorityCert, s.serverNames, s.certValidity)
		}
		s.mutex.Unlock()
	}
}

func (s *storageImpl) SetAuthorityCert(cert *Cert) {
	s.authorityCert = cert
}

func (s *storageImpl) GetAuthorityCert() *Cert {
	return s.authorityCert
}

func (s *storageImpl) SetRootCert(cert *Cert) {
	s.rootCert = cert
}

func (s *storageImpl) GetRootCert() *Cert {
	return s.rootCert
}

func (s *storageImpl) AddTrustedCert(cert *Cert) {
	s.trustedCerts = append(s.trustedCerts, cert)
}

func (s *storageImpl) GetTrustedCerts() []*Cert {
	return s.trustedCerts
}

func (s *storageImpl) GetStopChan() chan os.Signal {
	return s.stopChan
}
