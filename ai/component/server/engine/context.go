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

package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
	"unicode"

	"dubbo-admin-ai/schema"
)

const (
	contextMaxStringLength = 1000
	contextMaxArrayItems   = 10
	contextMaxDepth        = 12
	redactedContextValue   = "[REDACTED]"
	maxDepthContextValue   = "[MAX_DEPTH]"
)

var sensitiveContextKeys = []string{
	"password",
	"passwd",
	"token",
	"secret",
	"cookie",
	"authorization",
	"apikey",
	"privatekey",
	"kubeconfig",
}

var sensitiveDescriptorKeys = map[string]struct{}{
	"key":  {},
	"name": {},
}

var semanticValueKeys = map[string]struct{}{
	"value":        {},
	"values":       {},
	"currentvalue": {},
	"defaultvalue": {},
}

func (r *ChatRequest) ParseContext() (*schema.AIContextSnapshot, error) {
	if len(r.Context) == 0 || bytes.Equal(bytes.TrimSpace(r.Context), []byte("null")) {
		return nil, nil
	}
	if len(r.Context) > schema.AIContextMaxBytes {
		return nil, fmt.Errorf("context exceeds %d bytes", schema.AIContextMaxBytes)
	}

	decoder := json.NewDecoder(bytes.NewReader(r.Context))
	decoder.DisallowUnknownFields()
	var snapshot schema.AIContextSnapshot
	if err := decoder.Decode(&snapshot); err != nil {
		return nil, fmt.Errorf("invalid context: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, err
	}
	if err := validateAIContext(&snapshot); err != nil {
		return nil, err
	}

	// Re-sanitize at the trust boundary so non-browser clients cannot bypass frontend filtering.
	sanitizeAIContext(&snapshot)
	return &snapshot, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("invalid context: multiple JSON values")
		}
		return fmt.Errorf("invalid context: %w", err)
	}
	return nil
}

func validateAIContext(snapshot *schema.AIContextSnapshot) error {
	if snapshot.Version != schema.AIContextVersion {
		return fmt.Errorf("unsupported context version: %d", snapshot.Version)
	}
	if _, err := time.Parse(time.RFC3339Nano, snapshot.CapturedAt); err != nil {
		return fmt.Errorf("invalid context capturedAt: %w", err)
	}
	if strings.TrimSpace(snapshot.Page.Path) == "" {
		return fmt.Errorf("context page.path is required")
	}
	if strings.TrimSpace(snapshot.Scope.Mesh) == "" {
		return fmt.Errorf("context scope.mesh is required")
	}
	if len(snapshot.Evidence) > contextMaxArrayItems {
		return fmt.Errorf("context evidence exceeds %d sections", contextMaxArrayItems)
	}
	for index := range snapshot.Evidence {
		section := &snapshot.Evidence[index]
		if strings.TrimSpace(section.ID) == "" || strings.TrimSpace(section.Source) == "" {
			return fmt.Errorf("context evidence[%d] requires id and source", index)
		}
		if section.Data == nil {
			return fmt.Errorf("context evidence[%d].data is required", index)
		}
		if section.CapturedAt != "" {
			if _, err := time.Parse(time.RFC3339Nano, section.CapturedAt); err != nil {
				return fmt.Errorf("invalid context evidence[%d].capturedAt: %w", index, err)
			}
		}
	}
	return nil
}

func sanitizeAIContext(snapshot *schema.AIContextSnapshot) {
	snapshot.CapturedAt = sanitizeContextString(snapshot.CapturedAt)
	snapshot.Global.Locale = sanitizeContextString(snapshot.Global.Locale)
	snapshot.Page.RouteName = sanitizeContextString(snapshot.Page.RouteName)
	snapshot.Page.Path = sanitizeContextString(snapshot.Page.Path)
	snapshot.Page.FullPath = sanitizeContextString(snapshot.Page.FullPath)
	snapshot.Page.ActiveTab = sanitizeContextString(snapshot.Page.ActiveTab)
	snapshot.Page.Params = sanitizeContextMap(snapshot.Page.Params, 0)
	snapshot.Page.Query = sanitizeContextMap(snapshot.Page.Query, 0)
	snapshot.Scope.Mesh = sanitizeContextString(snapshot.Scope.Mesh)
	snapshot.Scope.Application = sanitizeContextString(snapshot.Scope.Application)
	snapshot.Scope.Service = sanitizeContextString(snapshot.Scope.Service)
	snapshot.Scope.Instance = sanitizeContextString(snapshot.Scope.Instance)
	snapshot.Scope.Rule = sanitizeContextString(snapshot.Scope.Rule)

	if snapshot.State != nil {
		snapshot.State.Filters = sanitizeContextMap(snapshot.State.Filters, 0)
		snapshot.State.Selection = sanitizeContextMap(snapshot.State.Selection, 0)
		snapshot.State.UnsavedChanges = sanitizeContextMap(snapshot.State.UnsavedChanges, 0)
	}
	for index := range snapshot.Evidence {
		section := &snapshot.Evidence[index]
		section.ID = sanitizeContextString(section.ID)
		section.Source = sanitizeContextString(section.Source)
		section.CapturedAt = sanitizeContextString(section.CapturedAt)
		section.Data = sanitizeContextMap(section.Data, 0)
	}
	if snapshot.Truncation != nil {
		limit := min(len(snapshot.Truncation.OmittedSections), contextMaxArrayItems)
		omittedSections := make([]string, 0, limit)
		for _, section := range snapshot.Truncation.OmittedSections[:limit] {
			omittedSections = append(omittedSections, sanitizeContextString(section))
		}
		snapshot.Truncation.OmittedSections = omittedSections
	}
}

func sanitizeContextMap(value map[string]any, depth int) map[string]any {
	if value == nil {
		return nil
	}
	// Handle pair-shaped settings such as {"key":"token","value":"..."}.
	sensitiveDescriptor := hasSensitiveContextDescriptor(value)
	result := make(map[string]any, len(value))
	for key, item := range value {
		if isSensitiveContextKey(key) || (sensitiveDescriptor && isSemanticContextValueKey(key)) {
			result[key] = redactedContextValue
			continue
		}
		result[key] = sanitizeContextValue(item, depth+1)
	}
	return result
}

func hasSensitiveContextDescriptor(value map[string]any) bool {
	for key, item := range value {
		if _, ok := sensitiveDescriptorKeys[normalizeContextKey(key)]; !ok {
			continue
		}
		text, ok := item.(string)
		if ok && isSensitiveContextKey(text) {
			return true
		}
	}
	return false
}

func isSemanticContextValueKey(key string) bool {
	_, ok := semanticValueKeys[normalizeContextKey(key)]
	return ok
}

func sanitizeContextValue(value any, depth int) any {
	switch typed := value.(type) {
	case nil, bool, float64:
		return typed
	case string:
		return sanitizeContextString(typed)
	case []any:
		if depth >= contextMaxDepth {
			return maxDepthContextValue
		}
		limit := min(len(typed), contextMaxArrayItems)
		result := make([]any, 0, limit)
		for _, item := range typed[:limit] {
			result = append(result, sanitizeContextValue(item, depth+1))
		}
		return result
	case map[string]any:
		if depth >= contextMaxDepth {
			return maxDepthContextValue
		}
		return sanitizeContextMap(typed, depth)
	default:
		return nil
	}
}

func sanitizeContextString(value string) string {
	value = sanitizeContextURL(value)
	runes := []rune(value)
	if len(runes) <= contextMaxStringLength {
		return value
	}
	return string(runes[:contextMaxStringLength]) + "...[TRUNCATED]"
}

func sanitizeContextURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return value
	}
	parsed.User = nil
	query := parsed.Query()
	for key := range query {
		if isSensitiveContextKey(key) || strings.EqualFold(key, "username") {
			query.Set(key, redactedContextValue)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func isSensitiveContextKey(key string) bool {
	normalized := normalizeContextKey(key)
	for _, sensitiveKey := range sensitiveContextKeys {
		if strings.Contains(normalized, sensitiveKey) {
			return true
		}
	}
	return false
}

func normalizeContextKey(key string) string {
	return strings.Map(func(char rune) rune {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			return unicode.ToLower(char)
		}
		return -1
	}, key)
}
