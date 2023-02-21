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
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/apache/dubbo-admin/ca/cert"
	"github.com/apache/dubbo-admin/ca/k8s"
	ca "github.com/apache/dubbo-admin/ca/v1alpha1"
	"google.golang.org/grpc"
	"log"
	"math/big"
	"net"
	"os"
	"time"
)

type DubboCertificateServiceServerImpl struct {
	ca.UnimplementedDubboCertificateServiceServer

	rootCert *x509.Certificate
	pubKey   string
	privKey  *rsa.PrivateKey
}

func (s *DubboCertificateServiceServerImpl) CreateCertificate(c context.Context, req *ca.DubboCertificateRequest) (*ca.DubboCertificateResponse, error) {
	csr, _ := LoadCSR(req.Csr)
	log.Printf("Receive csr request " + req.Csr)
	csrTemplate := x509.Certificate{
		Signature:          csr.Signature,
		SignatureAlgorithm: csr.SignatureAlgorithm,

		PublicKeyAlgorithm: csr.PublicKeyAlgorithm,
		PublicKey:          csr.PublicKey,

		SerialNumber: big.NewInt(2019),
		Issuer:       s.rootCert.Subject,
		Subject:      csr.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(0, 0, 1),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}
	csrTemplate.DNSNames = csr.DNSNames

	// TODO support ecdsa
	result, err := x509.CreateCertificate(rand.Reader, &csrTemplate, s.rootCert, csrTemplate.PublicKey, s.privKey)
	if err != nil {
		log.Fatal(err)
	}

	pubPEM := new(bytes.Buffer)
	pem.Encode(pubPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: result,
	})
	pub := pubPEM.String()
	log.Printf("Sign csr request " + pub)

	puba := pub
	return &ca.DubboCertificateResponse{
		PublicKey:  puba,
		TrustCerts: []string{s.pubKey},
		ExpireTime: time.Now().AddDate(0, 0, 1).UnixMilli(),
	}, nil
}

func LoadCSR(csrString string) (*x509.CertificateRequest, error) {
	block, _ := pem.Decode([]byte(csrString))
	csr, _ := x509.ParseCertificateRequest(block.Bytes)

	return csr, nil
}

// TODO read namespace from env
const namespace = "dubbo-system"

func main() {
	// TODO bypass k8s work
	k8sClient := &k8s.Client{}
	if !k8sClient.Init() {
		log.Printf("Failed to create kuberentes client.")
		return
	}

	// TODO inject pod based on Webhook

	var caCert *x509.Certificate
	var pub string
	var pri *rsa.PrivateKey

	certStr, priStr := k8sClient.GetCA(namespace)
	if certStr != "" && priStr != "" {
		caCert = cert.DecodeCert(certStr)
		pri = cert.DecodePri(priStr)
		pub = certStr
	}
	// TODO check cert if expired

	if caCert == nil || pri == nil || pub == "" {
		caCert, pub, pri = cert.CreateCA()
	}

	// TODO lock if multi server
	k8sClient.UpdateCA(pub, cert.EncodePri(pri), namespace)

	impl := &DubboCertificateServiceServerImpl{
		rootCert: caCert,
		pubKey:   pub,
		privKey:  pri,
	}

	grpcServer := grpc.NewServer()
	ca.RegisterDubboCertificateServiceServer(grpcServer, impl)

	lis, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal(err)
	}
	go grpcServer.Serve(lis)

	// TODO add task to update ca
	log.Printf("Writing ca to config maps.")
	if k8sClient.UpdateCAPub(pub) {
		log.Printf("Write ca to config maps success.")
	} else {
		log.Printf("Write ca to config maps failed.")
	}
	c := make(chan os.Signal)
	<-c
}
