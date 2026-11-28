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
	"errors"
	"fmt"
)

var ErrSensitiveDerivedData = errors.New("sensitive content is not allowed in derived data")

var knowledgeIndexFields = []string{
	"source", "title", "updated_at", "file_size", "file_type", "page",
	"header_path", "header_level", "chunk_index", "chunk_start", "chunk_size",
}

// DerivedDataPolicy validates content before it enters an index or future
// derived-memory store and retains only explicitly allowed metadata fields.
type DerivedDataPolicy struct {
	allowed   map[string]struct{}
	sanitizer *Sanitizer
}

func NewDerivedDataPolicy(allowedFields ...string) *DerivedDataPolicy {
	allowed := make(map[string]struct{}, len(allowedFields))
	for _, field := range allowedFields {
		allowed[field] = struct{}{}
	}
	return &DerivedDataPolicy{allowed: allowed, sanitizer: DefaultSanitizer()}
}

func NewKnowledgeIndexPolicy() *DerivedDataPolicy {
	return NewDerivedDataPolicy(knowledgeIndexFields...)
}

func (p *DerivedDataPolicy) Prepare(content string, metadata map[string]any) (map[string]any, []Finding, error) {
	if p == nil {
		return nil, nil, ErrSensitiveDerivedData
	}
	findings, found := p.sanitizer.ContainsSecret(content)
	if found {
		return nil, findings, ErrSensitiveDerivedData
	}
	clean := make(map[string]any)
	for key, value := range metadata {
		if _, ok := p.allowed[key]; !ok {
			continue
		}
		// Only scalar values and scalar lists (for example header_path) may be
		// persisted. Maps and nested containers can hide sensitive structure
		// from the field-level inspection below.
		if !allowedMetadataValue(value) {
			return nil, nil, fmt.Errorf("%w: metadata field %q must be a scalar or scalar list", ErrSensitiveDerivedData, key)
		}
		result := p.sanitizer.SanitizeStructured(value)
		if len(result.Findings) > 0 {
			return nil, result.Findings, ErrSensitiveDerivedData
		}
		// Preserve scalar types expected by index backends. The structured pass
		// above is inspection only; the allowlist limits what can be persisted.
		// Allowed slices are cloned so later mutation of the caller's metadata
		// cannot change already-validated prepared documents.
		clean[key] = cloneMetadataValue(value)
	}
	return clean, nil, nil
}

// allowedMetadataValue limits persisted metadata to scalars and lists of
// scalars. Index backends use header_path as []string, so flat lists remain
// valid; maps and nested lists do not.
func allowedMetadataValue(value any) bool {
	switch typed := value.(type) {
	case nil, string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	case []string:
		return true
	case []any:
		for _, item := range typed {
			if !isScalarMetadataValue(item) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// isScalarMetadataValue reports whether value is a scalar permitted in a flat
// metadata list. It intentionally does not recurse: lists of lists are outside
// the backend contract and could hide structure from the field inspection.
func isScalarMetadataValue(value any) bool {
	switch value.(type) {
	case nil, string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	default:
		return false
	}
}

// cloneMetadataValue copies the slice forms of a validated metadata value so
// the returned map does not share backing storage with the caller.
func cloneMetadataValue(value any) any {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		// allowedMetadataValue guarantees scalar elements, which are immutable.
		return append([]any(nil), typed...)
	default:
		return value
	}
}

// ValidateIdentifier applies the same secret boundary to external document
// identifiers before they enter a derived index.
func (p *DerivedDataPolicy) ValidateIdentifier(identifier string) ([]Finding, error) {
	if p == nil {
		return nil, ErrSensitiveDerivedData
	}
	findings, found := p.sanitizer.ContainsSecret(identifier)
	if found {
		return findings, ErrSensitiveDerivedData
	}
	return nil, nil
}
