/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package security

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const testSecret = "correct-horse-battery-staple"

func TestSanitizeTextCoversSecretVariants(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("password=" + testSecret))
	tests := []string{
		"Authorization: Bearer " + testSecret,
		"cookie=session=" + testSecret,
		"API_KEY=" + testSecret,
		"password%3D" + testSecret,
		"token=eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTYifQ.signature123",
		"-----BEGIN PRIVATE KEY-----\n" + testSecret + "\n-----END PRIVATE KEY-----",
		encoded,
		"key=sk-abcdefghijklmnopqrstuvwxyz123456",
		"https://example.test/?access%5Ftoken%3D" + testSecret,
	}
	for _, input := range tests {
		t.Run(input[:min(12, len(input))], func(t *testing.T) {
			result := DefaultSanitizer().SanitizeText(input)
			if len(result.Findings) == 0 {
				t.Fatalf("SanitizeText(%q) reported no findings", input)
			}
			if strings.Contains(result.Value, testSecret) || result.Value == input {
				t.Fatalf("SanitizeText() leaked input: %q", result.Value)
			}
			for _, finding := range result.Findings {
				if strings.Contains(finding.RuleID, testSecret) || strings.Contains(finding.Path, testSecret) {
					t.Fatalf("finding leaked secret: %#v", finding)
				}
			}
		})
	}
}

func TestSanitizeHeadersURLAndNestedValues(t *testing.T) {
	headers := http.Header{
		"Authorization": {"Bearer " + testSecret},
		"X-API-Key":     {testSecret},
		"X-Metadata":    {"password=" + testSecret},
	}
	cleanHeaders := DefaultSanitizer().SanitizeHeaders(headers)
	if got := cleanHeaders.Value.Get("Authorization"); strings.Contains(got, testSecret) || got == "" {
		t.Fatalf("authorization = %q", got)
	}
	if got := cleanHeaders.Value.Get("X-API-Key"); strings.Contains(got, testSecret) || got == "" {
		t.Fatalf("X-API-Key = %q", got)
	}

	parsed, err := url.Parse("https://user:" + testSecret + "@example.test/path?access_token=" + testSecret + "&q=ok")
	if err != nil {
		t.Fatal(err)
	}
	cleanURL := DefaultSanitizer().SanitizeURL(parsed)
	if strings.Contains(cleanURL.Value.String(), testSecret) || cleanURL.Value.Query().Get("q") != "ok" {
		t.Fatalf("sanitized URL = %s", cleanURL.Value)
	}

	input := map[string]any{
		"outer": []any{map[string]any{"PaSsWoRd": testSecret, "message": "token=" + testSecret}},
	}
	result := DefaultSanitizer().SanitizeStructured(input)
	encoded, err := json.Marshal(result.Value)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), testSecret) || len(result.Findings) != 2 {
		t.Fatalf("structured result = %s, findings = %#v", encoded, result.Findings)
	}
	if input["outer"].([]any)[0].(map[string]any)["PaSsWoRd"] != testSecret {
		t.Fatal("SanitizeStructured mutated canonical input")
	}
}

func TestDerivedDataPolicyRejectsSecretsAndFiltersMetadata(t *testing.T) {
	policy := NewDerivedDataPolicy("source", "title")
	metadata, findings, err := policy.Prepare("safe document", map[string]any{
		"source": "docs/guide.md",
		"title":  "Guide",
		"owner":  "must-not-be-indexed",
	})
	if err != nil || len(findings) != 0 {
		t.Fatalf("Prepare() error = %v, findings = %#v", err, findings)
	}
	if len(metadata) != 2 || metadata["owner"] != nil {
		t.Fatalf("metadata = %#v", metadata)
	}
	if _, findings, err = policy.Prepare("api_key="+testSecret, nil); !errors.Is(err, ErrSensitiveDerivedData) || len(findings) == 0 {
		t.Fatalf("secret Prepare() error = %v, findings = %#v", err, findings)
	}
}

func TestSanitizerBoundsEncodedCandidatesAndFindingPaths(t *testing.T) {
	key := "correct-horse-battery-staple-password"
	result := DefaultSanitizer().SanitizeStructured(map[string]any{key: testSecret})
	for _, finding := range result.Findings {
		if strings.Contains(finding.Path, key) {
			t.Fatalf("finding path contains untrusted key: %#v", finding)
		}
	}
	longCandidate := strings.Repeat("A", 1001)
	bounded := DefaultSanitizer().SanitizeText(longCandidate)
	if len(bounded.Findings) == 0 || bounded.Value == longCandidate || strings.Contains(bounded.Value, longCandidate) {
		t.Fatalf("long candidate was not conservatively redacted: %#v", bounded)
	}
}

func TestRedactingHandlerSanitizesMessagesAttributesAndErrors(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(NewRedactingHandler(slog.NewTextHandler(&output, nil)))
	logger.Error("request failed: token="+testSecret,
		"authorization", "Bearer "+testSecret,
		"error", errors.New("password="+testSecret),
		"nested", map[string]any{"api_key": testSecret})
	if strings.Contains(output.String(), testSecret) {
		t.Fatalf("log output leaked secret: %s", output.String())
	}
}

func FuzzSanitizeTextNeverReturnsMatchedSecret(f *testing.F) {
	f.Add("token", testSecret)
	f.Add("Authorization", "Bearer abc.def.ghi")
	f.Fuzz(func(t *testing.T, key, secret string) {
		if secret == "" || len(secret) > 4096 || strings.ContainsAny(secret, "\r\n;& ,") {
			t.Skip()
		}
		input := key + "=" + secret
		result := DefaultSanitizer().SanitizeText(input)
		if isSensitiveKey(key) && strings.Contains(result.Value, secret) {
			t.Fatalf("sanitized value contains matched secret")
		}
	})
}
