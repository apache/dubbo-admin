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

package security

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
)

// RedactingHandler applies the baseline Sanitizer to every AI log attribute.
// It is intentionally not configurable: feature flags cannot disable baseline
// secret protection.
type RedactingHandler struct {
	next      slog.Handler
	sanitizer *Sanitizer
	// redactAll is set when the logger is scoped to a sensitive group name
	// (for example logger.WithGroup("password")). Every descendant attribute
	// is then redacted regardless of its own key.
	redactAll bool
}

func NewRedactingHandler(next slog.Handler) slog.Handler {
	if next == nil {
		panic("security: nil slog handler")
	}
	return &RedactingHandler{next: next, sanitizer: DefaultSanitizer()}
}

func (h *RedactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *RedactingHandler) Handle(ctx context.Context, record slog.Record) error {
	clean := slog.NewRecord(record.Time, record.Level, h.sanitizer.SanitizeText(record.Message).Value, record.PC)
	record.Attrs(func(attr slog.Attr) bool {
		clean.AddAttrs(h.sanitizeAttr(attr))
		return true
	})
	return h.next.Handle(ctx, clean)
}

func (h *RedactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clean := make([]slog.Attr, len(attrs))
	for i := range attrs {
		clean[i] = h.sanitizeAttr(attrs[i])
	}
	return &RedactingHandler{next: h.next.WithAttrs(clean), sanitizer: h.sanitizer, redactAll: h.redactAll}
}

func (h *RedactingHandler) WithGroup(name string) slog.Handler {
	groupName := name
	redactAll := h.redactAll
	if isSensitiveKey(name) {
		groupName = redactedValue
		redactAll = true
	}
	return &RedactingHandler{next: h.next.WithGroup(groupName), sanitizer: h.sanitizer, redactAll: redactAll}
}

func (h *RedactingHandler) sanitizeAttr(attr slog.Attr) slog.Attr {
	attr.Value = attr.Value.Resolve()
	if h.redactAll || isSensitiveKey(attr.Key) {
		return slog.String(attr.Key, redactedValue)
	}
	if attr.Value.Kind() == slog.KindGroup {
		// Group() returns the caller's backing slice; copy before rewriting.
		members := append([]slog.Attr(nil), attr.Value.Group()...)
		for i := range members {
			members[i] = h.sanitizeAttr(members[i])
		}
		return slog.Group(attr.Key, attrsToAny(members)...)
	}
	switch attr.Value.Kind() {
	case slog.KindString:
		return slog.String(attr.Key, h.sanitizer.SanitizeText(attr.Value.String()).Value)
	case slog.KindAny:
		value := attr.Value.Any()
		switch typed := value.(type) {
		case error:
			return slog.String(attr.Key, h.sanitizer.SanitizeText(typed.Error()).Value)
		case *url.URL:
			clean := h.sanitizer.SanitizeURL(typed)
			return slog.String(attr.Key, clean.Value.String())
		case url.URL:
			clean := h.sanitizer.SanitizeURL(&typed)
			return slog.String(attr.Key, clean.Value.String())
		case http.Header:
			clean := h.sanitizer.SanitizeHeaders(typed)
			return slog.Any(attr.Key, clean.Value)
		}
		clean := h.sanitizer.SanitizeStructured(value)
		return slog.Any(attr.Key, clean.Value)
	default:
		return attr
	}
}

func attrsToAny(attrs []slog.Attr) []any {
	values := make([]any, len(attrs))
	for i := range attrs {
		values[i] = attrs[i]
	}
	return values
}

// SafeError returns a privacy-safe error string for telemetry or external
// responses. It does not modify the original error.
func SafeError(err error) string {
	if err == nil {
		return ""
	}
	return DefaultSanitizer().SanitizeText(fmt.Sprint(err)).Value
}
