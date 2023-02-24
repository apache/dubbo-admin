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
	"github.com/apache/dubbo-admin/ca/pkg/config"
	"github.com/apache/dubbo-admin/ca/pkg/logger"
	"math"
	"os"
	"reflect"
	"sync"
	"time"
)

type Storage struct {
	Mutex    *sync.Mutex
	StopChan chan os.Signal

	CaValidity   int64
	CertValidity int64

	RootCert      *Cert
	AuthorityCert *Cert

	TrustedCert []*Cert
	ServerNames []string
	ServerCerts *Cert
}

type Cert struct {
	Cert       *x509.Certificate
	CertPem    string
	PrivateKey *rsa.PrivateKey

	tlsCert *tls.Certificate
}

func NewStorage(options *config.Options) *Storage {
	return &Storage{
		Mutex:    &sync.Mutex{},
		StopChan: make(chan os.Signal, 1),

		AuthorityCert: &Cert{},
		TrustedCert:   []*Cert{},
		CertValidity:  options.CertValidity,
		CaValidity:    options.CaValidity,
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
		tlsCert, err := tls.X509KeyPair([]byte(c.CertPem), []byte(EncodePri(c.PrivateKey)))
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
	tlsCert, err := tls.X509KeyPair([]byte(c.CertPem), []byte(EncodePri(c.PrivateKey)))
	if err != nil {
		logger.Sugar.Warnf("Failed to load x509 cert. %v", err)
	}
	c.tlsCert = &tlsCert
	return c.tlsCert
}

func (s *Storage) GetServerCert(serverName string) *tls.Certificate {
	nameSigned := serverName == ""
	for _, name := range s.ServerNames {
		if name == serverName {
			nameSigned = true
			break
		}
	}
	if nameSigned && s.ServerCerts != nil && s.ServerCerts.IsValid() {
		return s.ServerCerts.GetTlsCert()
	}
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	if !nameSigned {
		s.ServerNames = append(s.ServerNames, serverName)
	}
	s.ServerCerts = SignServerCert(s.AuthorityCert, s.ServerNames, s.CertValidity)
	return s.ServerCerts.GetTlsCert()
}

func (s *Storage) RefreshServerCert() {
	interval := math.Min(math.Floor(float64(s.CertValidity)/100), 10_000)
	for true {
		time.Sleep(time.Duration(interval) * time.Millisecond)
		s.Mutex.Lock()
		if s.ServerCerts == nil || !s.ServerCerts.IsValid() {
			logger.Sugar.Infof("Server cert is invalid, refresh it.")
			s.ServerCerts = SignServerCert(s.AuthorityCert, s.ServerNames, s.CertValidity)
		}
		s.Mutex.Unlock()

		select {
		case <-s.StopChan:
			return
		default:
			continue
		}
	}
}
