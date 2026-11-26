/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with this
 * work for additional information regarding copyright ownership.
 * The ASF licenses this file under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with it.
 * You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by law or agreed in writing, software is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
 */
package context

import (
	"context"
	"dubbo-admin-ai/store"
	memorystore "dubbo-admin-ai/store/memory"
	"encoding/json"
	"github.com/firebase/genkit/go/ai"
	"strings"
	"testing"
	"time"
)

func TestMemorySourceMatchingBudgetAndRoles(t *testing.T) {
	for _, scenario := range []string{"selected", "unmatched", "budget", "disabled", "admin", "correction"} {
		t.Run(scenario, func(t *testing.T) {
			compiler, req := fixture(t)
			req.NotAfter = time.Now().Add(time.Second)
			req.SessionTenantID = "tenant"
			req.SessionOwnerID = "owner"
			req.MemoryEnabled = true
			backend := memorystore.NewMemoryStore()
			for _, project := range []string{"A", "B"} {
				_, err := backend.SaveMemory(context.Background(), req.Access, store.MemoryWrite{Generation: 1, StateVersion: 1, ScopeType: "user", MemoryType: "task_context", FactKey: "project.background", Value: "project-" + project + "-background", Applicability: store.MatchContext{ProjectRef: project}})
				if err != nil {
					t.Fatal(err)
				}
			}
			limit := 4096
			req.MatchContext = store.MatchContext{ProjectRef: "A"}
			if scenario == "budget" {
				limit = 1
			}
			if scenario == "unmatched" {
				req.MatchContext = store.MatchContext{}
			}
			if scenario == "disabled" {
				req.MemoryEnabled = false
			}
			if scenario == "correction" {
				req.Turns[len(req.Turns)-1].Messages = []*ai.Message{ai.NewUserTextMessage("project.background=new project context")}
			}
			if scenario == "admin" {
				req.SessionOwnerID = "another-owner"
				req.Access.Subject.Scopes = []string{"ai:sessions:admin"}
				for i := range req.Turns {
					req.Turns[i].OwnerID = "another-owner"
				}
			}
			if err := compiler.Register(MemorySource{Store: backend, MaxItems: 16, MaxBytes: 16384}, SourcePolicy{Scope: "user", Order: 4, MaxTokens: limit, Kinds: []string{"evidence"}}); err != nil {
				t.Fatal(err)
			}
			result, err := compiler.Compile(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			data, _ := json.Marshal(result.Messages)
			if strings.Contains(string(data), "project-B-background") {
				t.Fatal("unrelated project injected")
			}
			selected := scenario == "selected"
			if strings.Contains(string(data), "project-A-background") != selected {
				t.Fatal("unexpected selection", string(data))
			}
			if selected && len(result.Dependencies) != 1 {
				t.Fatal("selected dependency lost")
			}
			if !selected && len(result.Dependencies) != 0 {
				t.Fatal("omitted dependency propagated")
			}
			if result.Messages[0].Role != "system" || strings.Contains(result.Messages[0].Text, "project-A-background") {
				t.Fatal("Memory changed system role")
			}
			if scenario == "budget" {
				found := false
				for _, d := range result.Report.Decisions {
					if d.SourceID == "memory" && d.ReasonCode == "SOURCE_TOKEN_LIMIT" {
						found = true
					}
				}
				if !found {
					t.Fatal("independent budget not reported")
				}
			}
		})
	}
}
