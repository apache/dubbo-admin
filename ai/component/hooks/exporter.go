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

import (
	"context"
	"encoding/json"

	"github.com/firebase/genkit/go/ai"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// The attributes Genkit writes on the spans of its own actions. Genkit
// serializes the request and the response of a model call into them, so they
// carry the same content the capture policy governs.
const (
	genkitInputAttribute  = "genkit:input"
	genkitOutputAttribute = "genkit:output"
)

// contentScrubbingExporter applies the capture policy to the content
// attributes that Genkit writes on its own spans.
//
// The tracing hook is not the only writer of a trace. Genkit records the
// request and the response of every model call on the span of its own action,
// and those attributes bypass the hook: the Manager never sees them, so
// neither the content policy nor the reasoning policy is applied to them.
// Scrubbing them at the export boundary is what makes the policies hold for
// the whole trace instead of for the part of it the hook owns.
//
// Only the Genkit attributes are touched. The attributes the hook writes —
// gen_ai.input.messages, gen_ai.output.messages and the rest — are already
// governed by the policies.
type contentScrubbingExporter struct {
	inner            sdktrace.SpanExporter
	captureContent   string
	captureReasoning string
}

// ExportSpans scrubs the Genkit content attributes of the batch and hands the
// batch to the wrapped exporter.
//
// The spans are copied before they are rewritten, because a recorded span is
// read-only. The SDK's own span snapshot is used for the copy: it carries
// every field a recorded span exposes, so the exported batch stays
// faithful.
func (e *contentScrubbingExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	stubs := tracetest.SpanStubsFromReadOnlySpans(spans)
	for i := range stubs {
		stubs[i].Attributes = e.scrubAttributes(stubs[i].Attributes)
	}
	return e.inner.ExportSpans(ctx, stubs.Snapshots())
}

// Shutdown shuts the wrapped exporter down.
func (e *contentScrubbingExporter) Shutdown(ctx context.Context) error {
	return e.inner.Shutdown(ctx)
}

// scrubAttributes applies the capture policy to the Genkit content attributes
// of a span and leaves every other attribute untouched.
func (e *contentScrubbingExporter) scrubAttributes(attrs []attribute.KeyValue) []attribute.KeyValue {
	scrubbed := make([]attribute.KeyValue, 0, len(attrs))
	for _, attr := range attrs {
		key := string(attr.Key)
		if key != genkitInputAttribute && key != genkitOutputAttribute {
			scrubbed = append(scrubbed, attr)
			continue
		}
		if e.captureContent == CaptureNone {
			continue
		}
		if attr.Value.Type() != attribute.STRING {
			// Genkit writes the attributes as strings. An attribute of another
			// type was not written by Genkit, so it is left as it is.
			scrubbed = append(scrubbed, attr)
			continue
		}
		switch key {
		case genkitInputAttribute:
			scrubbed = append(scrubbed, attribute.String(string(attr.Key), scrubGenkitRequest(attr.Value.AsString(), e.captureReasoning)))
		case genkitOutputAttribute:
			scrubbed = append(scrubbed, attribute.String(string(attr.Key), scrubGenkitResponse(attr.Value.AsString(), e.captureReasoning)))
		}
	}
	return scrubbed
}

// scrubGenkitRequest applies the reasoning policy to a serialized Genkit model
// request.
//
// The document is the serialization of ai.ModelRequest, so the reasoning parts
// are found with Genkit's own types instead of a hand-written guess at its
// JSON shape. A document that does not parse is returned unchanged: a parsing
// failure must never drop captured content.
func scrubGenkitRequest(document string, policy string) string {
	if policy == CaptureFull {
		return document
	}
	var request ai.ModelRequest
	if err := json.Unmarshal([]byte(document), &request); err != nil {
		return document
	}
	changed := false
	for _, message := range request.Messages {
		if message == nil {
			continue
		}
		content, partsChanged := scrubGenkitParts(message.Content, policy)
		if partsChanged {
			message.Content = content
			changed = true
		}
	}
	if !changed {
		return document
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return document
	}
	return string(encoded)
}

// scrubGenkitResponse applies the reasoning policy to a serialized Genkit model
// response.
func scrubGenkitResponse(document string, policy string) string {
	if policy == CaptureFull {
		return document
	}
	var response ai.ModelResponse
	if err := json.Unmarshal([]byte(document), &response); err != nil {
		return document
	}
	if response.Message == nil {
		return document
	}
	content, changed := scrubGenkitParts(response.Message.Content, policy)
	if !changed {
		return document
	}
	response.Message.Content = content
	encoded, err := json.Marshal(response)
	if err != nil {
		return document
	}
	return string(encoded)
}

// scrubGenkitParts applies the reasoning policy to the parts of a Genkit
// message and reports whether the message was changed.
//
// A reasoning part is a part of Genkit's own reasoning kind. Under the none
// policy it is removed, and a message that held nothing else gets the same
// placeholder the hook uses, so a reader does not mistake it for a message
// without content. Under the truncated policy the part is kept and its text
// is shortened.
func scrubGenkitParts(parts []*ai.Part, policy string) ([]*ai.Part, bool) {
	kept := make([]*ai.Part, 0, len(parts))
	changed := false
	for _, part := range parts {
		if part == nil || part.Kind != ai.PartReasoning {
			kept = append(kept, part)
			continue
		}
		changed = true
		if policy == CaptureTruncated {
			part.Text, _ = truncateReasoningString(part.Text)
			kept = append(kept, part)
		}
	}
	if changed && len(kept) == 0 {
		kept = append(kept, ai.NewTextPart(reasoningOmittedText))
	}
	return kept, changed
}
