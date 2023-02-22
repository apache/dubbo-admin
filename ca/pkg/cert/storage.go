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
	"time"
)

type Storage struct {
	CaValidity   int64
	CertValidity int64

	RootCert      *Cert
	AuthorityCert *Cert

	ServerCerts map[string]*tls.Certificate
}

type Cert struct {
	Cert       *x509.Certificate
	CertPem    string
	PrivateKey *rsa.PrivateKey
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
