// Package providerusage preserves accounting availability before Genkit's
// OpenAI-compatible adapter converts missing usage to an allocated zero value.
package providerusage

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/openai/openai-go/option"
)

type contextKey struct{}

type Tokens struct {
	Input  int
	Output int
	Total  int
}

type Observation struct {
	mu       sync.Mutex
	observed bool
	tokens   *Tokens
}

func WithObservation(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey{}, &Observation{})
}

// FromContext reports whether the provider boundary was observed, separately
// from whether the provider supplied a complete token accounting object.
func FromContext(ctx context.Context) (tokens *Tokens, observed bool) {
	o, _ := ctx.Value(contextKey{}).(*Observation)
	if o == nil {
		return nil, false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.tokens == nil {
		return nil, o.observed
	}
	copy := *o.tokens
	return &copy, o.observed
}

// Middleware observes only usage metadata and leaves response bytes unchanged.
// It is scoped to one model call through its request context, not global state.
func Middleware(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
	o, _ := req.Context().Value(contextKey{}).(*Observation)
	if o == nil {
		return next(req)
	}
	o.mu.Lock()
	o.observed, o.tokens = true, nil // reset for each SDK retry
	o.mu.Unlock()
	resp, err := next(req)
	if err == nil && resp != nil && resp.Body != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		resp.Body = &usageBody{ReadCloser: resp.Body, observation: o,
			stream: strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream")}
	}
	return resp, err
}

// A bounded response/record buffer is transient and never logged or persisted.
// Large records stay unknown while their original bytes continue to the SDK.
const maxRecordBytes = 1 << 20

type usageBody struct {
	io.ReadCloser
	observation *Observation
	stream      bool
	buffer      []byte
	oversized   bool
}

func (b *usageBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	data := p[:n]
	if b.stream {
		for len(data) > 0 {
			i := bytes.IndexByte(data, '\n')
			if i < 0 {
				b.append(data)
				break
			}
			b.append(data[:i])
			b.record()
			data = data[i+1:]
		}
	} else {
		b.append(data)
	}
	if err == io.EOF {
		b.record()
	}
	return n, err
}

func (b *usageBody) append(data []byte) {
	if b.oversized {
		return
	}
	if len(b.buffer)+len(data) > maxRecordBytes {
		b.buffer = nil
		b.oversized = true
		return
	}
	b.buffer = append(b.buffer, data...)
}

func (b *usageBody) record() {
	defer func() { b.buffer = nil; b.oversized = false }()
	if b.oversized {
		return
	}
	data := bytes.TrimSpace(b.buffer)
	if b.stream {
		if !bytes.HasPrefix(data, []byte("data:")) {
			return
		}
		data = bytes.TrimSpace(data[len("data:"):])
	}
	var envelope struct {
		Usage *struct {
			Input  *int `json:"prompt_tokens"`
			Output *int `json:"completion_tokens"`
			Total  *int `json:"total_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.Usage == nil {
		return
	}
	u := envelope.Usage
	if u.Input == nil || u.Output == nil || u.Total == nil || *u.Input < 0 || *u.Output < 0 || *u.Total < 0 {
		return
	}
	b.observation.mu.Lock()
	b.observation.tokens = &Tokens{Input: *u.Input, Output: *u.Output, Total: *u.Total}
	b.observation.mu.Unlock()
}
