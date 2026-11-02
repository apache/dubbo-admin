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

package hooks

import (
	"encoding/json"
	"fmt"
	"strings"
)

// reasoningPartType is the part discriminator the generative AI semantic
// conventions define for reasoning/thinking content received from a model.
const reasoningPartType = "reasoning"

// reasoningPartsKey is the message field that carries the parts of a chat
// message in the generative AI semantic conventions.
const reasoningPartsKey = "parts"

// truncatedReasoningLimit is the number of characters kept from a single
// reasoning string when the truncated policy applies. It matches the limit
// used for the other captured JSON strings.
const truncatedReasoningLimit = 512

// ApplyReasoningPolicy filters or truncates the reasoning parts of a
// generative AI messages document according to the configured policy.
//
// The document is the serialized JSON of the gen_ai.input.messages or
// gen_ai.output.messages attribute. A reasoning part is an element of a
// "parts" array whose "type" is "reasoning"; every other part is preserved.
//
//   - CaptureNone removes the reasoning parts from the document.
//   - CaptureTruncated keeps them and shortens their content.
//   - CaptureFull returns the document unchanged.
//
// An unset or unknown policy is treated like CaptureNone so reasoning content
// never reaches a trace by accident. A document that is not valid JSON is
// returned unchanged: a parsing failure must never drop captured content.
func ApplyReasoningPolicy(document string, policy string) string {
	switch policy {
	case CaptureFull:
		return document
	case CaptureTruncated:
		return rewriteReasoningParts(document, truncateReasoningElement)
	default:
		return rewriteReasoningParts(document, nil)
	}
}

// normalizeCaptureReasoning resolves the reasoning capture policy of a hook
// registration. An unset policy defaults to CaptureNone so reasoning content
// is never captured unless a hook asks for it.
func normalizeCaptureReasoning(policy string) (string, error) {
	switch policy {
	case "":
		return CaptureNone, nil
	case CaptureNone, CaptureTruncated, CaptureFull:
		return policy, nil
	default:
		return "", fmt.Errorf("capture_reasoning must be one of %q, %q, or %q", CaptureNone, CaptureTruncated, CaptureFull)
	}
}

// rewriteReasoningParts rewrites the reasoning parts of a messages document
// and returns the document unchanged when it holds no reasoning part. The
// traversal only relies on the "parts" array and the "type" discriminator so
// it does not depend on the remaining message fields.
func rewriteReasoningParts(document string, rewrite func(element map[string]any) bool) string {
	// The document is decoded before the parts are inspected. A plain string
	// search for the type discriminator would be bypassed by JSON escapes: the
	// raw bytes of "type":"reasoning" do not contain the word "reasoning", so
	// the policy would not be applied and the reasoning content would reach the
	// trace unchanged.
	decoder := json.NewDecoder(strings.NewReader(document))
	// Numbers keep their original notation so a rewritten document stays
	// faithful to the captured payload.
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return document
	}
	if decoder.More() {
		// The document holds trailing data and is not a single JSON value.
		return document
	}
	rewritten, changed := filterReasoningParts(value, rewrite)
	if !changed {
		return document
	}
	encoded, err := json.Marshal(rewritten)
	if err != nil {
		return document
	}
	return string(encoded)
}

// filterReasoningParts applies rewrite to every reasoning part of the
// document and reports whether the document was modified.
func filterReasoningParts(value any, rewrite func(element map[string]any) bool) (any, bool) {
	switch typed := value.(type) {
	case []any:
		changed := false
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			child, childChanged := filterReasoningParts(item, rewrite)
			changed = changed || childChanged
			items = append(items, child)
		}
		if !changed {
			return typed, false
		}
		return items, true
	case map[string]any:
		changed := false
		for key, item := range typed {
			if key == reasoningPartsKey {
				parts, ok := item.([]any)
				if !ok {
					continue
				}
				filtered, partsChanged := filterReasoningElements(parts, rewrite)
				if partsChanged {
					typed[key] = filtered
					changed = true
				}
				continue
			}
			child, childChanged := filterReasoningParts(item, rewrite)
			if childChanged {
				typed[key] = child
				changed = true
			}
		}
		return typed, changed
	}
	return value, false
}

// filterReasoningElements rewrites the reasoning elements of a parts array
// and preserves every other element. A rewrite that reports false drops the
// element from the array.
func filterReasoningElements(parts []any, rewrite func(element map[string]any) bool) ([]any, bool) {
	filtered := make([]any, 0, len(parts))
	changed := false
	for _, part := range parts {
		element, ok := part.(map[string]any)
		if !ok || element["type"] != reasoningPartType {
			filtered = append(filtered, part)
			continue
		}
		changed = true
		if rewrite != nil && rewrite(element) {
			filtered = append(filtered, element)
		}
	}
	return filtered, changed
}

// truncateReasoningElement shortens the content of a reasoning part and keeps
// the part in the document.
func truncateReasoningElement(element map[string]any) bool {
	content, ok := element["content"].(string)
	if !ok {
		return true
	}
	truncated, changed := truncateReasoningString(content)
	if changed {
		element["content"] = truncated
	}
	return true
}

// truncateReasoningString shortens a reasoning string to the per-string limit
// and reports whether the string was shortened. The limit counts characters
// so a multi-byte reasoning text keeps the documented number of them.
func truncateReasoningString(content string) (string, bool) {
	characters := []rune(content)
	if len(characters) <= truncatedReasoningLimit {
		return content, false
	}
	return string(characters[:truncatedReasoningLimit-1]) + "…", true
}
