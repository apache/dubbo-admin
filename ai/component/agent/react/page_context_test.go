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
	"strings"
	"testing"

	"dubbo-admin-ai/component/memory"
	"dubbo-admin-ai/schema"

	"github.com/firebase/genkit/go/ai"
)

func TestInjectCurrentPageContext(t *testing.T) {
	history := []*ai.Message{
		ai.NewUserMessage(ai.NewTextPart("previous question")),
		ai.NewModelMessage(ai.NewTextPart("previous answer")),
		ai.NewUserMessage(ai.NewTextPart("current question")),
		ai.NewModelMessage(ai.NewTextPart("current thought")),
	}
	snapshot := &schema.AIContextSnapshot{
		Version:    schema.AIContextVersion,
		CapturedAt: "2026-07-19T13:00:00Z",
		Global:     schema.AIContextGlobal{Locale: "cn"},
		Page:       schema.AIContextPage{Path: "/home"},
		Scope:      schema.AIContextScope{Mesh: "nacos2.5"},
	}

	messages, err := injectCurrentPageContext(withCurrentPageContext(context.Background(), snapshot), history)
	if err != nil {
		t.Fatalf("injectCurrentPageContext() error = %v", err)
	}
	if len(messages) != 5 || len(history) != 4 {
		t.Fatalf("message lengths = (%d, %d), want (5, 4)", len(messages), len(history))
	}
	contextMessage := messages[2]
	if contextMessage.Role != ai.RoleUser || len(contextMessage.Content) != 1 {
		t.Fatalf("unexpected context message: %#v", contextMessage)
	}
	if messages[3].Content[0].Text != "current question" || messages[4].Content[0].Text != "current thought" {
		t.Fatalf("context changed the current turn order: %#v", messages)
	}
	var envelope currentPageContextEnvelope
	if err := json.Unmarshal([]byte(contextMessage.Content[0].Text), &envelope); err != nil {
		t.Fatalf("unmarshal context message: %v", err)
	}
	if envelope.Trust != "untrusted_observation" || envelope.Context.Scope.Mesh != "nacos2.5" {
		t.Fatalf("unexpected context envelope: %#v", envelope)
	}
}

func TestNewInteractionCarriesPageContext(t *testing.T) {
	snapshot := &schema.AIContextSnapshot{
		Version: schema.AIContextVersion,
		Page:    schema.AIContextPage{Path: "/home"},
		Scope:   schema.AIContextScope{Mesh: "nacos2.5"},
	}
	ra := &ReActAgent{memoryCtx: memory.NewMemoryContext(memory.ChatHistoryKey)}

	ctx, _, history, err := ra.newInteraction(&schema.UserInput{
		Content: "current question",
		Context: snapshot,
	}, "session")
	if err != nil {
		t.Fatalf("newInteraction() error = %v", err)
	}
	messages, err := injectCurrentPageContext(ctx, history.WindowMemory("session"))
	if err != nil {
		t.Fatalf("injectCurrentPageContext() error = %v", err)
	}
	if len(messages) != 2 || messages[1].Content[0].Text != "current question" {
		t.Fatalf("unexpected interaction messages: %#v", messages)
	}
}

func TestUserInputContextIsNotSerialized(t *testing.T) {
	input := schema.UserInput{
		Content: "hello",
		Context: &schema.AIContextSnapshot{Version: schema.AIContextVersion},
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal UserInput: %v", err)
	}
	if strings.Contains(string(data), "context") {
		t.Fatalf("serialized history contains page context: %s", data)
	}
}
