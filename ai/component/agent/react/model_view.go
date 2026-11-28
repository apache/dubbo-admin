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

package react

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"dubbo-admin-ai/security"

	"github.com/firebase/genkit/go/ai"
)

// modelInputView creates an ephemeral filtered copy. The canonical messages
// loaded from the conversation Store are never changed or written back.
func modelInputView(messages []*ai.Message) ([]*ai.Message, error) {
	encoded, err := json.Marshal(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to copy model input: %w", err)
	}
	var view []*ai.Message
	if err := json.Unmarshal(encoded, &view); err != nil {
		return nil, fmt.Errorf("failed to decode model input copy: %w", err)
	}
	sanitizer := security.DefaultSanitizer()
	for messageIndex, message := range view {
		if message == nil {
			continue
		}
		var source *ai.Message
		if messageIndex < len(messages) {
			source = messages[messageIndex]
		}
		message.Metadata = sanitizeMap(sanitizer, message.Metadata)
		for partIndex, part := range message.Content {
			if part == nil {
				continue
			}
			// The JSON round-trip drops Part.ContentType, so restore it from
			// the canonical part. Messages reloaded from the conversation store
			// have already lost the type, so also recognize the tool-output
			// envelope written by toolEngine.ToolOutput.
			if source != nil && partIndex < len(source.Content) && source.Content[partIndex] != nil {
				part.ContentType = source.Content[partIndex].ContentType
			}
			switch {
			case isJSONTextPart(part), isToolOutputJSON(part):
				// Declared or envelope-identified JSON fails closed when it is
				// too large, unparseable, or nested beyond the parser bound.
				part.Text = sanitizer.SanitizeJSONPart(part.Text).Value
			case part.Kind != ai.PartMedia:
				// Media parts hold opaque binary content (for example a base64
				// payload); text redaction would corrupt them.
				part.Text = sanitizer.SanitizeText(part.Text).Value
			}
			part.Metadata = sanitizeMap(sanitizer, part.Metadata)
			part.Custom = sanitizeMap(sanitizer, part.Custom)
			if part.ToolRequest != nil {
				part.ToolRequest.Input = sanitizer.SanitizeStructured(part.ToolRequest.Input).Value
			}
			if part.ToolResponse != nil {
				part.ToolResponse.Output = sanitizer.SanitizeStructured(part.ToolResponse.Output).Value
			}
			if part.Resource != nil {
				if parsed, parseErr := url.Parse(part.Resource.Uri); parseErr == nil {
					filtered := sanitizer.SanitizeURL(parsed).Value.String()
					part.Resource.Uri = sanitizer.SanitizeText(filtered).Value
				} else {
					part.Resource.Uri = sanitizer.SanitizeText(part.Resource.Uri).Value
				}
			}
		}
	}
	return view, nil
}

func isJSONTextPart(part *ai.Part) bool {
	return part != nil && part.Kind == ai.PartText && strings.HasPrefix(part.ContentType, "application/json")
}

// toolOutputEnvelopeProbeBytes bounds how much of a tool output is inspected
// when deciding whether it is the toolEngine.ToolOutput envelope. Only the
// first two member names and the tool name value are read.
const toolOutputEnvelopeProbeBytes = 64 << 10

// isToolOutputJSON recognizes the toolEngine.ToolOutput envelope
// ({"tool_name":...,"result":...|"summary":...}). Messages loaded from the
// conversation store have lost Part.ContentType because genkit's text-part
// encoding omits it, so the JSON shape is the remaining signal. The check
// parses only the first members instead of searching for substrings, so
// ordinary prose such as `{notes: "tool_name" and "result"}` is not
// misclassified and redacted whole.
func isToolOutputJSON(part *ai.Part) bool {
	if part == nil || part.Kind != ai.PartText {
		return false
	}
	text := strings.TrimSpace(part.Text)
	if len(text) == 0 || text[0] != '{' {
		return false
	}
	if len(text) > toolOutputEnvelopeProbeBytes {
		text = text[:toolOutputEnvelopeProbeBytes]
	}
	decoder := json.NewDecoder(strings.NewReader(text))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return false
	}
	first, err := decoder.Token()
	if err != nil {
		return false
	}
	if name, ok := first.(string); !ok || name != "tool_name" {
		return false
	}
	toolName, err := decoder.Token()
	if err != nil {
		return false
	}
	if _, ok := toolName.(string); !ok {
		return false
	}
	second, err := decoder.Token()
	if err != nil {
		return false
	}
	name, ok := second.(string)
	return ok && (name == "result" || name == "summary")
}

func sanitizeMap(sanitizer *security.Sanitizer, value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	result := sanitizer.SanitizeStructured(value)
	clean, ok := result.Value.(map[string]any)
	if !ok {
		return nil
	}
	return clean
}
