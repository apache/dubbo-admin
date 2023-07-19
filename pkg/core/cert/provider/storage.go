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
	"crypto/tls"
	"crypto/x509"
	"math"
	"reflect"
	"sync"
	"time"

	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/core/logger"
)

type storageImpl struct {
	config *dubbo_cp.Config

	certClient Client

	mutex *sync.Mutex

	rootCert      *Cert
	authorityCert *Cert

	trustedCerts []*Cert
	serverNames  []string
	serverCerts  *Cert
}

type Storage interface {
	GetServerCert(serverName string) *tls.Certificate
	RefreshServerCert(<-chan struct{})

	SetAuthorityCert(*Cert)
	GetAuthorityCert() *Cert

	SetRootCert(*Cert)
	GetRootCert() *Cert

	AddTrustedCert(*Cert)
	GetTrustedCerts() []*Cert

	GetConfig() *dubbo_cp.Config
	GetCertClient() Client

	Start(stop <-chan struct{}) error
	NeedLeaderElection() bool
}

func (s *storageImpl) Start(stop <-chan struct{}) error {
	go s.RefreshServerCert(stop)
	go func(stop <-chan struct{}) {
		interval := math.Min(math.Floor(float64(s.config.Security.CaValidity)/100), 10_000)
		for {
			time.Sleep(time.Duration(interval) * time.Millisecond)
			if s.GetAuthorityCert().NeedRefresh() {
				logger.Sugar().Infof("Authority cert is invalid, refresh it.")
				// TODO lock if multi cp-server
				// TODO refresh signed cert

				NewleaderElection().Election(s, s.config, s.certClient.GetKubClient())
				if s.config.KubeConfig.IsKubernetesConnected {
					s.certClient.UpdateAuthorityCert(s.GetAuthorityCert().CertPem, EncodePrivateKey(s.GetAuthorityCert().PrivateKey), s.config.KubeConfig.Namespace)
					s.certClient.UpdateWebhookConfig(s.config, s)
					if s.certClient.UpdateAuthorityPublicKey(s.GetAuthorityCert().CertPem) {
						logger.Sugar().Infof("Write ca to config maps success.")
					} else {
						logger.Sugar().Warnf("Write ca to config maps failed.")
					}
				}
			}

			select {
			case <-stop:
				return
			default:
				continue
			}
		}
	}(stop)
	return nil
}

func (s *storageImpl) NeedLeaderElection() bool {
	return false
}

type Cert struct {
	Cert       *x509.Certificate
	CertPem    string
	PrivateKey *ecdsa.PrivateKey

	tlsCert *tls.Certificate
}

func NewStorage(options *dubbo_cp.Config, certClient Client) *storageImpl {
	return &storageImpl{
		mutex: &sync.Mutex{},

		authorityCert: &Cert{},
		trustedCerts:  []*Cert{},
		config:        options,
		certClient:    certClient,
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

func (s *storageImpl) RefreshServerCert(stop <-chan struct{}) {
	interval := math.Min(math.Floor(float64(s.config.Security.CertValidity)/100), 10_000)
	for true {
		select {
		case <-stop:
			return
		default:
		}

		time.Sleep(time.Duration(interval) * time.Millisecond)
		func() {
			s.mutex.Lock()
			defer s.mutex.Unlock()
			if s.authorityCert == nil || !s.authorityCert.IsValid() {
				// ignore if authority cert is invalid
				return
			}
			if s.serverCerts == nil || !s.serverCerts.IsValid() {
				logger.Sugar().Infof("Server cert is invalid, refresh it.")
				s.serverCerts = SignServerCert(s.authorityCert, s.serverNames, s.config.Security.CertValidity)
			}
		}()
	}
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

	s.serverCerts = SignServerCert(s.authorityCert, s.serverNames, s.config.Security.CertValidity)
	return s.serverCerts.GetTlsCert()
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

func (s *storageImpl) GetConfig() *dubbo_cp.Config {
	return s.config
}

func (s *storageImpl) GetCertClient() Client {
	return s.certClient
}
