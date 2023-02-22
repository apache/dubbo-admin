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
