package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	ca "github.com/apache/dubbo-admin/ca/v1alpha1"
	"google.golang.org/grpc"
	"log"
	"math/big"
	"net"
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

func main() {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			CommonName:    "domain name",
			Organization:  []string{"Company, INC."},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Golden Gate Bridge"},
			PostalCode:    []string{"94016"},
		},
		NotBefore: time.Now(),                   // 生效时间
		NotAfter:  time.Now().AddDate(10, 0, 0), // 过期时间 年月日
		IsCA:      true,                         // 表示用于CA
		// openssl 中的 extendedKeyUsage = clientAuth, serverAuth 字段
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		// openssl 中的 keyUsage 字段
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
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	log.Printf(caPEM.String())

	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})
	log.Printf(caPrivKeyPEM.String())

	impl := &DubboCertificateServiceServerImpl{
		rootCert: cert,
		pubKey:   caPEM.String(),
		privKey:  caPrivKey,
	}
	//impl.CreateCertificate(nil, &ca.DubboCertificateRequest{Csr: "-----BEGIN CERTIFICATE REQUEST-----\nMIHTMHsCAQAwGTEXMBUGA1UECgwOY2x1c3Rlci5kb21haW4wWTATBgcqhkjOPQIB\nBggqhkjOPQMBBwNCAAQzg1QJajZxbYJOODjl+33guXFHR1Ryit2H5B6qRTC9Dpsl\nYSccYbRzWUnr4m0BLJyXMnZoEEV5zDo67eWzzEhnoAAwCgYIKoZIzj0EAwIDSAAw\nRQIhAM5oYu1r6ceV2SFgJUVrwYsq8ztuN4C0BUM9M3eJJmPfAiBVvnwRCMBkGhOs\nD+RtZ3fXn6aOxQvUMEZiywj9OcYnVA==\n-----END CERTIFICATE REQUEST-----"})

	grpcServer := grpc.NewServer()
	ca.RegisterDubboCertificateServiceServer(grpcServer, impl)

	lis, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal(err)
	}
	grpcServer.Serve(lis)
}
