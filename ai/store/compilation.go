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

package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"dubbo-admin-ai/security"
	"github.com/firebase/genkit/go/ai"
)

var ErrContextConflict = errors.New("context version conflict")

const MaxToolBatchBytes = 1 << 20

// TurnContext preserves the storage boundary; callers must not infer it from roles.
type TurnContext struct {
	PreviousSequence uint64
	Sequence         uint64
	Dependencies     []MemoryDependency
	ByteLimited      bool
	TenantID         string
	OwnerID          string
	ID               uint64
	Status           string
	Messages         []*ai.Message
	Batches          []ToolBatch
}

// ToolBatch is a versioned canonical execution ledger, independent of SDK messages.
// It survives failed-turn cleanup and is removed with its source Session.
type ToolBatch struct {
	ModelCallID   string
	ID            string
	TenantID      string
	OwnerID       string
	SessionID     string
	TurnID        uint64
	InteractionID string
	Sequence      uint64
	Version       uint64
	SchemaVersion string
	Calls         []ToolCall
}

type ToolCall struct {
	ID        string
	Name      string
	Input     json.RawMessage
	Output    json.RawMessage
	Status    string
	ErrorCode string
}

// ToolRecovery contains only batches whose source Turn has been released.
// Omitted signals a bounded window or byte truncation, never absent execution.
type ToolRecovery struct {
	Batches []ToolBatch
	Omitted bool
	// TerminalSequences is server-owned journal provenance, keyed by Turn ID.
	// Missing positions (legacy records) must never be inferred from batch order.
	TerminalSequences map[uint64]uint64
}

// CompilationStore is deliberately separate from the legacy MessageStore API.
type CompilationStore interface {
	// RecoveryToolBatches returns a recent batch suffix from released attempts,
	// excluding completed/current Turns. active fixes the upper Turn boundary.
	// Limits apply to batch payload bytes; Omitted is separate bounded metadata.
	RecoveryToolBatches(context.Context, security.AccessContext, string, uint64, int, int) (ToolRecovery, error)
	// ContextTurns caps loaded payloads; up to limit+1 Turn envelopes/omissions
	// are separately count-bounded. Compiler budgets rendered messages, not these
	// storage envelopes or its compilation report.
	ContextTurns(context.Context, security.AccessContext, string, uint64, int, int) ([]TurnContext, error)
	PinTemplate(context.Context, security.AccessContext, string, string) (string, error)
	CreateToolBatch(context.Context, security.AccessContext, ToolBatch) error
	CompleteToolCall(context.Context, security.AccessContext, string, string, uint64, int, ToolCall) error
	ToolBatches(context.Context, security.AccessContext, string, uint64, int) ([]ToolBatch, error)
}

func ValidateToolBatch(access security.AccessContext, b ToolBatch) error {
	if b.ID == "" || b.SessionID == "" || b.TurnID == 0 || b.Sequence == 0 || b.Version != 1 || b.SchemaVersion != "v1" || len(b.Calls) == 0 || len(b.Calls) > 64 || b.TenantID != access.Subject.TenantID || b.OwnerID != access.Subject.UserID {
		return fmt.Errorf("invalid tool batch")
	}
	seen := map[string]bool{}
	for _, c := range b.Calls {
		if c.ID == "" || c.Name == "" || c.Status != "pending" || !json.Valid(c.Input) || len(c.Output) != 0 || seen[c.ID] {
			return fmt.Errorf("invalid pending tool call")
		}
		seen[c.ID] = true
	}
	return nil
}

func ValidateTemplateRef(ref string) error {
	if len(ref) == 0 || len(ref) > 256 {
		return fmt.Errorf("invalid template reference")
	}
	return nil
}

func UpdateToolCall(b *ToolBatch, expected uint64, index int, call ToolCall) error {
	if b.Version != expected || index < 0 || index >= len(b.Calls) {
		return ErrContextConflict
	}
	old := b.Calls[index]
	var oldJSON, newJSON bytes.Buffer
	oldErr := json.Compact(&oldJSON, old.Input)
	newErr := json.Compact(&newJSON, call.Input)
	if old.Status != "pending" || old.ID != call.ID || old.Name != call.Name || oldErr != nil || newErr != nil || !bytes.Equal(oldJSON.Bytes(), newJSON.Bytes()) {
		return ErrContextConflict
	}
	if call.Status != "succeeded" && call.Status != "failed" && call.Status != "interrupted" {
		return fmt.Errorf("invalid terminal tool status")
	}
	if !json.Valid(call.Output) {
		return fmt.Errorf("invalid tool output")
	}
	call.Input = old.Input
	b.Calls[index] = call
	b.Version++
	return nil
}
