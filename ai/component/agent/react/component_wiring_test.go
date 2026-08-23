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
	"context"
	"testing"

	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/component/tools"
	"dubbo-admin-ai/runtime"
)

func TestAgentComponentKeepsEmptyManagerForLateHookRegistration(t *testing.T) {
	rt := runtime.NewRuntime()
	toolsComponent, err := tools.NewToolsComponent(tools.ToolConfig{})
	if err != nil {
		t.Fatalf("create tools component: %v", err)
	}
	rt.RegisterComponent(toolsComponent)

	hooksComponent := hooks.NewComponent(hooks.Spec{})
	if err := hooksComponent.Init(rt); err != nil {
		t.Fatalf("initialize hooks component: %v", err)
	}
	rt.RegisterComponent(hooksComponent)

	agentComponentRaw, err := NewAgentComponent("react", "test/model", "", 1, 1, "", nil, nil)
	if err != nil {
		t.Fatalf("create agent component: %v", err)
	}
	agentComponent := agentComponentRaw.(*AgentComponent)
	if err := agentComponent.Init(rt); err != nil {
		t.Fatalf("initialize agent component: %v", err)
	}
	if agentComponent.Agent.hookManager != hooksComponent.GetManager() {
		t.Fatal("agent did not retain the hooks manager")
	}

	calls := 0
	if err := hooksComponent.GetManager().Register(hooks.Registration{
		Events: []hooks.Event{hooks.EventInteractionStart},
		Hook: func(ctx context.Context, _ hooks.State) context.Context {
			calls++
			return ctx
		},
	}); err != nil {
		t.Fatalf("register late hook: %v", err)
	}
	emitHook(agentComponent.Agent.hookManager, context.Background(), hooks.State{Event: hooks.EventInteractionStart})
	if calls != 1 {
		t.Fatalf("late hook calls = %d, want 1", calls)
	}
}
