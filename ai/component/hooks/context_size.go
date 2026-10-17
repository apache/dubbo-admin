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

package hooks

import "sync"

// ContextSize describes only the selected conversation messages, including tool
// payloads. It excludes the prompt template, tool definitions and provider-specific
// framing. EstimatedTokens is a heuristic, not provider usage or a quota budget.
// Negative JSONBytes or EstimatedTokens indicate that measurement is unavailable.
type ContextSize struct {
	Messages        int
	JSONBytes       int
	EstimatedTokens int
}

type lazyContextSize struct {
	once     sync.Once
	provider func() ContextSize
	size     ContextSize
}

// WithContextSize attaches numeric metadata that is computed only on demand.
// A hook can inspect it without opting into payload capture.
func (s State) WithContextSize(provider func() ContextSize) State {
	if provider != nil {
		s.contextSize = &lazyContextSize{provider: provider}
	}
	return s
}

// ContextSize returns the selected-context measurement if one was supplied.
// Copies of one event share a single calculation, including across observers.
func (s State) ContextSize() (ContextSize, bool) {
	if s.contextSize == nil {
		return ContextSize{}, false
	}
	s.contextSize.once.Do(func() {
		s.contextSize.size = s.contextSize.provider()
		s.contextSize.provider = nil
	})
	return s.contextSize.size, true
}
