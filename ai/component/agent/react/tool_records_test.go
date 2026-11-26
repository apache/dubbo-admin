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

	"dubbo-admin-ai/store"
)

func TestBoundToolOutputKeepsEncodedBatchBounded(t *testing.T) {
	batch := store.ToolBatch{ID: "batch", SessionID: "session", TurnID: 1, TenantID: "tenant", OwnerID: "owner", InteractionID: "interaction", Sequence: 1, Version: 1, SchemaVersion: "v1", Calls: []store.ToolCall{{ID: "call", Name: "lookup", Input: json.RawMessage(`{}`), Status: "pending"}}}

	small := batch.Calls[0]
	small.Status = "succeeded"
	small.Output = json.RawMessage(`{"ok":true}`)
	if got, truncated := boundToolOutput(batch, 0, small); truncated || string(got.Output) != `{"ok":true}` {
		t.Fatalf("fitting result was modified: %+v", got)
	}

	call := batch.Calls[0]
	call.Status = "succeeded"
	call.Output = json.RawMessage(`"` + strings.Repeat("x", 2<<20) + `"`)
	call, truncated := boundToolOutput(batch, 0, call)
	if !truncated || call.ErrorCode != "TOOL_OUTPUT_TRUNCATED" || !json.Valid(call.Output) {
		t.Fatalf("oversized result was not degraded: %+v", call)
	}
	probe := batch
	probe.Calls = append([]store.ToolCall(nil), batch.Calls...)
	probe.Calls[0] = call
	encoded, err := json.Marshal(probe)
	if err != nil || len(encoded) > store.MaxToolBatchBytes {
		t.Fatalf("bounded batch = %d bytes, err %v", len(encoded), err)
	}
	var envelope map[string]any
	if err := json.Unmarshal(call.Output, &envelope); err != nil || envelope["truncated"] != true {
		t.Fatalf("missing truncation evidence: %s", call.Output)
	}
}
