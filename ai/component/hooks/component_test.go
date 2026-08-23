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
	"errors"
	"testing"
	"time"

	"dubbo-admin-ai/runtime"

	"gopkg.in/yaml.v3"
)

type deadlineFlushProvider struct {
	shutdownCalled     bool
	shutdownContextErr error
}

func (p *deadlineFlushProvider) ForceFlush(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func (p *deadlineFlushProvider) Shutdown(ctx context.Context) error {
	p.shutdownCalled = true
	p.shutdownContextErr = ctx.Err()
	return nil
}

func TestHookFactoryAndLifecycle(t *testing.T) {
	var spec yaml.Node
	if err := yaml.Unmarshal([]byte("logging:\n  enabled: true\n"), &spec); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	component, err := HookFactory(spec.Content[0])
	if err != nil {
		t.Fatalf("HookFactory() error = %v", err)
	}
	hooksComponent, ok := component.(*Component)
	if !ok {
		t.Fatalf("component type = %T, want *hooks.Component", component)
	}
	if err := hooksComponent.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if err := hooksComponent.Init(runtime.NewRuntime()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if hooksComponent.GetManager() == nil {
		t.Fatal("GetManager() returned nil")
	}
	if err := hooksComponent.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := hooksComponent.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestComponentRejectsNilRuntime(t *testing.T) {
	if err := NewComponent(Spec{}).Init(nil); err == nil {
		t.Fatal("Init(nil) succeeded")
	}
}

func TestStopTracerProviderGivesShutdownFreshTimeout(t *testing.T) {
	provider := &deadlineFlushProvider{}
	err := stopTracerProvider(provider, 20*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("stopTracerProvider() error = %v, want flush deadline", err)
	}
	if !provider.shutdownCalled {
		t.Fatal("Shutdown() was not called after ForceFlush timeout")
	}
	if provider.shutdownContextErr != nil {
		t.Fatalf("Shutdown() received expired context: %v", provider.shutdownContextErr)
	}
}

func TestComponentReturnsNilManagerWhenAllHooksDisabled(t *testing.T) {
	component := NewComponent(Spec{})
	if err := component.Init(runtime.NewRuntime()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if component.GetManager() == nil {
		t.Fatal("GetManager() must remain available for custom registration")
	}
	if component.GetActiveManager() != nil {
		t.Fatal("GetActiveManager() returned a manager with no registered hooks")
	}
}

func TestComponentRequestsContentOnlyForTracingCapture(t *testing.T) {
	t.Run("logging only", func(t *testing.T) {
		component := NewComponent(Spec{Logging: LoggingSpec{Enabled: true}})
		if err := component.Init(runtime.NewRuntime()); err != nil {
			t.Fatalf("Init() error = %v", err)
		}
		if component.GetActiveManager().NeedsContent(EventModelCallStart, "") {
			t.Fatal("logging hook requested model content")
		}
	})

	t.Run("tracing full capture", func(t *testing.T) {
		component := NewComponent(Spec{Tracing: TracingSpec{
			Enabled: true, ServiceName: "test", SampleRatio: 1, CaptureContent: CaptureFull,
		}})
		if err := component.Init(runtime.NewRuntime()); err != nil {
			t.Fatalf("Init() error = %v", err)
		}
		t.Cleanup(func() {
			if err := component.Stop(); err != nil {
				t.Errorf("Stop() error = %v", err)
			}
		})
		manager := component.GetActiveManager()
		if !manager.NeedsContent(EventModelCallStart, "") {
			t.Fatal("tracing hook did not request model content")
		}
		if !manager.NeedsContent(EventToolCallStart, "lookup_service") {
			t.Fatal("tracing hook did not request tool content")
		}
	})
}

func TestComponentValidatesTraceProtocol(t *testing.T) {
	base := TracingSpec{
		Enabled: true, ServiceName: "test", SampleRatio: 1, CaptureContent: CaptureNone,
	}
	for _, protocol := range []string{"", ProtocolGRPC, ProtocolHTTP} {
		spec := base
		spec.Protocol = protocol
		if err := NewComponent(Spec{Tracing: spec}).Validate(); err != nil {
			t.Fatalf("Validate() rejected protocol %q: %v", protocol, err)
		}
	}
	base.Protocol = "invalid"
	if err := NewComponent(Spec{Tracing: base}).Validate(); err == nil {
		t.Fatal("Validate() accepted an invalid tracing protocol")
	}
}
