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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

const (
	SanitizerRuleVersion = "2026-11-28.v4"
	redactedValue        = "[REDACTED]"
	maxStructuredBytes   = 1 << 20
	maxStructuredDepth   = 32
	// maxJSONParseDepth bounds parser recursion. It is higher than
	// maxStructuredDepth so ordinary JSON is still parsed and redacted at the
	// structured depth, while pathological nesting cannot exhaust the stack.
	maxJSONParseDepth = 256
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

const (
	// strongCredentialKeyExpr matches keys that always denote a credential,
	// with an optional environment or vendor prefix (DB_PASSWORD,
	// OPENAI_API_KEY, github_token).
	strongCredentialKeyExpr = `\b(?:[A-Za-z0-9_.\-]*(?:authorization|token|api[-_]?key|apikey|password|passwd|secret|private[-_]?key|key[-_]?data|certificate[-_]?data|access[-_]?key))`
	// weakCredentialKeyExpr matches ambiguous short keys. They only count with
	// an '=' separator, a quoted value, or a structured value, so ordinary
	// prose such as "Pass: all tests passed" is not redacted.
	weakCredentialKeyExpr = `\b(?:[A-Za-z0-9_.\-]*[-_.])?(?:pass|pwd|kubeconfig)\b`
	kubeconfigKeyExpr     = `\b(?:[A-Za-z0-9_.\-]*[-_.])?kubeconfig\b`
	// Double-quoted values use backslash escapes. Single-quoted YAML values
	// escape a quote by doubling it (''), so both forms are accepted.
	doubleQuotedValueExpr = `"(?:[^"\\\r\n]|\\.)*"`
	singleQuotedValueExpr = `'(?:[^'\\\r\n]|''|\\.)*'`
	// A truncated provider error or log line can contain an opening quote with
	// no closing quote. That form runs to the end of the line so it stays
	// fail-closed. It is ordered after the terminated forms, which win when a
	// closing quote exists.
	unterminatedQuotedValueExpr = `["'][^\r\n]*`
	// The unquoted branch must start with a non-whitespace character: without
	// this, a value such as "password: 'secret" would match only the space and
	// leave the secret visible. Once started, spaces are part of the value.
	// The unquoted branch also excludes the opening brackets of a JSON object
	// or array. Otherwise "password":[a,b] would consume only the '[' and leave
	// the array elements visible. Balanced JSON is handled structurally before
	// the text rules run.
	unquotedValueExpr = `[^\s\r\n,;&"'\]\}\[\{，。！？、；][^\r\n,;&"'\]\}\[\{，。！？、；]*`
	// credentialValueExpr consumes a quoted scalar (escape-aware), an
	// unterminated quote to end of line, an unquoted scalar, or an already
	// redacted marker. The marker alternative is matched here (instead of the
	// unquoted branch, which excludes '[') so an incomplete marker followed by
	// more content is still redacted rather than silently accepted. The
	// unquoted branch stops at record separators and sentence punctuation so
	// prose after a credential is preserved.
	credentialValueExpr = doubleQuotedValueExpr + `|` + singleQuotedValueExpr + `|` + unterminatedQuotedValueExpr + `|` + `\[REDACTED[^\]\r\n]*\]?` + `|` + unquotedValueExpr
)

var (
	// errJSONDepth marks a document that exceeded maxJSONParseDepth. Callers
	// that know the input was truncated by the structural bound treat it as
	// fail-closed rather than as ordinary non-JSON text.
	errJSONDepth      = errors.New("JSON nesting too deep")
	privateKeyPattern = regexp.MustCompile(`(?is)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`)
	// Authorization headers are redacted as a whole value. The value runs to
	// the end of the line: RFC 6265 credentials may be quoted and followed by
	// more fields, so a quoted-value branch would stop early and leak the rest.
	authorizationHeaderPattern = regexp.MustCompile(`(?i)\b((?:proxy[-_]?)?authorization)\b(\s*["']?\s*:\s*)(?:[^\r\n]*)`)
	// A bare scheme only counts as a credential when the token has credential
	// shape, so ordinary prose such as "basic usage" or "Bearer tokens" is not
	// rejected by the derived-index policy.
	bearerTokenPattern = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/=-]{20,}\b`)
	jwtPattern         = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{4,}\.[A-Za-z0-9_-]{4,}\.[A-Za-z0-9_-]{4,}\b`)
	apiKeyPattern      = regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]{16,}|gh[pousr]_[A-Za-z0-9]{16,}|AIza[A-Za-z0-9_-]{20,}|xox[baprs]-[A-Za-z0-9-]{16,}|A[KS]IA[0-9A-Z]{16})\b`)
	// Credential assignments are split by key strength so ambiguous keys do
	// not match ordinary prose.
	credentialAssignmentPattern      = regexp.MustCompile(`(?i)(` + strongCredentialKeyExpr + `)(\s*["']?\s*(?::|=|%3[dD])\s*)(?:` + credentialValueExpr + `)`)
	weakCredentialEqualsPattern      = regexp.MustCompile(`(?i)(` + weakCredentialKeyExpr + `)(\s*["']?\s*(?:=|%3[dD])\s*)(?:` + credentialValueExpr + `)`)
	weakCredentialQuotedKeyPattern   = regexp.MustCompile(`(?i)(["']` + weakCredentialKeyExpr + `["'])(\s*:\s*)(?:` + credentialValueExpr + `)`)
	weakCredentialQuotedColonPattern = regexp.MustCompile(`(?i)(` + weakCredentialKeyExpr + `)(\s*["']?\s*:\s*)(?:` + doubleQuotedValueExpr + `|` + singleQuotedValueExpr + `|` + unterminatedQuotedValueExpr + `)`)
	kubeconfigStructuredPattern      = regexp.MustCompile(`(?i)(` + kubeconfigKeyExpr + `)(\s*["']?\s*:\s*)(?:[\[{][^\r\n]*)`)
	// Cookie headers carry a semicolon-delimited cookie list, so the value must
	// run to the end of the line; a quoted-value branch would stop at the
	// closing quote and leave later cookies (for example JSESSIONID) exposed.
	cookieHeaderPattern = regexp.MustCompile(`(?i)\b(set[-_]?cookie|cookie)\b(\s*["']?\s*:\s*)(?:[^\r\n]*)`)
	// cookie=... / cookie%3D... forms have no header line boundary, so redact
	// through the rest of the cookie list while preserving other query
	// parameters separated by '&'.
	cookieAssignmentPattern = regexp.MustCompile(`(?i)\b(set[-_]?cookie|cookie)\b(\s*["']?\s*(?:=|%3[dD])\s*)(?:[^&\r\n]*)`)
	// URL userinfo carries credentials in plain text, outside *url.URL values.
	urlUserInfoPattern = regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.\-]*://[^/\s:@]*:)([^/\s@]+)(@)`)
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

// SanitizeJSONPart sanitizes text that is declared as JSON. Documents that are
// too large, unparseable, or nested beyond the parser bound are redacted whole:
// falling back to the text rules would leave array elements and nested escaped
// documents uninspected.
func (s *Sanitizer) SanitizeJSONPart(value string) SanitizedText {
	if len(value) > maxStructuredBytes {
		return redactedSanitizedText()
	}
	clean, findings, ok := s.sanitizeJSONText(value, "$", 0, true)
	if !ok {
		return redactedSanitizedText()
	}
	return SanitizedText{Value: clean, RuleVersion: SanitizerRuleVersion, Findings: findings}
}

func redactedSanitizedText() SanitizedText {
	return SanitizedText{
		Value:       redactedValue,
		RuleVersion: SanitizerRuleVersion,
		Findings:    []Finding{{RuleID: RuleSensitiveField, Path: "$", Count: 1}},
	}
}

func (s *Sanitizer) SanitizeHeaders(headers http.Header) SanitizedHeaders {
	clean := make(http.Header, len(headers))
	var findings []Finding
	for key, values := range headers {
		cleanKey := http.CanonicalHeaderKey(key)
		path := "header.field"
		if isSensitiveKey(key) {
			clean[cleanKey] = []string{redactedValue}
			if !allRedactedValues(values) {
				findings = appendFinding(findings, RuleSensitiveField, path, len(values))
			}
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
		username := copy.User.Username()
		password, hasPassword := copy.User.Password()
		cleanUsername, usernameFindings := s.sanitizeText(username, "url.user.username", true, true)
		findings = mergeFindings(findings, usernameFindings)
		if hasPassword {
			copy.User = url.UserPassword(cleanUsername, redactedValue)
			if !isRedactedCredentialValue(password) {
				findings = appendFinding(findings, RuleSensitiveField, "url.user.password", 1)
			}
		} else {
			copy.User = url.User(cleanUsername)
		}
	}
	// Every URL component can carry a token: a username, a host subdomain, a
	// path segment, a fragment, or an opaque value. Sanitize each before the
	// string is logged or forwarded to a provider.
	if cleanHost, hostFindings := s.sanitizeText(copy.Host, "url.host", true, true); cleanHost != copy.Host {
		copy.Host = cleanHost
		findings = mergeFindings(findings, hostFindings)
	}
	if cleanPath, pathFindings := s.sanitizeText(copy.Path, "url.path", true, true); cleanPath != copy.Path {
		copy.Path = cleanPath
		// RawPath is only valid when it encodes Path; let String re-encode.
		copy.RawPath = ""
		findings = mergeFindings(findings, pathFindings)
	}
	if cleanFragment, fragmentFindings := s.sanitizeText(copy.Fragment, "url.fragment", true, true); cleanFragment != copy.Fragment {
		copy.Fragment = cleanFragment
		copy.RawFragment = ""
		findings = mergeFindings(findings, fragmentFindings)
	}
	if cleanOpaque, opaqueFindings := s.sanitizeText(copy.Opaque, "url.opaque", true, true); cleanOpaque != copy.Opaque {
		copy.Opaque = cleanOpaque
		findings = mergeFindings(findings, opaqueFindings)
	}
	query := copy.Query()
	for key, values := range query {
		path := "url.query.field"
		if isSensitiveKey(key) {
			query[key] = []string{redactedValue}
			if !allRedactedValues(values) {
				findings = appendFinding(findings, RuleSensitiveField, path, len(values))
			}
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
			if isSensitiveKeyDecoded(key) {
				if str, isString := typed[key].(string); isString && isRedactedCredentialValue(str) {
					// Already redacted; keep the marker without a new finding.
					clean[key] = str
					continue
				}
				if _, isString := typed[key].(string); !isString && isWeakPassKey(key) {
					// Test-result style fields such as {"pass": true} are not
					// credentials. Still recurse in case the value hides one.
					child, childFindings := s.sanitizeValue(typed[key], childPath, depth+1)
					clean[key] = child
					findings = mergeFindings(findings, childFindings)
					continue
				}
				clean[key] = redactedValue
				findings = appendFinding(findings, RuleSensitiveField, childPath, 1)
				continue
			}
			cleanKey, keyFindings := s.sanitizeTextRules(key, childPath, true)
			if len(keyFindings) > 0 {
				// The key itself carries credential material; redact the pair.
				clean[redactedValue] = redactedValue
				findings = mergeFindings(findings, keyFindings)
				findings = appendFinding(findings, RuleSensitiveField, childPath, 1)
				continue
			}
			child, childFindings := s.sanitizeValue(typed[key], childPath, depth+1)
			clean[cleanKey] = child
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
		// Nested JSON text (for example an MCP result[].text payload) must be
		// inspected structurally before the text rules see it.
		return s.sanitizeStringValue(typed, path, depth, true)
	default:
		return value, nil
	}
}

func (s *Sanitizer) sanitizeText(value, path string, inspectEncoded, parseJSON bool) (string, []Finding) {
	if parseJSON {
		if cleaned, findings, ok := s.sanitizeJSONText(value, path, 0, inspectEncoded); ok {
			return cleaned, findings
		}
		// Mixed text (an error line, NDJSON, or a shell-quoted JSON argument)
		// cannot be parsed as one document. Sanitize the balanced JSON fragments
		// structurally first, then apply the text rules to the rest.
		if cleaned, findings, ok := s.sanitizeEmbeddedJSON(value, path, inspectEncoded); ok {
			textCleaned, textFindings := s.sanitizeTextRules(cleaned, path, inspectEncoded)
			return textCleaned, mergeFindings(findings, textFindings)
		}
	}
	return s.sanitizeTextRules(value, path, inspectEncoded)
}

// sanitizeStringValue sanitizes a string that may itself contain a JSON
// document. JSON is inspected first so the text rules cannot damage its
// structure; non-JSON strings fall through to the text rules.
func (s *Sanitizer) sanitizeStringValue(value, path string, depth int, inspectEncoded bool) (string, []Finding) {
	if depth < maxStructuredDepth {
		if cleaned, findings, ok := s.sanitizeJSONText(value, path, depth+1, inspectEncoded); ok {
			return cleaned, findings
		}
	}
	return s.sanitizeTextRules(value, path, inspectEncoded)
}

func (s *Sanitizer) sanitizeTextRules(value, path string, inspectEncoded bool) (string, []Finding) {
	clean := value
	var findings []Finding
	clean, findings = replaceMatches(clean, privateKeyPattern, RulePrivateKey, path, findings, func(string) string { return redactedValue })
	clean, findings = replaceMatches(clean, authorizationHeaderPattern, RuleAuthorization, path, findings, redactCredentialValue(authorizationHeaderPattern))
	clean, findings = replaceMatches(clean, cookieHeaderPattern, RuleCredentialAssignment, path, findings, redactCredentialValue(cookieHeaderPattern))
	clean, findings = replaceMatches(clean, cookieAssignmentPattern, RuleCredentialAssignment, path, findings, redactCredentialValue(cookieAssignmentPattern))
	clean, findings = s.redactAssignments(clean, path, credentialAssignmentPattern, findings)
	clean, findings = s.redactAssignments(clean, path, weakCredentialEqualsPattern, findings)
	clean, findings = replaceMatches(clean, weakCredentialQuotedKeyPattern, RuleCredentialAssignment, path, findings, redactCredentialValue(weakCredentialQuotedKeyPattern))
	clean, findings = replaceMatches(clean, weakCredentialQuotedColonPattern, RuleCredentialAssignment, path, findings, redactCredentialValue(weakCredentialQuotedColonPattern))
	clean, findings = replaceMatches(clean, kubeconfigStructuredPattern, RuleCredentialAssignment, path, findings, redactCredentialValue(kubeconfigStructuredPattern))
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
		clean, findings = s.redactEncodedCandidates(clean, path, findings)
		if decoded := decodePercentSequences(clean); decoded != clean {
			// Decoded JSON may use escaped keys such as "password", so
			// inspect it structurally before falling back to the text rules.
			_, decodedFindings := s.sanitizeText(decoded, path, false, true)
			if len(decodedFindings) > 0 {
				return redactedValue, appendFinding(findings, RuleEncodedSecret, path, 1)
			}
		}
	}
	return clean, findings
}

// sanitizeJSONText structurally inspects a complete JSON document. It keeps
// the original bytes and key order when nothing changed and re-encodes with
// the original member order when a value is redacted.
func (s *Sanitizer) sanitizeJSONText(value, path string, depth int, inspectEncoded bool) (string, []Finding, bool) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) == 0 || len(trimmed) > maxStructuredBytes {
		return "", nil, false
	}
	// A top-level JSON string can itself contain an encoded JSON document,
	// for example `"{\"password\":\"...\"}"` or a shell-quoted curl body.
	if trimmed[0] != '{' && trimmed[0] != '[' && trimmed[0] != '"' {
		return "", nil, false
	}
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.UseNumber()
	original, err := parseJSONNode(decoder)
	if err != nil {
		return "", nil, false
	}
	// Reject trailing content after the first JSON value.
	if _, err := decoder.Token(); err != io.EOF {
		return "", nil, false
	}
	clean, findings := s.sanitizeJSONNode(original, path, depth, inspectEncoded)
	if reflect.DeepEqual(original, clean) {
		return value, findings, true
	}
	var builder strings.Builder
	builder.Grow(len(value))
	clean.encode(&builder)
	return builder.String(), findings, true
}

// maxJSONFragmentAttempts bounds how many embedded JSON candidates are parsed
// for one text value, so a pathological input cannot make sanitization
// quadratic.
const maxJSONFragmentAttempts = 256

// sanitizeEmbeddedJSON finds balanced JSON values inside mixed text and
// sanitizes each structurally. It handles prefixed errors ("Error: {...}"),
// NDJSON, and JSON documents quoted as shell arguments. The returned bool is
// false when no JSON fragment was found, letting the caller fall back to the
// text rules.
func (s *Sanitizer) sanitizeEmbeddedJSON(value, path string, inspectEncoded bool) (string, []Finding, bool) {
	var builder strings.Builder
	builder.Grow(len(value))
	var findings []Finding
	last := 0
	found := false
	for i, attempts := 0, 0; i < len(value); {
		if !startsJSONFragment(value, i) {
			i++
			continue
		}
		if attempts >= maxJSONFragmentAttempts {
			// A JSON candidate remains but the parse budget is exhausted.
			// Falling back to the text rules would leave arrays and objects
			// uninspected, so fail closed for the whole value.
			return redactedValue, appendFinding(findings, RuleSensitiveField, path, 1), true
		}
		attempts++
		window := value[i:]
		truncated := false
		if len(window) > maxStructuredBytes {
			window = window[:maxStructuredBytes]
			truncated = true
		}
		cleaned, consumed, fragmentFindings, err := s.sanitizeJSONFragment(window, path, inspectEncoded)
		if err != nil {
			if truncated && (errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, errJSONDepth)) {
				// The candidate may be a valid JSON document that is larger
				// than the structural bound (or nested beyond it), so no
				// complete value can be inspected. Fail closed instead of
				// leaving arrays or objects to the text rules.
				return redactedValue, appendFinding(findings, RuleSensitiveField, path, 1), true
			}
			i++
			continue
		}
		builder.WriteString(value[last:i])
		builder.WriteString(cleaned)
		findings = mergeFindings(findings, fragmentFindings)
		last = i + consumed
		i = last
		found = true
	}
	if !found {
		return "", nil, false
	}
	builder.WriteString(value[last:])
	return builder.String(), findings, true
}

func startsJSONFragment(value string, i int) bool {
	switch value[i] {
	case '{', '[':
		return true
	case '"':
		// A JSON document encoded as a JSON/shell string: "{\"...\"}".
		return i+1 < len(value) && (value[i+1] == '{' || value[i+1] == '[')
	default:
		return false
	}
}

// sanitizeJSONFragment parses one JSON value at the start of value and returns
// the cleaned fragment plus the number of bytes it consumed.
func (s *Sanitizer) sanitizeJSONFragment(value, path string, inspectEncoded bool) (string, int, []Finding, error) {
	if len(value) > maxStructuredBytes {
		return "", 0, nil, fmt.Errorf("JSON fragment exceeds %d bytes", maxStructuredBytes)
	}
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber()
	node, err := parseJSONNode(decoder)
	if err != nil {
		return "", 0, nil, err
	}
	consumed := int(decoder.InputOffset())
	if consumed <= 0 || consumed > len(value) {
		return "", 0, nil, fmt.Errorf("invalid JSON fragment length %d", consumed)
	}
	clean, findings := s.sanitizeJSONNode(node, path, 0, inspectEncoded)
	if reflect.DeepEqual(node, clean) {
		return value[:consumed], consumed, findings, nil
	}
	var builder strings.Builder
	builder.Grow(consumed)
	clean.encode(&builder)
	return builder.String(), consumed, findings, nil
}

func (s *Sanitizer) sanitizeJSONNode(node *jsonNode, path string, depth int, inspectEncoded bool) (*jsonNode, []Finding) {
	if depth >= maxStructuredDepth {
		return redactedJSONNode(), []Finding{{RuleID: RuleSensitiveField, Path: path, Count: 1}}
	}
	switch node.kind {
	case jsonObjectKind:
		clean := &jsonNode{kind: jsonObjectKind, obj: make([]jsonMember, 0, len(node.obj))}
		var findings []Finding
		for _, member := range node.obj {
			childPath := path + ".field"
			if isSensitiveKeyDecoded(member.key) {
				if member.value.kind == jsonStringKind && isRedactedCredentialValue(member.value.str) {
					// Already redacted; keep the marker without a new finding.
					clean.obj = append(clean.obj, member)
					continue
				}
				if member.value.kind != jsonStringKind && isWeakPassKey(member.key) {
					// Test-result style fields such as {"pass": true} are not
					// credentials. Still recurse in case the value hides one.
					child, childFindings := s.sanitizeJSONNode(member.value, childPath, depth+1, inspectEncoded)
					clean.obj = append(clean.obj, jsonMember{key: member.key, value: child})
					findings = mergeFindings(findings, childFindings)
					continue
				}
				clean.obj = append(clean.obj, jsonMember{key: member.key, value: redactedJSONNode()})
				findings = appendFinding(findings, RuleSensitiveField, childPath, 1)
				continue
			}
			cleanKey, keyFindings := s.sanitizeTextRules(member.key, childPath, true)
			if len(keyFindings) > 0 {
				// The key itself carries credential material; redact the pair.
				clean.obj = append(clean.obj, jsonMember{key: redactedValue, value: redactedJSONNode()})
				findings = mergeFindings(findings, keyFindings)
				findings = appendFinding(findings, RuleSensitiveField, childPath, 1)
				continue
			}
			child, childFindings := s.sanitizeJSONNode(member.value, childPath, depth+1, inspectEncoded)
			clean.obj = append(clean.obj, jsonMember{key: cleanKey, value: child})
			findings = mergeFindings(findings, childFindings)
		}
		return clean, findings
	case jsonArrayKind:
		clean := &jsonNode{kind: jsonArrayKind, arr: make([]*jsonNode, len(node.arr))}
		var findings []Finding
		for i, child := range node.arr {
			cleanChild, childFindings := s.sanitizeJSONNode(child, fmt.Sprintf("%s[%d]", path, i), depth+1, inspectEncoded)
			clean.arr[i] = cleanChild
			findings = mergeFindings(findings, childFindings)
		}
		return clean, findings
	case jsonStringKind:
		// This covers values such as an escaped MCP result payload:
		// "{\"password\":\"...\"}" is decoded here and inspected structurally.
		clean, findings := s.sanitizeStringValue(node.str, path, depth, inspectEncoded)
		return &jsonNode{kind: jsonStringKind, str: clean}, findings
	default:
		return node, nil
	}
}

// JSON node representation that preserves member order, which encoding/json
// maps do not.

type jsonKind uint8

const (
	jsonObjectKind jsonKind = iota
	jsonArrayKind
	jsonStringKind
	jsonScalarKind
)

type jsonNode struct {
	kind jsonKind
	str  string
	raw  string
	obj  []jsonMember
	arr  []*jsonNode
}

type jsonMember struct {
	key   string
	value *jsonNode
}

func redactedJSONNode() *jsonNode {
	return &jsonNode{kind: jsonStringKind, str: redactedValue}
}

func parseJSONNode(decoder *json.Decoder) (*jsonNode, error) {
	return parseJSONNodeDepth(decoder, 0)
}

func parseJSONNodeDepth(decoder *json.Decoder, depth int) (*jsonNode, error) {
	if depth > maxJSONParseDepth {
		return nil, fmt.Errorf("%w: exceeds %d levels", errJSONDepth, maxJSONParseDepth)
	}
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch typed := token.(type) {
	case json.Delim:
		switch typed {
		case '{':
			node := &jsonNode{kind: jsonObjectKind}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, fmt.Errorf("JSON object key is not a string")
				}
				child, err := parseJSONNodeDepth(decoder, depth+1)
				if err != nil {
					return nil, err
				}
				node.obj = append(node.obj, jsonMember{key: key, value: child})
			}
			if _, err := decoder.Token(); err != nil {
				return nil, err
			}
			return node, nil
		case '[':
			node := &jsonNode{kind: jsonArrayKind}
			for decoder.More() {
				child, err := parseJSONNodeDepth(decoder, depth+1)
				if err != nil {
					return nil, err
				}
				node.arr = append(node.arr, child)
			}
			if _, err := decoder.Token(); err != nil {
				return nil, err
			}
			return node, nil
		default:
			return nil, fmt.Errorf("unexpected JSON delimiter %q", typed)
		}
	case string:
		return &jsonNode{kind: jsonStringKind, str: typed}, nil
	case json.Number:
		return &jsonNode{kind: jsonScalarKind, raw: typed.String()}, nil
	case bool:
		if typed {
			return &jsonNode{kind: jsonScalarKind, raw: "true"}, nil
		}
		return &jsonNode{kind: jsonScalarKind, raw: "false"}, nil
	case nil:
		return &jsonNode{kind: jsonScalarKind, raw: "null"}, nil
	default:
		return nil, fmt.Errorf("unsupported JSON token %T", token)
	}
}

func (n *jsonNode) encode(builder *strings.Builder) {
	switch n.kind {
	case jsonObjectKind:
		builder.WriteByte('{')
		for i, member := range n.obj {
			if i > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(encodeJSONString(member.key))
			builder.WriteByte(':')
			member.value.encode(builder)
		}
		builder.WriteByte('}')
	case jsonArrayKind:
		builder.WriteByte('[')
		for i, child := range n.arr {
			if i > 0 {
				builder.WriteByte(',')
			}
			child.encode(builder)
		}
		builder.WriteByte(']')
	case jsonStringKind:
		builder.WriteString(encodeJSONString(n.str))
	default:
		builder.WriteString(n.raw)
	}
}

func encodeJSONString(value string) string {
	var builder strings.Builder
	encoder := json.NewEncoder(&builder)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
	return strings.TrimSuffix(builder.String(), "\n")
}

func (s *Sanitizer) redactEncodedCandidates(value, path string, findings []Finding) (string, []Finding) {
	const (
		minCandidate           = 20
		maxCandidate           = 1000
		maxOversizedInputBytes = 4 << 20
	)
	var builder strings.Builder
	builder.Grow(len(value))
	last := 0
	changed := false
	for i := 0; i < len(value); {
		if !isEncodedCandidateByte(value[i]) {
			i++
			continue
		}
		start := i
		var seen [256]bool
		distinct := 0
		alnum := false
		for i < len(value) && isEncodedCandidateByte(value[i]) {
			if !seen[value[i]] {
				seen[value[i]] = true
				distinct++
			}
			if isASCIIAlnum(value[i]) {
				alnum = true
			}
			i++
		}
		runEnd := i
		padding := 0
		for i < len(value) && value[i] == '=' && padding < 2 {
			i++
			padding++
		}
		runLength := runEnd - start
		candidate := value[start:i]
		redactFrom := start
		redactTo := i
		redact := false
		if runLength > maxCandidate {
			// Runs without any alphanumeric character are separators or art
			// (for example a long "+---+---" table rule), not encoded secrets.
			if distinct > 1 && alnum {
				if runLength <= maxOversizedInputBytes {
					redact = oversizedCandidateContainsSecret(candidate)
				} else {
					redact = true
				}
			}
		} else if runLength >= minCandidate {
			if secretStart, secretEnd, ok := s.encodedSecretRange(candidate, path); ok {
				redactFrom = start + secretStart
				redactTo = start + secretEnd
				redact = true
			}
		}
		if redact {
			builder.WriteString(value[last:redactFrom])
			builder.WriteString(redactedValue)
			findings = appendFinding(findings, RuleEncodedSecret, path, 1)
			last = redactTo
			changed = true
		}
	}
	if !changed {
		return value, findings
	}
	builder.WriteString(value[last:])
	return builder.String(), findings
}

func isEncodedCandidateByte(b byte) bool {
	return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '+' || b == '/' || b == '_' || b == '-'
}

func isASCIIAlnum(b byte) bool {
	return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9'
}

// encodedSecretRange returns the slice of an encoded candidate that contains a
// secret. Slash boundaries are tried from right to left so a URL or file path
// prefix is preserved. At each boundary the suffix after the '/' is tried
// before the suffix that includes it, because the slash may itself be part of
// the base64 alphabet (for example an encoded payload whose first byte is
// 0xff). The whole candidate is the final fallback for an encoded secret that
// contains '/' internally.
func (s *Sanitizer) encodedSecretRange(candidate, path string) (int, int, bool) {
	starts := make([]int, 0, 8)
	for i := len(candidate) - 1; i >= 0; i-- {
		if candidate[i] != '/' {
			continue
		}
		if i+1 < len(candidate) {
			starts = append(starts, i+1)
		}
		starts = append(starts, i)
	}
	starts = append(starts, 0)

	seen := make(map[int]struct{}, len(starts))
	for _, start := range starts {
		if _, ok := seen[start]; ok {
			continue
		}
		seen[start] = struct{}{}
		segment := strings.TrimRight(candidate[start:], "/")
		if segment == "" {
			continue
		}
		if s.encodedSegmentHasSecret(segment, path) {
			return start, start + len(segment), true
		}
	}
	return 0, 0, false
}

func (s *Sanitizer) encodedSegmentHasSecret(segment, path string) bool {
	decoded, ok := decodeCandidate(segment)
	if !ok {
		return false
	}
	// Decoded JSON keys can be escaped ("password"), so inspect the
	// decoded fragment both structurally and as text.
	_, findings := s.sanitizeText(decoded, path, false, true)
	return len(findings) > 0
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
	trimmed := strings.TrimSpace(decoded)
	if len(trimmed) > maxStructuredBytes && (trimmed[0] == '{' || trimmed[0] == '[') {
		// A JSON document too large to inspect structurally must not reach
		// the text-rule fallback, which cannot traverse arrays or objects.
		return true
	}
	_, findings := defaultSanitizer.sanitizeText(decoded, "$", false, true)
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

// decodePercentSequences decodes every valid %XX escape and leaves stray '%'
// bytes untouched, so a single malformed escape (for example "80%") does not
// disable inspection of the rest of the text.
func decodePercentSequences(value string) string {
	if !strings.Contains(value, "%") {
		return value
	}
	var builder strings.Builder
	builder.Grow(len(value))
	for i := 0; i < len(value); i++ {
		if value[i] == '%' && i+2 < len(value) && isHexDigit(value[i+1]) && isHexDigit(value[i+2]) {
			builder.WriteByte(hexValue(value[i+1])<<4 | hexValue(value[i+2]))
			i += 2
			continue
		}
		builder.WriteByte(value[i])
	}
	return builder.String()
}

func isHexDigit(b byte) bool {
	return b >= '0' && b <= '9' || b >= 'a' && b <= 'f' || b >= 'A' && b <= 'F'
}

func hexValue(b byte) byte {
	switch {
	case b >= '0' && b <= '9':
		return b - '0'
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10
	default:
		return b - 'A' + 10
	}
}

func redactCredentialValue(pattern *regexp.Regexp) func(string) string {
	return func(match string) string {
		parts := pattern.FindStringSubmatch(match)
		if len(parts) < 3 {
			return redactedValue
		}
		value := match[len(parts[1])+len(parts[2]):]
		if isRedactedCredentialValue(value) {
			return match
		}
		return parts[1] + parts[2] + quotedRedactedValue(value)
	}
}

// isRedactedCredentialValue accepts only a complete existing marker. The bare
// prefix is included because some value branches stop before a trailing ']'
// (for example ["password=[REDACTED]"]). A prefix followed by other content is
// not a marker, so it is redacted again instead of leaking the remainder.
// isRedactedCredentialValue accepts only a complete marker. The unquoted value
// branch can stop before the closing ']' (for example inside a JSON array);
// redactAssignments handles that truncated form locally so the shared helper
// used by JSON, headers, URLs, and structured values stays strict.
func isRedactedCredentialValue(value string) bool {
	switch strings.TrimSpace(value) {
	case redactedValue, doubleQuotedValue(redactedValue), singleQuotedValue(redactedValue):
		return true
	default:
		return false
	}
}

func doubleQuotedValue(value string) string {
	return `"` + value + `"`
}

func singleQuotedValue(value string) string {
	return `'` + value + `'`
}

// allRedactedValues reports whether every value is already the redaction
// marker, so re-sanitizing a redacted document does not record a finding.
func allRedactedValues(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if !isRedactedCredentialValue(value) {
			return false
		}
	}
	return true
}

// redactAssignments redacts key/value assignments while preserving text that
// follows the credential. For '=' style logfmt records it stops the value at
// the next "key=" token, so "password=secret host=10.0.0.1" keeps the host
// field. Later assignments in the same match are scanned again instead of
// being swallowed by the first redaction.
func (s *Sanitizer) redactAssignments(value, path string, pattern *regexp.Regexp, findings []Finding) (string, []Finding) {
	var builder strings.Builder
	builder.Grow(len(value))
	pos := 0
	for pos < len(value) {
		locations := pattern.FindStringSubmatchIndex(value[pos:])
		if locations == nil {
			builder.WriteString(value[pos:])
			break
		}
		start := pos + locations[0]
		builder.WriteString(value[pos:start])
		key := value[pos+locations[2] : pos+locations[3]]
		separator := value[pos+locations[4] : pos+locations[5]]
		valueStart := pos + locations[5]
		valueEnd := pos + locations[1]
		rawValue := value[valueStart:valueEnd]
		if isRedactedCredentialValue(rawValue) {
			// Already redacted. Keep scanning after the value so a later
			// assignment on the same line is not skipped.
			builder.WriteString(value[start:valueEnd])
			pos = valueEnd
			continue
		}
		consumed := len(rawValue)
		// Inside a quoted value, whitespace and embedded "key=" pairs belong
		// to the credential (for example a signed token or a DSN). Only an
		// unquoted value uses the logfmt boundary.
		if isEqualsSeparator(separator) && !isQuotedValue(rawValue) {
			consumed = assignmentValueBoundary(rawValue)
		}
		if consumed <= 0 {
			builder.WriteString(value[start:valueEnd])
			pos = valueEnd
			continue
		}
		builder.WriteString(key)
		builder.WriteString(separator)
		builder.WriteString(quotedRedactedValue(rawValue[:consumed]))
		findings = appendFinding(findings, RuleCredentialAssignment, path, 1)
		pos = valueStart + consumed
	}
	return builder.String(), findings
}

func isEqualsSeparator(separator string) bool {
	return strings.Contains(separator, "=") || strings.Contains(strings.ToLower(separator), "%3d")
}

// isQuotedValue reports whether the value branch captured a quoted scalar. The
// regex only produces this form for a terminated pair or an unterminated quote
// that runs to the end of the line, so both are replaced as a whole.
func isQuotedValue(value string) bool {
	return strings.HasPrefix(value, `"`) || strings.HasPrefix(value, `'`)
}

// assignmentValueBoundary returns how much of an unquoted '=' value belongs to
// the credential. A whitespace-delimited token that starts a new assignment
// ("host=10.0.0.1") ends the value; ordinary words with spaces do not, so a
// secret such as "password=my secret value" stays fully redacted.
func assignmentValueBoundary(value string) int {
	end := 0
	for end < len(value) {
		if value[end] != ' ' && value[end] != '\t' {
			end++
			continue
		}
		next := end
		for next < len(value) && (value[next] == ' ' || value[next] == '\t') {
			next++
		}
		if next >= len(value) {
			return next
		}
		if looksLikeAssignmentStart(value[next:]) {
			return end
		}
		end = next
	}
	return end
}

func looksLikeAssignmentStart(value string) bool {
	i := 0
	for i < len(value) && (isASCIIAlnum(value[i]) || value[i] == '_' || value[i] == '-' || value[i] == '.') {
		i++
	}
	if i == 0 {
		return false
	}
	for i < len(value) && (value[i] == ' ' || value[i] == '\t') {
		i++
	}
	if i >= len(value) {
		return false
	}
	if value[i] == '=' {
		return true
	}
	return value[i] == '%' && i+2 < len(value) && value[i+1] == '3' && (value[i+2] == 'd' || value[i+2] == 'D')
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
	changed := 0
	cleaned := pattern.ReplaceAllStringFunc(value, func(match string) string {
		replacement := replace(match)
		if replacement != match {
			changed++
		}
		return replacement
	})
	if changed == 0 {
		return value, findings
	}
	return cleaned, appendFinding(findings, ruleID, path, changed)
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

func normalizeSensitiveKey(key string) string {
	return strings.NewReplacer("-", "", "_", "", ".", "", " ", "").Replace(strings.ToLower(strings.TrimSpace(key)))
}

func isSensitiveKey(key string) bool {
	normalized := normalizeSensitiveKey(key)
	switch normalized {
	case "authorization", "proxyauthorization", "cookie", "setcookie", "token", "accesstoken", "refreshtoken", "apikey", "password", "passwd", "pass", "pwd", "secret", "clientsecret", "clientkeydata", "clientcertificatedata", "privatekey", "kubeconfig":
		return true
	default:
		for _, suffix := range []string{"authorization", "cookie", "token", "apikey", "password", "passwd", "secret", "privatekey", "keydata", "certificatedata", "accesskey", "kubeconfig", "pwd"} {
			if strings.HasSuffix(normalized, suffix) {
				return true
			}
		}
		return false
	}
}

// isSensitiveKeyDecoded also accepts percent-encoded key names such as
// "%70assword". JSON object keys reach the structured path before the text
// rules can decode them, so the key must be checked directly.
func isSensitiveKeyDecoded(key string) bool {
	candidate := key
	for i := 0; i < 3; i++ {
		if isSensitiveKey(candidate) {
			return true
		}
		decoded := decodePercentSequences(candidate)
		if decoded == candidate {
			return false
		}
		candidate = decoded
	}
	return isSensitiveKey(candidate)
}

// isWeakPassKey reports whether a short key such as "pass" or "pwd" is only
// credential-like because of its name. Structured values of these keys are
// often test results ({"pass": true}), so non-string values are kept.
func isWeakPassKey(key string) bool {
	normalized := normalizeSensitiveKey(key)
	if normalized == "pass" || normalized == "pwd" {
		return true
	}
	return strings.HasSuffix(normalized, "pass") || strings.HasSuffix(normalized, "pwd")
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
