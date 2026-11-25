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
	"reflect"
	"regexp"
	"sort"
	"strings"
)

const (
	SanitizerRuleVersion = "2026-11-24.v1"
	redactedValue        = "[REDACTED]"
	redactedPrefix       = "[REDACTED"
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

// credentialKeyExpr matches a sensitive key with an optional environment or
// vendor prefix (DB_PASSWORD, OPENAI_API_KEY, github_token). The left \b keeps
// the unprefixed alternatives from matching inside unrelated words such as
// "compass" or "bypass".
const credentialKeyExpr = `\b(?:[A-Za-z0-9_.\-]*(?:authorization|token|api[-_]?key|apikey|password|passwd|secret|private[-_]?key|key[-_]?data|certificate[-_]?data|access[-_]?key|kubeconfig)|pass|pwd)`

var (
	privateKeyPattern = regexp.MustCompile(`(?is)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`)
	// Authorization headers are redacted as a whole value. Matching only the
	// leading scheme word would leave the credential itself exposed.
	authorizationHeaderPattern = regexp.MustCompile(`(?i)\b((?:proxy[-_]?)?authorization)\b(\s*["']?\s*:\s*)(?:"(?:[^"\\\r\n]|\\.)*"|'(?:[^'\\\r\n]|\\.)*'|[^\r\n,;&]+)`)
	// A bare scheme only counts as a credential when the token has credential
	// shape, so ordinary prose such as "basic usage" or "Bearer tokens" is not
	// rejected by the derived-index policy.
	bearerTokenPattern = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/=-]{20,}\b`)
	jwtPattern         = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{4,}\.[A-Za-z0-9_-]{4,}\.[A-Za-z0-9_-]{4,}\b`)
	apiKeyPattern      = regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]{16,}|gh[pousr]_[A-Za-z0-9]{16,}|AIza[A-Za-z0-9_-]{20,}|xox[baprs]-[A-Za-z0-9-]{16,}|A[KS]IA[0-9A-Z]{16})\b`)
	// The unquoted value stops before quotes, closing JSON punctuation, and
	// record separators. It deliberately allows spaces so a YAML plain scalar
	// such as "password: my secret value" is redacted as a whole.
	credentialAssignmentPattern = regexp.MustCompile(`(?i)(` + credentialKeyExpr + `)(\s*["']?\s*(?::|=|%3[dD])\s*)(?:"(?:[^"\\\r\n]|\\.)*"|'(?:[^'\\\r\n]|\\.)*'|[^\r\n,;&"'\]\}]+)`)
	// Cookie headers carry a semicolon-delimited cookie list, so the value must
	// run to the end of the line; stopping at ';' would leave later cookies
	// (for example JSESSIONID) exposed.
	cookieHeaderPattern = regexp.MustCompile(`(?i)\b(set[-_]?cookie|cookie)\b(\s*["']?\s*:\s*)(?:"(?:[^"\\\r\n]|\\.)*"|'(?:[^'\\\r\n]|\\.)*'|[^\r\n]*)`)
	// cookie=... / cookie%3D... forms have no header line boundary, so redact
	// through the rest of the cookie list while preserving other query
	// parameters separated by '&'.
	cookieAssignmentPattern = regexp.MustCompile(`(?i)\b(set[-_]?cookie|cookie)\b(\s*["']?\s*(?:=|%3[dD])\s*)(?:"(?:[^"\\\r\n]|\\.)*"|'(?:[^'\\\r\n]|\\.)*'|[^&\r\n]*)`)
	// URL userinfo carries credentials in plain text, outside *url.URL values.
	urlUserInfoPattern      = regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.\-]*://[^/\s:@]*:)([^/\s@]+)(@)`)
	encodedCandidatePattern = regexp.MustCompile(`\b[A-Za-z0-9+/_-]{20,1000}={0,2}\b`)
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
	clean, findings := s.sanitizeText(value, "$", true, true)
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
			sanitized, matches := s.sanitizeText(value, path, true, true)
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
			sanitized, valuesFindings := s.sanitizeText(value, path, true, true)
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
		// An uninspectable value must not be forwarded. fmt.Sprint would change
		// the value's type and can expose unexported fields, so redact it whole.
		return SanitizedValue{
			Value:       redactedValue,
			RuleVersion: SanitizerRuleVersion,
			Findings:    []Finding{{RuleID: RuleSensitiveField, Path: "$", Count: 1}},
		}
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return SanitizedValue{
			Value:       redactedValue,
			RuleVersion: SanitizerRuleVersion,
			Findings:    []Finding{{RuleID: RuleSensitiveField, Path: "$", Count: 1}},
		}
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
		return s.sanitizeText(typed, path, true, false)
	default:
		return value, nil
	}
}

func (s *Sanitizer) sanitizeText(value, path string, inspectEncoded, parseJSON bool) (string, []Finding) {
	clean, findings := s.sanitizeTextRules(value, path, inspectEncoded)
	if parseJSON {
		if jsonClean, jsonFindings, ok := s.sanitizeJSONText(clean, path, findings); ok {
			return jsonClean, jsonFindings
		}
	}
	return clean, findings
}

func (s *Sanitizer) sanitizeTextRules(value, path string, inspectEncoded bool) (string, []Finding) {
	clean := value
	var findings []Finding
	clean, findings = replaceMatches(clean, privateKeyPattern, RulePrivateKey, path, findings, func(string) string { return redactedValue })
	clean, findings = replaceMatches(clean, authorizationHeaderPattern, RuleAuthorization, path, findings, redactCredentialValue(authorizationHeaderPattern))
	clean, findings = replaceMatches(clean, cookieHeaderPattern, RuleCredentialAssignment, path, findings, redactCredentialValue(cookieHeaderPattern))
	clean, findings = replaceMatches(clean, cookieAssignmentPattern, RuleCredentialAssignment, path, findings, redactCredentialValue(cookieAssignmentPattern))
	clean, findings = replaceMatches(clean, credentialAssignmentPattern, RuleCredentialAssignment, path, findings, redactCredentialValue(credentialAssignmentPattern))
	clean, findings = replaceMatches(clean, bearerTokenPattern, RuleAuthorization, path, findings, func(match string) string {
		kind := strings.Fields(match)
		if len(kind) == 0 {
			return redactedValue
		}
		return kind[0] + " " + redactedValue
	})
	clean, findings = replaceMatches(clean, jwtPattern, RuleJWT, path, findings, func(string) string { return redactedValue })
	clean, findings = replaceMatches(clean, apiKeyPattern, RuleAPIKey, path, findings, func(string) string { return redactedValue })
	clean, findings = replaceMatches(clean, urlUserInfoPattern, RuleCredentialAssignment, path, findings, redactURLUserInfo)
	if inspectEncoded {
		clean, findings = redactOversizedEncodedCandidates(clean, path, findings)
		if decoded, err := url.QueryUnescape(clean); err == nil && decoded != clean {
			_, decodedFindings := s.sanitizeText(decoded, path, false, false)
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
			_, matches := s.sanitizeText(decoded, path, false, false)
			if len(matches) == 0 {
				return candidate
			}
			findings = appendFinding(findings, RuleEncodedSecret, path, 1)
			return redactedValue
		})
	}
	return clean, findings
}

// sanitizeJSONText adds structured redaction for JSON text that the regex
// rules could not see (for example escaped key names). It re-encodes only when
// structured inspection changed a value, so ordinary JSON keeps its original
// bytes and key order.
func (s *Sanitizer) sanitizeJSONText(value, path string, findings []Finding) (string, []Finding, bool) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) == 0 || len(trimmed) > maxStructuredBytes {
		return "", nil, false
	}
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return "", nil, false
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return "", nil, false
	}
	cleanValue, jsonFindings := s.sanitizeJSONKeys(decoded, path, 0)
	if reflect.DeepEqual(decoded, cleanValue) {
		return value, findings, true
	}
	var builder strings.Builder
	encoder := json.NewEncoder(&builder)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(cleanValue); err != nil {
		return "", nil, false
	}
	return strings.TrimSuffix(builder.String(), "\n"), mergeFindings(findings, jsonFindings), true
}

// sanitizeJSONKeys applies the sensitive-key rule to an already text-sanitized
// JSON value. String contents are intentionally left alone: the regex pass
// already handled them, and re-running it would double-redact values that
// already contain the redaction marker.
func (s *Sanitizer) sanitizeJSONKeys(value any, path string, depth int) (any, []Finding) {
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
			child, childFindings := s.sanitizeJSONKeys(typed[key], childPath, depth+1)
			clean[key] = child
			findings = mergeFindings(findings, childFindings)
		}
		return clean, findings
	case []any:
		clean := make([]any, len(typed))
		var findings []Finding
		for i := range typed {
			child, childFindings := s.sanitizeJSONKeys(typed[i], fmt.Sprintf("%s[%d]", path, i), depth+1)
			clean[i] = child
			findings = mergeFindings(findings, childFindings)
		}
		return clean, findings
	default:
		return value, nil
	}
}

func redactOversizedEncodedCandidates(value, path string, findings []Finding) (string, []Finding) {
	const (
		maxCandidate           = 1000
		maxOversizedInputBytes = 4 << 20
	)
	isCandidate := func(b byte) bool {
		return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '+' || b == '/' || b == '_' || b == '-'
	}
	var builder strings.Builder
	builder.Grow(len(value))
	last := 0
	for start := 0; start < len(value); {
		if !isCandidate(value[start]) {
			start++
			continue
		}
		end := start
		var seen [256]bool
		distinct := 0
		for end < len(value) && isCandidate(value[end]) {
			if !seen[value[end]] {
				seen[value[end]] = true
				distinct++
			}
			end++
		}
		if end-start > maxCandidate && distinct > 1 {
			redact := true
			if end-start <= maxOversizedInputBytes {
				redact = oversizedCandidateContainsSecret(value[start:end])
			}
			if redact {
				builder.WriteString(value[last:start])
				builder.WriteString(redactedValue)
				findings = appendFinding(findings, RuleEncodedSecret, path, 1)
				last = end
			}
		}
		start = end
	}
	if last == 0 {
		return value, findings
	}
	builder.WriteString(value[last:])
	return builder.String(), findings
}

// oversizedCandidateContainsSecret decodes an oversized candidate and checks
// the decoded text. Candidates that cannot be decoded are treated as secrets
// (fail closed); decoded binary payloads such as embedded images usually
// produce no findings and are preserved.
func oversizedCandidateContainsSecret(candidate string) bool {
	decoded, ok := decodeOversizedCandidate(candidate)
	if !ok {
		return true
	}
	_, findings := defaultSanitizer.sanitizeText(decoded, "$", false, false)
	return len(findings) > 0
}

func decodeOversizedCandidate(value string) (string, bool) {
	encodings := []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding}
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(value)
		if err == nil && len(decoded) > 0 {
			return string(decoded), true
		}
	}
	return "", false
}

func redactCredentialValue(pattern *regexp.Regexp) func(string) string {
	return func(match string) string {
		parts := pattern.FindStringSubmatch(match)
		if len(parts) < 3 {
			return redactedValue
		}
		value := match[len(parts[1])+len(parts[2]):]
		// Re-sanitizing an already redacted value must be a no-op. The
		// unquoted branch can leave the trailing ']' outside the match.
		if strings.HasPrefix(value, redactedPrefix) {
			return match
		}
		return parts[1] + parts[2] + quotedRedactedValue(value)
	}
}

// quotedRedactedValue preserves the quoting of the value itself, not the key,
// so single-quoted YAML keys do not change the value's scalar style.
func quotedRedactedValue(value string) string {
	switch {
	case strings.HasPrefix(value, `"`):
		return `"` + redactedValue + `"`
	case strings.HasPrefix(value, `'`):
		return `'` + redactedValue + `'`
	default:
		return redactedValue
	}
}

func redactURLUserInfo(match string) string {
	parts := urlUserInfoPattern.FindStringSubmatch(match)
	if len(parts) < 4 {
		return redactedValue
	}
	return parts[1] + redactedValue + parts[3]
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
	case "authorization", "proxyauthorization", "cookie", "setcookie", "token", "accesstoken", "refreshtoken", "apikey", "password", "passwd", "pass", "pwd", "secret", "clientsecret", "clientkeydata", "clientcertificatedata", "privatekey", "kubeconfig":
		return true
	default:
		for _, suffix := range []string{"authorization", "cookie", "token", "apikey", "password", "passwd", "secret", "privatekey", "keydata", "certificatedata", "accesskey"} {
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
