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
	batch := store.ToolBatch{ID: uuid.NewString(), TenantID: state.access.Subject.TenantID, OwnerID: state.access.Subject.UserID, SessionID: s.Session, TurnID: state.turnID, InteractionID: state.interactionID, Sequence: uint64(s.Iteration), Version: 1, SchemaVersion: "v1"}
	batch.ModelCallID = s.ModelCallID
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
		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), persistenceTimeout)
		writeErr := ra.compilationStore.CompleteToolCall(persistCtx, state.access, s.Session, batch.ID, batch.Version, i, call)
		cancel()
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
