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

package context

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"dubbo-admin-ai/security"
	"dubbo-admin-ai/store"
)

type registration struct {
	source     ContextSource
	descriptor SourceDescriptor
	policy     SourcePolicy
}
type Compiler struct {
	sources   []registration
	templates map[string]Template
}

func New(templates ...Template) (*Compiler, error) {
	c := &Compiler{templates: map[string]Template{}}
	namedVersions := map[string]string{}
	for _, t := range templates {
		if t.ID == "" || t.Version == "" || len(t.ID) > 64 || len(t.Version) > 32 {
			return nil, fmt.Errorf("template identity required")
		}
		// Copy all caller-owned maps before retaining a registered version.
		data := JSON(t)
		var copy Template
		if err := json.Unmarshal(data, &copy); err != nil {
			return nil, err
		}
		key := copy.ID + "@" + copy.Version
		if old, ok := namedVersions[key]; ok && old != copy.Ref() {
			return nil, fmt.Errorf("template version content changed: %s", key)
		}
		namedVersions[key] = copy.Ref()
		c.templates[copy.Ref()] = copy
	}
	return c, nil
}

// Register is called during construction, before any concurrent compilation.
func (c *Compiler) Register(source ContextSource, p SourcePolicy) error {
	if source == nil {
		return fmt.Errorf("nil source")
	}
	d := source.Descriptor()
	if d.ID == "" || d.Version == "" || len(p.Kinds) == 0 {
		return fmt.Errorf("invalid source registration")
	}
	for _, r := range c.sources {
		if r.descriptor.ID == d.ID {
			return fmt.Errorf("duplicate source %s", d.ID)
		}
	}
	p.Kinds = append([]string(nil), p.Kinds...)
	d.RequiredPermission = append([]string(nil), d.RequiredPermission...)
	if p.Trust == "" {
		p.Trust = "untrusted"
	}
	if p.Authority == "" {
		p.Authority = "context-data"
	}
	if p.Sensitivity == "" {
		p.Sensitivity = "model-input-filtered"
	}
	if p.Scope == "" {
		p.Scope = "session"
	}
	if p.Scope != "session" && p.Scope != "user" {
		return fmt.Errorf("unsupported source scope")
	}
	c.sources = append(c.sources, registration{source, d, p})
	sort.Slice(c.sources, func(i, j int) bool {
		a, b := c.sources[i], c.sources[j]
		if a.policy.Order != b.policy.Order {
			return a.policy.Order < b.policy.Order
		}
		return a.descriptor.ID < b.descriptor.ID
	})
	return nil
}

type candidate struct {
	dependencies     []store.MemoryDependency
	sourceID         string
	sourceTokenLimit int
	summarySequence  uint64
	historyPrevious  uint64
	historySequence  uint64
	coversThrough    uint64
	messages         []Message
	required         bool
	tokens           int
	decision         int
	bytes            int
	sequence         string
	unavailable      bool
}

// Estimate counts UTF-8 bytes conservatively, plus protocol framing. This is an
// estimate, not a tokenizer guarantee; the method/version travels in the report.
func Estimate(value any) int { return len(JSON(value)) + 16 }

func (c *Compiler) Compile(ctx context.Context, req CompileRequest) (*CompiledRequest, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := req.Access.ValidateBusiness(); err != nil {
		return nil, err
	}
	if req.SessionID == "" || req.TurnID == 0 || req.MaxItems <= 0 || req.MaxItems > 10000 {
		return nil, fmt.Errorf("invalid bounded collection request")
	}
	if req.SessionTenantID == "" || req.SessionOwnerID == "" {
		for _, turn := range req.Turns {
			if turn.ID == req.TurnID {
				req.SessionTenantID, req.SessionOwnerID = turn.TenantID, turn.OwnerID
				break
			}
		}
	}
	if !req.Access.CanAccessOwner(req.SessionTenantID, req.SessionOwnerID) {
		return nil, security.ErrResourceNotAccessible
	}
	if req.ContextTokens <= 0 || req.OutputTokens <= 0 || req.SafetyMargin < 0 ||
		req.SafetyMargin >= req.ContextTokens || req.OutputTokens > req.ContextTokens-req.SafetyMargin {
		return nil, ErrInvalidBudget
	}
	if req.MaxBytes <= 0 {
		req.MaxBytes = 8 << 20
	}
	if req.MaxCollectBytes <= 0 {
		req.MaxCollectBytes = 8 << 20
	}
	t, ok := c.templates[req.TemplateRef]
	if !ok {
		return nil, ErrTemplateVersion
	}
	// The configuration is a server-owned value pool shared by template versions.
	// Resolve against the pinned version; Render retains its strict input contract.
	values := make(map[string]string, len(t.Variables))
	for key := range t.Variables {
		if value, exists := req.Variables[key]; exists {
			values[key] = value
		}
	}
	system, err := t.Render(req.Phase, values)
	if err != nil {
		return nil, err
	}
	var tools []Tool
	for _, tool := range req.Tools {
		for _, name := range t.AllowedTools {
			if name == tool.Name {
				tools = append(tools, tool)
				break
			}
		}
	}
	if req.Phase == "answer" {
		tools = nil
	}
	sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })
	for i, v := range tools {
		if v.Name == "" || (i > 0 && tools[i-1].Name == v.Name) {
			return nil, fmt.Errorf("invalid tool set")
		}
	}
	if _, err := json.Marshal(tools); err != nil {
		return nil, fmt.Errorf("invalid tool schema: %w", err)
	}
	out := &CompiledRequest{Model: req.Model, Phase: req.Phase, TemplateRef: req.TemplateRef, PolicyVersion: "core-v1", Tools: tools, OutputTokens: req.OutputTokens, SafetyMargin: req.SafetyMargin, Temperature: req.Temperature, TopP: req.TopP}
	out.ContextTokens = req.ContextTokens
	out.Messages = []Message{{Role: "system", Text: system}}
	out.Report = CompilationReport{InteractionID: req.InteractionID, CompilerVersion: "v2", PolicyVersion: out.PolicyVersion, TemplateRef: req.TemplateRef, EstimatorVersion: "utf8-json-bytes-v1", ReservedOutput: req.OutputTokens, SafetyMargin: req.SafetyMargin}
	out.Report.ContextTokens = req.ContextTokens
	if req.SummaryReason != "" {
		out.Report.Degraded = true
		out.Report.DegradationReasons = append(out.Report.DegradationReasons, req.SummaryReason)
	}
	base := Estimate(out.Messages) + Estimate(tools) + 64
	available := req.ContextTokens - req.OutputTokens - req.SafetyMargin
	var candidates []candidate
	seen := map[string]bool{}
	for _, r := range c.sources {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		permitted := true
		for _, scope := range r.descriptor.RequiredPermission {
			permitted = permitted && req.Access.Subject.HasScope(scope)
		}
		if !permitted {
			if r.policy.Required {
				return nil, security.ErrResourceNotAccessible
			}
			out.Report.Degraded = true
			out.Report.DegradationReasons = append(out.Report.DegradationReasons, "SOURCE_NOT_AUTHORIZED:"+r.descriptor.ID)
			continue
		}
		// A Source cannot mutate the identity or canonical snapshot seen by a
		// later Source through shared slices, maps or message pointers.
		var sourceRequest CollectRequest
		if err := json.Unmarshal(JSON(req.CollectRequest), &sourceRequest); err != nil {
			return nil, err
		}
		result, err := r.source.Collect(ctx, sourceRequest)
		if err != nil {
			if !r.policy.OptionalFailure {
				return nil, err
			}
			out.Report.Degraded = true
			out.Report.DegradationReasons = append(out.Report.DegradationReasons, "SOURCE_UNAVAILABLE:"+r.descriptor.ID)
			continue
		}
		if len(result.Items)+len(out.Report.Decisions) > req.MaxItems {
			return nil, fmt.Errorf("source exceeded item limit")
		}
		if result.ReasonCode != "" {
			out.Report.SourceReasons = append(out.Report.SourceReasons, r.descriptor.ID+":"+result.ReasonCode)
		}
		for _, item := range result.Items {
			item.Scope = r.policy.Scope
			item.Trust = r.policy.Trust
			item.Authority = r.policy.Authority
			item.Sensitivity = r.policy.Sensitivity
			d := CompilationDecision{ItemID: item.ID, SourceID: r.descriptor.ID, SourceVersion: r.descriptor.Version, Scope: item.Scope, Trust: item.Trust, Authority: item.Authority, Sensitivity: item.Sensitivity, CanonicalID: item.CanonicalID, GroupID: item.GroupID, ItemVersion: item.Version}
			identityTenant, identityOwner := req.SessionTenantID, req.SessionOwnerID
			if r.policy.Scope == "user" {
				identityTenant, identityOwner = req.Access.Subject.TenantID, req.Access.Subject.UserID
			}
			valid := item.ID != "" && !seen[r.descriptor.ID+":"+item.ID] && item.SourceID == r.descriptor.ID && item.SourceVersion == r.descriptor.Version && item.TenantID == identityTenant && item.OwnerID == identityOwner && item.SessionID == req.SessionID && item.Version > 0
			seen[r.descriptor.ID+":"+item.ID] = true
			if !valid || (item.ExpiresAt != nil && !item.ExpiresAt.After(req.NotAfter)) || (!item.ObservedAt.IsZero() && item.ObservedAt.After(req.NotAfter)) {
				d.Status = "rejected"
				d.ReasonCode = "ITEM_IDENTITY_OR_STATE_INVALID"
				out.Report.Decisions = append(out.Report.Decisions, d)
				if r.policy.Required {
					return nil, security.ErrResourceNotAccessible
				}
				continue
			}
			// Bound content before rendering, independently of the count-bounded
			// collection envelope. Oversized optional groups become omissions;
			// adding provenance or omission metadata cannot abort ordinary chat.
			content, encodeErr := json.Marshal(item.Events)
			if encodeErr != nil {
				return nil, fmt.Errorf("source returned invalid content")
			}
			if len(content) > req.MaxCollectBytes {
				item.UnavailableReason = "SOURCE_BYTE_LIMIT"
			}
			if item.UnavailableReason != "" {
				if item.UnavailableReason != "SOURCE_BYTE_LIMIT" && item.UnavailableReason != "RECOVERY_LIMIT" && item.UnavailableReason != "MEMORY_CURRENT_INPUT_CONFLICT" {
					return nil, fmt.Errorf("unknown source omission code")
				}
				if r.policy.Required {
					return nil, ErrMandatoryBudget
				}
				d.Status = "omitted"
				d.ReasonCode = item.UnavailableReason
				out.Report.Decisions = append(out.Report.Decisions, d)
				out.Report.Degraded = true
				if r.policy.RecentSuffix {
					candidates = append(candidates, candidate{decision: len(out.Report.Decisions) - 1, sequence: r.descriptor.ID, unavailable: true})
				}
				continue
			}
			messages, trimmed, redacted, err := renderEvents(item.Events, r.policy.Kinds)
			if err != nil {
				return nil, err
			}
			d.EstimatedTokens = Estimate(messages)
			d.Status = "included"
			d.ReasonCode = "SELECTED"
			d.Redacted = redacted
			if redacted {
				d.Status = "redacted"
				d.ReasonCode = "MODEL_INPUT_SANITIZED"
			}
			if trimmed {
				d.Status = "trimmed"
				d.ReasonCode = "STRUCTURED_TOOL_LIMIT"
			}
			index := len(out.Report.Decisions)
			if item.FilteredReason != "" {
				d.Status = "filtered"
				d.ReasonCode = item.FilteredReason
				out.Report.Degraded = true
			}
			out.Report.Decisions = append(out.Report.Decisions, d)
			v := candidate{messages: messages, required: r.policy.Required, tokens: d.EstimatedTokens, bytes: len(JSON(messages)) + 1, decision: index}
			v.dependencies = append([]store.MemoryDependency(nil), item.Dependencies...)
			v.sourceID = r.descriptor.ID
			v.sourceTokenLimit = r.policy.MaxTokens
			if r.policy.History {
				v.historySequence = item.HistorySequence
				v.historyPrevious = item.HistoryPrevious
			}
			if r.policy.HistoryCoverage {
				v.coversThrough = item.CoversThrough
			}
			if r.policy.SummaryDedup {
				v.summarySequence = item.SummarySequence
			}
			if r.policy.RecentSuffix {
				v.sequence = r.descriptor.ID
			}
			candidates = append(candidates, v)
		}
	}
	used := base
	usedBytes := len(JSON(out.Messages)) + len(JSON(tools))
	for _, v := range candidates {
		if v.required {
			used += v.tokens
			usedBytes += v.bytes
		}
	}
	if used > available || usedBytes > req.MaxBytes {
		return nil, ErrMandatoryBudget
	}
	// First try summary + uncovered history. If the summary is excluded, select
	// again over the original bounded history. Existence alone never deduplicates.
	coverage := uint64(0)
	summaryIndex := -1
	for i, v := range candidates {
		if v.coversThrough > 0 {
			if summaryIndex >= 0 {
				return nil, fmt.Errorf("multiple history coverage candidates")
			}
			coverage = v.coversThrough
			summaryIndex = i
		}
	}
	selectCandidates := func(cover uint64, omitSummary bool) (map[int]bool, int, map[int]string) {
		selected := map[int]bool{}
		reasons := map[int]string{}
		blocked := map[string]bool{}
		sourceTokens := map[string]int{}
		tokens, bytes := used, usedBytes
		for i := len(candidates) - 1; i >= 0; i-- {
			v := candidates[i]
			if (v.historySequence > 0 && v.historySequence <= cover) || (v.summarySequence > 0 && v.summarySequence <= cover) {
				reasons[i] = "SUMMARY_COVERED"
				continue
			}
			if v.unavailable {
				blocked[v.sequence] = true
				continue
			}
			if v.required {
				selected[i] = true
				continue
			}
			if omitSummary && v.coversThrough > 0 {
				reasons[i] = "SUMMARY_NOT_SELECTED"
				continue
			}
			if v.sourceTokenLimit > 0 && sourceTokens[v.sourceID]+v.tokens > v.sourceTokenLimit {
				reasons[i] = "SOURCE_TOKEN_LIMIT"
				continue
			}
			if !blocked[v.sequence] && tokens+v.tokens <= available && bytes+v.bytes <= req.MaxBytes {
				tokens += v.tokens
				bytes += v.bytes
				selected[i] = true
				sourceTokens[v.sourceID] += v.tokens
			} else {
				if v.sequence != "" {
					blocked[v.sequence] = true
				}
				reasons[i] = "GROUP_BUDGET_EXCEEDED"
				if bytes+v.bytes > req.MaxBytes {
					reasons[i] = "SOURCE_BYTE_LIMIT"
				}
			}
		}
		return selected, tokens, reasons
	}
	selected, total, reasons := selectCandidates(coverage, false)
	if summaryIndex >= 0 && !selected[summaryIndex] {
		selected, total, reasons = selectCandidates(0, true)
	}
	for i, reason := range reasons {
		d := &out.Report.Decisions[candidates[i].decision]
		d.Status = "omitted"
		d.ReasonCode = reason
		if reason != "SUMMARY_COVERED" {
			out.Report.Degraded = true
		}
	}
	for i, v := range candidates {
		if selected[i] {
			out.Messages = append(out.Messages, v.messages...)
			out.Dependencies = store.MergeMemoryDependencies(out.Dependencies, v.dependencies)
		}
	}
	if req.HistorySnapshot.Upper > 0 {
		out.Report.Summary = &SummaryReport{SnapshotUpper: req.HistorySnapshot.Upper, Target: req.HistorySnapshot.Target, Calls: req.SummaryCalls, Backlog: req.SummaryBacklog}
		if req.Summary != nil {
			out.Report.Summary.AvailableCovered = req.Summary.Covered
			out.Report.Summary.FragmentPending = req.Summary.Progress != nil
		}
		covered := uint64(0)
		if summaryIndex >= 0 && selected[summaryIndex] {
			covered = coverage
			out.Report.Summary.SelectedCovered = coverage
		}
		through := req.HistorySnapshot.Upper
		for i, v := range candidates {
			if selected[i] && v.historySequence > 0 {
				through = v.historyPrevious
				break
			}
		}
		if through > covered {
			out.Report.HistoryOmissions = []HistoryOmission{{After: covered, Through: through, ReasonCode: "BOUNDED_HISTORY_GAP"}}
			out.Report.Degraded = true
			out.Report.DegradationReasons = append(out.Report.DegradationReasons, "BOUNDED_HISTORY_GAP")
		}
	}
	out.Report.InputTokens = total
	if len(out.Dependencies) > 256 {
		return nil, store.ErrResourceExhausted
	}
	return out, nil
}

func renderEvents(events []Event, kinds []string) ([]Message, bool, bool, error) {
	var result []Message
	trimmed := false
	redacted := false
	for _, event := range events {
		allowed := false
		for _, kind := range kinds {
			if kind == event.Kind {
				allowed = true
			}
		}
		if !allowed {
			return nil, false, false, fmt.Errorf("source content kind not permitted")
		}
		switch event.Kind {
		case "user", "model", "legacy", "evidence":
			role := "model"
			if event.Kind == "user" {
				role = "user"
			}
			clean := security.DefaultSanitizer().SanitizeText(event.Text)
			redacted = redacted || len(clean.Findings) > 0
			text := clean.Value
			if event.Kind == "legacy" {
				text = "[Legacy tool evidence: request association unavailable]\n" + text
			}
			if event.Kind == "evidence" {
				text = "[Untrusted context data]\n" + text
			}
			result = append(result, Message{Role: role, Text: text})
		case "tool_batch", "recovered_batch":
			if event.Batch == nil {
				return nil, false, false, fmt.Errorf("missing tool batch")
			}
			b := *event.Batch
			complete := true
			for _, call := range b.Calls {
				if call.Status == "pending" {
					complete = false
				}
			}
			if !complete || event.Kind == "recovered_batch" {
				bounded, changed, sanitized, err := boundedBatchEvidence(b)
				if err != nil {
					return nil, false, false, err
				}
				trimmed = trimmed || changed
				redacted = redacted || sanitized
				clean := security.DefaultSanitizer().SanitizeStructured(bounded)
				redacted = redacted || len(clean.Findings) > 0
				label := "[Incomplete tool batch; execution outcome unknown. Do not assume success or replay.]\n"
				if event.Kind == "recovered_batch" {
					label = "[Recovered tool batch from a released attempt; pending/interrupted execution outcomes are unknown. Do not automatically replay.]\n"
				}
				result = append(result, Message{Role: "model", Text: label + string(JSON(clean.Value))})
				continue
			}
			calls := make([]store.ToolCall, len(b.Calls))
			ids := map[string]bool{}
			for i, call := range b.Calls {
				if call.ID == "" || ids[call.ID] || (call.Status != "succeeded" && call.Status != "failed" && call.Status != "interrupted") {
					return nil, false, false, fmt.Errorf("invalid canonical tool record")
				}
				ids[call.ID] = true
				var input, output any
				if err := json.Unmarshal(call.Input, &input); err != nil {
					return nil, false, false, err
				}
				if err := json.Unmarshal(call.Output, &output); err != nil {
					return nil, false, false, err
				}
				cleanInput := security.DefaultSanitizer().SanitizeStructured(input)
				cleanOutput := security.DefaultSanitizer().SanitizeStructured(output)
				redacted = redacted || len(cleanInput.Findings) > 0 || len(cleanOutput.Findings) > 0
				limitedInput, inputChanged := limitValue(cleanInput.Value, 0)
				limitedOutput, outputChanged := limitValue(cleanOutput.Value, 0)
				changed := inputChanged || outputChanged
				trimmed = trimmed || changed
				call.Input = JSON(limitedInput)
				call.Output = JSON(map[string]any{"status": call.Status, "result": limitedOutput, "error_code": call.ErrorCode, "source_call_id": call.ID, "truncated": changed})
				calls[i] = call
			}
			result = append(result, Message{Role: "model", Requests: calls}, Message{Role: "tool", Responses: calls})
		default:
			return nil, false, false, fmt.Errorf("unknown content kind")
		}
	}
	return result, trimmed, redacted, nil
}

// Bound result data in evidence just as in native tool responses, while keeping
// the original batch/call identities, arguments and uncertainty states intact.
func boundedBatchEvidence(b store.ToolBatch) (store.ToolBatch, bool, bool, error) {
	b.Calls = append([]store.ToolCall(nil), b.Calls...)
	trimmed, redacted := false, false
	for i, call := range b.Calls {
		if len(call.Input) == 0 && len(call.Output) == 0 {
			continue
		}
		var input any
		if len(call.Input) > 0 {
			if err := json.Unmarshal(call.Input, &input); err != nil {
				return b, false, false, err
			}
		}
		var output any
		if len(call.Output) > 0 {
			if err := json.Unmarshal(call.Output, &output); err != nil {
				return b, false, false, err
			}
		}
		cleanInput := security.DefaultSanitizer().SanitizeStructured(input)
		clean := security.DefaultSanitizer().SanitizeStructured(output)
		limitedInput, inputChanged := limitValue(cleanInput.Value, 0)
		limited, changed := limitValue(clean.Value, 0)
		changed = changed || inputChanged
		trimmed = trimmed || changed
		redacted = redacted || len(clean.Findings) > 0 || len(cleanInput.Findings) > 0
		b.Calls[i].Input = JSON(limitedInput)
		b.Calls[i].Output = JSON(map[string]any{"status": call.Status, "result": limited, "error_code": call.ErrorCode, "source_call_id": call.ID, "truncated": changed})
	}
	return b, trimmed, redacted, nil
}

func limitValue(v any, depth int) (any, bool) {
	if depth > 6 {
		return "[depth limit]", true
	}
	switch x := v.(type) {
	case string:
		r := []rune(x)
		if len(r) > 2048 {
			return string(r[:2048]) + "[truncated]", true
		}
	case []any:
		limit := len(x)
		changed := limit > 32
		if limit > 32 {
			limit = 32
		}
		var out []any
		for i, v := range x[:limit] {
			var cut bool
			limited, cut := limitValue(v, depth+1)
			changed = changed || cut
			if cut && out == nil {
				out = append([]any(nil), x[:i]...)
			}
			if out != nil {
				out = append(out, limited)
			}
		}
		if !changed {
			return x, false
		}
		if out == nil {
			out = append([]any(nil), x[:limit]...)
		}
		return out, true
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		changed := len(keys) > 32
		if len(keys) > 32 {
			keys = keys[:32]
		}
		var out map[string]any
		for i, k := range keys {
			v, cut := limitValue(x[k], depth+1)
			changed = changed || cut
			if cut && out == nil {
				out = make(map[string]any, len(keys))
				for _, existing := range keys[:i] {
					out[existing] = x[existing]
				}
			}
			if out != nil {
				out[k] = v
			}
		}
		if !changed {
			return x, false
		}
		if out == nil {
			out = make(map[string]any, len(keys))
			for _, k := range keys {
				out[k] = x[k]
			}
		}
		return out, true
	}
	return v, false
}
