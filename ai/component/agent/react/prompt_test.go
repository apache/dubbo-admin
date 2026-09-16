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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

func TestBuildPromptsSeparatesPhasePolicies(t *testing.T) {
	const (
		promptFile   = "shared-policy.txt"
		sharedPolicy = "# Shared Policy\nCapacity is 99% available."
		toolName     = "lookup_service"
	)

	promptDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(promptDir, promptFile), []byte(sharedPolicy), 0o600); err != nil {
		t.Fatalf("write shared policy: %v", err)
	}

	g := genkit.Init(context.Background())
	spec := &AgentSpec{
		PromptBasePath: promptDir,
		PromptFile:     promptFile,
		Temperature:    0,
		MaxTokens:      512,
	}
	act, answer, err := (&ReActAgent{}).buildPrompts(g, spec, "test/model", []ai.ToolRef{ai.ToolName(toolName)})
	if err != nil {
		t.Fatalf("buildPrompts() error: %v", err)
	}

	actOpts, err := act.Render(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("render act prompt: %v", err)
	}
	answerOpts, err := answer.Render(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("render answer prompt: %v", err)
	}

	actSystem := onlySystemMessage(t, actOpts.Messages)
	answerSystem := onlySystemMessage(t, answerOpts.Messages)

	for name, system := range map[string]string{"act": actSystem, "answer": answerSystem} {
		if !strings.Contains(system, sharedPolicy) {
			t.Errorf("%s system prompt does not contain shared policy: %q", name, system)
		}
		if strings.Contains(system, "%!") || !strings.Contains(system, "99% available") {
			t.Errorf("%s system prompt corrupted percent text: %q", name, system)
		}
	}
	if !strings.Contains(actSystem, actPolicy) || strings.Contains(actSystem, finalAnswerPolicy) {
		t.Errorf("act system prompt has incorrect phase policy: %q", actSystem)
	}
	if !strings.Contains(answerSystem, finalAnswerPolicy) || strings.Contains(answerSystem, actPolicy) {
		t.Errorf("answer system prompt has incorrect phase policy: %q", answerSystem)
	}

	if len(actOpts.Tools) != 1 || actOpts.Tools[0] != toolName {
		t.Errorf("act prompt tools = %v, want [%s]", actOpts.Tools, toolName)
	}
	if !actOpts.ReturnToolRequests {
		t.Error("act prompt must return tool requests to the ReAct loop")
	}
	if len(answerOpts.Tools) != 0 {
		t.Errorf("answer prompt unexpectedly binds tools: %v", answerOpts.Tools)
	}
	if answerOpts.ReturnToolRequests {
		t.Error("answer prompt must not return tool requests")
	}
}

func onlySystemMessage(t *testing.T, messages []*ai.Message) string {
	t.Helper()
	if len(messages) != 1 {
		t.Fatalf("rendered prompt has %d messages, want only the system message", len(messages))
	}
	if messages[0].Role != ai.RoleSystem {
		t.Fatalf("rendered message role = %q, want %q", messages[0].Role, ai.RoleSystem)
	}
	return messages[0].Text()
}
