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
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
)

// ThinkingFormat selects how thinking/reasoning parameters are spelled on the
// wire. Endpoints that advertise themselves as OpenAI-compatible still diverge
// here: the same switch is named "enable_thinking" on Qwen and "thinking" on
// DeepSeek-V3.1, and "reasoning_effort" on OpenAI proper.
type ThinkingFormat string

const (
	// ThinkingFormatOpenAI sends reasoning_effort. The fallback when unset.
	ThinkingFormatOpenAI ThinkingFormat = "openai"
	// ThinkingFormatQwen sends a top-level enable_thinking boolean.
	ThinkingFormatQwen ThinkingFormat = "qwen"
	// ThinkingFormatQwenChatTemplate sends
	// chat_template_kwargs:{enable_thinking,preserve_thinking}.
	ThinkingFormatQwenChatTemplate ThinkingFormat = "qwen-chat-template"
	// ThinkingFormatDeepSeek sends thinking:{type:"enabled"|"disabled"}.
	ThinkingFormatDeepSeek ThinkingFormat = "deepseek"
	// ThinkingFormatChatTemplate sends chat_template_kwargs. Used by vLLM,
	// which keys the flag per model family (enable_thinking vs thinking).
	ThinkingFormatChatTemplate ThinkingFormat = "chat-template"
	// ThinkingFormatOpenRouter sends reasoning:{effort}; OpenRouter
	// normalizes reasoning across the providers behind it.
	ThinkingFormatOpenRouter ThinkingFormat = "openrouter"
	// ThinkingFormatString sends a top-level thinking string.
	ThinkingFormatString ThinkingFormat = "string-thinking"
)

// SendBackThinkingMode controls how thinking content is replayed on later
// turns of a multi-turn tool-calling conversation.
type SendBackThinkingMode string

const (
	// SendBackAuto replays in whichever field the content was received in, and
	// falls back to XML tags when no field matched. The default.
	SendBackAuto SendBackThinkingMode = "auto"
	// SendBackField forces replay through Compat.ThinkingField.
	SendBackField SendBackThinkingMode = "field"
	// SendBackTags replays wrapped in Compat.ThinkingTags.
	SendBackTags SendBackThinkingMode = "tags"
	// SendBackNever drops thinking content entirely.
	SendBackNever SendBackThinkingMode = "never"
)

// ThinkingFields is the probe order for reading thinking content out of a
// response, most specific first.
//
// This array is the whole reason ADR-005 exists. vLLM renamed
// `reasoning_content` to `reasoning` and kept the old name only as an input
// alias, so a client that reads `reasoning_content` from a response silently
// gets an empty string — no error, no log line. Probing an ordered list and
// taking the first non-empty value is immune to that rename and to the
// endpoints that return both spellings with identical content.
var ThinkingFields = []string{"reasoning", "reasoning_content", "reasoning_text"}

// DefaultThinkingTags are the XML delimiters used when thinking content is
// embedded in the text stream rather than carried in a dedicated field.
var DefaultThinkingTags = [2]string{"<think>", "</think>"}

// Compat describes the request-shaping quirks of one provider or model.
//
// Field naming follows capability semantics (ADR-002): a field states what the
// endpoint supports, and its doc comment states what happens when it is
// disabled. Tri-state booleans are *bool so that "unset" stays distinguishable
// from "explicitly false" when provider-level and model-level configs merge.
//
// Defaults are conservative rather than OpenAI-like (ADR-003): an unset field
// picks the spelling the widest set of endpoints accepts. There is no
// base-URL sniffing, because a gateway in front of a provider defeats it.
type Compat struct {
	// ThinkingFormat selects the thinking parameter spelling. Unset behaves as
	// ThinkingFormatOpenAI.
	ThinkingFormat ThinkingFormat `yaml:"thinking_format,omitempty"`

	// ThinkingField is the response field carrying thinking content.
	//
	// Parsing tries this first, then falls back to ThinkingFields. Required
	// when SendBackThinking is "field" — Validate enforces that pairing, so a
	// DeepSeek-style endpoint cannot start up half-configured and then fail
	// mid-conversation with HTTP 400.
	//
	// Sources: https://api-docs.deepseek.com/guides/thinking_mode
	//          https://docs.vllm.ai/en/latest/features/tool_calling/
	ThinkingField string `yaml:"thinking_field,omitempty"`

	// SendBackThinking controls replay of thinking content on later turns.
	// Unset behaves as SendBackAuto.
	//
	// DeepSeek-compatible endpoints need "field": with tools enabled they
	// return 400 ("reasoning_content ... must be passed back") if a
	// tool-calling assistant message omits the field.
	SendBackThinking SendBackThinkingMode `yaml:"send_back_thinking,omitempty"`

	// ThinkingBudgetField caps thinking tokens via a top-level field. Unset
	// sends nothing.
	//
	// This matters more for agents than for chat: thinking and the answer
	// share max_tokens, so an uncapped reasoning-heavy turn can consume the
	// whole response and emit neither an answer nor a tool call — which just
	// spins the next iteration of the loop. Accepted values are
	// "thinking_budget" (Qwen/DashScope/SGLang), "thinking_token_budget"
	// (vLLM) and "thinking_budget_tokens" (llama.cpp).
	ThinkingBudgetField string `yaml:"thinking_budget_field,omitempty"`

	// ThinkingTags are the XML delimiters for tags-mode replay. Unset uses
	// DefaultThinkingTags.
	ThinkingTags [2]string `yaml:"thinking_tags,omitempty"`

	// IgnoreLeadingWhitespace drops leading whitespace from streamed content.
	//
	// Works around models that emit "<think>\n</think>\n\n", or an empty text
	// part, ahead of a tool call — which an agent loop can otherwise mistake
	// for the final answer and exit on. Observed with Ollama + Qwen3.
	IgnoreLeadingWhitespace *bool `yaml:"ignore_leading_whitespace,omitempty"`

	// MaxTokensField is "max_tokens" or "max_completion_tokens". Unset uses
	// "max_tokens", which the older and wider set of endpoints accepts.
	MaxTokensField string `yaml:"max_tokens_field,omitempty"`

	// SupportsDeveloperRole reports whether the endpoint accepts the
	// "developer" role. Unset is false, i.e. plain "system".
	//
	// vLLM performs no role validation and hands the string straight to the
	// Jinja chat template; templates for Qwen, DeepSeek and GLM match only
	// system/user/assistant/tool, so a "developer" message leaves the prompt
	// with no error at all.
	SupportsDeveloperRole *bool `yaml:"supports_developer_role,omitempty"`

	// SupportsStrictMode reports whether tool definitions may carry
	// "strict". Unset is false, which drops the field.
	//
	// vLLM is the exception worth knowing about: with tool_choice="auto" its
	// tool-call grammar is driven entirely by "strict", so disabling it
	// weakens structured tool calls. Enable it there deliberately.
	SupportsStrictMode *bool `yaml:"supports_strict_mode,omitempty"`

	// SupportsStore reports whether the "store" field is accepted. Unset is
	// false, which drops the field. LiteLLM-style proxies commonly reject it.
	SupportsStore *bool `yaml:"supports_store,omitempty"`

	// SupportsForcedToolChoice reports whether tool_choice="required" or a
	// named tool is accepted. Unset is true.
	//
	// When false, a forced tool choice the agent resolved for itself degrades
	// to "auto" rather than failing the request; a choice the user configured
	// explicitly is a configuration error and is rejected at startup.
	//
	// Known to reject "required": the Qwen family, Moonshot, DeepSeek-R1, and
	// DeepSeek while thinking is enabled.
	// Source: https://docs.qwencloud.com/developer-guides/text-generation/function-calling
	SupportsForcedToolChoice *bool `yaml:"supports_forced_tool_choice,omitempty"`

	// SupportsForcedToolChoiceWithThinking reports whether a forced tool
	// choice is accepted while the model is thinking. Unset is true.
	//
	// DeepSeek-V4 is false and answers "Thinking mode does not support this
	// tool_choice". Requests count as thinking when the model reasons by
	// default, not only when thinking was explicitly requested.
	SupportsForcedToolChoiceWithThinking *bool `yaml:"supports_forced_tool_choice_with_thinking,omitempty"`

	// RequiresToolResultName requires tool-result messages to carry "name".
	RequiresToolResultName *bool `yaml:"requires_tool_result_name,omitempty"`

	// RequiresAssistantAfterToolResult requires a synthetic assistant message
	// between tool results and a following user message.
	RequiresAssistantAfterToolResult *bool `yaml:"requires_assistant_after_tool_result,omitempty"`

	// BackfillThinkingField injects an empty thinking field into assistant
	// messages that carry tool calls but are missing it.
	//
	// This is the workaround for the DeepSeek 400 above. The endpoint checks
	// that the field is present, not that it holds anything, so an empty
	// string passes — and unlike replaying the original text it costs no
	// context and keeps thinking content out of stored messages. Applied only
	// to tool-calling turns, which are the ones that trigger the rejection.
	BackfillThinkingField *bool `yaml:"backfill_thinking_field,omitempty"`
}

// reasoningEffortValues are the abstract effort levels a caller may request.
// Endpoints name them differently and support different subsets; ApplyRequest
// maps from this set onto whatever the endpoint actually accepts.
var reasoningEffortValues = []string{"minimal", "low", "medium", "high", "xhigh"}

// ValidThinkingFormats lists the accepted thinking_format values, used to
// produce a usable error message instead of a silent no-op.
var ValidThinkingFormats = []ThinkingFormat{
	ThinkingFormatOpenAI,
	ThinkingFormatQwen,
	ThinkingFormatQwenChatTemplate,
	ThinkingFormatDeepSeek,
	ThinkingFormatChatTemplate,
	ThinkingFormatOpenRouter,
	ThinkingFormatString,
}

// ValidThinkingBudgetFields lists the accepted thinking budget field names.
var ValidThinkingBudgetFields = []string{
	"thinking_budget",
	"thinking_token_budget",
	"thinking_budget_tokens",
}

// Validate checks the cross-field constraints that a JSON Schema cannot
// express, and fails startup rather than letting a misconfiguration surface
// as a confusing 400 mid-conversation (ADR-008). Every message names the
// offending provider and states how to fix it.
func (c *Compat) Validate(providerName string) error {
	if c.ThinkingFormat != "" && !containsThinkingFormat(ValidThinkingFormats, c.ThinkingFormat) {
		return fmt.Errorf(
			"provider %q: unknown thinking_format %q; valid values are %v",
			providerName, c.ThinkingFormat, ValidThinkingFormats)
	}

	if c.SendBackThinking == SendBackField && c.ThinkingField == "" {
		return fmt.Errorf(
			"provider %q: send_back_thinking=%q requires thinking_field to be set; "+
				"DeepSeek-compatible endpoints return 400 on tool-calling turns whose "+
				"assistant messages omit the thinking field. "+
				"Set providers.%s.compat.thinking_field (e.g. reasoning_content)",
			providerName, SendBackField, providerName)
	}

	switch c.SendBackThinking {
	case "", SendBackAuto, SendBackField, SendBackTags, SendBackNever:
	default:
		return fmt.Errorf(
			"provider %q: unknown send_back_thinking %q; valid values are auto, field, tags, never",
			providerName, c.SendBackThinking)
	}

	switch c.MaxTokensField {
	case "", "max_tokens", "max_completion_tokens":
	default:
		return fmt.Errorf(
			"provider %q: max_tokens_field must be \"max_tokens\" or \"max_completion_tokens\", got %q",
			providerName, c.MaxTokensField)
	}

	if c.ThinkingBudgetField != "" && !containsString(ValidThinkingBudgetFields, c.ThinkingBudgetField) {
		return fmt.Errorf(
			"provider %q: unknown thinking_budget_field %q; valid values are %v "+
				"(thinking_budget=Qwen/DashScope/SGLang, thinking_token_budget=vLLM, "+
				"thinking_budget_tokens=llama.cpp)",
			providerName, c.ThinkingBudgetField, ValidThinkingBudgetFields)
	}

	if c.ThinkingFormat == ThinkingFormatQwen || c.ThinkingFormat == ThinkingFormatQwenChatTemplate {
		if boolValue(c.SupportsStrictMode) {
			return fmt.Errorf(
				"provider %q: thinking_format %q cannot be combined with supports_strict_mode=true; "+
					"the Qwen family rejects tool definitions carrying \"strict\"",
				providerName, c.ThinkingFormat)
		}
	}

	return nil
}

// Resolved fills in every default so consumers never re-derive the behaviour of
// an unset field. Tri-state *bool fields collapse to plain bools here: unset
// and explicit-false mean the same thing once merging is done.
func (c *Compat) Resolved() ResolvedCompat {
	return ResolvedCompat{
		ThinkingFormat:                c.effectiveThinkingFormat(),
		MaxTokensField:                c.effectiveMaxTokensField(),
		ThinkingField:                 c.ThinkingField,
		SendBackThinking:              c.effectiveSendBackThinking(),
		ThinkingTags:                  c.effectiveThinkingTags(),
		ThinkingBudgetField:           c.ThinkingBudgetField,
		IgnoreLeadingWhitespace:       boolValue(c.IgnoreLeadingWhitespace),
		SupportsDeveloperRole:         boolValue(c.SupportsDeveloperRole),
		SupportsStrictMode:            boolValue(c.SupportsStrictMode),
		SupportsStore:                 boolValue(c.SupportsStore),
		SupportsForcedToolChoice:      boolValueOr(c.SupportsForcedToolChoice, true),
		ForcedToolChoiceWithThinking:  boolValueOr(c.SupportsForcedToolChoiceWithThinking, true),
		RequiresToolResultName:        boolValue(c.RequiresToolResultName),
		RequiresAssistantAfterToolRst: boolValue(c.RequiresAssistantAfterToolResult),
		BackfillThinkingField:         boolValue(c.BackfillThinkingField),
	}
}

// ResolvedCompat is a Compat with every default filled in, so that consumers
// never have to re-derive the "unset" behaviour of a field.
type ResolvedCompat struct {
	ThinkingFormat                ThinkingFormat
	MaxTokensField                string
	ThinkingField                 string
	SendBackThinking              SendBackThinkingMode
	ThinkingTags                  [2]string
	IgnoreLeadingWhitespace       bool
	SupportsDeveloperRole         bool
	SupportsStrictMode            bool
	SupportsStore                 bool
	SupportsForcedToolChoice      bool
	ForcedToolChoiceWithThinking  bool
	RequiresToolResultName        bool
	RequiresAssistantAfterToolRst bool
	BackfillThinkingField         bool
	ThinkingBudgetField           string
}

// ResolveCompat merges a provider-level Compat with a model-level one. The
// model config overrides the provider config field by field; fields the model
// leaves unset keep the provider value (ADR-004, two levels only).
// Either argument may be nil.
func ResolveCompat(provider, model *Compat) ResolvedCompat {
	base := &Compat{}
	if provider != nil {
		*base = *provider
	}
	if model != nil {
		mergeCompat(base, model)
	}
	return base.Resolved()
}

// mergeCompat overwrites only the fields the override actually sets. A nil
// *bool is "unset" and leaves the existing value alone; a non-nil one replaces
// it, including when it points at false.
func mergeCompat(base, override *Compat) {
	if override.ThinkingFormat != "" {
		base.ThinkingFormat = override.ThinkingFormat
	}
	if override.ThinkingField != "" {
		base.ThinkingField = override.ThinkingField
	}
	if override.SendBackThinking != "" {
		base.SendBackThinking = override.SendBackThinking
	}
	if override.ThinkingBudgetField != "" {
		base.ThinkingBudgetField = override.ThinkingBudgetField
	}
	if override.ThinkingTags != ([2]string{}) {
		base.ThinkingTags = override.ThinkingTags
	}
	if override.MaxTokensField != "" {
		base.MaxTokensField = override.MaxTokensField
	}
	if override.SupportsDeveloperRole != nil {
		base.SupportsDeveloperRole = override.SupportsDeveloperRole
	}
	if override.SupportsForcedToolChoice != nil {
		base.SupportsForcedToolChoice = override.SupportsForcedToolChoice
	}
	if override.SupportsForcedToolChoiceWithThinking != nil {
		base.SupportsForcedToolChoiceWithThinking = override.SupportsForcedToolChoiceWithThinking
	}
	if override.IgnoreLeadingWhitespace != nil {
		base.IgnoreLeadingWhitespace = override.IgnoreLeadingWhitespace
	}
	if override.SupportsStrictMode != nil {
		base.SupportsStrictMode = override.SupportsStrictMode
	}
	if override.SupportsStore != nil {
		base.SupportsStore = override.SupportsStore
	}
	if override.RequiresToolResultName != nil {
		base.RequiresToolResultName = override.RequiresToolResultName
	}
	if override.RequiresAssistantAfterToolResult != nil {
		base.RequiresAssistantAfterToolResult = override.RequiresAssistantAfterToolResult
	}
	if override.BackfillThinkingField != nil {
		base.BackfillThinkingField = override.BackfillThinkingField
	}
}

func (c *Compat) effectiveThinkingFormat() ThinkingFormat {
	if c.ThinkingFormat == "" {
		return ThinkingFormatOpenAI
	}
	return c.ThinkingFormat
}

func (c *Compat) effectiveMaxTokensField() string {
	if c.MaxTokensField == "" {
		return "max_tokens"
	}
	return c.MaxTokensField
}

func (c *Compat) effectiveSendBackThinking() SendBackThinkingMode {
	if c.SendBackThinking == "" {
		return SendBackAuto
	}
	return c.SendBackThinking
}

func (c *Compat) effectiveThinkingTags() [2]string {
	if c.ThinkingTags == ([2]string{}) {
		return DefaultThinkingTags
	}
	return c.ThinkingTags
}

// SupportsForced reports whether a forced tool choice may be sent for this
// request. thinking states whether the model is reasoning, which includes
// models that reason by default.
//
// The two capability bits are separate because endpoints differ in which
// restriction applies: some reject "required" outright, others only while
// thinking (ADR-007).
func (r ResolvedCompat) SupportsForced(thinking bool) bool {
	if !r.SupportsForcedToolChoice {
		return false
	}
	if thinking && !r.ForcedToolChoiceWithThinking {
		return false
	}
	return true
}

// ApplyRequest rewrites cfg so the request matches this endpoint's spelling of
// the common fields. maxTokens is the caller's own limit; pass 0 to leave it
// unset. reasoningEffort is the abstract effort level, or "" for no reasoning.
//
// Two mechanisms are used, because openai-go models the standard fields as real
// struct fields and everything else as extra JSON:
//
//   - MaxTokensField selects MaxTokens or MaxCompletionTokens. Both exist on
//     ChatCompletionNewParams and serialize under their own names, so no
//     hand-built JSON is needed.
//   - SetExtraFields carries non-standard top-level keys such as
//     enable_thinking and chat_template_kwargs. genkit's compat_oai accepts a
//     typed ChatCompletionNewParams, so this is the only channel that reaches
//     the wire; passing a map[string]any through ai.WithConfig instead would
//     round-trip through json.Unmarshal and drop every unknown key.
func (r ResolvedCompat) ApplyRequest(cfg *openai.ChatCompletionNewParams, maxTokens int, reasoningEffort string) {
	if cfg == nil {
		return
	}

	// Length field. A zero param.Opt is "omitted", and the struct tags carry
	// omitzero, so assigning one spelling and clearing the other leaves
	// exactly the configured key in the request body.
	switch r.MaxTokensField {
	case "max_completion_tokens":
		cfg.MaxTokens = param.Opt[int64]{}
		if maxTokens > 0 {
			cfg.MaxCompletionTokens = openai.Int(int64(maxTokens))
		}
	default:
		cfg.MaxCompletionTokens = param.Opt[int64]{}
		if maxTokens > 0 {
			cfg.MaxTokens = openai.Int(int64(maxTokens))
		}
	}

	extra := map[string]any{}

	thinking := reasoningEffort != ""
	if thinking {
		switch r.ThinkingFormat {
		case ThinkingFormatQwen:
			extra["enable_thinking"] = true
		case ThinkingFormatQwenChatTemplate:
			extra["chat_template_kwargs"] = map[string]any{
				"enable_thinking":   true,
				"preserve_thinking": true,
			}
		case ThinkingFormatChatTemplate:
			// vLLM keys the flag per model family, so the value is forwarded
			// verbatim under the neutral name and the operator picks the right
			// one via models.yaml.
			extra["chat_template_kwargs"] = map[string]any{
				"thinking": true,
			}
		case ThinkingFormatDeepSeek:
			extra["thinking"] = map[string]any{"type": "enabled"}
		case ThinkingFormatOpenRouter:
			extra["reasoning"] = map[string]any{"effort": reasoningEffort}
		case ThinkingFormatString:
			extra["thinking"] = reasoningEffort
		default:
			extra["reasoning_effort"] = reasoningEffort
		}
	} else {
		switch r.ThinkingFormat {
		case ThinkingFormatQwen:
			extra["enable_thinking"] = false
		case ThinkingFormatQwenChatTemplate:
			extra["chat_template_kwargs"] = map[string]any{
				"enable_thinking":   false,
				"preserve_thinking": true,
			}
		case ThinkingFormatChatTemplate:
			extra["chat_template_kwargs"] = map[string]any{"thinking": false}
		case ThinkingFormatDeepSeek:
			extra["thinking"] = map[string]any{"type": "disabled"}
		}
	}

	// The budget goes out whenever one is configured, whether or not reasoning
	// was requested, because some endpoints think by default. Measured on
	// qwen3.5-plus (2026-10-04): a request that never mentions thinking still
	// burned 1648 reasoning tokens, drawn from the same max_tokens pool as the
	// answer. Gating the budget on an explicit effort would leave that case
	// uncapped. DashScope treats -1 as "endpoint default".
	if r.ThinkingBudgetField != "" {
		extra[r.ThinkingBudgetField] = -1
	}

	if len(extra) > 0 {
		cfg.SetExtraFields(extra)
	}
}

// ExtractThinking pulls thinking content out of a raw response object, probing
// the configured field first and then ThinkingFields in order. It returns the
// content and the name of the field it came from, so a later turn can replay
// it through the same field.
//
// Taking the first non-empty value rather than concatenating matters because
// some endpoints return both spellings with identical content, which would
// otherwise be duplicated into the prompt.
func (r ResolvedCompat) ExtractThinking(msg map[string]any) (content, field string) {
	candidates := ThinkingFields
	if r.ThinkingField != "" {
		candidates = append([]string{r.ThinkingField}, ThinkingFields...)
	}
	for _, name := range candidates {
		if v, ok := msg[name].(string); ok && v != "" {
			return v, name
		}
	}
	return "", ""
}

func boolValue(v *bool) bool {
	return v != nil && *v
}

func boolValueOr(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

func containsThinkingFormat(list []ThinkingFormat, want ThinkingFormat) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func containsString(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
