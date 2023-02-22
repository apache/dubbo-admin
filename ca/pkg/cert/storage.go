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
	"log"
	"math"
	"sync"
	"time"
)

type Storage struct {
	Mutex        *sync.Mutex
	CaValidity   int64
	CertValidity int64

	RootCert      *Cert
	AuthorityCert *Cert

	TrustedCert []*Cert
	ServerCerts map[string]*Cert
}

type Cert struct {
	Cert       *x509.Certificate
	CertPem    string
	PrivateKey *rsa.PrivateKey

	tlsCert *tls.Certificate
}

func (c *Cert) IsValid() bool {
	if c.Cert == nil || c.CertPem == "" || c.PrivateKey == nil {
		return false
	}
	if time.Now().Before(c.Cert.NotBefore) || time.Now().After(c.Cert.NotAfter) {
		return false
	}
	if c.Cert.PublicKey == c.PrivateKey.Public() {
		return false
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
	if c.Cert.PublicKey == c.PrivateKey.Public() {
		return true
	}
	return false
}

func (c *Cert) GetTlsCert() *tls.Certificate {
	if c.tlsCert != nil {
		return c.tlsCert
	}
	tlsCert, err := tls.X509KeyPair([]byte(c.CertPem), []byte(EncodePri(c.PrivateKey)))
	if err != nil {
		log.Printf("Failed to load x509 cert. %v", err)
	}
	c.tlsCert = &tlsCert
	return c.tlsCert
}

func (s *Storage) GetServerCert(serverName string) *tls.Certificate {
	if cert, exist := s.ServerCerts[serverName]; exist && cert.IsValid() {
		return cert.GetTlsCert()
	}
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	log.Printf("Generate certificate for %s", serverName)
	cert := SignServerCert(s.AuthorityCert, serverName, s.CertValidity)
	s.ServerCerts[serverName] = cert
	return cert.GetTlsCert()
}

func (s *Storage) RefreshServerCert() {
	interval := math.Min(math.Floor(float64(s.CertValidity)/100), 10_000)
	for true {
		time.Sleep(time.Duration(interval) * time.Millisecond)
		s.Mutex.Lock()
		wg := sync.WaitGroup{}
		wg.Add(len(s.ServerCerts))
		for serverName, cert := range s.ServerCerts {
			cert := cert
			serverName := serverName
			go func() {
				defer wg.Done()
				if cert.NeedRefresh() {
					log.Printf("Server cert for %s is invalid, refresh it.", serverName)
					cert = SignServerCert(s.AuthorityCert, serverName, s.CertValidity)
					s.ServerCerts[serverName] = cert
				}
			}()
		}
		wg.Wait()
		s.Mutex.Unlock()
	}
}
