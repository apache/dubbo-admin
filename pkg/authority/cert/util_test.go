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
	"testing"

	"github.com/apache/dubbo-admin/pkg/logger"
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

	target, err := SignFromCSR(request, nil, cert, 365*24*60*60*1000)
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

	if DecodeCert("-----BEGIN CERTIFICATE-----\n"+
		"123\n"+
		"-----END CERTIFICATE-----") != nil {
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

	if DecodePrivateKey("-----BEGIN RSA PRIVATE KEY-----\n"+
		"MIIEpgIBAAKCAQEAwQl8A5KYyOmXsz+Mk05NLWS9jHDhvJC1ekWgqOApwrb0Ecio\n"+
		"tv5dirqAtuEX+dGRVftxJdtZHWto+gKy3H6Ae866FBFt7TWgTZFkt0XW3tMmUmNG\n"+
		"bdzHAuZGK9+RlNNTNBTZJAx338kxM7/lqqOgEZig5SmX2Xt3u+DQjJPlsWB/lKDD\n"+
		"OKOc93lGo/8chdmMv70inE/xv6LQ9nugRvBe1XfXafuHEUVyj2rzF1v9y7yF5Tek\n"+
		"70wK/KV+O7ukBRc4SPwJ7YAWuofMhFneNtWGNHYaLShJBhvC+E7JXD+prJfHNdSc\n"+
		"ORnTz/LjMWsLbD1lhr/p7vrWXujDSGM6ZDR6EwIDAQABAoIBAQCjqjPwH4HUjmDl\n"+
		"RBMe7bt3qjsfcLGjm5mSQqh1piEiCtYioduR01ZiAcCRzYTzdWBg4x/Ktg/3ZpMJ\n"+
		"rfISCltLHTodO63U+auhOI2I6fjE0YdjQPJ8wTwmVDDYj+Qxp36a4LY93yhfn4hM\n"+
		"1P2XUMWtRZfc1AgAB7O7ol+PYPHVEX4n9ugbRDkn7/hpi05JPAOnGNimKDi61PpS\n"+
		"rWpkAKYCC6q2hLTOW+EKvfNqUjuK/YAzPQD14zP7KRQ9kkezAluwwVbwwaI2jJ4x\n"+
		"n6jHwPMOH1eKTQMtUg6Xxv59jBrcPmtD38dZvzzjBZDZYu4xcWJeeY4oP8/UE7uE\n"+
		"pTFACvBRAoGBAMlErLppeVZcgcW9gOP98qV2CbWK/+aB3oJQEQBlKB3UtQSjCnZ7\n"+
		"lLzxgMtDD+tcPocY5FY52MzJQ2UFgScSzW04JuBQPbsHcGmuzv/cahuB/S+xwB6m\n"+
		"I2RXbFkgPPirJ9mqTeuNMwcXgAhoVbPV3otMq45EsxHubATit7QvczabAoGBAPWH\n"+
		"yt0uxcf/j2k7EH3Ug5TkVKI8IftCM0fRs9eUzy2zPKTVRdTbQY75J/E2zkEQat2B\n"+
		"8hEONkkV/ScLV5Oon4oeBxCRq17h37H5znkW2yNYSMNLcqUN58ZcVxsRSPj/Eoq5\n"+
		"Ngotll+JmITrxtd6NpFcGqrDQ/KV9uM1AoqN4EXpAoGBALAXeLRD8dhAaX4TdgCD\n"+
		"v9dKNeZzLb+EYqRK3wUke/vVjWb4KwBM0W6aMWAlVXlLpJ1YhvZ1+Bv7/w4UydHg\n"+
		"3oCvfzwEmG3ZbV3ZhtxPATr9+QHQl9F49EAnSPGVhiLexKfpG/F6AWo0Al3Ywxrr\n"+
		"hKEFvJdlvfJzUmjX33gzh67/AoGBAMAnqBJ2On+NeFUozn1LxjbOg5X8bbPQWYXJ\n"+
		"jnAXnBTuA3YVG3O8rJASWroi5ERzbs8wlZvXfZCxTtAxxjZfb4yOd4T2HCJDr+f/\n"+
		"0yFdS99bhoahE3YtbckGF32th2inZ4F99db9WoQmkWDljVax5ObaKFygORsvVmr2\n"+
		"36hD5NORAoGBALKQZ6j9RYCC3NiV46K6GhN7RMu70u+/ET2skEdChn3Mu0OTYzqa\n"+
		"+qOCXvV+RWEiUoa2JX7UkSagEs+1O404afQv2+qnhdUOskxzUD+smQJBGOrXmdMq\n"+
		"ubzSn24LsPYWYGWsgl3AJ+n8rmVMXgPaWZQD9qHkZD9Oe2wwI9W+4K74\n"+
		"-----END RSA PRIVATE KEY-----\n") == nil {
		t.Fatal("DecodePrivateKey should not return nil")
		return
	}
}
