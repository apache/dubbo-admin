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

type CertStorage struct {
	config *dubbo_cp.Config

	certClient Client

	mutex *sync.Mutex

	rootCert      *Cert
	authorityCert *Cert

	trustedCerts []*Cert
	serverNames  []string
	serverCerts  *Cert
}

func calculateInterval(caValidity int64) time.Duration {
	interval := math.Max(math.Min(float64(caValidity/100), 10_000), 1)
	return time.Duration(interval) * time.Millisecond
}

func (s *CertStorage) Start(stop <-chan struct{}) error {
	go s.RefreshServerCert(stop)
	go func(stop <-chan struct{}) {
		ticker := time.NewTicker(calculateInterval(s.config.Security.CaValidity))
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if s.GetAuthorityCert().NeedRefresh() {
					logger.Sugar().Infof("[Authority] Authority cert is invalid, refresh it.")
					// TODO lock if multi cp-server
					// TODO refresh signed cert

					err := NewleaderElection().Election(s, s.config, s.certClient.GetKubClient())
					if err != nil {
						logger.Sugar().Error("[Authority] Leader Election failed")
					}
					if s.config.KubeConfig.IsKubernetesConnected {
						s.certClient.UpdateAuthorityCert(s.GetAuthorityCert().CertPem, EncodePrivateKey(s.GetAuthorityCert().PrivateKey), s.config.KubeConfig.Namespace)
						s.certClient.UpdateWebhookConfig(s.config, s)
						if s.certClient.UpdateAuthorityPublicKey(s.GetAuthorityCert().CertPem) {
							logger.Sugar().Infof("[Authority] Write ca to config maps success.")
						} else {
							logger.Sugar().Warnf("[Authority] Write ca to config maps failed.")
						}
					}
				}
			}
		}
	}(stop)
	return nil
}

func (s *CertStorage) NeedLeaderElection() bool {
	return false
}

type Cert struct {
	Cert       *x509.Certificate
	CertPem    string
	PrivateKey *ecdsa.PrivateKey

	tlsCert *tls.Certificate
}

func NewStorage(options *dubbo_cp.Config, certClient Client) *CertStorage {
	return &CertStorage{
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
		logger.Sugar().Warnf("[Authority] Failed to load x509 cert. %v", err)
	}
	c.tlsCert = &tlsCert
	return c.tlsCert
}

func (s *CertStorage) RefreshServerCert(stop <-chan struct{}) {
	interval := math.Max(math.Min(math.Floor(float64(s.config.Security.CertValidity)/100), 10_000), 1)
	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			func() {
				s.mutex.Lock()
				defer s.mutex.Unlock()
				if s.authorityCert == nil || !s.authorityCert.IsValid() {
					// ignore if authority cert is invalid
					return
				}
				if s.serverCerts == nil || !s.serverCerts.IsValid() {
					logger.Sugar().Infof("[Authority] Server cert is invalid, refresh it.")
					s.serverCerts = SignServerCert(s.authorityCert, s.serverNames, s.config.Security.CertValidity)
				}
			}()
		}
	}
}

func (s *CertStorage) GetServerCert(serverName string) *tls.Certificate {
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

func (s *CertStorage) SetAuthorityCert(cert *Cert) {
	s.authorityCert = cert
}

func (s *CertStorage) GetAuthorityCert() *Cert {
	return s.authorityCert
}

func (s *CertStorage) SetRootCert(cert *Cert) {
	s.rootCert = cert
}

func (s *CertStorage) GetRootCert() *Cert {
	return s.rootCert
}

func (s *CertStorage) AddTrustedCert(cert *Cert) {
	s.trustedCerts = append(s.trustedCerts, cert)
}

func (s *CertStorage) GetTrustedCerts() []*Cert {
	return s.trustedCerts
}

func (s *CertStorage) GetConfig() *dubbo_cp.Config {
	return s.config
}

func (s *CertStorage) GetCertClient() Client {
	return s.certClient
}
