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
	"testing"

	"github.com/firebase/genkit/go/ai"
)

func TestGenAIMessageContentUsesSemanticConventionShape(t *testing.T) {
	messages := []*ai.Message{
		ai.NewUserMessage(ai.NewTextPart("weather?")),
		ai.NewModelMessage(ai.NewToolRequestPart(&ai.ToolRequest{
			Ref: "call-1", Name: "weather", Input: map[string]any{"city": "Paris"},
		})),
	}
	content, err := json.Marshal(genAIInputMessages(messages))
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded[0]["role"] != "user" || decoded[1]["role"] != "assistant" {
		t.Fatalf("roles = %v, want user/assistant", decoded)
	}
	parts := decoded[1]["parts"].([]any)
	toolCall := parts[0].(map[string]any)
	if toolCall["type"] != "tool_call" || toolCall["id"] != "call-1" || toolCall["name"] != "weather" {
		t.Fatalf("tool call part = %v", toolCall)
	}
}

func TestGenAIOutputIncludesFinishReason(t *testing.T) {
	response := &ai.ModelResponse{
		Message:      ai.NewModelTextMessage("done"),
		FinishReason: ai.FinishReasonStop,
	}
	content, err := json.Marshal(genAIOutputMessages(response))
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded[0]["finish_reason"] != "stop" {
		t.Fatalf("output = %v, want finish_reason stop", decoded)
	}
}

func TestGenAIInputIncludesReasoningPart(t *testing.T) {
	messages := []*ai.Message{
		ai.NewModelMessage(
			ai.NewReasoningPart("thinking about the answer", []byte("signature")),
			ai.NewTextPart("the answer"),
		),
	}
	content, err := json.Marshal(genAIInputMessages(messages))
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatal(err)
	}
	parts := decoded[0]["parts"].([]any)
	reasoning := parts[0].(map[string]any)
	if reasoning["type"] != "reasoning" || reasoning["content"] != "thinking about the answer" {
		t.Fatalf("reasoning part = %v", reasoning)
	}
	if len(reasoning) != 2 {
		t.Fatalf("reasoning part = %v, want only a type and a content field", reasoning)
	}
	text := parts[1].(map[string]any)
	if text["type"] != "text" || text["content"] != "the answer" {
		t.Fatalf("text part = %v", text)
	}
}

func TestGenAIOutputIncludesReasoningPart(t *testing.T) {
	response := &ai.ModelResponse{
		Message:      ai.NewModelMessage(ai.NewReasoningPart("thinking", nil)),
		FinishReason: ai.FinishReasonStop,
	}
	content, err := json.Marshal(genAIOutputMessages(response))
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatal(err)
	}
	parts := decoded[0]["parts"].([]any)
	reasoning := parts[0].(map[string]any)
	if reasoning["type"] != "reasoning" || reasoning["content"] != "thinking" {
		t.Fatalf("reasoning part = %v", reasoning)
	}
}

func TestGenAIInputSkipsEmptyReasoningPart(t *testing.T) {
	messages := []*ai.Message{
		ai.NewModelMessage(
			ai.NewReasoningPart("", nil),
			ai.NewTextPart("the answer"),
		),
	}
	content, err := json.Marshal(genAIInputMessages(messages))
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatal(err)
	}
	parts := decoded[0]["parts"].([]any)
	if len(parts) != 1 || parts[0].(map[string]any)["type"] != "text" {
		t.Fatalf("parts = %v, want only the text part", parts)
	}
}
