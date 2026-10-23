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

	"dubbo-admin-ai/store"
)

// SnapshotSource reads the authorized, bounded snapshot fixed by the Agent at
// interaction start. Active tool batches are updated only after durable writes.
type SnapshotSource struct{ Kind string }

func (s SnapshotSource) Descriptor() SourceDescriptor {
	return SourceDescriptor{ID: s.Kind, Version: "v1"}
}
func (s SnapshotSource) Collect(ctx context.Context, req CollectRequest) (CollectResult, error) {
	var result CollectResult
	for _, turn := range req.Turns {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		active := turn.ID == req.TurnID
		if (s.Kind == "history" && active) || (s.Kind != "history" && !active) {
			continue
		}
		item := ContextItem{ID: fmt.Sprintf("%s:%d", s.Kind, turn.ID), SourceID: s.Kind, SourceVersion: "v1", TenantID: turn.TenantID, OwnerID: turn.OwnerID, SessionID: req.SessionID, CanonicalID: fmt.Sprintf("turn:%s:%d", req.SessionID, turn.ID), GroupID: fmt.Sprintf("turn:%d", turn.ID), Version: 1}
		item.HistorySequence = turn.Sequence
		item.HistoryPrevious = turn.PreviousSequence
		if turn.ByteLimited {
			item.UnavailableReason = "SOURCE_BYTE_LIMIT"
			result.Items = append(result.Items, item)
			continue
		}
		if s.Kind != "tools" {
			for _, message := range turn.Messages {
				if message == nil {
					continue
				}
				kind := ""
				switch string(message.Role) {
				case "user":
					kind = "user"
				case "model":
					kind = "model"
				default:
					return result, fmt.Errorf("unsupported canonical history role")
				}
				for _, part := range message.Content {
					if part == nil {
						continue
					}
					if part.ToolRequest != nil || part.ToolResponse != nil {
						return result, fmt.Errorf("SDK tool messages require canonical batch migration")
					}
					if !part.IsText() {
						return result, fmt.Errorf("unsupported canonical content part")
					}
					if part.Text != "" {
						k := kind
						if kind == "model" && (part.ContentType == "application/json" || legacyToolEvidence(part.Text)) {
							k = "legacy"
						}
						item.Events = append(item.Events, Event{Kind: k, Text: part.Text})
					}
				}
			}
		}
		if s.Kind != "input" {
			for i := range turn.Batches {
				batch := turn.Batches[i]
				if err := verifyBatch(req, turn, batch); err != nil {
					return result, err
				}
				item.Events = append(item.Events, Event{Kind: "tool_batch", Batch: &batch})
				item.Version += batch.Version
			}
		}
		// New canonical batches occur after the user input and before the final
		// answer. Legacy evidence retains its original message ordering.
		if s.Kind == "history" && len(turn.Batches) > 0 {
			var users, answers, batches []Event
			for _, e := range item.Events {
				switch e.Kind {
				case "tool_batch":
					batches = append(batches, e)
				case "model":
					answers = append(answers, e)
				default:
					users = append(users, e)
				}
			}
			item.Events = append(append(users, batches...), answers...)
		}
		if len(item.Events) > 0 {
			result.Items = append(result.Items, item)
		}
	}
	if s.Kind == "input" && len(result.Items) != 1 {
		return result, fmt.Errorf("current input snapshot missing")
	}
	return result, nil
}

// Genkit's message JSON codec does not retain NewJSONPart's ContentType.
// Recognize only the historical result envelope; never reconstruct requests.
func legacyToolEvidence(text string) bool {
	var value map[string]json.RawMessage
	if json.Unmarshal([]byte(text), &value) != nil {
		return false
	}
	var name string
	if json.Unmarshal(value["tool_name"], &name) != nil || name == "" {
		return false
	}
	_, result := value["result"]
	_, summary := value["summary"]
	return result || summary
}

func verifyBatch(req CollectRequest, turn store.TurnContext, b store.ToolBatch) error {
	if b.SessionID != req.SessionID || b.TurnID != turn.ID || b.TenantID != req.Access.Subject.TenantID || b.OwnerID != req.Access.Subject.UserID || b.SchemaVersion != "v1" || b.Version == 0 {
		return fmt.Errorf("invalid tool batch provenance")
	}
	return nil
}

func RegisterCoreSources(c *Compiler) error {
	if err := c.Register(RecoverySource{Notice: true}, SourcePolicy{Order: 11, Required: true, Kinds: []string{"evidence"}}); err != nil {
		return err
	}
	if err := c.Register(RecoverySource{}, SourcePolicy{Order: 12, SummaryDedup: true, RecentSuffix: true, Kinds: []string{"recovered_batch"}}); err != nil {
		return err
	}
	for _, v := range []struct {
		kind string
		p    SourcePolicy
	}{
		{"history", SourcePolicy{Order: 10, History: true, RecentSuffix: true, Kinds: []string{"user", "model", "legacy", "tool_batch"}}},
		{"input", SourcePolicy{Order: 20, Required: true, Kinds: []string{"user", "model", "legacy"}}},
		{"tools", SourcePolicy{Order: 30, Required: true, Kinds: []string{"tool_batch"}}},
	} {
		if err := c.Register(SnapshotSource{v.kind}, v.p); err != nil {
			return err
		}
	}
	return nil
}

// RecoverySource labels released attempts as evidence, never as new tool calls.
// A mandatory notice survives even when optional result details cannot fit.
type RecoverySource struct{ Notice bool }

func (s RecoverySource) Descriptor() SourceDescriptor {
	id := "recovery"
	if s.Notice {
		id = "recovery_notice"
	}
	return SourceDescriptor{ID: id, Version: "v1"}
}

func (s RecoverySource) Collect(ctx context.Context, req CollectRequest) (CollectResult, error) {
	var result CollectResult
	if len(req.Recovery.Batches) == 0 && !req.Recovery.Omitted {
		return result, nil
	}
	makeItem := func(id, text string) ContextItem {
		return ContextItem{ID: id, SourceID: s.Descriptor().ID, SourceVersion: "v1", TenantID: req.Access.Subject.TenantID, OwnerID: req.Access.Subject.UserID, SessionID: req.SessionID, Version: 1, Events: []Event{{Kind: "evidence", Text: text}}}
	}
	if s.Notice {
		result.Items = append(result.Items, makeItem("notice", "Previous released attempts have tool execution records. Successful calls may already have effects; pending/interrupted calls have unknown execution outcomes. Do not assume they did not execute or automatically replay them. Recovery details are bounded and may be omitted; absence of details is not evidence of non-execution."))
		if req.Recovery.Omitted {
			result.Items[0].Events[0].Text += " Some recovery records were omitted by the storage window or byte limit."
		}
		return result, nil
	}
	if req.Recovery.Omitted {
		item := makeItem("omitted", "")
		item.Events = nil
		item.UnavailableReason = "RECOVERY_LIMIT"
		result.Items = append(result.Items, item)
	}
	for _, b := range req.Recovery.Batches {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if err := verifyBatch(req, store.TurnContext{ID: b.TurnID}, b); err != nil {
			return result, err
		}
		item := makeItem(b.ID, "")
		item.Events = []Event{{Kind: "recovered_batch", Batch: &b}}
		item.CanonicalID = "batch:" + b.ID
		item.GroupID = item.CanonicalID
		item.Version = b.Version
		item.SummarySequence = req.Recovery.TerminalSequences[b.TurnID]
		result.Items = append(result.Items, item)
	}
	return result, nil
}
