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
	"strings"
	"testing"

	"github.com/firebase/genkit/go/ai"
)

func TestModelInputViewSanitizesCopyWithoutChangingCanonicalMessage(t *testing.T) {
	const secret = "model-view-secret"
	canonical := ai.NewMessage(ai.RoleUser, map[string]any{"password": secret},
		ai.NewTextPart("Authorization: Bearer "+secret),
		ai.NewToolRequestPart(&ai.ToolRequest{Name: "lookup", Input: map[string]any{"api_key": secret}}),
	)
	view, err := modelInputView([]*ai.Message{canonical})
	if err != nil {
		t.Fatal(err)
	}
	if len(view) != 1 || view[0] == canonical {
		t.Fatalf("modelInputView() did not create a copy: %#v", view)
	}
	if strings.Contains(view[0].Content[0].Text, secret) {
		t.Fatalf("model view leaked secret: %#v", view[0])
	}
	if canonical.Content[0].Text != "Authorization: Bearer "+secret || canonical.Metadata["password"] != secret {
		t.Fatalf("canonical message was changed: %#v", canonical)
	}
}

func TestModelInputViewSanitizesJSONText(t *testing.T) {
	const secret = "model-json-secret"
	canonical := ai.NewMessage(ai.RoleUser, nil,
		ai.NewTextPart(`{"password":"`+secret+`","enabled":true}`),
	)
	view, err := modelInputView([]*ai.Message{canonical})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(view[0].Content[0].Text, secret) {
		t.Fatalf("model view leaked JSON secret: %q", view[0].Content[0].Text)
	}
	if canonical.Content[0].Text != `{"password":"`+secret+`","enabled":true}` {
		t.Fatalf("canonical JSON message was changed: %q", canonical.Content[0].Text)
	}
}

func TestModelInputViewSanitizesJSONToolOutput(t *testing.T) {
	const secret = "model-json-part-secret"
	for _, input := range []string{
		`{"privateKey":"` + secret + `"}`,
		`{"kubeconfig":"` + secret + `"}`,
		`{"DB_PASSWORD":"` + secret + `"}`,
	} {
		canonical := ai.NewMessage(ai.RoleUser, nil, ai.NewJSONPart(input))
		view, err := modelInputView([]*ai.Message{canonical})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(view[0].Content[0].Text, secret) {
			t.Fatalf("model view leaked JSON part secret: %q", view[0].Content[0].Text)
		}
		if canonical.Content[0].Text != input {
			t.Fatalf("canonical JSON part was changed: %q", canonical.Content[0].Text)
		}
	}
}

func TestModelInputViewRedactsEmbeddedAndQuotedJSON(t *testing.T) {
	const secret = "model-embedded-secret"
	inputs := []string{
		`Error: {"password":["` + secret + `","second-secret"]}`,
		`"{\"password\":\"` + secret + `\"}"`,
		`curl -d "{\"password\":\"` + secret + `\"}"`,
		`TOKEN="v1 sig=` + secret + `"`,
		`CONNECTION_SECRET='host=10.0.0.1 pass_phrase=` + secret + `'`,
	}
	for _, input := range inputs {
		canonical := ai.NewMessage(ai.RoleUser, nil, ai.NewTextPart(input))
		view, err := modelInputView([]*ai.Message{canonical})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(view[0].Content[0].Text, secret) {
			t.Fatalf("model view leaked credential %q: %q", input, view[0].Content[0].Text)
		}
		if canonical.Content[0].Text != input {
			t.Fatalf("canonical message was changed: %q", canonical.Content[0].Text)
		}
	}
}

func TestModelInputViewDoesNotTreatProseAsToolOutput(t *testing.T) {
	const input = `{notes: mentions "tool_name" and "result"}`
	canonical := ai.NewMessage(ai.RoleUser, nil, ai.NewTextPart(input))

	view, err := modelInputView([]*ai.Message{canonical})
	if err != nil {
		t.Fatal(err)
	}
	if view[0].Content[0].Text != input {
		t.Fatalf("ordinary prose was misclassified as a tool output: %q", view[0].Content[0].Text)
	}
	if canonical.Content[0].Text != input {
		t.Fatalf("canonical message was changed")
	}
}

func TestModelInputViewSanitizesDataPart(t *testing.T) {
	const secret = "data-part-secret"
	input := `{"password":"` + secret + `"}`
	canonical := ai.NewMessage(ai.RoleUser, nil, ai.NewDataPart(input))

	view, err := modelInputView([]*ai.Message{canonical})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(view[0].Content[0].Text, secret) {
		t.Fatalf("data part leaked a secret: %q", view[0].Content[0].Text)
	}
	if canonical.Content[0].Text != input {
		t.Fatalf("canonical data part was changed: %q", canonical.Content[0].Text)
	}
}

func TestModelInputViewFailsClosedForOversizedJSONPart(t *testing.T) {
	const secret = "model-oversized-secret"
	input := `{"password":"` + secret + `","pad":"` + strings.Repeat("x", 1<<20) + `"}`
	canonical := ai.NewMessage(ai.RoleModel, nil, ai.NewJSONPart(input))

	view, err := modelInputView([]*ai.Message{canonical})
	if err != nil {
		t.Fatal(err)
	}
	if view[0].Content[0].Text != "[REDACTED]" {
		t.Fatalf("oversized JSON part was not redacted whole: %q", view[0].Content[0].Text)
	}
	if canonical.Content[0].Text != input {
		t.Fatalf("canonical JSON part was changed")
	}
}

// persistedMessage simulates the conversation store, which clones messages
// through encoding/json. Genkit's text-part encoding omits ContentType, so a
// NewJSONPart reloaded from the store looks like a plain text part.
func persistedMessage(t *testing.T, message *ai.Message) *ai.Message {
	t.Helper()
	encoded, err := json.Marshal(message)
	if err != nil {
		t.Fatal(err)
	}
	var reloaded ai.Message
	if err := json.Unmarshal(encoded, &reloaded); err != nil {
		t.Fatal(err)
	}
	return &reloaded
}

func TestModelInputViewRedactsPersistedToolOutputEnvelope(t *testing.T) {
	const secret = "persisted-array-secret"
	input := `{"tool_name":"mcp_get_config","result":[{"password":["` + secret + `","second"]}],"summary":"ok"}`
	canonical := ai.NewMessage(ai.RoleModel, nil, ai.NewJSONPart(input))
	reloaded := persistedMessage(t, canonical)
	if reloaded.Content[0].ContentType != "" {
		t.Fatalf("expected the store round-trip to drop ContentType, got %q", reloaded.Content[0].ContentType)
	}

	view, err := modelInputView([]*ai.Message{reloaded})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(view[0].Content[0].Text, secret) {
		t.Fatalf("persisted tool output leaked an array secret: %q", view[0].Content[0].Text)
	}
	if canonical.Content[0].Text != input {
		t.Fatalf("canonical tool output was changed")
	}
}

func TestModelInputViewFailsClosedForPersistedOversizedToolOutput(t *testing.T) {
	const secret = "persisted-oversized-secret"
	input := `{"tool_name":"mcp_get_config","result":[{"password":["` + secret + `","second"]}],"summary":"ok","pad":"` + strings.Repeat("x", 1<<20) + `"}`
	canonical := ai.NewMessage(ai.RoleModel, nil, ai.NewJSONPart(input))
	reloaded := persistedMessage(t, canonical)

	view, err := modelInputView([]*ai.Message{reloaded})
	if err != nil {
		t.Fatal(err)
	}
	if view[0].Content[0].Text != "[REDACTED]" {
		t.Fatalf("persisted oversized tool output was not redacted whole: %q", view[0].Content[0].Text)
	}
	if canonical.Content[0].Text != input {
		t.Fatalf("canonical tool output was changed")
	}
}

func TestModelInputViewPreservesMediaPayload(t *testing.T) {
	payload := ai.NewMediaPart("image/png", strings.Repeat("iVBORw0KGgo", 200))
	canonical := ai.NewMessage(ai.RoleUser, nil, payload)
	view, err := modelInputView([]*ai.Message{canonical})
	if err != nil {
		t.Fatal(err)
	}
	if view[0].Content[0].Text != payload.Text {
		t.Fatalf("media payload was modified")
	}
}

func TestModelInputViewRedactsCookieListsAndSingleQuotedYAML(t *testing.T) {
	const secret = "review-boundary-secret"
	inputs := []string{
		"Cookie: theme=dark; JSESSIONID=" + secret,
		`'password': '` + secret + `'`,
	}
	for _, input := range inputs {
		canonical := ai.NewMessage(ai.RoleUser, nil, ai.NewTextPart(input))

		view, err := modelInputView([]*ai.Message{canonical})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(view[0].Content[0].Text, secret) {
			t.Fatalf("model view leaked credential: %q", view[0].Content[0].Text)
		}
		if canonical.Content[0].Text != input {
			t.Fatalf("canonical message was changed: %q", canonical.Content[0].Text)
		}
	}
}

func TestModelInputViewRedactsEscapedCredentialValues(t *testing.T) {
	const suffix = "review-suffix-secret"
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "escaped quote",
			input: `{"password":"prefix\"` + suffix + `","enabled":true}`,
		},
		{
			name:  "escaped backslash",
			input: `{"password":"prefix\\` + suffix + `","enabled":true}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonical := ai.NewMessage(ai.RoleUser, nil, ai.NewTextPart(tt.input))

			view, err := modelInputView([]*ai.Message{canonical})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(view[0].Content[0].Text, suffix) {
				t.Fatalf("model view leaked escaped credential suffix: %q", view[0].Content[0].Text)
			}
			if canonical.Content[0].Text != tt.input {
				t.Fatalf("canonical message was changed: %q", canonical.Content[0].Text)
			}
		})
	}
}

func TestModelInputViewRedactsNestedEscapedJSONToolOutput(t *testing.T) {
	const secret = "mcp-secret-123"
	input := `{"tool_name":"mcp_get_config","result":[{"type":"text","text":"{\"password\":\"` + secret + `\",\"enabled\":true}"}]}`
	canonical := ai.NewMessage(ai.RoleModel, nil, ai.NewJSONPart(input))

	view, err := modelInputView([]*ai.Message{canonical})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(view[0].Content[0].Text, secret) {
		t.Fatalf("model view leaked nested escaped JSON secret: %q", view[0].Content[0].Text)
	}
	if canonical.Content[0].Text != input {
		t.Fatalf("canonical JSON part was changed: %q", canonical.Content[0].Text)
	}
}

func TestModelInputViewRedactsCredentialVariants(t *testing.T) {
	const secret = "review-variant-secret"
	inputs := []string{
		`Cookie: "theme=dark"; JSESSIONID=` + secret,
		`Authorization: "x"; JSESSIONID=` + secret,
		`'password': 'ab''` + secret + `'`,
		`{"%70assword":"` + secret + `"}`,
	}
	for _, input := range inputs {
		canonical := ai.NewMessage(ai.RoleUser, nil, ai.NewTextPart(input))
		view, err := modelInputView([]*ai.Message{canonical})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(view[0].Content[0].Text, secret) {
			t.Fatalf("model view leaked credential variant %q: %q", input, view[0].Content[0].Text)
		}
		if canonical.Content[0].Text != input {
			t.Fatalf("canonical message was changed: %q", canonical.Content[0].Text)
		}
	}
}
