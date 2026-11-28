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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/openai/openai-go/option"
)

// reasoningCaptureKey is the context key of the per-call reasoning capture.
type reasoningCaptureKey struct{}

// reasoningCapture holds the reasoning content of a single model call.
//
// An OpenAI-compatible endpoint reports the thinking content of a model in a
// non-standard field, reasoning_content. The adapter that parses the response
// does not map that field onto a reasoning part, so the field is read from the
// response body instead. The capture travels in the context of the call, so
// concurrent calls cannot see each other's reasoning.
type reasoningCapture struct {
	mu       sync.Mutex
	contents []string
}

// add records the reasoning content of one response.
func (c *reasoningCapture) add(content string) {
	if content == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.contents = append(c.contents, content)
}

// content returns the reasoning content of the call.
func (c *reasoningCapture) content() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return strings.Join(c.contents, "\n")
}

// captureReasoningMiddleware reads the reasoning content out of a chat
// completion response.
//
// The middleware sees the response before the adapter parses it, which is the
// only place the reasoning content is still present: the adapter maps the
// message body onto a text part and leaves the field behind.
//
// A streaming response is left alone. Its body is consumed incrementally, so
// reading it here would turn the stream into a buffer and change how the
// caller receives the answer.
func captureReasoningMiddleware(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
	resp, err := next(req)
	if err != nil || resp == nil || resp.Body == nil {
		return resp, err
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		return resp, nil
	}
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	// The body is handed back unchanged: the adapter still has to read it.
	resp.Body = io.NopCloser(bytes.NewReader(body))
	if readErr != nil {
		return resp, nil
	}
	capture, ok := req.Context().Value(reasoningCaptureKey{}).(*reasoningCapture)
	if !ok {
		return resp, nil
	}
	for _, content := range reasoningContents(body) {
		capture.add(content)
	}
	return resp, nil
}

// reasoningContents returns the reasoning content of a chat completion
// response body.
//
// The field is read by name because it is not part of the OpenAI chat
// completion schema: the endpoints that report it differ in where they put
// it, so both the message of a complete response and the delta of a chunk
// are inspected.
func reasoningContents(body []byte) []string {
	var completion struct {
		Choices []struct {
			Message struct {
				ReasoningContent string `json:"reasoning_content"`
			} `json:"message"`
			Delta struct {
				ReasoningContent string `json:"reasoning_content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &completion); err != nil {
		return nil
	}
	var contents []string
	for _, choice := range completion.Choices {
		if choice.Message.ReasoningContent != "" {
			contents = append(contents, choice.Message.ReasoningContent)
		}
		if choice.Delta.ReasoningContent != "" {
			contents = append(contents, choice.Delta.ReasoningContent)
		}
	}
	return contents
}

// reasoningModel adds the reasoning content of a response to the parts of the
// message it returns.
//
// The reasoning content becomes a reasoning part, which is the shape the
// generative AI semantic conventions define and the shape the capture policy
// acts on. Without it a reasoning capture policy has nothing to filter, and
// the thinking content of a model never reaches a trace.
type reasoningModel struct {
	inner ai.Model
}

// Name returns the registry name of the model.
func (m reasoningModel) Name() string {
	return m.inner.Name()
}

// Register registers the model with the given registry.
func (m reasoningModel) Register(r api.Registry) {
	m.inner.Register(r)
}

// Generate calls the wrapped model and adds the reasoning content of the
// response to the message it returns.
func (m reasoningModel) Generate(ctx context.Context, req *ai.ModelRequest, cb ai.ModelStreamCallback) (*ai.ModelResponse, error) {
	capture := &reasoningCapture{}
	ctx = context.WithValue(ctx, reasoningCaptureKey{}, capture)
	response, err := m.inner.Generate(ctx, req, cb)
	if err != nil || response == nil || response.Message == nil {
		return response, err
	}
	reasoning := capture.content()
	if reasoning == "" {
		return response, nil
	}
	response.Message.Content = append(response.Message.Content, ai.NewReasoningPart(reasoning, nil))
	return response, nil
}
