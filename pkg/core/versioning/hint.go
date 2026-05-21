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
	"sync"
	"time"

	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type AdminHint struct {
	RuleKind         coremodel.ResourceKind
	Mesh             string
	ResourceKey      string
	RuleName         string
	ContentHash      string
	SpecJSON         string
	Source           Source
	Author           string
	Reason           string
	Operation        Operation
	RolledBackFromID *int64
	ExpiresAt        time.Time
}

type hintKey struct {
	kind        coremodel.ResourceKind
	resourceKey string
	hash        string
}

type AdminHintRegistry struct {
	mu    sync.Mutex
	now   func() time.Time
	hints map[hintKey]AdminHint
}

func NewAdminHintRegistry() *AdminHintRegistry {
	return &AdminHintRegistry{
		now:   time.Now,
		hints: make(map[hintKey]AdminHint),
	}
}

func (r *AdminHintRegistry) Put(kind coremodel.ResourceKind, resourceKey, contentHash string, hint AdminHint) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hints[hintKey{kind: kind, resourceKey: resourceKey, hash: contentHash}] = hint
}

func (r *AdminHintRegistry) Take(kind coremodel.ResourceKind, resourceKey, contentHash string) (AdminHint, bool) {
	if r == nil {
		return AdminHint{}, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := hintKey{kind: kind, resourceKey: resourceKey, hash: contentHash}
	hint, ok := r.hints[key]
	if !ok {
		return AdminHint{}, false
	}
	delete(r.hints, key)
	if !hint.ExpiresAt.IsZero() && r.now().After(hint.ExpiresAt) {
		return AdminHint{}, false
	}
	return hint, true
}

func (r *AdminHintRegistry) pruneExpiredLocked() {
	now := r.now()
	for key, hint := range r.hints {
		if !hint.ExpiresAt.IsZero() && now.After(hint.ExpiresAt) {
			delete(r.hints, key)
		}
	}
}
