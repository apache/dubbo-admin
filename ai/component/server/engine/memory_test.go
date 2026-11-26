/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with this
 * work for additional information regarding copyright ownership.
 * The ASF licenses this file under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with it.
 * You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by law or agreed in writing, software is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
 */
package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dubbo-admin-ai/security"
	"dubbo-admin-ai/store"
	memorystore "dubbo-admin-ai/store/memory"
	"dubbo-admin-ai/testutils"
)

func TestMemoryAPIAndChatMatching(t *testing.T) {
	a := testutils.TestAccess()
	backend := memorystore.NewMemoryStore()
	capture := &captureAgent{}
	auth, err := security.NewStaticAuthenticator(a.Subject)
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewRouter(capture, backend, backend, auth)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = router.Close() })
	router.SetMatchContextValidator(ProjectCatalog{{ID: "project-a", TenantID: a.Subject.TenantID, OwnerIDs: []string{a.Subject.UserID}}})
	request := func(method, path, body, key string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "/api/v1/ai"+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		w := httptest.NewRecorder()
		router.GetEngine().ServeHTTP(w, r)
		return w
	}
	for _, body := range []string{`{"scope_type":"resource"}`, `{"generation":1,"state_version":1,"scope_type":"user","memory_type":"preference","fact_key":"response.language","value":"Chinese","owner_id":"other"}`, `{"generation":1,"state_version":1,"scope_type":"user","memory_type":"task_context","fact_key":"project.background","value":"context","applicability":{"projectRef":"missing"}}`} {
		w := request("POST", "/memories", body, "")
		if w.Code < 400 || w.Code >= 500 {
			t.Fatal("invalid Memory accepted", w.Code, w.Body.String())
		}
	}
	created := request("POST", "/memories", `{"generation":1,"state_version":1,"scope_type":"user","memory_type":"task_context","fact_key":"project.background","value":"stable project background","applicability":{"projectRef":" project-a "}}`, "")
	if created.Code != http.StatusOK {
		t.Fatal(created.Code, created.Body.String())
	}
	var result struct {
		Data store.ExplicitMemory `json:"data"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Data.Applicability.ProjectRef != "project-a" {
		t.Fatal("not normalized", result)
	}
	listed := request("GET", "/memories?limit=1", "", "")
	if listed.Code != 200 || !strings.Contains(listed.Body.String(), result.Data.ID) {
		t.Fatal(listed.Code, listed.Body.String())
	}
	ctx := context.Background()
	if err := backend.Create(ctx, a, testutils.TestSession("matching", time.Now())); err != nil {
		t.Fatal(err)
	}
	chat := `{"sessionID":"matching","message":"help diagnose","matchContext":{"projectRef":" project-a ","taskType":"diagnosis"}}`
	w := request("POST", "/chat/stream", chat, "match-key")
	if w.Code != 200 || capture.input == nil || capture.input.MatchContext.ProjectRef != "project-a" || capture.input.MatchContext.TaskType != "diagnosis" {
		t.Fatal("matching did not reach UserInput", w.Code, w.Body.String(), capture.input)
	}
	changed := request("POST", "/chat/stream", `{"sessionID":"matching","message":"help diagnose","matchContext":{"projectRef":"project-a","taskType":"inspection"}}`, "match-key")
	if changed.Code != 409 || !strings.Contains(changed.Body.String(), "IDEMPOTENCY_CONFLICT") {
		t.Fatal("matching omitted from idempotency", changed.Code, changed.Body.String())
	}
	for _, match := range []string{`{"projectRef":"missing"}`, `{"taskType":"unknown"}`, `{"owner_id":"someone"}`} {
		w := request("POST", "/chat/stream", `{"sessionID":"matching","message":"hello","matchContext":`+match+`}`, "invalid-match")
		if w.Code < 400 {
			t.Fatal("invalid match accepted", w.Code, w.Body.String())
		}
	}
	clear := request("POST", "/memories/clear", `{"generation":1,"state_version":1}`, "")
	if clear.Code != 200 {
		t.Fatal(clear.Code, clear.Body.String())
	}
	stale := request("PUT", "/memories/"+result.Data.ID, `{"generation":1,"state_version":1,"version":1,"scope_type":"user","memory_type":"preference","fact_key":"response.language","value":"English"}`, "")
	if stale.Code != 409 {
		t.Fatal("old generation update", stale.Code, stale.Body.String())
	}
}

func TestMatchHashCompatibility(t *testing.T) {
	legacy := hashChatRequest(ChatRequest{SessionID: "session", Message: "hello"})
	empty := hashChatRequest(ChatRequest{SessionID: "session", Message: "hello", MatchContext: store.MatchContext{}})
	if legacy != empty {
		t.Fatal("empty matching changed v1 hash")
	}
	a, _ := store.NormalizeMatchContext(store.MatchContext{ProjectRef: " project-a ", TaskType: "diagnosis"})
	b, _ := store.NormalizeMatchContext(store.MatchContext{ProjectRef: "project-a", TaskType: "diagnosis"})
	if hashChatRequest(ChatRequest{SessionID: "session", Message: "hello", MatchContext: a}) != hashChatRequest(ChatRequest{SessionID: "session", Message: "hello", MatchContext: b}) {
		t.Fatal("equivalent matching differs")
	}
	if legacy == hashChatRequest(ChatRequest{SessionID: "session", Message: "hello", MatchContext: a}) {
		t.Fatal("matching not hashed")
	}
}
