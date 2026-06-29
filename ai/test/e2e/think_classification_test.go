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

package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"dubbo-admin-ai/component/agent/react"
	appruntime "dubbo-admin-ai/runtime"
)

// TestThinkClassificationIsolated validates the think stage's intent
// classification for documentation-grade questions in ISOLATION: each question
// runs in its own fresh session with no prior conversation history, so the
// result reflects only the prompt's classification ability — free of the
// multi-turn bias (a long shared session nudges the model toward MEMORY_SEARCH)
// and the API stalls that the full multi-turn HTTP/SSE test carries.
//
// It is fast (only the think stage, no act/observe/tool calls) and, by running
// each question several times, surfaces nondeterminism as a distribution rather
// than a single flaky pass/fail.
func TestThinkClassificationIsolated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping classification e2e test in short mode")
	}

	configPath, _ := createTestConfig(t)
	rt, err := appruntime.Bootstrap(configPath, registerFactories)
	if err != nil {
		t.Fatalf("Failed to bootstrap runtime: %v", err)
	}
	defer rt.StopAll()

	agentComp, err := rt.GetComponent("agent")
	if err != nil {
		t.Fatalf("Failed to get agent component: %v", err)
	}
	ac, ok := agentComp.(*react.AgentComponent)
	if !ok {
		t.Fatalf("agent component is not *react.AgentComponent, got %T", agentComp)
	}
	if ac.Agent == nil {
		t.Fatalf("agent component has no ReActAgent")
	}

	// Documentation-grade questions whose answers live in the seeded knowledge
	// base (component/rag/seeds/*.md). The think stage SHOULD classify these as
	// DOCUMENTATION_QUERY and route them to query_knowledge_base — not memory.
	cases := []struct {
		name     string
		question string
	}{
		{"ProviderConfigKeys", "What are the exact dubbo.provider config keys and their default values for timeout, retries and loadbalance?"},
		{"SerializationOptions", "Which serialization protocols does Dubbo support and which one is the default?"},
		{"RegisterModeDefault", "In Dubbo 3, what is the default register-mode and what values can it take?"},
		{"AdminConnectionConfig", "Which addresses must Dubbo Admin be configured with to manage a cluster, and what are the config keys?"},
	}

	const runs = 3                  // repeat each question to expose nondeterminism
	const wantIntent = "DOCUMENTATION_QUERY"

	type tally struct {
		intentCounts map[string]int
		ragHits      int // times suggested_tools contained query_knowledge_base
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			tl := tally{intentCounts: map[string]int{}}
			for i := 0; i < runs; i++ {
				// Unique fresh session per run -> no shared history, no bias.
				sessionID := fmt.Sprintf("think-iso-%s-%d", c.name, i)
				start := time.Now()
				out, err := ac.Agent.ThinkOnce(c.question, sessionID)
				elapsed := time.Since(start)
				if err != nil {
					t.Errorf("run %d: ThinkOnce error: %v", i, err)
					continue
				}
				tl.intentCounts[string(out.Intent)]++
				if containsTool(out.SuggestedTools, "query_knowledge_base") {
					tl.ragHits++
				}
				t.Logf("run %d (%.1fs): intent=%s suggested_tools=%v",
					i, elapsed.Seconds(), out.Intent, out.SuggestedTools)
			}

			t.Logf("summary: intents=%s, query_knowledge_base in %d/%d runs",
				formatIntentCounts(tl.intentCounts), tl.ragHits, runs)

			// Require a stable majority classified as DOCUMENTATION_QUERY.
			if tl.intentCounts[wantIntent]*2 <= runs {
				t.Errorf("%s: expected majority %s, got %s",
					c.name, wantIntent, formatIntentCounts(tl.intentCounts))
			}
		})
	}
}

func containsTool(tools []string, name string) bool {
	for _, tn := range tools {
		if tn == name {
			return true
		}
	}
	return false
}

func formatIntentCounts(m map[string]int) string {
	var parts []string
	for k, v := range m {
		parts = append(parts, fmt.Sprintf("%s=%d", k, v))
	}
	return strings.Join(parts, " ")
}
