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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

const (
	SanitizerRuleVersion = "2026-10-10.v1"
	redactedValue        = "[REDACTED]"
	maxStructuredBytes   = 1 << 20
	maxStructuredDepth   = 32
)

const (
	RuleSensitiveField       = "secret.sensitive-field"
	RuleAuthorization        = "secret.authorization"
	RuleCredentialAssignment = "secret.credential-assignment"
	RulePrivateKey           = "secret.private-key"
	RuleJWT                  = "secret.jwt"
	RuleAPIKey               = "secret.api-key"
	RuleEncodedSecret        = "secret.encoded"
)

var (
	privateKeyPattern           = regexp.MustCompile(`(?is)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`)
	authorizationPattern        = regexp.MustCompile(`(?i)\b(?:bearer|basic)\s+[A-Za-z0-9._~+/=-]+`)
	jwtPattern                  = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{4,}\.[A-Za-z0-9_-]{4,}\.[A-Za-z0-9_-]{4,}\b`)
	apiKeyPattern               = regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]{16,}|gh[pousr]_[A-Za-z0-9]{16,}|AIza[A-Za-z0-9_-]{20,}|xox[baprs]-[A-Za-z0-9-]{16,})\b`)
	credentialAssignmentPattern = regexp.MustCompile(`(?i)\b(authorization|proxy[-_]?authorization|cookie|set[-_]?cookie|token|access[-_]?token|refresh[-_]?token|api[-_]?key|apikey|password|passwd|secret|client[-_]?secret|client[-_]?key[-_]?data|client[-_]?certificate[-_]?data)(\s*(?::|=|%3[dD])\s*)(?:"[^"\r\n]*"|'[^'\r\n]*'|[^\s,;&\r\n]+)`)
	encodedCandidatePattern     = regexp.MustCompile(`\b[A-Za-z0-9+/_-]{20,1000}={0,2}\b`)
)

// Finding describes a match without retaining the matched secret.
type Finding struct {
	RuleID string
	Path   string
	Count  int
}

type SanitizedText struct {
	Value       string
	RuleVersion string
	Findings    []Finding
}

type SanitizedHeaders struct {
	Value       http.Header
	RuleVersion string
	Findings    []Finding
}

type SanitizedURL struct {
	Value       *url.URL
	RuleVersion string
	Findings    []Finding
}

type SanitizedValue struct {
	Value       any
	RuleVersion string
	Findings    []Finding
}

// Sanitizer is immutable and safe for concurrent use.
type Sanitizer struct{}

var defaultSanitizer = &Sanitizer{}

func DefaultSanitizer() *Sanitizer { return defaultSanitizer }

func NewSanitizer() *Sanitizer { return &Sanitizer{} }

func (s *Sanitizer) SanitizeText(value string) SanitizedText {
	clean, findings := s.sanitizeText(value, "$", true)
	return SanitizedText{Value: clean, RuleVersion: SanitizerRuleVersion, Findings: findings}
}

func (s *Sanitizer) ContainsSecret(value string) ([]Finding, bool) {
	result := s.SanitizeText(value)
	return result.Findings, len(result.Findings) > 0
}

// ModelInput returns an ephemeral, redacted copy suitable for a model,
// embedding, or reranking provider. It never mutates the caller's value.
func (s *Sanitizer) ModelInput(value string) string {
	return s.SanitizeText(value).Value
}

func (s *Sanitizer) SanitizeHeaders(headers http.Header) SanitizedHeaders {
	clean := make(http.Header, len(headers))
	var findings []Finding
	for key, values := range headers {
		cleanKey := http.CanonicalHeaderKey(key)
		path := "header.field"
		if isSensitiveKey(key) {
			clean[cleanKey] = []string{redactedValue}
			findings = appendFinding(findings, RuleSensitiveField, path, len(values))
			continue
		}
		for _, value := range values {
			sanitized, matches := s.sanitizeText(value, path, true)
			clean[cleanKey] = append(clean[cleanKey], sanitized)
			findings = mergeFindings(findings, matches)
		}
	}
	return SanitizedHeaders{Value: clean, RuleVersion: SanitizerRuleVersion, Findings: findings}
}

func (s *Sanitizer) SanitizeURL(input *url.URL) SanitizedURL {
	if input == nil {
		return SanitizedURL{RuleVersion: SanitizerRuleVersion}
	}
	copy := *input
	var findings []Finding
	if copy.User != nil {
		if _, set := copy.User.Password(); set {
			copy.User = url.UserPassword(copy.User.Username(), redactedValue)
			findings = appendFinding(findings, RuleSensitiveField, "url.user.password", 1)
		}
	}
	query := copy.Query()
	for key, values := range query {
		path := "url.query.field"
		if isSensitiveKey(key) {
			query[key] = []string{redactedValue}
			findings = appendFinding(findings, RuleSensitiveField, path, len(values))
			continue
		}
		for i, value := range values {
			sanitized, valuesFindings := s.sanitizeText(value, path, true)
			query[key][i] = sanitized
			findings = mergeFindings(findings, valuesFindings)
		}
	}
	copy.RawQuery = query.Encode()
	return SanitizedURL{Value: &copy, RuleVersion: SanitizerRuleVersion, Findings: findings}
}

// SanitizeStructured returns a JSON-compatible copy. It never mutates value.
func (s *Sanitizer) SanitizeStructured(value any) SanitizedValue {
	encoded, err := json.Marshal(value)
	if err != nil || len(encoded) > maxStructuredBytes {
		result := s.SanitizeText(fmt.Sprint(value))
		return SanitizedValue{Value: result.Value, RuleVersion: result.RuleVersion, Findings: result.Findings}
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		result := s.SanitizeText(string(encoded))
		return SanitizedValue{Value: result.Value, RuleVersion: result.RuleVersion, Findings: result.Findings}
	}
	clean, findings := s.sanitizeValue(decoded, "$", 0)
	return SanitizedValue{Value: clean, RuleVersion: SanitizerRuleVersion, Findings: findings}
}

func (s *Sanitizer) sanitizeValue(value any, path string, depth int) (any, []Finding) {
	if depth >= maxStructuredDepth {
		return redactedValue, []Finding{{RuleID: RuleSensitiveField, Path: path, Count: 1}}
	}
	switch typed := value.(type) {
	case map[string]any:
		clean := make(map[string]any, len(typed))
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var findings []Finding
		for _, key := range keys {
			childPath := path + ".field"
			if isSensitiveKey(key) {
				clean[key] = redactedValue
				findings = appendFinding(findings, RuleSensitiveField, childPath, 1)
				continue
			}
			child, childFindings := s.sanitizeValue(typed[key], childPath, depth+1)
			clean[key] = child
			findings = mergeFindings(findings, childFindings)
		}
		return clean, findings
	case []any:
		clean := make([]any, len(typed))
		var findings []Finding
		for i := range typed {
			child, childFindings := s.sanitizeValue(typed[i], fmt.Sprintf("%s[%d]", path, i), depth+1)
			clean[i] = child
			findings = mergeFindings(findings, childFindings)
		}
		return clean, findings
	case string:
		return s.sanitizeText(typed, path, true)
	default:
		return value, nil
	}
}

func (s *Sanitizer) sanitizeText(value, path string, inspectEncoded bool) (string, []Finding) {
	clean := value
	var findings []Finding
	clean, findings = replaceMatches(clean, privateKeyPattern, RulePrivateKey, path, findings, func(string) string { return redactedValue })
	clean, findings = replaceMatches(clean, authorizationPattern, RuleAuthorization, path, findings, func(match string) string {
		kind := strings.Fields(match)
		if len(kind) == 0 {
			return redactedValue
		}
		return kind[0] + " " + redactedValue
	})
	clean, findings = replaceMatches(clean, credentialAssignmentPattern, RuleCredentialAssignment, path, findings, func(match string) string {
		parts := credentialAssignmentPattern.FindStringSubmatch(match)
		if len(parts) < 3 {
			return redactedValue
		}
		return parts[1] + parts[2] + redactedValue
	})
	clean, findings = replaceMatches(clean, jwtPattern, RuleJWT, path, findings, func(string) string { return redactedValue })
	clean, findings = replaceMatches(clean, apiKeyPattern, RuleAPIKey, path, findings, func(string) string { return redactedValue })
	if inspectEncoded {
		clean, findings = redactOversizedEncodedCandidates(clean, path, findings)
		if decoded, err := url.QueryUnescape(clean); err == nil && decoded != clean {
			_, decodedFindings := s.sanitizeText(decoded, path, false)
			if len(decodedFindings) > 0 {
				return redactedValue, appendFinding(findings, RuleEncodedSecret, path, 1)
			}
		}
	}
	if inspectEncoded {
		clean = encodedCandidatePattern.ReplaceAllStringFunc(clean, func(candidate string) string {
			decoded, ok := decodeCandidate(candidate)
			if !ok {
				return candidate
			}
			_, matches := s.sanitizeText(decoded, path, false)
			if len(matches) == 0 {
				return candidate
			}
			findings = appendFinding(findings, RuleEncodedSecret, path, 1)
			return redactedValue
		})
	}
	return clean, findings
}

func redactOversizedEncodedCandidates(value, path string, findings []Finding) (string, []Finding) {
	const maxCandidate = 1000
	isCandidate := func(b byte) bool {
		return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '+' || b == '/' || b == '_' || b == '-'
	}
	for start := 0; start < len(value); {
		if !isCandidate(value[start]) {
			start++
			continue
		}
		end := start
		for end < len(value) && isCandidate(value[end]) {
			end++
		}
		if end-start > maxCandidate {
			value = value[:start] + redactedValue + value[end:]
			findings = appendFinding(findings, RuleEncodedSecret, path, 1)
			start += len(redactedValue)
			continue
		}
		start = end
	}
	return value, findings
}

func replaceMatches(value string, pattern *regexp.Regexp, ruleID, path string, findings []Finding, replace func(string) string) (string, []Finding) {
	matches := pattern.FindAllStringIndex(value, -1)
	if len(matches) == 0 {
		return value, findings
	}
	findings = appendFinding(findings, ruleID, path, len(matches))
	return pattern.ReplaceAllStringFunc(value, replace), findings
}

func decodeCandidate(value string) (string, bool) {
	if len(value) > 1000 {
		return "", false
	}
	encodings := []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding}
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(value)
		if err == nil && len(decoded) > 0 && len(decoded) <= maxStructuredBytes {
			return string(decoded), true
		}
	}
	return "", false
}

func isSensitiveKey(key string) bool {
	normalized := strings.NewReplacer("-", "", "_", "", ".", "", " ", "").Replace(strings.ToLower(strings.TrimSpace(key)))
	switch normalized {
	case "authorization", "proxyauthorization", "cookie", "setcookie", "token", "accesstoken", "refreshtoken", "apikey", "password", "passwd", "secret", "clientsecret", "clientkeydata", "clientcertificatedata", "privatekey", "kubeconfig":
		return true
	default:
		for _, suffix := range []string{"authorization", "cookie", "token", "apikey", "password", "passwd", "secret", "privatekey", "keydata", "certificatedata"} {
			if strings.HasSuffix(normalized, suffix) {
				return true
			}
		}
		return false
	}
}

func appendFinding(findings []Finding, ruleID, path string, count int) []Finding {
	if count <= 0 {
		return findings
	}
	for i := range findings {
		if findings[i].RuleID == ruleID && findings[i].Path == path {
			findings[i].Count += count
			return findings
		}
	}
	return append(findings, Finding{RuleID: ruleID, Path: path, Count: count})
}

func mergeFindings(target, source []Finding) []Finding {
	for _, finding := range source {
		target = appendFinding(target, finding.RuleID, finding.Path, finding.Count)
	}
	return target
}
