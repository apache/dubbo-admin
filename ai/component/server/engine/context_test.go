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

package engine

import (
	"encoding/json"
	"strings"
	"testing"

	"dubbo-admin-ai/schema"
)

func validContextJSON(t *testing.T) json.RawMessage {
	t.Helper()
	value := map[string]any{
		"version":    schema.AIContextVersion,
		"capturedAt": "2026-07-19T13:00:00Z",
		"global":     map[string]any{"locale": "cn"},
		"page": map[string]any{
			"path":     "/home",
			"fullPath": "https://admin:secret@example.com/home?token=value&keyword=shop",
			"query":    map[string]any{"authorization": "Bearer value", "keyword": "shop"},
		},
		"scope": map[string]any{"mesh": "nacos2.5"},
		"state": map[string]any{
			"filters": map[string]any{"password": "plain", "items": []any{1, 2, 3}},
		},
		"evidence": []any{
			map[string]any{
				"id":         "cluster-overview",
				"source":     "cluster-overview-api",
				"capturedAt": "2026-07-19T13:00:00Z",
				"data": map[string]any{
					"api_key":  "secret",
					"endpoint": "https://user:pass@example.com/api?cookie=value",
					"content": map[string]any{
						"tags": []any{
							map[string]any{
								"name": "gray",
								"match": []any{
									map[string]any{
										"key":   "env",
										"value": map[string]any{"exact": "gray"},
									},
								},
							},
						},
					},
					"properties": []any{
						map[string]any{"key": "access-token", "value": "token-value"},
						map[string]any{"name": "DB_PASSWORD", "currentValue": "password-value"},
						map[string]any{"key": "environment", "value": "production"},
					},
				},
			},
		},
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal context: %v", err)
	}
	return data
}

func TestChatRequestParseContext(t *testing.T) {
	req := ChatRequest{Context: validContextJSON(t)}
	context, err := req.ParseContext()
	if err != nil {
		t.Fatalf("ParseContext() error = %v", err)
	}

	if context.State.Filters["password"] != redactedContextValue {
		t.Fatalf("password was not redacted: %#v", context.State.Filters)
	}
	if context.Page.Query["authorization"] != redactedContextValue {
		t.Fatalf("authorization was not redacted: %#v", context.Page.Query)
	}
	if context.Evidence[0].Data["api_key"] != redactedContextValue {
		t.Fatalf("api key was not redacted: %#v", context.Evidence[0].Data)
	}
	if strings.Contains(context.Page.FullPath, "admin:secret") || strings.Contains(context.Page.FullPath, "token=value") {
		t.Fatalf("page URL credentials were not redacted: %s", context.Page.FullPath)
	}
	endpoint, _ := context.Evidence[0].Data["endpoint"].(string)
	if strings.Contains(endpoint, "user:pass") || strings.Contains(endpoint, "cookie=value") {
		t.Fatalf("evidence URL credentials were not redacted: %s", endpoint)
	}
	evidenceJSON, _ := json.Marshal(context.Evidence[0].Data)
	if strings.Contains(string(evidenceJSON), maxDepthContextValue) || !strings.Contains(string(evidenceJSON), `"exact":"gray"`) {
		t.Fatalf("nested evidence was not preserved: %s", evidenceJSON)
	}
	if strings.Contains(string(evidenceJSON), "token-value") || strings.Contains(string(evidenceJSON), "password-value") {
		t.Fatalf("semantic sensitive values were not redacted: %s", evidenceJSON)
	}
	if !strings.Contains(string(evidenceJSON), `"value":"production"`) {
		t.Fatalf("non-sensitive semantic value was not preserved: %s", evidenceJSON)
	}
}

func TestChatRequestParseContextValidation(t *testing.T) {
	tests := []struct {
		name       string
		context    json.RawMessage
		errContain string
	}{
		{name: "missing context", context: nil},
		{name: "unsupported version", context: json.RawMessage(`{"version":2,"capturedAt":"2026-07-19T13:00:00Z","global":{},"page":{"path":"/home"},"scope":{"mesh":"mesh"}}`), errContain: "unsupported context version"},
		{name: "unknown field", context: json.RawMessage(`{"version":1,"capturedAt":"2026-07-19T13:00:00Z","global":{},"page":{"path":"/home","unknown":true},"scope":{"mesh":"mesh"}}`), errContain: "unknown field"},
		{name: "oversized", context: json.RawMessage(strings.Repeat("x", schema.AIContextMaxBytes+1)), errContain: "exceeds"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context, err := (&ChatRequest{Context: test.context}).ParseContext()
			if test.errContain == "" {
				if err != nil || context != nil {
					t.Fatalf("ParseContext() = (%#v, %v), want (nil, nil)", context, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.errContain) {
				t.Fatalf("ParseContext() error = %v, want containing %q", err, test.errContain)
			}
		})
	}
}
