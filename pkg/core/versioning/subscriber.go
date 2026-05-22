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

package versioning

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type Subscriber struct {
	kind           coremodel.ResourceKind
	store          Store
	hints          *AdminHintRegistry
	maxVersions    int64
	coalesceWindow time.Duration
	mu             sync.Mutex
	pending        map[string]*pendingEvent
}

type pendingEvent struct {
	event events.Event
	timer *time.Timer
}

func NewSubscriber(kind coremodel.ResourceKind, store Store, hints *AdminHintRegistry, maxVersions int64, coalesceWindow time.Duration) *Subscriber {
	return &Subscriber{
		kind:           kind,
		store:          store,
		hints:          hints,
		maxVersions:    maxVersions,
		coalesceWindow: coalesceWindow,
		pending:        make(map[string]*pendingEvent),
	}
}

func (s *Subscriber) ResourceKind() coremodel.ResourceKind {
	return s.kind
}

func (s *Subscriber) Name() string {
	return "rule-version-" + s.kind.ToString()
}

func (s *Subscriber) AsyncEnabled() bool {
	return true
}

func (s *Subscriber) ProcessEvent(event events.Event) error {
	res := event.NewObj()
	if event.Type() == cache.Deleted {
		res = event.OldObj()
	}
	if res == nil {
		return nil
	}
	if s.coalesceWindow <= 0 {
		return s.record(event)
	}
	key := res.ResourceKey()
	s.mu.Lock()
	if p := s.pending[key]; p != nil {
		p.event = event
		s.mu.Unlock()
		return nil
	}
	p := &pendingEvent{event: event}
	p.timer = time.AfterFunc(s.coalesceWindow, func() {
		s.flush(key)
	})
	s.pending[key] = p
	s.mu.Unlock()
	return nil
}

func (s *Subscriber) FlushAll() {
	s.mu.Lock()
	keys := make([]string, 0, len(s.pending))
	for key := range s.pending {
		keys = append(keys, key)
	}
	s.mu.Unlock()
	for _, key := range keys {
		s.flush(key)
	}
}

func (s *Subscriber) flush(key string) {
	s.mu.Lock()
	p := s.pending[key]
	if p == nil {
		s.mu.Unlock()
		return
	}
	delete(s.pending, key)
	if p.timer != nil {
		p.timer.Stop()
	}
	event := p.event
	s.mu.Unlock()
	if err := s.record(event); err != nil {
		logger.Errorf("failed to record rule version for %s: %v", key, err)
	}
}

func (s *Subscriber) record(event events.Event) error {
	res := event.NewObj()
	op := OperationUpdate
	switch event.Type() {
	case cache.Added:
		op = OperationCreate
	case cache.Updated:
		op = OperationUpdate
	case cache.Deleted:
		op = OperationDelete
		res = event.OldObj()
	default:
		return nil
	}
	if res == nil {
		return nil
	}
	hash, specJSON, err := NormalizeResource(res)
	if op == OperationDelete {
		hash = HashSpecJSON(DeleteSpecJSON)
		specJSON = DeleteSpecJSON
		err = nil
	}
	if err != nil {
		return err
	}
	ruleKind := res.ResourceKind()
	mesh := res.ResourceMesh()
	resourceKey := res.ResourceKey()
	ruleName := res.ResourceMeta().Name
	source := SourceUpstream
	author := "system:upstream"
	reason := ""
	var rolledBackFromID *int64
	if ctx := event.Context(); ctx != nil {
		if registry := ctx[events.SourceRegistryContextKey]; registry != "" {
			author = "system:" + registry
		}
	}
	if hint, ok := s.hints.Take(res.ResourceKind(), res.ResourceKey(), hash); ok {
		source = hint.Source
		if source == "" {
			source = SourceAdmin
		}
		author = hint.Author
		reason = hint.Reason
		if hint.Operation != "" {
			op = hint.Operation
		}
		rolledBackFromID = hint.RolledBackFromID
		if op == OperationDelete {
			if hint.RuleKind != "" {
				ruleKind = hint.RuleKind
			}
			if hint.Mesh != "" {
				mesh = hint.Mesh
			}
			if hint.ResourceKey != "" {
				resourceKey = hint.ResourceKey
			}
			if hint.RuleName != "" {
				ruleName = hint.RuleName
			}
		}
	}
	if author == "" {
		author = "system:unknown"
	}
	_, err = s.store.InsertVersion(InsertRequest{
		RuleKind:         ruleKind,
		Mesh:             mesh,
		ResourceKey:      resourceKey,
		RuleName:         ruleName,
		SpecJSON:         specJSON,
		ContentHash:      hash,
		Source:           source,
		Operation:        op,
		Author:           author,
		Reason:           reason,
		RolledBackFromID: rolledBackFromID,
		CreatedAt:        time.Now(),
	}, s.maxVersions)
	return err
}

func RecordBootstrap(store Store, maxVersions int64, res coremodel.Resource) error {
	meta, err := store.CurrentMeta(res.ResourceKind(), res.ResourceKey())
	if err != nil {
		return err
	}
	if meta != nil {
		return nil
	}
	hash, specJSON, err := NormalizeResource(res)
	if err != nil {
		return err
	}
	_, err = store.InsertVersion(InsertRequest{
		RuleKind:    res.ResourceKind(),
		Mesh:        res.ResourceMesh(),
		ResourceKey: res.ResourceKey(),
		RuleName:    res.ResourceMeta().Name,
		SpecJSON:    specJSON,
		ContentHash: hash,
		Source:      SourceBootstrap,
		Operation:   OperationCreate,
		Author:      "system:bootstrap",
		CreatedAt:   time.Now(),
	}, maxVersions)
	if err != nil {
		return fmt.Errorf("bootstrap version for %s failed: %w", res.ResourceKey(), err)
	}
	return nil
}
