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
	"encoding/pem"
	"net/url"
	"testing"

	"github.com/apache/dubbo-admin/pkg/core/endpoint"
	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/stretchr/testify/assert"
)

func TestCSR(t *testing.T) {
	t.Parallel()

	csr, privateKey, err := GenerateCSR()
	if err != nil {
		t.Fatal(err)
		return
	}

	request, err := LoadCSR(csr)
	if err != nil {
		t.Fatal(err)
		return
	}

	cert := GenerateAuthorityCert(nil, 365*24*60*60*1000)

	target, err := SignFromCSR(request, &endpoint.Endpoint{SpiffeID: "spiffe://cluster.local"}, cert, 365*24*60*60*1000)
	if err != nil {
		t.Fatal(err)
		return
	}

	certificate := DecodeCert(target)

	check := &Cert{
		Cert:       certificate,
		PrivateKey: privateKey,
		CertPem:    target,
	}

	if !check.IsValid() {
		t.Fatal("Cert is not valid")
		return
	}

	assert.Equal(t, 1, len(certificate.URIs))
	assert.Equal(t, &url.URL{Scheme: "spiffe", Host: "cluster.local"}, certificate.URIs[0])

	target, err = SignFromCSR(request, &endpoint.Endpoint{SpiffeID: "://"}, cert, 365*24*60*60*1000)
	assert.Nil(t, err)

	certificate = DecodeCert(target)

	check = &Cert{
		Cert:       certificate,
		PrivateKey: privateKey,
		CertPem:    target,
	}

	assert.True(t, check.IsValid())

	assert.Equal(t, 0, len(certificate.URIs))
}

func TestDecodeCert(t *testing.T) {
	t.Parallel()

	logger.Init()

	if DecodeCert("") != nil {
		t.Fatal("DecodeCert should return nil")
		return
	}

	if DecodeCert("123") != nil {
		t.Fatal("DecodeCert should return nil")
		return
	}

	certPem := new(bytes.Buffer)
	err := pem.Encode(certPem, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: []byte("123"),
	})
	assert.Nil(t, err)

	if DecodeCert(certPem.String()) != nil {
		t.Fatal("DecodeCert should return nil")
		return
	}

	if DecodeCert("-----BEGIN CERTIFICATE-----\n"+
		"MIICSjCCAbOgAwIBAgIJAJHGGR4dGioHMA0GCSqGSIb3DQEBCwUAMFYxCzAJBgNV\n"+
		"BAYTAkFVMRMwEQYDVQQIEwpTb21lLVN0YXRlMSEwHwYDVQQKExhJbnRlcm5ldCBX\n"+
		"aWRnaXRzIFB0eSBMdGQxDzANBgNVBAMTBnRlc3RjYTAeFw0xNDExMTEyMjMxMjla\n"+
		"Fw0yNDExMDgyMjMxMjlaMFYxCzAJBgNVBAYTAkFVMRMwEQYDVQQIEwpTb21lLVN0\n"+
		"YXRlMSEwHwYDVQQKExhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQxDzANBgNVBAMT\n"+
		"BnRlc3RjYTCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAwEDfBV5MYdlHVHJ7\n"+
		"+L4nxrZy7mBfAVXpOc5vMYztssUI7mL2/iYujiIXM+weZYNTEpLdjyJdu7R5gGUu\n"+
		"g1jSVK/EPHfc74O7AyZU34PNIP4Sh33N+/A5YexrNgJlPY+E3GdVYi4ldWJjgkAd\n"+
		"Qah2PH5ACLrIIC6tRka9hcaBlIECAwEAAaMgMB4wDAYDVR0TBAUwAwEB/zAOBgNV\n"+
		"HQ8BAf8EBAMCAgQwDQYJKoZIhvcNAQELBQADgYEAHzC7jdYlzAVmddi/gdAeKPau\n"+
		"sPBG/C2HCWqHzpCUHcKuvMzDVkY/MP2o6JIW2DBbY64bO/FceExhjcykgaYtCH/m\n"+
		"oIU63+CFOTtR7otyQAWHqXa7q4SbCDlG7DyRFxqG0txPtGvy12lgldA2+RgcigQG\n"+
		"Dfcog5wrJytaQ6UA0wE=\n"+
		"-----END CERTIFICATE-----\n") == nil {
		t.Fatal("DecodeCert should not return nil")
		return
	}
}

func TestDecodePrivateKey(t *testing.T) {
	t.Parallel()

	logger.Init()
	if DecodePrivateKey("") != nil {
		t.Fatal("DecodePrivateKey should return nil")
		return
	}

	if DecodePrivateKey("123") != nil {
		t.Fatal("DecodePrivateKey should return nil")
		return
	}

	if DecodePrivateKey("-----BEGIN PRIVATE KEY-----\n"+
		"123\n"+
		"-----END PRIVATE KEY-----\n") != nil {
		t.Fatal("DecodePrivateKey should return nil")
		return
	}

	if DecodePrivateKey("-----BEGIN PRIVATE KEY-----\n"+
		"MIICdQIBADANBgkqhkiG9w0BAQEFAASCAl8wggJbAgEAAoGBAMBA3wVeTGHZR1Ry\n"+
		"e/i+J8a2cu5gXwFV6TnObzGM7bLFCO5i9v4mLo4iFzPsHmWDUxKS3Y8iXbu0eYBl\n"+
		"LoNY0lSvxDx33O+DuwMmVN+DzSD+Eod9zfvwOWHsazYCZT2PhNxnVWIuJXViY4JA\n"+
		"HUGodjx+QAi6yCAurUZGvYXGgZSBAgMBAAECgYAxRi8i9BlFlufGSBVoGmydbJOm\n"+
		"bwLKl9dP3o33ODSP9hok5y6A0w5plWk3AJSF1hPLleK9VcSKYGYnt0clmPVHF35g\n"+
		"bx2rVK8dOT0mn7rz9Zr70jcSz1ETA2QonHZ+Y+niLmcic9At6hRtWiewblUmyFQm\n"+
		"GwggIzi7LOyEUHrEcQJBAOXxyQvnLvtKzXiqcsW/K6rExqVJVk+KF0fzzVyMzTJx\n"+
		"HRBxUVgvGdEJT7j+7P2kcTyafve0BBzDSPIaDyiJ+Y0CQQDWCb7jASFSbu5M3Zcd\n"+
		"Gkr4ZKN1XO3VLQX10b22bQYdF45hrTN2tnzRvVUR4q86VVnXmiGiTqmLkXcA2WWf\n"+
		"pHfFAkAhv9olUBo6MeF0i3frBEMRfm41hk0PwZHnMqZ6pgPcGnQMnMU2rzsXzkkQ\n"+
		"OwJnvAIOxhJKovZTjmofdqmw5odlAkBYVUdRWjsNUTjJwj3GRf6gyq/nFMYWz3EB\n"+
		"RWFdM1ttkDYzu45ctO2IhfHg4sPceDMO1s6AtKQmNI9/azkUjITdAkApNa9yFRzc\n"+
		"TBaDNPd5KVd58LVIzoPQ6i7uMHteLXJUWqSroji6S3s4gKMFJ/dO+ZXIlgQgfJJJ\n"+
		"ZDL4cdrdkeoM\n"+
		"-----END PRIVATE KEY-----\n") != nil {
		t.Fatal("DecodePrivateKey should return nil")
		return
	}

	if DecodePrivateKey("-----BEGIN EC PRIVATE KEY-----\n"+
		"MHcCAQEEIMS+Yc+9GMD0v7a2yz8EwEoF2vsM7d54aeV5jKjHGFzioAoGCCqGSM49\n"+
		"AwEHoUQDQgAEe6MTHP7f5BKtVMEswm59WTZXyDD7cAbPdeBDtljJRIl6yAYgBtFN\n"+
		"9RT54nIlNiPnH3P8DKyuvSE3jmsG3IHhcg==\n"+
		"-----END EC PRIVATE KEY-----\n") == nil {
		t.Fatal("DecodePrivateKey should not return nil")
		return
	}
}

func TestDecodePublicKey(t *testing.T) {
	t.Parallel()

	key := DecodePrivateKey("-----BEGIN EC PRIVATE KEY-----\n" +
		"MHcCAQEEIIyys+L2OLSPvIjqbSJXkjbl6QtFysqhuHWsHwmfpADloAoGCCqGSM49\n" +
		"AwEHoUQDQgAE4/2iaB+J+yBSdwtbKtyymbOiEXwNPB3v8EYRJBahICOYZFbWz4MK\n" +
		"3eV88hF7Q91yec8SpAyG2HXVUTKBCh53wg==\n" +
		"-----END EC PRIVATE KEY-----")

	assert.NotNil(t, key)

	assert.Equal(t, "-----BEGIN EC PUBLIC KEY-----\n"+
		"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE4/2iaB+J+yBSdwtbKtyymbOiEXwN\n"+
		"PB3v8EYRJBahICOYZFbWz4MK3eV88hF7Q91yec8SpAyG2HXVUTKBCh53wg==\n"+
		"-----END EC PUBLIC KEY-----\n", EncodePublicKey(&key.PublicKey))

	assert.Equal(t, "", EncodePublicKey(&ecdsa.PublicKey{}))
}
