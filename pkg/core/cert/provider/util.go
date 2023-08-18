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
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"log"
	"math/big"
	"net/url"
	"time"

	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/core/logger"
)

const (
	UNITag int = 6
)

// The OID for the SAN extension (See
// http://www.alvestrand.no/objectid/2.5.29.17.html).
var oidSubjectAlternativeName = asn1.ObjectIdentifier{2, 5, 29, 17}

func DecodeCert(cert string) *x509.Certificate {
	block, _ := pem.Decode([]byte(cert))
	if block == nil {
		logger.Sugar().Warnf("[Authority] Failed to parse public key.")
		return nil
	}
	p, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to parse public key. " + err.Error())
		return nil
	}
	return p
}

func DecodePrivateKey(cert string) *ecdsa.PrivateKey {
	block, _ := pem.Decode([]byte(cert))
	if block == nil {
		logger.Sugar().Warnf("[Authority] Failed to parse private key.")
		return nil
	}
	p, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to parse private key. " + err.Error())
		return nil
	}
	return p
}

func GenerateAuthorityCert(rootCert *Cert, caValidity int64) *Cert {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			CommonName:   "Dubbo RA",
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

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		log.Fatal(err)
	}

	caPEM := new(bytes.Buffer)
	err = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to encode certificate. " + err.Error())
		panic(err)
	}

	return &Cert{
		Cert:       DecodeCert(caPEM.String()),
		CertPem:    caPEM.String(),
		PrivateKey: privateKey,
	}
}

func SignServerCert(authorityCert *Cert, serverName []string, certValidity int64) *Cert {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
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
	cert.DNSNames = serverName

	c, err := x509.CreateCertificate(rand.Reader, cert, authorityCert.Cert, &privateKey.PublicKey, authorityCert.PrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	certPem := new(bytes.Buffer)
	err = pem.Encode(certPem, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: c,
	})
	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to encode certificate. " + err.Error())
		panic(err)
	}
	return &Cert{
		Cert:       cert,
		CertPem:    certPem.String(),
		PrivateKey: privateKey,
	}
}

func GenerateCSR() (string, *ecdsa.PrivateKey, error) {
	csrTemplate := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   "Dubbo",
			Organization: []string{"Apache Dubbo"},
		},
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
		return "", nil, err
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, privateKey)
	if err != nil {
		return "", nil, err
	}

	csr := new(bytes.Buffer)
	err = pem.Encode(csr, &pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrBytes,
	})

	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to encode certificate. " + err.Error())
		return "", nil, err
	}
	return csr.String(), privateKey, nil
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

func SignFromCSR(csr *x509.CertificateRequest, endpoint *endpoint.Endpoint, authorityCert *Cert, certValidity int64) (string, error) {
	csrTemplate := &x509.Certificate{
		PublicKeyAlgorithm: csr.PublicKeyAlgorithm,
		PublicKey:          csr.PublicKey,

		SerialNumber: big.NewInt(3),
		Issuer:       authorityCert.Cert.Subject,
		Subject:      csr.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Duration(certValidity) * time.Millisecond),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}
	if endpoint != nil {
		AppendEndpoint(endpoint, csrTemplate)
	}

	// TODO support ecdsa
	result, err := x509.CreateCertificate(rand.Reader, csrTemplate, authorityCert.Cert, csrTemplate.PublicKey, authorityCert.PrivateKey)
	if err != nil {
		return "", err
	}

	certPem := new(bytes.Buffer)
	err = pem.Encode(certPem, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: result,
	})
	if err != nil {
		return "", err
	}
	cert := certPem.String()

	return cert, nil
}

func AppendEndpoint(endpoint *endpoint.Endpoint, cert *x509.Certificate) {
	if endpoint.ID != "" {
		cert.Subject.CommonName = endpoint.ID
	}
	if endpoint.SpiffeID != "" {
		spiffeId, err := url.Parse(endpoint.SpiffeID)
		if err != nil {
			logger.Sugar().Warnf("[Authority] failed to parse the spiffe id (err: %s)", err)
			return
		}
		cert.URIs = append(cert.URIs, spiffeId)
	}
}

func EncodePrivateKey(caPrivKey *ecdsa.PrivateKey) string {
	caPrivKeyPEM := new(bytes.Buffer)
	pri, err := x509.MarshalECPrivateKey(caPrivKey)
	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to marshal EC private key. " + err.Error())
		return ""
	}
	err = pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: pri,
	})
	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to encode private key. " + err.Error())
		return ""
	}
	return caPrivKeyPEM.String()
}

func EncodePublicKey(pub *ecdsa.PublicKey) (res string) {
	caPrivKeyPEM := new(bytes.Buffer)
	defer func() {
		if err := recover(); err != nil {
			logger.Sugar().Warnf("[Authority] Failed to marshal EC public key. %v", err)
			res = ""
		}
	}()
	pri, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to marshal EC public key. " + err.Error())
		return ""
	}
	err = pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "EC PUBLIC KEY",
		Bytes: pri,
	})
	if err != nil {
		logger.Sugar().Warnf("[Authority] Failed to encode public key. " + err.Error())
		return ""
	}
	return caPrivKeyPEM.String()
}
