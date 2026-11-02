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
	for _, message := range view {
		if message == nil {
			continue
		}
		message.Metadata = sanitizeMap(sanitizer, message.Metadata)
		for _, part := range message.Content {
			if part == nil {
				continue
			}
			part.Text = sanitizer.SanitizeText(part.Text).Value
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
