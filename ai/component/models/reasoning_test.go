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

package models

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/openai/openai-go/option"
)

// TestReasoningContentReachesTheMessage walks the whole path the reasoning
// content takes: the endpoint reports it, the adapter parses the response,
// and the message the agent receives carries it as a reasoning part.
//
// The step that matters is the adapter: it maps the body of a message onto a
// text part and leaves the reasoning field behind, so a reasoning capture
// policy would have nothing to filter without the adaptation.
func TestReasoningContentReachesTheMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "1", "object": "chat.completion", "created": 1, "model": "review-model",
			"choices": [{"index": 0, "finish_reason": "stop",
				"message": {"role": "assistant", "content": "answer", "reasoning_content": "REVIEW_REASONING_SENTINEL"}}],
			"usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
		}`))
	}))
	defer server.Close()

	// The plugin the component builds, with the reasoning middleware.
	plugin := &compat_oai.OpenAICompatible{
		Provider: "review",
		Opts: []option.RequestOption{
			option.WithAPIKey("review-key"),
			option.WithBaseURL(server.URL + "/v1"),
			option.WithMiddleware(captureReasoningMiddleware),
		},
	}
	genkit.Init(context.Background(), genkit.WithPlugins(plugin))
	model := reasoningModel{inner: plugin.DefineModel("review", "review-model", ai.ModelOptions{})}

	response, err := model.Generate(context.Background(), &ai.ModelRequest{
		Messages: []*ai.Message{ai.NewMessage(ai.RoleUser, nil, ai.NewTextPart("hello"))},
	}, nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	var reasoning, text string
	for _, part := range response.Message.Content {
		if part.IsReasoning() {
			reasoning += part.Text
		}
		if part.IsText() {
			text += part.Text
		}
	}
	if reasoning != "REVIEW_REASONING_SENTINEL" {
		t.Fatalf("reasoning part = %q, want the reasoning content of the response", reasoning)
	}
	if text != "answer" {
		t.Fatalf("text part = %q, want the answer of the response", text)
	}
}

// TestReasoningContentIsAbsentWithoutTheField keeps the adaptation from
// inventing reasoning content: a response that does not report it produces no
// reasoning part.
func TestReasoningContentIsAbsentWithoutTheField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "1", "object": "chat.completion", "created": 1, "model": "review-model",
			"choices": [{"index": 0, "finish_reason": "stop",
				"message": {"role": "assistant", "content": "answer"}}],
			"usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
		}`))
	}))
	defer server.Close()

	plugin := &compat_oai.OpenAICompatible{
		Provider: "review",
		Opts: []option.RequestOption{
			option.WithAPIKey("review-key"),
			option.WithBaseURL(server.URL + "/v1"),
			option.WithMiddleware(captureReasoningMiddleware),
		},
	}
	genkit.Init(context.Background(), genkit.WithPlugins(plugin))
	model := reasoningModel{inner: plugin.DefineModel("review", "review-model", ai.ModelOptions{})}

	response, err := model.Generate(context.Background(), &ai.ModelRequest{
		Messages: []*ai.Message{ai.NewMessage(ai.RoleUser, nil, ai.NewTextPart("hello"))},
	}, nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, part := range response.Message.Content {
		if part.IsReasoning() {
			t.Fatalf("response carries a reasoning part without reasoning content: %q", part.Text)
		}
	}
}

// TestReasoningCaptureKeepsConcurrentCallsApart makes sure the capture of one
// call cannot reach the response of another one.
func TestReasoningCaptureKeepsConcurrentCallsApart(t *testing.T) {
	var reply string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "1", "object": "chat.completion", "created": 1, "model": "review-model",
			"choices": [{"index": 0, "finish_reason": "stop",
				"message": {"role": "assistant", "content": "answer", "reasoning_content": "` + reply + `"}}],
			"usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
		}`))
	}))
	defer server.Close()

	plugin := &compat_oai.OpenAICompatible{
		Provider: "review",
		Opts: []option.RequestOption{
			option.WithAPIKey("review-key"),
			option.WithBaseURL(server.URL + "/v1"),
			option.WithMiddleware(captureReasoningMiddleware),
		},
	}
	genkit.Init(context.Background(), genkit.WithPlugins(plugin))
	model := reasoningModel{inner: plugin.DefineModel("review", "review-model", ai.ModelOptions{})}

	reply = "FIRST"
	first, err := model.Generate(context.Background(), &ai.ModelRequest{
		Messages: []*ai.Message{ai.NewMessage(ai.RoleUser, nil, ai.NewTextPart("hello"))},
	}, nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	reply = "SECOND"
	second, err := model.Generate(context.Background(), &ai.ModelRequest{
		Messages: []*ai.Message{ai.NewMessage(ai.RoleUser, nil, ai.NewTextPart("hello"))},
	}, nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if got := reasoningOf(first); got != "FIRST" {
		t.Fatalf("first call reasoning = %q, want %q", got, "FIRST")
	}
	if got := reasoningOf(second); got != "SECOND" {
		t.Fatalf("second call reasoning = %q, want %q", got, "SECOND")
	}
}

// reasoningOf returns the reasoning content of a response.
func reasoningOf(response *ai.ModelResponse) string {
	var reasoning strings.Builder
	for _, part := range response.Message.Content {
		if part.IsReasoning() {
			reasoning.WriteString(part.Text)
		}
	}
	return reasoning.String()
}
