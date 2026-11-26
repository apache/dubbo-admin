/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 */
package react

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"dubbo-admin-ai/testutils"
	"github.com/firebase/genkit/go/ai"
)

func TestLargeSummaryFallsBackToRecentHistoryAtProvider(t *testing.T) {
	provider := &providerFixture{summaries: true, summaryPadding: 3500}
	unused, backend, spec, g, _ := compilerAgentFixture(t, provider)
	unused.Stop()
	ctx := testutils.ContextWithTestAccess(context.Background())
	access := testutils.TestAccess()
	p, err := backend.PrepareInteraction(ctx, access, "compiled", "seed-budget", "seed-budget", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.AddHistoryToTurn(ctx, access, "compiled", p.TurnID, ai.NewUserTextMessage("initial goal: identify outage"), ai.NewModelTextMessage("seed answer")); err != nil {
		t.Fatal(err)
	}
	if err := backend.CompleteInteraction(ctx, access, "compiled", p.InteractionID, p.TurnID, "seed answer"); err != nil {
		t.Fatal(err)
	}
	spec.MaxIterations = 1
	spec.ContextWindowTurns = 2
	spec.ConversationSummary.Enabled = true
	spec.ConversationSummary.ContextTokens = 32768
	spec.ConversationSummary.TriggerHeadroomTurns = 1
	ra, err := NewReActAgentWithStore(g, backend, spec, newToolTimeoutResolver(5, nil), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ra.Stop)
	for i := 0; i < 4; i++ {
		runCompiledInteraction(t, ra, backend, fmt.Sprintf("budget-%d", i))
	}
	v, err := backend.GetSummary(ctx, access, "compiled")
	if err != nil || v == nil || len(v.Payload.Goals) == 0 || len(v.Payload.Goals[0].Text) < 3500 {
		t.Fatalf("large summary not committed: %+v %v", v, err)
	}
	ra.compilerSpec.ContextTokens = 4096
	// Retain the valid large summary but prevent further generation for the test.
	// newInteraction obtains it only through Advance, so use invalid output to
	// exercise the normal old-summary retention path instead of disabling it.
	provider.summaries = false
	runCompiledInteraction(t, ra, backend, "bounded-fallback")
	provider.mu.Lock()
	last := stringMustJSON(provider.requests[len(provider.requests)-1])
	provider.mu.Unlock()
	if strings.Contains(last, "earliest objective preserved") || !strings.Contains(last, "check the service") {
		t.Fatal("Provider did not receive recent history fallback", last)
	}
}
