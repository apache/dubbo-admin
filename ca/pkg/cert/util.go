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
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"time"
)

func DecodeCert(cert string) *x509.Certificate {
	block, _ := pem.Decode([]byte(cert))
	p, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Printf("Failed to parse public key. " + err.Error())
		return nil
	}
	return p
}

func DecodePub(cert string) *rsa.PublicKey {
	p, err := x509.ParsePKCS1PublicKey([]byte(cert))
	if err != nil {
		log.Printf("Failed to parse public key. " + err.Error())
		return nil
	}
	return p
}

func DecodePri(cert string) *rsa.PrivateKey {
	block, _ := pem.Decode([]byte(cert))

	p, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Printf("Failed to parse private key. " + err.Error())
		return nil
	}
	return p
}

func CreateCA(rootCert *Cert, caValidity int64) *Cert {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			CommonName:   "Dubbo",
			Organization: []string{"Apache Dubbo"},
		},
		Issuer: pkix.Name{
			CommonName:   "Dubbo CA",
			Organization: []string{"Apache Dubbo"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Duration(caValidity) * time.Millisecond),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Fatal(err)
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		log.Fatal(err)
	}

	caPEM := new(bytes.Buffer)
	err = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	if err != nil {
		log.Printf("Failed to encode certificate. " + err.Error())
		panic(err)
	}

	return &Cert{
		Cert:       cert,
		CertPem:    caPEM.String(),
		PrivateKey: caPrivKey,
	}
}

func SignServerCert(authorityCert *Cert, serverName string, certValidity int64) *Cert {
	privKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Fatal(err)
	}

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Issuer:       authorityCert.Cert.Subject,
		Subject: pkix.Name{
			CommonName:   "Dubbo",
			Organization: []string{"Apache Dubbo"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(time.Duration(certValidity) * time.Millisecond),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	cert.DNSNames = []string{serverName}

	c, err := x509.CreateCertificate(rand.Reader, cert, authorityCert.Cert, &privKey.PublicKey, authorityCert.PrivateKey)

	certPem := new(bytes.Buffer)
	err = pem.Encode(certPem, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: c,
	})
	if err != nil {
		log.Printf("Failed to encode certificate. " + err.Error())
		panic(err)
	}
	return &Cert{
		Cert:       cert,
		CertPem:    certPem.String(),
		PrivateKey: privKey,
	}
}

func LoadCSR(csrString string) (*x509.CertificateRequest, error) {
	block, _ := pem.Decode([]byte(csrString))
	if block == nil {
		return nil, nil
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, err
	}

	return csr, nil
}

func SignFromCSR(csr *x509.CertificateRequest, authorityCert *Cert, certValidity int64) (string, error) {
	csrTemplate := x509.Certificate{
		Signature:          csr.Signature,
		SignatureAlgorithm: csr.SignatureAlgorithm,

		PublicKeyAlgorithm: csr.PublicKeyAlgorithm,
		PublicKey:          csr.PublicKey,

		SerialNumber: big.NewInt(3),
		Issuer:       authorityCert.Cert.Subject,
		Subject:      csr.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Duration(certValidity) * time.Millisecond),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth | x509.ExtKeyUsageServerAuth},
	}
	csrTemplate.DNSNames = csr.DNSNames

	// TODO support ecdsa
	result, err := x509.CreateCertificate(rand.Reader, &csrTemplate, authorityCert.Cert, csrTemplate.PublicKey, authorityCert.PrivateKey)
	if err != nil {
		return "", err
	}

	pubPEM := new(bytes.Buffer)
	err = pem.Encode(pubPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: result,
	})
	if err != nil {
		return "", err
	}
	pub := pubPEM.String()
	log.Printf("Sign csr request " + pub)

	return pub, nil
}

func EncodePri(caPrivKey *rsa.PrivateKey) string {
	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})
	return caPrivKeyPEM.String()
}
