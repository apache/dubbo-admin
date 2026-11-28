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
	"strings"

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
// This array is the whole reason the probe order exists — see
// docs/ai-provider-compat.md#thinking-field-probe-order. vLLM renamed
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
// # Applied and reserved fields
//
// The fields fall into two groups, and the difference matters more than the
// field names do:
//
//   - Applied fields reach the request. ApplyRequest reads them, and they
//     change the bytes on the wire.
//   - Reserved fields are parsed and validated, but nothing applies them yet.
//     They describe parts of the request that genkit's compat_oai plugin
//     builds — the message roles, the tool definitions, the tool-choice value
//     and the shape of the message list — and that plugin takes a typed
//     request whose shape this layer cannot reach.
//
// A reserved field that is configured is rejected at startup, so a
// configuration can never promise a behaviour the request does not have.
// docs/ai-provider-compat.md lists every field, which group it is in, and
// where the reserved ones would be applied.
//
// # Conventions
//
// Field naming follows capability semantics: a field states what the endpoint
// supports, and its doc comment states what happens when it is disabled.
// Tri-state booleans are *bool so that "unset" stays distinguishable from
// "explicitly false" when provider-level and model-level configs merge.
//
// Defaults are conservative rather than OpenAI-like: an unset field picks the
// spelling the widest set of endpoints accepts. There is no base-URL
// sniffing, because a gateway in front of a provider defeats it.
type Compat struct {
	// ThinkingFormat selects the thinking parameter spelling. Unset behaves as
	// ThinkingFormatOpenAI. Applied.
	ThinkingFormat ThinkingFormat `yaml:"thinking_format,omitempty"`

	// ThinkingBudgetField names the top-level field that caps thinking tokens.
	// Unset sends nothing.
	//
	// This matters more for agents than for chat: thinking and the answer
	// share max_tokens, so an uncapped reasoning-heavy turn can consume the
	// whole response and emit neither an answer nor a tool call — which just
	// spins the next iteration of the loop. Accepted values are
	// "thinking_budget" (Qwen/DashScope/SGLang), "thinking_token_budget"
	// (vLLM) and "thinking_budget_tokens" (llama.cpp). Applied.
	ThinkingBudgetField string `yaml:"thinking_budget_field,omitempty"`

	// ThinkingBudget is the cap, in tokens, sent in the ThinkingBudgetField.
	// Unset (zero) sends no cap at all.
	//
	// The cap has to be a real number to be worth sending: DashScope reads -1
	// as "use the endpoint default", which is the same as sending nothing, and
	// the other two spellings have no meaning for -1 at all and may reject it.
	// A cap also competes with the answer for the same max_tokens pool, so
	// leave it unset unless the endpoint reasons by default and the turn
	// budget is tight. Applied.
	ThinkingBudget int `yaml:"thinking_budget,omitempty"`

	// ChatTemplateKey is the inner key of chat_template_kwargs that carries
	// the thinking flag. Unset means "enable_thinking" for the
	// qwen-chat-template format, and is required for the chat-template
	// format.
	//
	// vLLM keys the flag per model family — enable_thinking on the Qwen
	// templates, thinking on others — so the inner key has to be selectable
	// rather than hardcoded. Applied.
	ChatTemplateKey string `yaml:"chat_template_key,omitempty"`

	// MaxTokensField is "max_tokens" or "max_completion_tokens". Unset uses
	// "max_tokens", which the older and wider set of endpoints accepts.
	// Applied.
	MaxTokensField string `yaml:"max_tokens_field,omitempty"`

	// --- reserved fields ---
	//
	// The fields below are parsed and validated, but nothing applies them to
	// the request yet, because the part of the request they would change is
	// built by genkit's compat_oai plugin. Configuring one is rejected at
	// startup rather than silently ignored. docs/ai-provider-compat.md
	// explains each one and says where it would be applied.

	// ThinkingField is the response field carrying thinking content.
	//
	// Reading tries this first, then falls back to ThinkingFields.
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

	// ThinkingTags are the XML delimiters for tags-mode replay. Unset uses
	// DefaultThinkingTags.
	ThinkingTags [2]string `yaml:"thinking_tags,omitempty"`

	// IgnoreLeadingWhitespace drops leading whitespace from streamed content.
	//
	// Works around models that emit "<think>\n</think>\n\n", or an empty text
	// part, ahead of a tool call — which an agent loop can otherwise mistake
	// for the final answer and exit on. Observed with Ollama + Qwen3.
	IgnoreLeadingWhitespace *bool `yaml:"ignore_leading_whitespace,omitempty"`

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

// ReasoningEffortLevels are the abstract effort levels a config may request.
//
// The level is an abstract name, not an endpoint's own vocabulary. The formats
// that carry a level forward it as-is — the OpenAI format sends
// reasoning_effort, the OpenRouter format sends reasoning.effort — and the
// formats whose switch is a plain boolean drop it, so the endpoint picks its
// own budget.
//
// Which of these levels a given endpoint accepts is a property of that
// endpoint, and nothing here checks it: a level the endpoint does not know is
// rejected by the endpoint, in the same way an unsupported value of any other
// kind is. See docs/ai-provider-compat.md#the-effort-level-is-not-always-carried.
var ReasoningEffortLevels = []string{"minimal", "low", "medium", "high", "xhigh"}

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

// reservedCompatFields lists the Compat fields that are parsed and validated
// but not applied to the request.
//
// Each entry carries the reason the field is reserved, because the operator
// has to be able to tell "not implemented yet" apart from "wrong value" — and
// because a reserved field that is configured has to be refused rather than
// silently ignored. docs/ai-provider-compat.md lists the same fields with the
// place each of them would be applied.
var reservedCompatFields = []struct {
	name       string
	configured func(*Compat) bool
	reason     string
}{
	{
		name:       "thinking_field",
		configured: func(c *Compat) bool { return c.ThinkingField != "" },
		reason: "reading thinking content back out of a response is not wired up: " +
			"genkit's compat_oai plugin maps only the content and the tool calls, " +
			"so the reasoning field of a response never reaches this layer",
	},
	{
		name:       "send_back_thinking",
		configured: func(c *Compat) bool { return c.SendBackThinking != "" },
		reason: "replaying thinking content on later turns needs the message list, " +
			"and the message list is built by genkit's compat_oai plugin",
	},
	{
		name:       "thinking_tags",
		configured: func(c *Compat) bool { return c.ThinkingTags != ([2]string{}) },
		reason:     "the tags delimiters belong to the thinking replay, which is not wired up yet",
	},
	{
		name:       "ignore_leading_whitespace",
		configured: func(c *Compat) bool { return c.IgnoreLeadingWhitespace != nil },
		reason: "the whitespace is dropped while the response is streamed, and the " +
			"response is streamed by genkit's compat_oai plugin",
	},
	{
		name:       "supports_developer_role",
		configured: func(c *Compat) bool { return c.SupportsDeveloperRole != nil },
		reason: "the message roles are chosen by genkit's prompt options, " +
			"not by this layer",
	},
	{
		name:       "supports_strict_mode",
		configured: func(c *Compat) bool { return c.SupportsStrictMode != nil },
		reason: "the tool definitions are built by genkit's tool options, " +
			"so the \"strict\" flag on them is out of reach here",
	},
	{
		name:       "supports_store",
		configured: func(c *Compat) bool { return c.SupportsStore != nil },
		reason: "the request is assembled by genkit's compat_oai plugin, " +
			"which does not expose the \"store\" field",
	},
	{
		name:       "supports_forced_tool_choice",
		configured: func(c *Compat) bool { return c.SupportsForcedToolChoice != nil },
		reason: "the agent never forces a tool choice — it lets the endpoint pick — " +
			"and the tool-choice value belongs to genkit's plugin",
	},
	{
		name:       "supports_forced_tool_choice_with_thinking",
		configured: func(c *Compat) bool { return c.SupportsForcedToolChoiceWithThinking != nil },
		reason: "the agent never forces a tool choice — it lets the endpoint pick — " +
			"and the tool-choice value belongs to genkit's plugin",
	},
	{
		name:       "requires_tool_result_name",
		configured: func(c *Compat) bool { return c.RequiresToolResultName != nil },
		reason:     "the message list is built by genkit's compat_oai plugin",
	},
	{
		name:       "requires_assistant_after_tool_result",
		configured: func(c *Compat) bool { return c.RequiresAssistantAfterToolResult != nil },
		reason:     "the message list is built by genkit's compat_oai plugin",
	},
	{
		name:       "backfill_thinking_field",
		configured: func(c *Compat) bool { return c.BackfillThinkingField != nil },
		reason:     "the message list is built by genkit's compat_oai plugin",
	},
}

// Validate checks the cross-field constraints that a JSON Schema cannot
// express, and fails startup rather than letting a misconfiguration surface
// as a confusing 400 mid-conversation. Every message names the offending
// provider and states how to fix it.
//
// The reserved fields are checked last, so a mistake that has a more specific
// explanation is reported with that explanation rather than with the generic
// "not implemented yet".
func (c *Compat) Validate(providerName string) error {
	if c.ThinkingFormat != "" && !containsThinkingFormat(ValidThinkingFormats, c.ThinkingFormat) {
		return fmt.Errorf(
			"provider %q: unknown thinking_format %q; valid values are %v",
			providerName, c.ThinkingFormat, ValidThinkingFormats)
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

	if c.ThinkingBudget < 0 {
		return fmt.Errorf(
			"provider %q: thinking_budget must not be negative, got %d; "+
				"0 means no cap, a positive value is the cap in tokens",
			providerName, c.ThinkingBudget)
	}

	if c.ThinkingBudget > 0 && c.ThinkingBudgetField == "" {
		return fmt.Errorf(
			"provider %q: thinking_budget=%d needs thinking_budget_field to say which key "+
				"carries the cap; valid values are %v "+
				"(thinking_budget=Qwen/DashScope/SGLang, thinking_token_budget=vLLM, "+
				"thinking_budget_tokens=llama.cpp)",
			providerName, c.ThinkingBudget, ValidThinkingBudgetFields)
	}

	if c.ChatTemplateKey != "" && !isChatTemplateFormat(c.ThinkingFormat) {
		return fmt.Errorf(
			"provider %q: chat_template_key only applies to thinking_format %q or %q, got %q",
			providerName, ThinkingFormatQwenChatTemplate, ThinkingFormatChatTemplate, c.ThinkingFormat)
	}

	if c.ThinkingFormat == ThinkingFormatChatTemplate && c.ChatTemplateKey == "" {
		return fmt.Errorf(
			"provider %q: thinking_format %q needs chat_template_key; "+
				"vLLM keys the thinking flag per model family (enable_thinking on the Qwen "+
				"templates, thinking on others), so the inner key has to be chosen. "+
				"Set providers.%s.compat.chat_template_key",
			providerName, ThinkingFormatChatTemplate, providerName)
	}

	if c.ThinkingFormat == ThinkingFormatQwen || c.ThinkingFormat == ThinkingFormatQwenChatTemplate {
		if boolValue(c.SupportsStrictMode) {
			return fmt.Errorf(
				"provider %q: thinking_format %q cannot be combined with supports_strict_mode=true; "+
					"the Qwen family rejects tool definitions carrying \"strict\"",
				providerName, c.ThinkingFormat)
		}
	}

	return c.validateReserved(providerName)
}

// validateReserved refuses a configuration that sets a field the request path
// does not apply. Silently ignoring it would be the worst outcome: the
// operator would read a configuration that promises a behaviour the endpoint
// never sees.
func (c *Compat) validateReserved(providerName string) error {
	var offending []string
	for _, field := range reservedCompatFields {
		if field.configured(c) {
			offending = append(offending,
				fmt.Sprintf("%s (%s)", field.name, field.reason))
		}
	}
	if len(offending) == 0 {
		return nil
	}
	return fmt.Errorf(
		"provider %q: %d compat field(s) are configured but not applied to the request: %s; "+
			"remove them until the capability is wired. "+
			"See docs/ai-provider-compat.md for the capabilities that are applied today",
		providerName, len(offending), strings.Join(offending, "; "))
}

// Resolved fills in every default so consumers never re-derive the behaviour of
// an unset field. Tri-state *bool fields collapse to plain bools here: unset
// and explicit-false mean the same thing once merging is done.
//
// The reserved fields are resolved as well, even though nothing reads them yet.
// Resolving them the same way keeps the merge rules one-to-one with the
// config, so wiring a capability up later is a matter of reading the field
// rather than of changing how it is merged.
func (c *Compat) Resolved() ResolvedCompat {
	return ResolvedCompat{
		ThinkingFormat:                       c.effectiveThinkingFormat(),
		MaxTokensField:                       c.effectiveMaxTokensField(),
		ThinkingBudgetField:                  c.ThinkingBudgetField,
		ThinkingBudget:                       c.ThinkingBudget,
		ChatTemplateKey:                      c.effectiveChatTemplateKey(),
		ThinkingField:                        c.ThinkingField,
		SendBackThinking:                     c.effectiveSendBackThinking(),
		ThinkingTags:                         c.effectiveThinkingTags(),
		IgnoreLeadingWhitespace:              boolValue(c.IgnoreLeadingWhitespace),
		SupportsDeveloperRole:                boolValue(c.SupportsDeveloperRole),
		SupportsStrictMode:                   boolValue(c.SupportsStrictMode),
		SupportsStore:                        boolValue(c.SupportsStore),
		SupportsForcedToolChoice:             boolValueOr(c.SupportsForcedToolChoice, true),
		SupportsForcedToolChoiceWithThinking: boolValueOr(c.SupportsForcedToolChoiceWithThinking, true),
		RequiresToolResultName:               boolValue(c.RequiresToolResultName),
		RequiresAssistantAfterToolResult:     boolValue(c.RequiresAssistantAfterToolResult),
		BackfillThinkingField:                boolValue(c.BackfillThinkingField),
	}
}

// ResolvedCompat is a Compat with every default filled in, so that consumers
// never have to re-derive the "unset" behaviour of a field.
//
// The field names mirror the Compat ones, because the merge rules are
// one-to-one with the config. Only the applied fields change the request;
// the reserved ones are resolved but not yet read by anything.
type ResolvedCompat struct {
	ThinkingFormat                       ThinkingFormat
	MaxTokensField                       string
	ThinkingBudgetField                  string
	ThinkingBudget                       int
	ChatTemplateKey                      string
	ThinkingField                        string
	SendBackThinking                     SendBackThinkingMode
	ThinkingTags                         [2]string
	IgnoreLeadingWhitespace              bool
	SupportsDeveloperRole                bool
	SupportsStrictMode                   bool
	SupportsStore                        bool
	SupportsForcedToolChoice             bool
	SupportsForcedToolChoiceWithThinking bool
	RequiresToolResultName               bool
	RequiresAssistantAfterToolResult     bool
	BackfillThinkingField                bool
}

// ResolveCompat merges a provider-level Compat with a model-level one. The
// model config overrides the provider config field by field; fields the model
// leaves unset keep the provider value (two levels only — see
// docs/ai-provider-compat.md#two-level-merge).
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
	if override.ThinkingBudget != 0 {
		base.ThinkingBudget = override.ThinkingBudget
	}
	if override.ChatTemplateKey != "" {
		base.ChatTemplateKey = override.ChatTemplateKey
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

// effectiveChatTemplateKey picks the inner key of chat_template_kwargs that
// carries the thinking flag. The Qwen templates use enable_thinking, so that
// is the default there. The generic chat-template format has no safe default
// — vLLM serves model families that disagree on the name — so Validate
// insists the operator picks one.
func (c *Compat) effectiveChatTemplateKey() string {
	if c.ChatTemplateKey != "" {
		return c.ChatTemplateKey
	}
	if c.ThinkingFormat == ThinkingFormatQwenChatTemplate {
		return "enable_thinking"
	}
	return ""
}

// isChatTemplateFormat reports whether the format carries the thinking flag
// inside chat_template_kwargs.
func isChatTemplateFormat(format ThinkingFormat) bool {
	return format == ThinkingFormatQwenChatTemplate || format == ThinkingFormatChatTemplate
}

// SupportsForced reports whether a forced tool choice may be sent for this
// request. thinking states whether the model is reasoning, which includes
// models that reason by default.
//
// The two capability bits are separate because endpoints differ in which
// restriction applies: some reject "required" outright, others only while
// thinking (see docs/ai-provider-compat.md#separate-capability-bits).
func (r ResolvedCompat) SupportsForced(thinking bool) bool {
	if !r.SupportsForcedToolChoice {
		return false
	}
	if thinking && !r.SupportsForcedToolChoiceWithThinking {
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
				r.ChatTemplateKey:   true,
				"preserve_thinking": true,
			}
		case ThinkingFormatChatTemplate:
			// vLLM keys the flag per model family, so the inner key has to be
			// configurable rather than hardcoded; Validate insists that the
			// operator picks one for this format.
			if r.ChatTemplateKey != "" {
				extra["chat_template_kwargs"] = map[string]any{
					r.ChatTemplateKey: true,
				}
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
				r.ChatTemplateKey:   false,
				"preserve_thinking": true,
			}
		case ThinkingFormatChatTemplate:
			if r.ChatTemplateKey != "" {
				extra["chat_template_kwargs"] = map[string]any{r.ChatTemplateKey: false}
			}
		case ThinkingFormatDeepSeek:
			extra["thinking"] = map[string]any{"type": "disabled"}
		}
	}

	// The cap goes out whenever one is configured, whether or not reasoning was
	// requested, because some endpoints think by default. Measured on
	// qwen3.5-plus (2026-10-04): a request that never mentions thinking still
	// burned 1648 reasoning tokens, drawn from the same max_tokens pool as the
	// answer. Gating the cap on an explicit effort would leave that case
	// uncapped.
	//
	// Only a real number is sent. DashScope reads -1 as "endpoint default",
	// which caps nothing, and the other two spellings have no meaning for -1
	// at all and may reject it — a cap that is not a cap is not worth
	// sending.
	if r.ThinkingBudgetField != "" && r.ThinkingBudget > 0 {
		extra[r.ThinkingBudgetField] = r.ThinkingBudget
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
