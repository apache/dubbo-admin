/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with this
 * work for additional information regarding copyright ownership.
 * The ASF licenses this file under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with it.
 * You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by law or agreed in writing, software is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
 */
package react

import (
	"context"
	comp "dubbo-admin-ai/component/context"
	"dubbo-admin-ai/store"
	"errors"
	"fmt"
)

type MemoryConfig struct {
	Enabled   bool `json:"enabled" yaml:"enabled"`
	MaxItems  int  `json:"max_items" yaml:"max_items"`
	MaxBytes  int  `json:"max_bytes" yaml:"max_bytes"`
	MaxTokens int  `json:"max_tokens" yaml:"max_tokens"`
}

func (c *MemoryConfig) Defaults() {
	if c.MaxItems == 0 {
		c.MaxItems = 16
	}
	if c.MaxBytes == 0 {
		c.MaxBytes = 16384
	}
	if c.MaxTokens == 0 {
		c.MaxTokens = 4096
	}
}
func (c MemoryConfig) Validate() error {
	c.Defaults()
	if c.MaxItems < 1 || c.MaxItems > 128 || c.MaxBytes < 256 || c.MaxBytes > 2<<20 || c.MaxTokens < 64 || c.MaxTokens > 32768 {
		return fmt.Errorf("invalid explicit_memory limits")
	}
	return nil
}

func (ra *ReActAgent) initMemory(spec *AgentSpec) error {
	backend, ok := ra.messageStore.(store.ExplicitMemoryStore)
	if !ok {
		if spec.ExplicitMemory.Enabled {
			return fmt.Errorf("store does not support explicit Memory")
		}
		return nil
	}
	ra.memoryStore = backend
	cfg := spec.ExplicitMemory
	cfg.Defaults()
	return ra.compiler.Register(comp.MemorySource{Store: backend, MaxItems: cfg.MaxItems, MaxBytes: cfg.MaxBytes}, comp.SourcePolicy{Scope: "user", Order: 4, MaxTokens: cfg.MaxTokens, OptionalFailure: true, Kinds: []string{"evidence"}})
}
func (ra *ReActAgent) filterMemoryHistory(ctx context.Context, state *interactionState, id string) ([]store.TurnContext, error) {
	turns := append([]store.TurnContext(nil), state.turns...)
	if ra.memoryStore == nil {
		return turns, nil
	}
	for i, t := range turns {
		if len(t.Dependencies) == 0 {
			continue
		}
		err := ra.memoryStore.ValidateMemoryDependencies(ctx, state.access, id, t.Dependencies)
		if errors.Is(err, store.ErrMemoryDependency) {
			turns[i] = store.FilterDependentTurn(t)
		} else if err != nil {
			return nil, err
		}
	}
	return turns, nil
}

// filterMemoryRecovery removes recovered tool batches whose Memory provenance
// is no longer valid. Recovery is an ephemeral view just like history, so this
// must be repeated whenever a compile is retried after concurrent invalidation.
func (ra *ReActAgent) filterMemoryRecovery(ctx context.Context, state *interactionState, id string, recovery store.ToolRecovery) (store.ToolRecovery, error) {
	if ra.memoryStore == nil {
		return recovery, nil
	}
	kept := make([]store.ToolBatch, 0, len(recovery.Batches))
	for _, batch := range recovery.Batches {
		err := ra.memoryStore.ValidateMemoryDependencies(ctx, state.access, id, batch.Dependencies)
		if errors.Is(err, store.ErrMemoryDependency) {
			recovery.Omitted = true
			continue
		}
		if err != nil {
			return store.ToolRecovery{}, err
		}
		kept = append(kept, batch)
	}
	recovery.Batches = kept
	return recovery, nil
}
