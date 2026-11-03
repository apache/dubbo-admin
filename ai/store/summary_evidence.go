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

package store

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"dubbo-admin-ai/security"
)

var summarySensitiveKey = regexp.MustCompile(`(?i)\b(?:authorization|proxy[-_]?authorization|cookie|set[-_]?cookie|token|access[-_]?token|refresh[-_]?token|api[-_]?key|apikey|password|passwd|secret|client[-_]?secret|client[-_]?key[-_]?data|client[-_]?certificate[-_]?data)[\\"]*\s*:`)

// Bump when the detector, sanitizer or tool evidence transformation changes.
const SummaryInspectionRule = "evidence-v1:" + security.SanitizerRuleVersion
const SummaryInspectionBytes int64 = 64

// SummaryInspection is fixed-size, server-owned metadata attached to a
// canonical record. Messages use version 1; tool records use their ledger version.
// Only completed scans produce a result. Never accept these values from callers.
type SummaryInspection struct {
	Rule      string
	Version   uint64
	Sensitive bool
}

func (s SummaryInspection) Valid(version uint64) bool {
	return s.Rule == SummaryInspectionRule && s.Version == version
}

// InspectSummaryRecord scans before any evidence is released. Overlap preserves
// detection across storage-read and later summary-fragment boundaries.
func InspectSummaryRecord(version uint64, read func(int, int) (string, int, error)) (SummaryInspection, error) {
	result := SummaryInspection{Rule: SummaryInspectionRule, Version: version}
	_, total, err := read(0, 0)
	if err != nil {
		return SummaryInspection{}, err
	}
	tail := ""
	for pos := 0; pos < total; {
		chunk, _, err := read(pos, min(8192, total-pos))
		if err != nil {
			return SummaryInspection{}, err
		}
		if len(chunk) == 0 {
			return SummaryInspection{}, ErrContextConflict
		}
		view := tail + chunk
		_, found := security.DefaultSanitizer().ContainsSecret(view)
		upper := strings.ToUpper(view)
		if found || summarySensitiveKey.MatchString(view) || strings.Contains(upper, "PRIVATE KEY-----") || strings.Contains(upper, "-----BEGIN CERTIFICATE") {
			result.Sensitive = true
			return result, nil
		}
		pos += len(chunk)
		if len(view) > 2048 {
			tail = view[len(view)-2048:]
		} else {
			tail = view
		}
	}
	return result, nil
}

// ReadInspectedSummaryFragment uses a verified inspection for the same immutable
// source version. Commit still rereads the fragment and validates its digest.
func ReadInspectedSummaryFragment(check SummaryInspection, version uint64, offset, limit int, read func(int, int) (string, int, error)) (string, int, error) {
	if !check.Valid(version) || offset < 0 || limit < 0 {
		return "", 0, ErrContextConflict
	}
	if check.Sensitive {
		const omitted = "[Sensitive canonical record excluded from summary evidence; its contents and outcome must not be inferred.]"
		if offset >= len(omitted) {
			return "", 0, ErrContextConflict
		}
		return omitted[offset:min(offset+limit, len(omitted))], len(omitted), nil
	}
	return read(offset, limit)
}

// SummaryToolEvidence keeps canonical call identities and all uncertainty states,
// while limiting large structured result data before fragmenting the summary view.
func SummaryToolEvidence(batch ToolBatch) ([]byte, error) {
	batch.Calls = append([]ToolCall(nil), batch.Calls...)
	for i, call := range batch.Calls {
		if len(call.Output) == 0 {
			continue
		}
		var output any
		if err := json.Unmarshal(call.Output, &output); err != nil {
			return nil, err
		}
		clean := security.DefaultSanitizer().SanitizeStructured(output)
		limited := limitSummaryValue(clean.Value, 0)
		batch.Calls[i].Output = summaryJSON(map[string]any{"status": call.Status, "result": limited, "source_call_id": call.ID, "summary_view_limited": true})
	}
	return json.Marshal(batch)
}

func limitSummaryValue(v any, depth int) any {
	if depth > 6 {
		return "[depth limit]"
	}
	switch x := v.(type) {
	case string:
		r := []rune(x)
		if len(r) > 2048 {
			return string(r[:2048]) + "[truncated]"
		}
	case []any:
		out := make([]any, min(len(x), 32))
		for i := range out {
			out[i] = limitSummaryValue(x[i], depth+1)
		}
		return out
	case map[string]any: // JSON serialization sorts keys; retain a stable subset.
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := map[string]any{}
		for _, k := range keys[:min(len(keys), 32)] {
			out[k] = limitSummaryValue(x[k], depth+1)
		}
		return out
	}
	return v
}

// SummaryRecordOrder mirrors the canonical history view: user input, complete
// tool batches, then assistant answers. Legacy Turns without batches keep order.
func SummaryRecordOrder(roles []string, batches int) []int {
	var order []int
	if batches == 0 {
		for i := range roles {
			order = append(order, i)
		}
		return order
	}
	for i, role := range roles {
		if role != "model" {
			order = append(order, i)
		}
	}
	for i := 0; i < batches; i++ {
		order = append(order, len(roles)+i)
	}
	for i, role := range roles {
		if role == "model" {
			order = append(order, i)
		}
	}
	return order
}
