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
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"dubbo-admin-ai/store"
	"github.com/firebase/genkit/go/ai"
	"github.com/google/uuid"
)

func (ra *ReActAgent) execCanonicalTools(ctx context.Context, state *interactionState, s *interactionTrace, reqs []*ai.ToolRequest) (err error) {
	batch := store.ToolBatch{ID: uuid.NewString(), TenantID: state.sessionTenant, OwnerID: state.sessionOwner, SessionID: s.Session, TurnID: state.turnID, InteractionID: state.interactionID, Sequence: uint64(s.Iteration), Version: 1, SchemaVersion: "v1"}
	batch.ModelCallID = s.ModelCallID
	batch.Dependencies = append([]store.MemoryDependency(nil), state.usedMemory...)
	seen := map[string]bool{}
	for _, req := range reqs {
		if req == nil || !state.allowedTools[req.Name] {
			return fmt.Errorf("TOOL_NOT_AUTHORIZED")
		}
		if req.Ref == "" {
			return fmt.Errorf("TOOL_CALL_ID_MISSING")
		}
		if seen[req.Ref] {
			return fmt.Errorf("TOOL_CALL_ID_DUPLICATE")
		}
		seen[req.Ref] = true
		data, err := json.Marshal(req.Input)
		if err != nil {
			return err
		}
		batch.Calls = append(batch.Calls, store.ToolCall{ID: req.Ref, Name: req.Name, Input: data, Status: "pending"})
	}
	// All calls must be durable before the first potentially side-effecting tool.
	if err := ra.compilationStore.CreateToolBatch(ctx, state.access, batch); err != nil {
		return err
	}
	defer func() {
		if err == nil {
			return
		}
		// Record cancellation and partial completion with a fresh bounded context.
		// If persistence is unavailable, pending remains an uncertain outcome.
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), persistenceTimeout)
		defer cancel()
		for i, call := range batch.Calls {
			if call.Status != "pending" {
				continue
			}
			call.Status = "interrupted"
			call.Output = json.RawMessage(`null`)
			call.ErrorCode = "EXECUTION_INTERRUPTED"
			if e := ra.compilationStore.CompleteToolCall(cleanup, state.access, s.Session, batch.ID, batch.Version, i, call); e != nil {
				break
			}
			batch.Calls[i] = call
			batch.Version++
		}
	}()
	for i, req := range reqs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := ra.messageStore.IsTurnEmpty(ctx, state.access, s.Session, state.turnID); err != nil {
			return err
		}
		output, callErr := func() (result any, callErr error) {
			toolCtx, cancel := withTimeout(ctx, ra.toolTimeouts.For(req.Name))
			defer cancel()
			// Convert panics into a terminal failure without losing the ledger.
			defer func() {
				if recover() != nil {
					callErr = fmt.Errorf("tool panicked")
				}
			}()
			return ra.executeToolCall(toolCtx, s, req)
		}()
		call := batch.Calls[i]
		call.Status = "succeeded"
		if callErr != nil {
			call.Status = "failed"
			call.ErrorCode = "TOOL_EXECUTION_FAILED"
			s.Degraded = true
		}
		if errors.Is(callErr, context.Canceled) || ctx.Err() != nil {
			call.Status = "interrupted"
			call.ErrorCode = "EXECUTION_INTERRUPTED"
		}
		data, marshalErr := json.Marshal(output)
		if marshalErr != nil {
			return marshalErr
		}
		call.Output = data
		var truncated bool
		call, truncated = boundToolOutput(batch, i, call)
		if truncated {
			s.Degraded = true
		}
		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), persistenceTimeout)
		writeErr := ra.compilationStore.CompleteToolCall(persistCtx, state.access, s.Session, batch.ID, batch.Version, i, call)
		cancel()
		if errors.Is(writeErr, store.ErrResourceExhausted) {
			// The tool already ran; narrow the evidence and retry once so a
			// quota failure cannot discard the whole interaction.
			call = shrinkToolOutput(call)
			s.Degraded = true
			persistCtx, cancel = context.WithTimeout(context.WithoutCancel(ctx), persistenceTimeout)
			writeErr = ra.compilationStore.CompleteToolCall(persistCtx, state.access, s.Session, batch.ID, batch.Version, i, call)
			cancel()
		}
		if writeErr != nil {
			return writeErr
		}
		batch.Calls[i] = call
		batch.Version++
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	for i := range state.turns {
		if state.turns[i].ID == state.turnID {
			state.turns[i].Batches = append(state.turns[i].Batches, batch)
			return nil
		}
	}
	return fmt.Errorf("active context turn missing")
}

// toolBatchEncodingReserve covers small growth such as the batch version digit
// count so a bounded result does not race the store's exact payload check.
const toolBatchEncodingReserve = 32

// boundToolOutput keeps the complete encoded batch within the store ceiling.
// Tool execution already happened, so an oversized result degrades to explicit
// truncated evidence instead of aborting the interaction.
func boundToolOutput(batch store.ToolBatch, index int, call store.ToolCall) (store.ToolCall, bool) {
	probe := batch
	probe.Calls = append([]store.ToolCall(nil), batch.Calls...)
	probe.Calls[index] = call
	if encoded, err := json.Marshal(probe); err == nil && len(encoded)+toolBatchEncodingReserve <= store.MaxToolBatchBytes {
		return call, false
	}
	original := len(call.Output)
	// Measure the batch with the result replaced by a small placeholder, then
	// fit a JSON envelope into the remaining room.
	probe.Calls[index].Output = json.RawMessage(`""`)
	base, err := json.Marshal(probe)
	if err != nil {
		return shrinkToolOutput(call), true
	}
	budget := store.MaxToolBatchBytes - toolBatchEncodingReserve
	room := budget - (len(base) - len(`""`))
	preview := string(call.Output)
	if room < 0 {
		room = 0
	}
	if len(preview) > room {
		preview = preview[:room]
	}
	for len(preview) > 0 {
		envelope, encodeErr := json.Marshal(map[string]any{
			"truncated":      true,
			"reason":         "TOOL_OUTPUT_TOO_LARGE",
			"original_bytes": original,
			"preview":        preview,
		})
		if encodeErr != nil {
			break
		}
		if len(base)-len(`""`)+len(envelope) <= budget {
			call.Output = envelope
			call.ErrorCode = "TOOL_OUTPUT_TRUNCATED"
			return call, true
		}
		preview = preview[:len(preview)/2]
	}
	marker := json.RawMessage(fmt.Sprintf(`{"truncated":true,"reason":"TOOL_OUTPUT_TOO_LARGE","original_bytes":%d}`, original))
	if len(base)-len(`""`)+len(marker) <= budget {
		call.Output = marker
		call.ErrorCode = "TOOL_OUTPUT_TRUNCATED"
		return call, true
	}
	call = shrinkToolOutput(call)
	probe.Calls[index] = call
	if encoded, err := json.Marshal(probe); err == nil && len(encoded) <= store.MaxToolBatchBytes {
		return call, true
	}
	// An empty object with no error code is not larger than the pending null
	// result for a successful call, so the terminal update still fits; the
	// degraded turn keeps the truncation observable. CompleteToolCall remains
	// the final arbiter for unusual status/version combinations.
	call.Output = json.RawMessage(`{}`)
	call.ErrorCode = ""
	return call, true
}

func shrinkToolOutput(call store.ToolCall) store.ToolCall {
	call.Output = json.RawMessage(`{"truncated":true}`)
	call.ErrorCode = "TOOL_OUTPUT_TRUNCATED"
	return call
}
