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

package models

import (
	"encoding/json"
	"testing"

	"github.com/openai/openai-go"
)

// requestBody renders the params the way the HTTP layer would, so the tests
// assert the bytes on the wire rather than the struct we happened to build.
func requestBody(t *testing.T, cfg *openai.ChatCompletionNewParams) map[string]any {
	t.Helper()
	raw, err := cfg.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal request params: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	return body
}

func newParams() *openai.ChatCompletionNewParams {
	return &openai.ChatCompletionNewParams{Model: "test-model"}
}

func TestApplyRequest_ThinkingFormatSpelling(t *testing.T) {
	tests := []struct {
		name       string
		compat     Compat
		effort     string
		wantKey    string
		wantValue  any
		absentKeys []string
	}{
		{
			name:      "qwen sends a top-level enable_thinking",
			compat:    Compat{ThinkingFormat: ThinkingFormatQwen},
			effort:    "high",
			wantKey:   "enable_thinking",
			wantValue: true,
			// The Qwen family must not also receive the OpenAI spelling.
			absentKeys: []string{"reasoning_effort", "thinking", "chat_template_kwargs"},
		},
		{
			name:      "qwen off is an explicit false, not an omitted key",
			compat:    Compat{ThinkingFormat: ThinkingFormatQwen},
			effort:    "",
			wantKey:   "enable_thinking",
			wantValue: false,
		},
		{
			name:      "deepseek nests the switch under thinking.type",
			compat:    Compat{ThinkingFormat: ThinkingFormatDeepSeek},
			effort:    "medium",
			wantKey:   "thinking",
			wantValue: map[string]any{"type": "enabled"},
		},
		{
			name:      "chat-template forwards the neutral flag name",
			compat:    Compat{ThinkingFormat: ThinkingFormatChatTemplate},
			effort:    "high",
			wantKey:   "chat_template_kwargs",
			wantValue: map[string]any{"thinking": true},
		},
		{
			name:      "qwen-chat-template sets both flags",
			compat:    Compat{ThinkingFormat: ThinkingFormatQwenChatTemplate},
			effort:    "low",
			wantKey:   "chat_template_kwargs",
			wantValue: map[string]any{"enable_thinking": true, "preserve_thinking": true},
		},
		{
			name:      "openai uses reasoning_effort",
			compat:    Compat{ThinkingFormat: ThinkingFormatOpenAI},
			effort:    "high",
			wantKey:   "reasoning_effort",
			wantValue: "high",
		},
		{
			name:      "openrouter nests effort under reasoning",
			compat:    Compat{ThinkingFormat: ThinkingFormatOpenRouter},
			effort:    "low",
			wantKey:   "reasoning",
			wantValue: map[string]any{"effort": "low"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolved := ResolveCompat(&tc.compat, nil)
			cfg := newParams()
			resolved.ApplyRequest(cfg, 0, tc.effort)

			body := requestBody(t, cfg)
			got, ok := body[tc.wantKey]
			if !ok {
				t.Fatalf("expected key %q in %v", tc.wantKey, body)
			}
			if !jsonEqual(got, tc.wantValue) {
				t.Errorf("key %q = %#v, want %#v", tc.wantKey, got, tc.wantValue)
			}
			for _, absent := range tc.absentKeys {
				if _, present := body[absent]; present {
					t.Errorf("key %q should be absent, got %#v", absent, body[absent])
				}
			}
		})
	}
}

func TestApplyRequest_MaxTokensField(t *testing.T) {
	t.Run("defaults to max_tokens", func(t *testing.T) {
		cfg := newParams()
		ResolveCompat(nil, nil).ApplyRequest(cfg, 1500, "")
		body := requestBody(t, cfg)
		if got := body["max_tokens"]; got != float64(1500) {
			t.Errorf("max_tokens = %#v, want 1500", got)
		}
		if _, present := body["max_completion_tokens"]; present {
			t.Error("max_completion_tokens should be absent when max_tokens_field is unset")
		}
	})

	t.Run("honours max_completion_tokens", func(t *testing.T) {
		cfg := newParams()
		resolved := ResolveCompat(&Compat{MaxTokensField: "max_completion_tokens"}, nil)
		resolved.ApplyRequest(cfg, 1500, "")
		body := requestBody(t, cfg)
		if got := body["max_completion_tokens"]; got != float64(1500) {
			t.Errorf("max_completion_tokens = %#v, want 1500", got)
		}
		if _, present := body["max_tokens"]; present {
			t.Error("max_tokens should be absent when max_completion_tokens is configured")
		}
	})

	t.Run("zero max_tokens sends neither key", func(t *testing.T) {
		cfg := newParams()
		ResolveCompat(nil, nil).ApplyRequest(cfg, 0, "")
		body := requestBody(t, cfg)
		if _, present := body["max_tokens"]; present {
			t.Error("max_tokens should be absent when unset")
		}
	})

	t.Run("max_completion_tokens does not leak when caller set max_tokens", func(t *testing.T) {
		// Guards the regression where a caller pre-populates the other
		// spelling: the field that is not configured must be cleared, not left
		// to be sent alongside the configured one.
		cfg := newParams()
		cfg.MaxTokens = openai.Int(999)
		resolved := ResolveCompat(&Compat{MaxTokensField: "max_completion_tokens"}, nil)
		resolved.ApplyRequest(cfg, 100, "")
		body := requestBody(t, cfg)
		if _, present := body["max_tokens"]; present {
			t.Errorf("max_tokens should be cleared, got %#v", body["max_tokens"])
		}
	})
}

func TestApplyRequest_ThinkingBudgetCoversImplicitThinking(t *testing.T) {
	compat := Compat{
		ThinkingFormat:      ThinkingFormatQwen,
		ThinkingBudgetField: "thinking_budget",
	}
	resolved := ResolveCompat(&compat, nil)

	t.Run("sent while reasoning is on", func(t *testing.T) {
		cfg := newParams()
		resolved.ApplyRequest(cfg, 0, "medium")
		body := requestBody(t, cfg)
		if _, present := body["thinking_budget"]; !present {
			t.Errorf("thinking_budget should be sent when reasoning is on, got %v", body)
		}
	})

	// qwen3.5-plus reasons by default, so a request that never mentions
	// thinking still pays for it — measured 2026-10-04, an unprompted
	// troubleshooting question burned 1648 reasoning tokens out of the same
	// max_tokens pool as the answer. The budget has to travel in this case too,
	// otherwise the only requests that stay bounded are the ones that already
	// asked for thinking.
	t.Run("still sent when reasoning was not requested", func(t *testing.T) {
		cfg := newParams()
		resolved.ApplyRequest(cfg, 0, "")
		body := requestBody(t, cfg)
		if _, present := body["thinking_budget"]; !present {
			t.Errorf("thinking_budget must cap thinking that the endpoint enables on its own, got %v", body)
		}
		if got := body["enable_thinking"]; got != false {
			t.Errorf("enable_thinking = %#v, want false (the off switch is still sent)", got)
		}
	})
}

func TestApplyRequest_NoThinkingKeysByDefault(t *testing.T) {
	// An endpoint with no compat block configured must get a plain OpenAI
	// request, not a stray enable_thinking or thinking_budget.
	cfg := newParams()
	ResolveCompat(nil, nil).ApplyRequest(cfg, 0, "")
	body := requestBody(t, cfg)
	for _, key := range []string{"enable_thinking", "thinking", "thinking_budget", "chat_template_kwargs"} {
		if _, present := body[key]; present {
			t.Errorf("key %q should be absent by default, got %#v", key, body[key])
		}
	}
}

// Measured on qwen3.5-plus against DashScope compatible-mode, 2026-10-04.
// These pin the behaviours the shipped models.yaml depends on, so a change to
// the compat defaults that contradicts the endpoint shows up as a test failure
// rather than as a mystery 400 in production.
func TestDashScopeMeasuredBehaviour(t *testing.T) {
	dashscope := ResolveCompat(&Compat{
		ThinkingFormat:      ThinkingFormatQwen,
		ThinkingBudgetField: "thinking_budget",
		MaxTokensField:      "max_tokens",
		SupportsStrictMode:  bp(false),
	}, nil)

	t.Run("thinking switch is a top-level enable_thinking", func(t *testing.T) {
		// Measured: enable_thinking=true, stream=false -> HTTP 200.
		// The "only supports streaming" limit applies to open-weight Qwen3, not
		// to this hosted model, so the non-streaming agent call is fine.
		cfg := newParams()
		dashscope.ApplyRequest(cfg, 0, "high")
		body := requestBody(t, cfg)
		if body["enable_thinking"] != true {
			t.Errorf("enable_thinking = %#v, want true", body["enable_thinking"])
		}
		if _, present := body["reasoning_effort"]; present {
			t.Error("reasoning_effort must not be sent to this endpoint")
		}
	})

	t.Run("thinking budget is sent even with thinking off", func(t *testing.T) {
		// Measured: this model reasons by default — a request with no
		// enable_thinking still returned reasoning_content, and burned 1648
		// reasoning tokens. Only enable_thinking=false suppresses it.
		cfg := newParams()
		dashscope.ApplyRequest(cfg, 0, "")
		body := requestBody(t, cfg)
		if body["enable_thinking"] != false {
			t.Errorf("enable_thinking = %#v, want false to suppress default thinking", body["enable_thinking"])
		}
		if _, present := body["thinking_budget"]; !present {
			t.Error("thinking_budget must still travel as a guard")
		}
	})

	t.Run("forced tool choice is declared unsupported", func(t *testing.T) {
		// Measured: tool_choice="required" with a question that needs no tool
		// produced a prose answer and no tool call — identical to "auto". The
		// endpoint ignores it silently rather than rejecting it, so an agent
		// that relies on forcing gets no error and no tool call.
		no := false
		resolved := ResolveCompat(&Compat{SupportsForcedToolChoice: &no}, nil)
		if resolved.SupportsForced(false) {
			t.Error("SupportsForced should be false for this endpoint")
		}
		if resolved.SupportsForced(true) {
			t.Error("SupportsForced should be false while thinking too")
		}
	})

	t.Run("developer role is not accepted", func(t *testing.T) {
		// Measured: role=developer -> HTTP 400 "developer is not one of
		// ['system','assistant','user','tool','function']". The default of
		// false, i.e. plain "system", is the only correct choice.
		resolved := ResolveCompat(nil, nil)
		if resolved.SupportsDeveloperRole {
			t.Error("SupportsDeveloperRole must default to false")
		}
	})
}

func TestResolveCompat_MergeSemantics(t *testing.T) {

	t.Run("model overrides only the fields it sets", func(t *testing.T) {
		provider := &Compat{
			ThinkingFormat:           ThinkingFormatQwen,
			MaxTokensField:           "max_tokens",
			SupportsForcedToolChoice: bp(false),
			IgnoreLeadingWhitespace:  bp(true),
		}
		model := &Compat{SupportsForcedToolChoice: bp(true)}

		got := ResolveCompat(provider, model)
		if got.ThinkingFormat != ThinkingFormatQwen {
			t.Errorf("ThinkingFormat = %q, want qwen (model did not set it)", got.ThinkingFormat)
		}
		if !got.IgnoreLeadingWhitespace {
			t.Error("IgnoreLeadingWhitespace should be inherited from the provider")
		}
		if !got.SupportsForcedToolChoice {
			t.Error("SupportsForcedToolChoice should be overridden to true by the model")
		}
	})

	t.Run("model can turn a provider flag off explicitly", func(t *testing.T) {
		provider := &Compat{SupportsStrictMode: bp(true)}
		got := ResolveCompat(provider, &Compat{SupportsStrictMode: bp(false)})
		if got.SupportsStrictMode {
			t.Error("SupportsStrictMode should be false after an explicit model override")
		}
	})

	t.Run("tri-state keeps the provider value when the model is unset", func(t *testing.T) {
		provider := &Compat{SupportsDeveloperRole: bp(true)}
		got := ResolveCompat(provider, nil)
		if !got.SupportsDeveloperRole {
			t.Error("an unset model override must not reset a provider value")
		}
	})

	t.Run("nil provider and model give conservative defaults", func(t *testing.T) {
		got := ResolveCompat(nil, nil)
		if got.MaxTokensField != "max_tokens" {
			t.Errorf("MaxTokensField = %q, want max_tokens", got.MaxTokensField)
		}
		if got.SupportsDeveloperRole {
			t.Error("SupportsDeveloperRole should default to false")
		}
		if got.SupportsForcedToolChoice != true {
			t.Error("SupportsForcedToolChoice should default to true")
		}
		if got.SendBackThinking != SendBackAuto {
			t.Errorf("SendBackThinking = %q, want auto", got.SendBackThinking)
		}
		if got.ThinkingTags != DefaultThinkingTags {
			t.Errorf("ThinkingTags = %v, want %v", got.ThinkingTags, DefaultThinkingTags)
		}
	})
}

func TestSupportsForced(t *testing.T) {

	tests := []struct {
		name     string
		compat   Compat
		thinking bool
		want     bool
	}{
		{name: "default allows forcing", compat: Compat{}, thinking: false, want: true},
		{name: "endpoint rejects required", compat: Compat{SupportsForcedToolChoice: bp(false)}, thinking: false, want: false},
		{name: "thinking allowed by default", compat: Compat{}, thinking: true, want: true},
		{
			name:     "thinking rejected while reasoning",
			compat:   Compat{SupportsForcedToolChoiceWithThinking: bp(false)},
			thinking: true,
			want:     false,
		},
		{
			name:     "same config allows forcing when not thinking",
			compat:   Compat{SupportsForcedToolChoiceWithThinking: bp(false)},
			thinking: false,
			want:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveCompat(&tc.compat, nil).SupportsForced(tc.thinking); got != tc.want {
				t.Errorf("SupportsForced(%v) = %v, want %v", tc.thinking, got, tc.want)
			}
		})
	}
}

func TestExtractThinking(t *testing.T) {
	tests := []struct {
		name      string
		compat    Compat
		msg       map[string]any
		wantValue string
		wantField string
	}{
		{
			name:      "reads reasoning_content",
			msg:       map[string]any{"reasoning_content": "thought"},
			wantValue: "thought",
			wantField: "reasoning_content",
		},
		{
			// vLLM renamed the response field; a client reading only the old
			// name silently gets nothing.
			name:      "reads reasoning",
			msg:       map[string]any{"reasoning": "thought"},
			wantValue: "thought",
			wantField: "reasoning",
		},
		{
			// chutes.ai returns both spellings with identical content.
			name:      "does not duplicate when both are present",
			msg:       map[string]any{"reasoning_content": "t", "reasoning": "t"},
			wantValue: "t",
			wantField: "reasoning",
		},
		{
			name:      "configured field wins",
			compat:    Compat{ThinkingField: "reasoning_content"},
			msg:       map[string]any{"reasoning": "t", "reasoning_content": "c"},
			wantValue: "c",
			wantField: "reasoning_content",
		},
		{
			name:      "empty values are skipped",
			msg:       map[string]any{"reasoning": "", "reasoning_content": "real"},
			wantValue: "real",
			wantField: "reasoning_content",
		},
		{
			name: "non-string is ignored",
			msg:  map[string]any{"reasoning": map[string]any{"text": "x"}},
		},
		{
			name: "absent",
			msg:  map[string]any{"content": "hello"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotValue, gotField := ResolveCompat(&tc.compat, nil).ExtractThinking(tc.msg)
			if gotValue != tc.wantValue {
				t.Errorf("content = %q, want %q", gotValue, tc.wantValue)
			}
			if gotField != tc.wantField {
				t.Errorf("field = %q, want %q", gotField, tc.wantField)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		compat  Compat
		wantErr string
	}{
		{
			name:   "valid deepseek pairing",
			compat: Compat{ThinkingFormat: ThinkingFormatDeepSeek, SendBackThinking: SendBackField, ThinkingField: "reasoning_content"},
		},
		{
			name:   "valid default",
			compat: Compat{},
		},
		{
			name:    "field mode without a field name",
			compat:  Compat{SendBackThinking: SendBackField},
			wantErr: "requires thinking_field",
		},
		{
			name:    "unknown thinking format",
			compat:  Compat{ThinkingFormat: "qwen3"},
			wantErr: "unknown thinking_format",
		},
		{
			name:    "invalid max tokens field",
			compat:  Compat{MaxTokensField: "maxToken"},
			wantErr: "max_tokens_field must be",
		},
		{
			name:    "invalid send back mode",
			compat:  Compat{SendBackThinking: "sometimes"},
			wantErr: "unknown send_back_thinking",
		},
		{
			name:    "invalid budget field",
			compat:  Compat{ThinkingBudgetField: "budget"},
			wantErr: "unknown thinking_budget_field",
		},
		{
			name:    "qwen with strict mode",
			compat:  Compat{ThinkingFormat: ThinkingFormatQwen, SupportsStrictMode: bp(true)},
			wantErr: "cannot be combined with supports_strict_mode",
		},
		{
			// The error must name the provider so an operator can find the
			// offending block in a multi-provider file.
			name:    "error mentions the provider",
			compat:  Compat{SendBackThinking: SendBackField},
			wantErr: `provider "dashscope"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.compat.Validate("dashscope")
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected an error containing %q, got nil", tc.wantErr)
			}
			if !contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// bp returns a pointer to v, for the tri-state compat fields.
func bp(v bool) *bool { return &v }

func jsonEqual(a, b any) bool {
	left, errA := json.Marshal(a)
	right, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return string(left) == string(right)
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
