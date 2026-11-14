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
			// vLLM keys the flag per model family, so the inner key has to be
			// selectable: the Qwen templates want enable_thinking, the others
			// want thinking.
			name:      "chat-template uses the configured inner key",
			compat:    Compat{ThinkingFormat: ThinkingFormatChatTemplate, ChatTemplateKey: "enable_thinking"},
			effort:    "high",
			wantKey:   "chat_template_kwargs",
			wantValue: map[string]any{"enable_thinking": true},
		},
		{
			name:      "chat-template can use the neutral inner key",
			compat:    Compat{ThinkingFormat: ThinkingFormatChatTemplate, ChatTemplateKey: "thinking"},
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

func TestApplyRequest_ThinkingBudgetCarriesTheConfiguredCap(t *testing.T) {
	compat := Compat{
		ThinkingFormat:      ThinkingFormatQwen,
		ThinkingBudgetField: "thinking_budget",
		ThinkingBudget:      1024,
	}
	resolved := ResolveCompat(&compat, nil)

	t.Run("sends the configured cap while reasoning is on", func(t *testing.T) {
		cfg := newParams()
		resolved.ApplyRequest(cfg, 0, "medium")
		body := requestBody(t, cfg)
		if got := body["thinking_budget"]; got != float64(1024) {
			t.Errorf("thinking_budget = %#v, want 1024", got)
		}
	})

	// qwen3.5-plus reasons by default, so a request that never mentions
	// thinking still pays for it — measured 2026-10-04, an unprompted
	// troubleshooting question burned 1648 reasoning tokens out of the same
	// max_tokens pool as the answer. The cap has to travel in this case too,
	// otherwise the only requests that stay bounded are the ones that already
	// asked for thinking.
	t.Run("sends the cap when reasoning was not requested", func(t *testing.T) {
		cfg := newParams()
		resolved.ApplyRequest(cfg, 0, "")
		body := requestBody(t, cfg)
		if got := body["thinking_budget"]; got != float64(1024) {
			t.Errorf("thinking_budget = %#v, want 1024 — the cap has to cover the thinking "+
				"the endpoint enables on its own", got)
		}
		if got := body["enable_thinking"]; got != false {
			t.Errorf("enable_thinking = %#v, want false (the off switch is still sent)", got)
		}
	})

	// A cap that is not a cap is not worth sending: this endpoint reads -1 as
	// its own default, which bounds nothing, and the other two spellings have
	// no meaning for -1 and may reject it.
	t.Run("sends nothing when no cap is configured", func(t *testing.T) {
		cfg := newParams()
		ResolveCompat(&Compat{
			ThinkingFormat:      ThinkingFormatQwen,
			ThinkingBudgetField: "thinking_budget",
		}, nil).ApplyRequest(cfg, 0, "medium")
		body := requestBody(t, cfg)
		if _, present := body["thinking_budget"]; present {
			t.Errorf("thinking_budget should be absent without a configured cap, got %#v",
				body["thinking_budget"])
		}
	})

	t.Run("each spelling carries its own value", func(t *testing.T) {
		for _, field := range ValidThinkingBudgetFields {
			cfg := newParams()
			ResolveCompat(&Compat{
				ThinkingBudgetField: field,
				ThinkingBudget:      512,
			}, nil).ApplyRequest(cfg, 0, "high")
			body := requestBody(t, cfg)
			if got := body[field]; got != float64(512) {
				t.Errorf("%s = %#v, want 512", field, got)
			}
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
//
// The compat block below mirrors the shipped one: only the keys that reach
// the request. The measurements about the capabilities that are not wired up
// yet are recorded in docs/ai-provider-compat.md.
func TestDashScopeMeasuredBehaviour(t *testing.T) {
	dashscope := ResolveCompat(&Compat{
		ThinkingFormat: ThinkingFormatQwen,
		MaxTokensField: "max_tokens",
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

	t.Run("no thinking budget is sent because the shipped config sets no cap", func(t *testing.T) {
		// Measured: this model reasons by default — a request with no
		// enable_thinking still returned reasoning_content, and burned 1648
		// reasoning tokens. Only enable_thinking=false suppresses it.
		//
		// The budget is not sent because the shipped config sets no cap, and
		// this endpoint reads -1 as its own default, which bounds nothing.
		// Sending -1 would look like a guard while guarding nothing.
		cfg := newParams()
		dashscope.ApplyRequest(cfg, 0, "")
		body := requestBody(t, cfg)
		if body["enable_thinking"] != false {
			t.Errorf("enable_thinking = %#v, want false to suppress default thinking", body["enable_thinking"])
		}
		if _, present := body["thinking_budget"]; present {
			t.Errorf("thinking_budget should be absent without a configured cap, got %#v",
				body["thinking_budget"])
		}
	})

	t.Run("forced tool choice is declared unsupported", func(t *testing.T) {
		// Measured: tool_choice="required" with a question that needs no tool
		// produced a prose answer and no tool call — identical to "auto". The
		// endpoint ignores it silently rather than rejecting it, so an agent
		// that relies on forcing gets no error and no tool call.
		//
		// The field is reserved — the agent never forces a tool choice, so
		// there is nothing to degrade — but the semantics are pinned here for
		// the day the capability is wired up.
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
			name:   "valid default",
			compat: Compat{},
		},
		{
			name: "valid applied configuration",
			compat: Compat{
				ThinkingFormat:      ThinkingFormatDeepSeek,
				ThinkingBudgetField: "thinking_budget",
				ThinkingBudget:      1024,
				MaxTokensField:      "max_tokens",
			},
		},
		{
			name:   "valid chat template with an inner key",
			compat: Compat{ThinkingFormat: ThinkingFormatChatTemplate, ChatTemplateKey: "enable_thinking"},
		},
		{
			// The Qwen templates spell the flag enable_thinking, so the key
			// has a default there.
			name:   "valid qwen chat template without an inner key",
			compat: Compat{ThinkingFormat: ThinkingFormatQwenChatTemplate},
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
			name:    "negative budget",
			compat:  Compat{ThinkingBudgetField: "thinking_budget", ThinkingBudget: -1},
			wantErr: "thinking_budget must not be negative",
		},
		{
			// The cap has to say which key carries it, or there is nothing to
			// send it in.
			name:    "budget without a field name",
			compat:  Compat{ThinkingBudget: 1024},
			wantErr: "needs thinking_budget_field",
		},
		{
			name:    "chat template without an inner key",
			compat:  Compat{ThinkingFormat: ThinkingFormatChatTemplate},
			wantErr: "needs chat_template_key",
		},
		{
			name:    "chat template key without a chat template format",
			compat:  Compat{ThinkingFormat: ThinkingFormatQwen, ChatTemplateKey: "enable_thinking"},
			wantErr: "chat_template_key only applies to",
		},
		{
			name:    "qwen with strict mode",
			compat:  Compat{ThinkingFormat: ThinkingFormatQwen, SupportsStrictMode: bp(true)},
			wantErr: "cannot be combined with supports_strict_mode",
		},
		{
			// A reserved field must be refused rather than accepted and
			// ignored: a configuration that promises a behaviour the request
			// does not have is worse than one that fails to start.
			name:    "a reserved field is refused",
			compat:  Compat{SendBackThinking: SendBackField},
			wantErr: "not applied to the request",
		},
		{
			name:    "a refused field is named",
			compat:  Compat{SupportsStore: bp(true)},
			wantErr: "supports_store",
		},
		{
			// The error must name the provider so an operator can find the
			// offending block in a multi-provider file.
			name:    "error mentions the provider",
			compat:  Compat{ThinkingFormat: "qwen3"},
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

// reservedFieldSamples builds one configuration per reserved field, so the
// table in compat.go and this test stay in step: a field added to the table
// without a sample fails here, and a sample without a table entry is caught by
// the completeness check below.
var reservedFieldSamples = map[string]Compat{
	"thinking_field":                            {ThinkingField: "reasoning_content"},
	"send_back_thinking":                        {SendBackThinking: SendBackField},
	"thinking_tags":                             {ThinkingTags: DefaultThinkingTags},
	"ignore_leading_whitespace":                 {IgnoreLeadingWhitespace: bp(true)},
	"supports_developer_role":                   {SupportsDeveloperRole: bp(true)},
	"supports_strict_mode":                      {SupportsStrictMode: bp(true)},
	"supports_store":                            {SupportsStore: bp(true)},
	"supports_forced_tool_choice":               {SupportsForcedToolChoice: bp(false)},
	"supports_forced_tool_choice_with_thinking": {SupportsForcedToolChoiceWithThinking: bp(false)},
	"requires_tool_result_name":                 {RequiresToolResultName: bp(true)},
	"requires_assistant_after_tool_result":      {RequiresAssistantAfterToolResult: bp(true)},
	"backfill_thinking_field":                   {BackfillThinkingField: bp(true)},
}

// Every reserved field must be refused when it is configured, and must not
// stand in the way when it is not.
func TestValidate_ReservedFields(t *testing.T) {
	for _, field := range reservedCompatFields {
		t.Run(field.name, func(t *testing.T) {
			sample, ok := reservedFieldSamples[field.name]
			if !ok {
				t.Fatalf("no sample for the reserved field %q", field.name)
			}
			if !field.configured(&sample) {
				t.Fatalf("the reserved field %q is not detected as configured", field.name)
			}

			err := sample.Validate("dashscope")
			if err == nil {
				t.Fatalf("the reserved field %q must be refused when configured", field.name)
			}
			if !contains(err.Error(), field.name) {
				t.Errorf("error = %q, want it to name the field %q", err.Error(), field.name)
			}
			if !contains(err.Error(), "docs/ai-provider-compat.md") {
				t.Errorf("error = %q, want it to point at the design document", err.Error())
			}
		})
	}

	// A sample without a table entry would mean a field that nothing refuses.
	for name := range reservedFieldSamples {
		found := false
		for _, field := range reservedCompatFields {
			if field.name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("the sample %q has no entry in the reserved field table", name)
		}
	}

	// An empty configuration must not be refused.
	if err := (&Compat{}).Validate("dashscope"); err != nil {
		t.Errorf("an empty compat block must validate, got %v", err)
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
