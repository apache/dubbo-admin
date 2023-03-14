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
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"dubbo.apache.org/dubbo-go/v3/common"

	"github.com/apache/dubbo-admin/pkg/authority/rule"
	"github.com/apache/dubbo-admin/pkg/logger"
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
		logger.Sugar().Warnf("Failed to parse public key.")
		return nil
	}
	p, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		logger.Sugar().Warnf("Failed to parse public key. " + err.Error())
		return nil
	}
	return p
}

func DecodePrivateKey(cert string) *rsa.PrivateKey {
	block, _ := pem.Decode([]byte(cert))
	if block == nil {
		logger.Sugar().Warnf("Failed to parse private key.")
		return nil
	}
	p, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		logger.Sugar().Warnf("Failed to parse private key. " + err.Error())
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

	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
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
		logger.Sugar().Warnf("Failed to encode certificate. " + err.Error())
		panic(err)
	}

	return &Cert{
		Cert:       DecodeCert(caPEM.String()),
		CertPem:    caPEM.String(),
		PrivateKey: privateKey,
	}
}

func SignServerCert(authorityCert *Cert, serverName []string, certValidity int64) *Cert {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
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
		logger.Sugar().Warnf("Failed to encode certificate. " + err.Error())
		panic(err)
	}
	return &Cert{
		Cert:       cert,
		CertPem:    certPem.String(),
		PrivateKey: privateKey,
	}
}

func GenerateCSR() (string, *rsa.PrivateKey, error) {
	csrTemplate := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   "Dubbo",
			Organization: []string{"Apache Dubbo"},
		},
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
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
		logger.Sugar().Warnf("Failed to encode certificate. " + err.Error())
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

func SignFromCSR(csr *x509.CertificateRequest, endpoint *rule.Endpoint, authorityCert *Cert, certValidity int64) (string, error) {
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

func AppendEndpoint(endpoint *rule.Endpoint, cert *x509.Certificate) {
	if endpoint == nil {
		return
	}

	err := BuildSANExtension(endpoint, cert)
	if err != nil {
		logger.Sugar().Warnf("Failed to build SAN extension. " + err.Error())
		return
	}
}

func BuildSANExtension(endpoint *rule.Endpoint, cert *x509.Certificate) error {
	rawValues := []asn1.RawValue{}

	if endpoint.SpiffeID != "" {
		rawValues = append(rawValues, asn1.RawValue{
			Bytes: []byte(endpoint.SpiffeID),
			Class: asn1.ClassContextSpecific,
			Tag:   UNITag,
		})
	}

	u, err := common.NewURL("")
	if err != nil {
		return err
	}

	u.Protocol = "dubbo"
	if len(endpoint.Ips) != 0 {
		u.Ip = endpoint.Ips[0]
		u.AddParam("ips", strings.Join(endpoint.Ips, ","))
	} else {
		u.Ip = "localhost"
	}
	u.Port = "0"

	if endpoint.KubernetesEnv != nil {
		u.AddParam("kubernetesEnv.namespace", endpoint.KubernetesEnv.Namespace)
		u.AddParam("kubernetesEnv.deploymentName", endpoint.KubernetesEnv.DeploymentName)
		u.AddParam("kubernetesEnv.statefulSetName", endpoint.KubernetesEnv.StatefulSetName)
		u.AddParam("kubernetesEnv.podName", endpoint.KubernetesEnv.PodName)

		if len(endpoint.KubernetesEnv.PodLabels) != 0 {
			labels, err := json.Marshal(endpoint.KubernetesEnv.PodLabels)
			if err != nil {
				return err
			}
			u.AddParam("kubernetesEnv.podLabels", string(labels))
		}

		if len(endpoint.KubernetesEnv.PodAnnotations) != 0 {
			annotations, err := json.Marshal(endpoint.KubernetesEnv.PodAnnotations)
			if err != nil {
				return err
			}
			u.AddParam("kubernetesEnv.podAnnotations", string(annotations))
		}
	}

	rawValues = append(rawValues, asn1.RawValue{
		Bytes: []byte(u.String()),
		Class: asn1.ClassContextSpecific,
		Tag:   UNITag,
	})

	if len(rawValues) != 0 {
		bs, err := asn1.Marshal(rawValues)
		if err != nil {
			return fmt.Errorf("failed to marshal the raw values for SAN field (err: %s)", err)
		}

		// SAN is Critical because the subject is empty. This is compliant with X.509 and SPIFFE standards.
		cert.ExtraExtensions = append(cert.ExtraExtensions, pkix.Extension{Id: oidSubjectAlternativeName, Critical: true, Value: bs})
	}

	return nil
}

func EncodePrivateKey(caPrivKey *rsa.PrivateKey) string {
	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})
	return caPrivKeyPEM.String()
}
