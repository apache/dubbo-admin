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
	"io"
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

func TestSanitizeTextRedactsJSONCredentials(t *testing.T) {
	const secret = "review-fixture-secret"
	input := `{"password":"` + secret + `","enabled":true}`
	result := DefaultSanitizer().SanitizeText(input)
	if len(result.Findings) == 0 || strings.Contains(result.Value, secret) {
		t.Fatalf("JSON credential was not redacted: value=%q findings=%#v", result.Value, result.Findings)
	}
	if result.Value != `{"password":"[REDACTED]","enabled":true}` {
		t.Fatalf("JSON shape was not preserved: %q", result.Value)
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(input))
	encodedResult := DefaultSanitizer().SanitizeText(encoded)
	if len(encodedResult.Findings) == 0 || strings.Contains(encodedResult.Value, encoded) {
		t.Fatalf("encoded JSON credential was not redacted: value=%q findings=%#v", encodedResult.Value, encodedResult.Findings)
	}
}

func TestSanitizeTextRedactsEscapedQuotesInJSONCredentials(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "escaped quote inside value",
			input: `{"password":"prefix\"review-suffix-secret","enabled":true}`,
			want:  `{"password":"[REDACTED]","enabled":true}`,
		},
		{
			name:  "escaped backslash inside value",
			input: `{"secret":"prefix\\review-suffix-secret","enabled":true}`,
			want:  `{"secret":"[REDACTED]","enabled":true}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultSanitizer().SanitizeText(tt.input)
			if len(result.Findings) == 0 || strings.Contains(result.Value, "review-suffix-secret") {
				t.Fatalf("escaped credential leaked: value=%q findings=%#v", result.Value, result.Findings)
			}
			if result.Value != tt.want {
				t.Fatalf("SanitizeText() = %q, want %q", result.Value, tt.want)
			}
		})
	}
}

func TestSanitizeTextCoversCompoundKeysURLUserInfoAndJSONText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"environment password", `DB_PASSWORD=` + testSecret, `DB_PASSWORD=[REDACTED]`},
		{"environment api key", `OPENAI_API_KEY=` + testSecret, `OPENAI_API_KEY=[REDACTED]`},
		{"vendor token", `github_token=` + testSecret, `github_token=[REDACTED]`},
		{"aws secret", `AWS_SECRET_ACCESS_KEY=` + testSecret, `AWS_SECRET_ACCESS_KEY=[REDACTED]`},
		{"camel-case token", `{"idToken":"` + testSecret + `"}`, `{"idToken":"[REDACTED]"}`},
		{"short pass key", `pass=` + testSecret, `pass=[REDACTED]`},
		{"short pwd key", `pwd=` + testSecret, `pwd=[REDACTED]`},
		{"url userinfo", `https://user:` + testSecret + `@example.test/path`, `https://user:[REDACTED]@example.test/path`},
		{"json private key", `{"privateKey":"` + testSecret + `"}`, `{"privateKey":"[REDACTED]"}`},
		{"json kubeconfig", `{"kubeconfig":"` + testSecret + `"}`, `{"kubeconfig":"[REDACTED]"}`},
		{"aws access key id", `AKIAABCDEFGHIJKLMNOP`, `[REDACTED]`},
		{"unquoted value keeps json punctuation", `["password=` + testSecret + `"]`, `["password=[REDACTED]"]`},
		{"plain value with spaces", `password=my ` + testSecret + ` value`, `password=[REDACTED]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultSanitizer().SanitizeText(tt.input)
			if len(result.Findings) == 0 || strings.Contains(result.Value, testSecret) {
				t.Fatalf("credential leaked: value=%q findings=%#v", result.Value, result.Findings)
			}
			if result.Value != tt.want {
				t.Fatalf("SanitizeText() = %q, want %q", result.Value, tt.want)
			}
		})
	}

	// Ordinary prose must not be treated as a credential.
	for _, input := range []string{"basic usage of the API", "Bearer tokens are supported", "compass=south", "bypass=yes"} {
		result := DefaultSanitizer().SanitizeText(input)
		if len(result.Findings) != 0 || result.Value != input {
			t.Fatalf("prose was redacted: input=%q value=%q findings=%#v", input, result.Value, result.Findings)
		}
	}

	// Structured boundaries use the same key classification as the text rules.
	for _, key := range []string{"DB_PASSWORD", "OPENAI_API_KEY", "github_token", "AWS_SECRET_ACCESS_KEY", "idToken", "pass", "pwd", "privateKey", "kubeconfig"} {
		if !isSensitiveKey(key) {
			t.Fatalf("isSensitiveKey(%q) = false", key)
		}
	}
	for _, key := range []string{"compass", "bypass"} {
		if isSensitiveKey(key) {
			t.Fatalf("isSensitiveKey(%q) = true", key)
		}
	}
}

func TestSanitizeTextRedactsCookieListsAndSingleQuotedYAMLKeys(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "cookie header redacts every cookie in the list",
			input: "Cookie: theme=dark; JSESSIONID=" + testSecret,
			want:  "Cookie: [REDACTED]",
		},
		{
			name:  "set-cookie header redacts attributes too",
			input: "Set-Cookie: session=" + testSecret + "; Path=/; HttpOnly",
			want:  "Set-Cookie: [REDACTED]",
		},
		{
			name:  "cookie assignment redacts the semicolon-delimited list",
			input: "cookie=theme=dark; JSESSIONID=" + testSecret,
			want:  "cookie=[REDACTED]",
		},
		{
			name:  "single quoted YAML key and value",
			input: `'password': '` + testSecret + `'`,
			want:  `'password': '[REDACTED]'`,
		},
		{
			name:  "single quoted YAML key with plain value",
			input: `'api_key': ` + testSecret,
			want:  `'api_key': [REDACTED]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultSanitizer().SanitizeText(tt.input)
			if len(result.Findings) == 0 || strings.Contains(result.Value, testSecret) {
				t.Fatalf("credential leaked: value=%q findings=%#v", result.Value, result.Findings)
			}
			if result.Value != tt.want {
				t.Fatalf("SanitizeText() = %q, want %q", result.Value, tt.want)
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

func TestSanitizeURLRedactsUsernamePathFragmentAndOpaque(t *testing.T) {
	const secret = "sk-correct-horse-battery-staple"
	tests := []struct {
		name string
		raw  string
	}{
		{name: "username token", raw: "https://" + secret + "@example.test/"},
		{name: "path token", raw: "https://example.test/path/" + secret},
		{name: "fragment token", raw: "https://example.test/#token=" + secret},
		{name: "opaque value", raw: "mailto:password=" + secret + "@example.test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := url.Parse(tt.raw)
			if err != nil {
				t.Fatal(err)
			}
			result := DefaultSanitizer().SanitizeURL(parsed)
			if len(result.Findings) == 0 || strings.Contains(result.Value.String(), secret) {
				t.Fatalf("URL component leaked: value=%q findings=%#v", result.Value, result.Findings)
			}
			if parsed.String() != tt.raw {
				t.Fatalf("SanitizeURL mutated its input: %q", parsed)
			}
		})
	}

	normal, err := url.Parse("https://example.test/path?q=ok")
	if err != nil {
		t.Fatal(err)
	}
	clean := DefaultSanitizer().SanitizeURL(normal)
	if clean.Value.String() != normal.String() || len(clean.Findings) != 0 {
		t.Fatalf("ordinary URL was changed: value=%q findings=%#v", clean.Value, clean.Findings)
	}
}

func TestRedactingHandlerSanitizesTypedURLComponents(t *testing.T) {
	const secret = "sk-correct-horse-battery-staple"
	parsed, err := url.Parse("https://" + secret + "@example.test/path/" + secret + "#token=" + secret)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	logger := slog.New(NewRedactingHandler(slog.NewTextHandler(&output, nil)))
	logger.Info("request", "url", parsed)
	if strings.Contains(output.String(), secret) {
		t.Fatalf("typed URL log leaked a component secret: %s", output.String())
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
	if _, findings, err = policy.Prepare(`{"password":"`+testSecret+`"}`, nil); !errors.Is(err, ErrSensitiveDerivedData) || len(findings) == 0 {
		t.Fatalf("JSON secret Prepare() error = %v, findings = %#v", err, findings)
	}
	if _, findings, err = policy.Prepare("Cookie: theme=dark; JSESSIONID="+testSecret, nil); !errors.Is(err, ErrSensitiveDerivedData) || len(findings) == 0 {
		t.Fatalf("cookie Prepare() error = %v, findings = %#v", err, findings)
	}
	if _, findings, err = policy.Prepare(`'password': '`+testSecret+`'`, nil); !errors.Is(err, ErrSensitiveDerivedData) || len(findings) == 0 {
		t.Fatalf("YAML secret Prepare() error = %v, findings = %#v", err, findings)
	}
	for _, input := range []string{
		`DB_PASSWORD=` + testSecret,
		`AWS_SECRET_ACCESS_KEY=` + testSecret,
		`{"idToken":"` + testSecret + `"}`,
		`{"privateKey":"` + testSecret + `"}`,
		`{"kubeconfig":"` + testSecret + `"}`,
		`https://user:` + testSecret + `@example.test`,
	} {
		if _, findings, err = policy.Prepare(input, nil); !errors.Is(err, ErrSensitiveDerivedData) || len(findings) == 0 {
			t.Fatalf("Prepare(%q) error = %v, findings = %#v", input, err, findings)
		}
	}
}

func TestDerivedDataPolicyAllowsScalarListsOnly(t *testing.T) {
	policy := NewDerivedDataPolicy("header_path", "source")
	metadata, findings, err := policy.Prepare("safe document", map[string]any{
		"header_path": []string{"Guide", "Install"},
		"source":      "docs/guide.md",
	})
	if err != nil || len(findings) != 0 {
		t.Fatalf("Prepare() error = %v, findings = %#v", err, findings)
	}
	if len(metadata) != 2 {
		t.Fatalf("metadata = %#v", metadata)
	}

	if _, _, err := policy.Prepare("safe document", map[string]any{"source": map[string]any{"nested": "value"}}); !errors.Is(err, ErrSensitiveDerivedData) {
		t.Fatalf("nested map metadata accepted: %v", err)
	}
	if _, _, err := policy.Prepare("safe document", map[string]any{"header_path": []any{"ok", map[string]any{"nested": "value"}}}); !errors.Is(err, ErrSensitiveDerivedData) {
		t.Fatalf("nested list metadata accepted: %v", err)
	}
	if _, _, err := policy.Prepare("safe document", map[string]any{"header_path": []any{[]any{"nested"}}}); !errors.Is(err, ErrSensitiveDerivedData) {
		t.Fatalf("list of lists accepted: %v", err)
	}
}

func TestDerivedDataPolicyClonesMetadataSlices(t *testing.T) {
	headerPath := []string{"Guide"}
	metadata := map[string]any{"header_path": headerPath}
	clean, _, err := NewDerivedDataPolicy("header_path").Prepare("safe document", metadata)
	if err != nil {
		t.Fatal(err)
	}

	headerPath[0] = "password=mutated-in-original"
	cleanPath, ok := clean["header_path"].([]string)
	if !ok || cleanPath[0] != "Guide" {
		t.Fatalf("prepared metadata shares the caller's slice: %#v", clean["header_path"])
	}

	cleanPath[0] = "password=mutated-in-prepared"
	if headerPath[0] != "password=mutated-in-original" {
		t.Fatalf("caller's slice was aliased by prepared metadata: %#v", headerPath)
	}

	list := []any{"Guide"}
	cleanAny, _, err := NewDerivedDataPolicy("header_path").Prepare("safe document", map[string]any{"header_path": list})
	if err != nil {
		t.Fatal(err)
	}
	list[0] = "password=mutated-in-original"
	if cleanAny["header_path"].([]any)[0] != "Guide" {
		t.Fatalf("prepared []any metadata shares the caller's slice: %#v", cleanAny["header_path"])
	}
}

func TestSanitizeStructuredRedactsUninspectableValue(t *testing.T) {
	result := DefaultSanitizer().SanitizeStructured(make(chan int))
	if result.Value != redactedValue || len(result.Findings) == 0 {
		t.Fatalf("uninspectable value was not redacted: %#v", result)
	}
}

func TestSanitizeTextIsIdempotentForRedactedValues(t *testing.T) {
	inputs := []string{
		`["password=` + testSecret + `"]`,
		"token=" + testSecret,
		`Cookie: theme=dark; JSESSIONID=` + testSecret,
		`password="[REDACTED]"`,
		`{"password":"[REDACTED]"}`,
		`TOKEN="v1 sig=` + testSecret + `"`,
		`Error: {"password":["` + testSecret + `","second"]}`,
		`curl -d "{\"password\":\"` + testSecret + `\"}"`,
	}
	for _, input := range inputs {
		first := DefaultSanitizer().SanitizeText(input).Value
		second := DefaultSanitizer().SanitizeText(first).Value
		if second != first {
			t.Fatalf("sanitization is not idempotent: first=%q second=%q", first, second)
		}
	}
}

func TestSanitizeTextRedactsQuotedAssignmentValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "double quoted signed value",
			input: `TOKEN="v1 sig=` + testSecret + `"`,
			want:  `TOKEN="[REDACTED]"`,
		},
		{
			name:  "single quoted connection string",
			input: `CONNECTION_SECRET='host=10.0.0.1 pass_phrase=` + testSecret + `'`,
			want:  `CONNECTION_SECRET='[REDACTED]'`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultSanitizer().SanitizeText(tt.input)
			if len(result.Findings) == 0 || strings.Contains(result.Value, testSecret) {
				t.Fatalf("quoted credential leaked: value=%q findings=%#v", result.Value, result.Findings)
			}
			if result.Value != tt.want {
				t.Fatalf("SanitizeText() = %q, want %q", result.Value, tt.want)
			}
		})
	}
}

func TestSanitizeTextRedactsEmbeddedAndEscapedJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "prefixed JSON array value",
			input: `Error: {"password":["` + testSecret + `","second-secret"]}`,
			want:  `Error: {"password":"[REDACTED]"}`,
		},
		{
			name:  "top-level JSON string",
			input: `"{\"password\":\"` + testSecret + `\"}"`,
			want:  `"{\"password\":\"[REDACTED]\"}"`,
		},
		{
			name:  "shell-quoted JSON argument",
			input: `curl -d "{\"password\":\"` + testSecret + `\"}"`,
			want:  `curl -d "{\"password\":\"[REDACTED]\"}"`,
		},
		{
			name:  "double-encoded JSON string",
			input: `{"payload":"\"{\\\"password\\\":\\\"` + testSecret + `\\\"}\""}`,
			want:  `{"payload":"\"{\\\"password\\\":\\\"[REDACTED]\\\"}\""}`,
		},
		{
			name:  "NDJSON line",
			input: "line1\n" + `{"password":"` + testSecret + `","x":1}` + "\nline2\n",
			want:  "line1\n" + `{"password":"[REDACTED]","x":1}` + "\nline2\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultSanitizer().SanitizeText(tt.input)
			if len(result.Findings) == 0 || strings.Contains(result.Value, testSecret) {
				t.Fatalf("embedded credential leaked: value=%q findings=%#v", result.Value, result.Findings)
			}
			if result.Value != tt.want {
				t.Fatalf("SanitizeText() = %q, want %q", result.Value, tt.want)
			}
		})
	}
}

func TestSanitizeTextInspectsDecodedJSONKeys(t *testing.T) {
	plain := "{\"\\u0070assword\":\"encoded-key-secret\"}"
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "percent encoded JSON",
			input: `%7B%22%5Cu0070assword%22%3A%22encoded-key-secret%22%7D`,
		},
		{
			name:  "base64 encoded JSON",
			input: base64.StdEncoding.EncodeToString([]byte(plain)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultSanitizer().SanitizeText(tt.input)
			if len(result.Findings) == 0 || strings.Contains(result.Value, "encoded-key-secret") {
				t.Fatalf("decoded JSON credential leaked: value=%q findings=%#v", result.Value, result.Findings)
			}
		})
	}
}

func TestSanitizeTextPreservesPathPrefixForEncodedSecrets(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("password=" + testSecret))
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "file path", input: "/var/run/" + encoded, want: "/var/run/[REDACTED]"},
		{name: "URL path", input: "https://example.com/" + encoded, want: "https://example.com/[REDACTED]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultSanitizer().SanitizeText(tt.input)
			if len(result.Findings) == 0 || strings.Contains(result.Value, encoded) {
				t.Fatalf("encoded credential leaked: value=%q findings=%#v", result.Value, result.Findings)
			}
			if result.Value != tt.want {
				t.Fatalf("SanitizeText() = %q, want %q", result.Value, tt.want)
			}
		})
	}
}

func TestSanitizeTextFailsClosedWhenJSONFragmentBudgetExhausted(t *testing.T) {
	input := strings.Repeat("{", maxJSONFragmentAttempts) + `{"password":["` + testSecret + `"]}`
	result := DefaultSanitizer().SanitizeText(input)
	if len(result.Findings) == 0 || strings.Contains(result.Value, testSecret) {
		t.Fatalf("budget exhaustion leaked: value=%q findings=%#v", result.Value, result.Findings)
	}
	if result.Value != redactedValue {
		t.Fatalf("budget exhaustion did not fail closed: %q", result.Value)
	}
}

func TestSanitizeTextScansJSONFragmentsInLargeText(t *testing.T) {
	prefix := strings.Repeat("x", maxStructuredBytes+1)
	input := prefix + `{"password":["` + testSecret + `"]}`
	result := DefaultSanitizer().SanitizeText(input)
	if len(result.Findings) == 0 || strings.Contains(result.Value, testSecret) {
		t.Fatalf("large mixed text leaked: findings=%#v", result.Findings)
	}
	if !strings.HasPrefix(result.Value, prefix) {
		t.Fatalf("large text prefix was redacted")
	}
}

func TestSanitizeTextFailsClosedForOversizedEmbeddedJSON(t *testing.T) {
	input := strings.Repeat("x", 32) + `{"password":["` + strings.Repeat("a", maxStructuredBytes) + `"]}`
	result := DefaultSanitizer().SanitizeText(input)
	if result.Value != redactedValue || len(result.Findings) == 0 {
		t.Fatalf("oversized embedded JSON did not fail closed: value=%q findings=%#v", result.Value, result.Findings)
	}
}

func TestSanitizeTextRedactsEncodedSecretsWithSlashes(t *testing.T) {
	tests := []struct {
		name   string
		prefix []byte
	}{
		{name: "leading slash", prefix: []byte{0xff}},
		{name: "internal slash", prefix: []byte{0x00, 0x00, 0x00, 0xff}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := append(append([]byte(nil), tt.prefix...), []byte("password="+testSecret)...)
			encoded := base64.StdEncoding.EncodeToString(raw)
			if !strings.Contains(encoded, "/") {
				t.Fatalf("fixture does not contain a slash: %q", encoded)
			}

			input := "/var/run/" + encoded
			result := DefaultSanitizer().SanitizeText(input)
			if len(result.Findings) == 0 || strings.Contains(result.Value, encoded) {
				t.Fatalf("encoded secret leaked: value=%q findings=%#v", result.Value, result.Findings)
			}
			if !strings.HasPrefix(result.Value, "/var/run/") {
				t.Fatalf("path prefix was not preserved: %q", result.Value)
			}
		})
	}

	// An encoded secret may itself begin with '/', with no path prefix at all.
	encoded := base64.StdEncoding.EncodeToString(append([]byte{0xff}, []byte("password="+testSecret)...))
	result := DefaultSanitizer().SanitizeText(encoded)
	if len(result.Findings) == 0 || result.Value != redactedValue {
		t.Fatalf("leading-slash encoded secret was not redacted whole: value=%q findings=%#v", result.Value, result.Findings)
	}
}

func TestSanitizeTextRejectsIncompleteRedactionMarkers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "JSON value",
			input: `{"password":"[REDACTED"}`,
			want:  `{"password":"[REDACTED]"}`,
		},
		{
			name:  "authorization header",
			input: `Authorization: [REDACTED`,
			want:  `Authorization: [REDACTED]`,
		},
		{
			name:  "cookie header",
			input: `Cookie: [REDACTED`,
			want:  `Cookie: [REDACTED]`,
		},
		{
			name:  "unquoted assignment",
			input: `password=[REDACTED`,
			want:  `password=[REDACTED]`,
		},
		{
			name:  "unquoted assignment with trailing content",
			input: `password=[REDACTED malicious-data]`,
			want:  `password=[REDACTED]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultSanitizer().SanitizeText(tt.input)
			if len(result.Findings) == 0 || strings.Contains(result.Value, `"[REDACTED"`) {
				t.Fatalf("incomplete marker was accepted: value=%q findings=%#v", result.Value, result.Findings)
			}
			if result.Value != tt.want {
				t.Fatalf("SanitizeText() = %q, want %q", result.Value, tt.want)
			}
		})
	}

	structured := DefaultSanitizer().SanitizeStructured(map[string]any{"password": "[REDACTED"})
	clean, ok := structured.Value.(map[string]any)
	if !ok || clean["password"] != redactedValue || len(structured.Findings) == 0 {
		t.Fatalf("incomplete structured marker was accepted: %#v", structured)
	}
}

func TestSanitizeTextIgnoresRedactedPlaceholders(t *testing.T) {
	inputs := []string{
		`password="[REDACTED]"`,
		`{"password":"[REDACTED]","enabled":true}`,
		`Authorization: [REDACTED]`,
		`Cookie: [REDACTED]`,
	}
	for _, input := range inputs {
		result := DefaultSanitizer().SanitizeText(input)
		if len(result.Findings) != 0 || result.Value != input {
			t.Fatalf("placeholder was redacted again: input=%q value=%q findings=%#v", input, result.Value, result.Findings)
		}
	}

	headers := DefaultSanitizer().SanitizeHeaders(http.Header{
		"Authorization": {"[REDACTED]"},
		"X-Api-Key":     {"[REDACTED]"},
	})
	if len(headers.Findings) != 0 {
		t.Fatalf("redacted headers produced findings: %#v", headers.Findings)
	}

	parsed, err := url.Parse("https://user:%5BREDACTED%5D@example.test/?access_token=%5BREDACTED%5D")
	if err != nil {
		t.Fatal(err)
	}
	if cleanURL := DefaultSanitizer().SanitizeURL(parsed); len(cleanURL.Findings) != 0 {
		t.Fatalf("redacted URL produced findings: %#v", cleanURL.Findings)
	}

	// Documentation that shows a redacted example must remain indexable.
	policy := NewDerivedDataPolicy("source", "title")
	for _, input := range inputs {
		if _, findings, err := policy.Prepare(input, nil); err != nil || len(findings) != 0 {
			t.Fatalf("redacted placeholder was rejected for indexing: input=%q err=%v findings=%#v", input, err, findings)
		}
	}
}

func TestSanitizeJSONPartFailsClosedForUnparseableJSON(t *testing.T) {
	deep := strings.Repeat("[", maxJSONParseDepth+2) + `"` + testSecret + `"` + strings.Repeat("]", maxJSONParseDepth+2)
	tests := []struct {
		name  string
		input string
	}{
		{name: "truncated document", input: `{"password":"` + testSecret + `"`},
		{name: "deep nesting", input: deep},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultSanitizer().SanitizeJSONPart(tt.input)
			if result.Value != redactedValue || len(result.Findings) == 0 {
				t.Fatalf("unparseable JSON part was not redacted: value=%q findings=%#v", result.Value, result.Findings)
			}
		})
	}
}

func TestSanitizeJSONPartFailsClosedWhenOversized(t *testing.T) {
	oversized := `{"password":"` + testSecret + `","pad":"` + strings.Repeat("x", maxStructuredBytes) + `"}`
	result := DefaultSanitizer().SanitizeJSONPart(oversized)
	if result.Value != redactedValue || len(result.Findings) == 0 {
		t.Fatalf("oversized JSON part was not redacted: value=%q findings=%#v", result.Value, result.Findings)
	}

	small := `{"password":"` + testSecret + `"}`
	clean := DefaultSanitizer().SanitizeJSONPart(small)
	if len(clean.Findings) == 0 || strings.Contains(clean.Value, testSecret) {
		t.Fatalf("small JSON part leaked: value=%q findings=%#v", clean.Value, clean.Findings)
	}
}

func TestRedactingHandlerHandlesNilURL(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(NewRedactingHandler(slog.NewTextHandler(&output, nil)))
	var nilURL *url.URL
	logger.Info("request", "url", nilURL)
	if strings.Contains(output.String(), testSecret) {
		t.Fatalf("log output leaked secret: %s", output.String())
	}
}

func TestRedactingHandlerDoesNotMutateGroupAttrs(t *testing.T) {
	original := slog.Group("group", slog.String("password", testSecret))
	handler := NewRedactingHandler(slog.NewTextHandler(io.Discard, nil))
	_ = handler.WithAttrs([]slog.Attr{original})

	members := original.Value.Group()
	if len(members) != 1 || members[0].Value.String() != testSecret {
		t.Fatalf("caller's group attribute was mutated: %#v", members)
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

	// Single-character runs are separators or padding, not encoded secrets.
	for _, filler := range []string{"A", "-"} {
		candidate := strings.Repeat(filler, 1200)
		kept := DefaultSanitizer().SanitizeText(candidate)
		if len(kept.Findings) != 0 || kept.Value != candidate {
			t.Fatalf("%q run was conservatively redacted: %#v", filler, kept)
		}
	}

	// An oversized candidate that decodes to a credential is still redacted.
	payload := base64.StdEncoding.EncodeToString([]byte("password=" + testSecret + strings.Repeat("x", 900)))
	if len(payload) <= 1000 {
		t.Fatalf("fixture is not oversized: %d", len(payload))
	}
	oversized := DefaultSanitizer().SanitizeText(payload)
	if len(oversized.Findings) == 0 || oversized.Value == payload || strings.Contains(oversized.Value, payload) {
		t.Fatalf("oversized credential was not redacted: %#v", oversized)
	}

	// An oversized binary payload such as an embedded image is preserved.
	image := base64.StdEncoding.EncodeToString(append([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, make([]byte, 900)...))
	kept := DefaultSanitizer().SanitizeText(image)
	if len(kept.Findings) != 0 || kept.Value != image {
		t.Fatalf("binary payload was rejected: findings=%#v", kept.Findings)
	}
}

func TestRedactingHandlerSanitizesMessagesAttributesAndErrors(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(NewRedactingHandler(slog.NewTextHandler(&output, nil)))
	logger.Error("request failed: token="+testSecret,
		"authorization", "Bearer "+testSecret,
		"error", errors.New("password="+testSecret),
		"nested", map[string]any{"api_key": testSecret})
	logger.Info("JSON payload", "body", `{"password":"`+testSecret+`"}`)
	logger.Warn("cookie header", "header", "Cookie: theme=dark; JSESSIONID="+testSecret)
	logger.Info("YAML config", "body", `'password': '`+testSecret+`'`)
	logger.Warn("quoted cookie", "header", `Cookie: "theme=dark"; JSESSIONID=`+testSecret)
	logger.Info("YAML escaped quote", "body", `'password': 'ab''`+testSecret+`'`)
	logger.Info("encoded JSON key", "body", `{"%70assword":"`+testSecret+`"}`)
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
		// The sanitizer keeps the key and separator and replaces the value
		// with the marker. If the secret is a substring of either, a plain
		// Contains check would report a false leak.
		prefix := key + "="
		if strings.Contains(prefix, secret) || strings.Contains(redactedValue, secret) {
			t.Skip()
		}
		input := prefix + secret
		result := DefaultSanitizer().SanitizeText(input)
		if isSensitiveKey(key) && strings.Contains(result.Value, secret) {
			t.Fatalf("sanitized value contains matched secret")
		}
	})
}

func TestSanitizeTextRedactsNestedEscapedJSON(t *testing.T) {
	const secret = "mcp-secret-123"
	const privateKey = "mcp-priv-key"
	inputs := []string{
		`{"tool_name":"mcp_get_config","result":[{"type":"text","text":"{\"password\":\"` + secret + `\",\"privateKey\":\"` + privateKey + `\",\"enabled\":true}"}]}`,
		`{"data":"{\"\\u0070assword\":\"` + secret + `\"}"}`,
	}
	for _, input := range inputs {
		result := DefaultSanitizer().SanitizeText(input)
		if len(result.Findings) == 0 || strings.Contains(result.Value, secret) || strings.Contains(result.Value, privateKey) {
			t.Fatalf("nested JSON secret leaked: value=%q findings=%#v", result.Value, result.Findings)
		}
		if !json.Valid([]byte(result.Value)) {
			t.Fatalf("nested JSON sanitization produced invalid JSON: %q", result.Value)
		}
	}

	if _, findings, err := NewKnowledgeIndexPolicy().Prepare(inputs[0], nil); !errors.Is(err, ErrSensitiveDerivedData) || len(findings) == 0 {
		t.Fatalf("nested JSON was accepted by index policy: err=%v findings=%#v", err, findings)
	}
}

func TestSanitizeStructuredRedactsNestedJSONStrings(t *testing.T) {
	const secret = "structured-nested-secret"
	input := map[string]any{"data": `{"\u0070assword":"` + secret + `"}`}
	result := DefaultSanitizer().SanitizeStructured(input)
	encoded, err := json.Marshal(result.Value)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), secret) || len(result.Findings) == 0 {
		t.Fatalf("structured nested JSON leaked: value=%s findings=%#v", encoded, result.Findings)
	}
}

func TestSanitizeTextRedactsSensitiveKeySubtrees(t *testing.T) {
	inputs := []string{
		`{"password":["first-secret","second-secret"]}`,
		`{"kubeconfig":{"clusters":[{"server":"https://10.0.0.1"}],"users":[{"token_val":"raw-cluster-secret"}]}}`,
		`{"token":true}`,
		`{"password":12345}`,
	}
	for _, input := range inputs {
		result := DefaultSanitizer().SanitizeText(input)
		if len(result.Findings) == 0 {
			t.Fatalf("sensitive key was not reported: %q", input)
		}
		if !json.Valid([]byte(result.Value)) {
			t.Fatalf("sensitive key redaction produced invalid JSON: input=%q value=%q", input, result.Value)
		}
	}
	for _, leaked := range []string{"first-secret", "second-secret", "raw-cluster-secret", "12345"} {
		for _, input := range inputs {
			result := DefaultSanitizer().SanitizeText(input)
			if strings.Contains(result.Value, leaked) {
				t.Fatalf("sensitive subtree value %q leaked from %q: %q", leaked, input, result.Value)
			}
		}
	}
}

func TestSanitizeTextAvoidsProseFalsePositives(t *testing.T) {
	for _, input := range []string{"Pass: all tests passed", "Kubeconfig: path to cluster config"} {
		result := DefaultSanitizer().SanitizeText(input)
		if len(result.Findings) != 0 || result.Value != input {
			t.Fatalf("ordinary prose was redacted: input=%q value=%q findings=%#v", input, result.Value, result.Findings)
		}
	}

	input := "password=my-secret，帮我看看为什么连接超时？"
	result := DefaultSanitizer().SanitizeText(input)
	if !strings.Contains(result.Value, "帮我看看") || strings.Contains(result.Value, "my-secret") {
		t.Fatalf("sentence after credential was not preserved: %q", result.Value)
	}
}

func TestSanitizeTextRedactsPrefixedWeakKeys(t *testing.T) {
	for _, input := range []string{"MYSQL_PWD=" + testSecret, "DB_PASS=" + testSecret, `'pwd': ` + testSecret, `"pass": ` + testSecret} {
		result := DefaultSanitizer().SanitizeText(input)
		if len(result.Findings) == 0 || strings.Contains(result.Value, testSecret) {
			t.Fatalf("prefixed or quoted weak key leaked: input=%q value=%q findings=%#v", input, result.Value, result.Findings)
		}
	}
	if !isSensitiveKey("cluster_kubeconfig") {
		t.Fatal(`isSensitiveKey("cluster_kubeconfig") = false`)
	}
}

func TestSanitizeTextPreservesLogfmtContext(t *testing.T) {
	input := "password=" + testSecret + " host=10.0.0.1 reason=timeout"
	result := DefaultSanitizer().SanitizeText(input)
	if strings.Contains(result.Value, testSecret) {
		t.Fatalf("logfmt credential leaked: %q", result.Value)
	}
	if !strings.Contains(result.Value, "host=10.0.0.1") || !strings.Contains(result.Value, "reason=timeout") {
		t.Fatalf("logfmt context was swallowed: %q", result.Value)
	}

	// A later sensitive assignment on the same line must still be redacted.
	input = "password=" + testSecret + " token=" + testSecret + " user=alice"
	result = DefaultSanitizer().SanitizeText(input)
	if strings.Contains(result.Value, testSecret) {
		t.Fatalf("later logfmt credential leaked: %q", result.Value)
	}
	if !strings.Contains(result.Value, "user=alice") {
		t.Fatalf("logfmt context was swallowed: %q", result.Value)
	}
}

func TestSanitizeTextRescansAfterRedactionMarker(t *testing.T) {
	input := "password=[REDACTED token=" + testSecret
	result := DefaultSanitizer().SanitizeText(input)
	if strings.Contains(result.Value, testSecret) {
		t.Fatalf("assignment after a redaction marker was skipped: %q", result.Value)
	}
}

func TestSanitizeTextEncodedCandidateBoundaries(t *testing.T) {
	// 0xfb 0xff forces the base64 encoding to start with '+' and '/', which a
	// word-boundary regex would skip.
	raw := append([]byte{0xfb, 0xff}, []byte("password="+testSecret)...)
	encoded := base64.StdEncoding.EncodeToString(raw)
	if encoded[0] != '+' {
		t.Fatalf("fixture does not start with '+': %q", encoded[:4])
	}
	result := DefaultSanitizer().SanitizeText(encoded)
	if len(result.Findings) == 0 || strings.Contains(result.Value, encoded) {
		t.Fatalf("leading '+' base64 candidate was not redacted: value=%q findings=%#v", result.Value, result.Findings)
	}

	table := strings.Repeat("+---+", 300)
	kept := DefaultSanitizer().SanitizeText(table)
	if len(kept.Findings) != 0 || kept.Value != table {
		t.Fatalf("ASCII table rule was treated as a secret: findings=%#v", kept.Findings)
	}
}

func TestSanitizeTextToleratesStrayPercentEscapes(t *testing.T) {
	input := "CPU 80%, https://example.test/?pas%73word=" + testSecret
	result := DefaultSanitizer().SanitizeText(input)
	if len(result.Findings) == 0 || strings.Contains(result.Value, testSecret) {
		t.Fatalf("encoded query credential leaked after stray '%%': value=%q findings=%#v", result.Value, result.Findings)
	}
}

func TestSanitizeStructuredRedactsSensitiveMapKeys(t *testing.T) {
	const key = "sk-abcdefghijklmnopqrstuvwxyz123456"
	result := DefaultSanitizer().SanitizeStructured(map[string]any{key: "ok"})
	encoded, err := json.Marshal(result.Value)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) == 0 || strings.Contains(string(encoded), key) {
		t.Fatalf("sensitive map key leaked: value=%s findings=%#v", encoded, result.Findings)
	}
}

func TestRedactingHandlerRedactsSensitiveGroupsAndURLAuditAttrs(t *testing.T) {
	var output bytes.Buffer
	handler := NewRedactingHandler(slog.NewTextHandler(&output, nil))
	logger := slog.New(handler)

	parsed, err := url.Parse("https://user:" + testSecret + "@example.test/path?token=" + testSecret)
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("url", "profile", parsed, "headers", http.Header{"X-Token": {testSecret}})
	slog.New(handler).WithGroup("password").Info("grouped", "user", "alice", "note", testSecret)

	if strings.Contains(output.String(), testSecret) {
		t.Fatalf("log output leaked secret: %s", output.String())
	}
	if strings.Contains(output.String(), "alice") {
		t.Fatalf("attribute under a sensitive group was not redacted: %s", output.String())
	}
}

func TestSanitizeTextRedactsQuotedCookieAndAuthorizationValues(t *testing.T) {
	for _, input := range []string{
		`Cookie: "theme=dark"; JSESSIONID=` + testSecret,
		`Set-Cookie: "a=b"; JSESSIONID=` + testSecret,
		`Cookie="theme=dark"; JSESSIONID=` + testSecret,
		`Authorization: "x"; JSESSIONID=` + testSecret,
	} {
		result := DefaultSanitizer().SanitizeText(input)
		if len(result.Findings) == 0 || strings.Contains(result.Value, testSecret) {
			t.Fatalf("quoted header credential leaked: input=%q value=%q findings=%#v", input, result.Value, result.Findings)
		}
	}
}

func TestSanitizeTextRedactsYAMLEscapedSingleQuotes(t *testing.T) {
	for _, input := range []string{
		`'password': 'ab''` + testSecret + `'`,
		`'pass': 'ab''` + testSecret + `'`,
	} {
		result := DefaultSanitizer().SanitizeText(input)
		if len(result.Findings) == 0 || strings.Contains(result.Value, testSecret) {
			t.Fatalf("YAML escaped quote credential leaked: input=%q value=%q findings=%#v", input, result.Value, result.Findings)
		}
	}
}

func TestSanitizeTextRedactsPercentEncodedJSONKeys(t *testing.T) {
	input := `{"%70assword":"` + testSecret + `"}`
	result := DefaultSanitizer().SanitizeText(input)
	if len(result.Findings) == 0 || strings.Contains(result.Value, testSecret) {
		t.Fatalf("percent-encoded JSON key leaked: value=%q findings=%#v", result.Value, result.Findings)
	}
	if _, findings, err := NewKnowledgeIndexPolicy().Prepare(input, nil); !errors.Is(err, ErrSensitiveDerivedData) || len(findings) == 0 {
		t.Fatalf("percent-encoded JSON key was accepted by index policy: err=%v findings=%#v", err, findings)
	}
}

func TestSanitizeStructuredKeepsNonStringPassValues(t *testing.T) {
	result := DefaultSanitizer().SanitizeStructured(map[string]any{"pass": true, "pwd": 3})
	encoded, err := json.Marshal(result.Value)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), redactedValue) || len(result.Findings) != 0 {
		t.Fatalf("non-string pass/pwd field was redacted: value=%s findings=%#v", encoded, result.Findings)
	}
	result = DefaultSanitizer().SanitizeStructured(map[string]any{"pass": testSecret})
	encoded, _ = json.Marshal(result.Value)
	if strings.Contains(string(encoded), testSecret) || len(result.Findings) == 0 {
		t.Fatalf("string pass field was not redacted: value=%s findings=%#v", encoded, result.Findings)
	}
}

func TestSanitizeTextRedactsUnterminatedQuotedValues(t *testing.T) {
	inputs := []string{
		`pass="` + testSecret,
		`password: '` + testSecret,
		`password="prefix\"` + testSecret,
		`pass="00`,
	}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			result := DefaultSanitizer().SanitizeText(input)
			if len(result.Findings) == 0 {
				t.Fatalf("unterminated quoted value was not reported: %q", input)
			}
			if strings.Contains(result.Value, testSecret) || (input == `pass="00` && strings.Contains(result.Value, "00")) {
				t.Fatalf("unterminated quoted value leaked: input=%q value=%q findings=%#v", input, result.Value, result.Findings)
			}
		})
	}
}

func TestSanitizeTextUnterminatedFallbackKeepsTerminatedQuotes(t *testing.T) {
	input := `password: '` + testSecret + `'`
	result := DefaultSanitizer().SanitizeText(input)
	if result.Value != `password: '[REDACTED]'` {
		t.Fatalf("terminated quoted value was handled by the fallback: %q", result.Value)
	}
}
