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
	"fmt"
	"log/slog"
	"time"

	"dubbo-admin-ai/runtime"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type LoggingSpec struct {
	Enabled bool `yaml:"enabled"`
}

const (
	CaptureNone      = "none"
	CaptureTruncated = "truncated"
	CaptureFull      = "full"
	ProtocolGRPC     = "grpc"
	ProtocolHTTP     = "http/protobuf"
)

type TracingSpec struct {
	Enabled        bool    `yaml:"enabled"`
	Protocol       string  `yaml:"protocol"`
	ServiceName    string  `yaml:"service_name"`
	SampleRatio    float64 `yaml:"sample_ratio"`
	CaptureContent string  `yaml:"capture_content"`
}

type Spec struct {
	Logging LoggingSpec `yaml:"logging"`
	Tracing TracingSpec `yaml:"tracing"`
}

type Component struct {
	instanceName string
	spec         Spec
	manager      *Manager
	provider     *sdktrace.TracerProvider
}

type tracerProviderLifecycle interface {
	ForceFlush(context.Context) error
	Shutdown(context.Context) error
}

func NewComponent(spec Spec) *Component {
	return &Component{spec: spec, manager: NewManager(nil)}
}

func (c *Component) Name() string {
	if c.instanceName != "" {
		return c.instanceName
	}
	return "hooks"
}

func (c *Component) SetName(name string) { c.instanceName = name }

func (c *Component) Validate() error {
	if !c.spec.Tracing.Enabled {
		return nil
	}
	if c.spec.Tracing.ServiceName == "" {
		return fmt.Errorf("tracing service_name is required")
	}
	if c.spec.Tracing.SampleRatio < 0 || c.spec.Tracing.SampleRatio > 1 {
		return fmt.Errorf("tracing sample_ratio must be between 0 and 1")
	}
	switch c.spec.Tracing.exportProtocol() {
	case ProtocolGRPC, ProtocolHTTP:
	default:
		return fmt.Errorf("tracing protocol must be %q or %q", ProtocolGRPC, ProtocolHTTP)
	}
	switch c.spec.Tracing.CaptureContent {
	case CaptureNone, CaptureTruncated, CaptureFull:
		return nil
	default:
		return fmt.Errorf("tracing capture_content must be one of %q, %q, or %q", CaptureNone, CaptureTruncated, CaptureFull)
	}
}

func (c *Component) Init(rt *runtime.Runtime) error {
	if rt == nil {
		return fmt.Errorf("runtime is required")
	}
	c.manager = NewManager(rt.GetLogger())
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	if c.spec.Logging.Enabled {
		if err := c.manager.Register(Registration{
			Events: AllEvents(),
			Tools:  []string{AllTools},
			Hook:   NewLoggingHook(rt.GetLogger()),
		}); err != nil {
			return fmt.Errorf("register logging hook: %w", err)
		}
	}
	if c.spec.Tracing.Enabled {
		exporter, err := newTraceExporter(rt.GetContext(), c.spec.Tracing.exportProtocol())
		if err != nil {
			return fmt.Errorf("create OTLP trace exporter: %w", err)
		}
		res := resource.NewSchemaless(attribute.String("service.name", c.spec.Tracing.ServiceName))
		c.provider = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(c.spec.Tracing.SampleRatio))),
		)
		otel.SetTracerProvider(c.provider)
		registration := NewTracingRegistration(
			c.provider.Tracer("dubbo-admin-ai/hooks"),
			c.spec.Tracing.CaptureContent,
		)
		if err := c.manager.Register(registration); err != nil {
			return fmt.Errorf("register tracing hook: %w", err)
		}
	}
	return nil
}

func (s TracingSpec) exportProtocol() string {
	if s.Protocol == "" {
		return ProtocolGRPC
	}
	return s.Protocol
}

func newTraceExporter(ctx context.Context, protocol string) (sdktrace.SpanExporter, error) {
	switch protocol {
	case ProtocolGRPC:
		return otlptracegrpc.New(ctx)
	case ProtocolHTTP:
		return otlptracehttp.New(ctx)
	default:
		return nil, fmt.Errorf("unsupported OTLP trace protocol %q", protocol)
	}
}

func (c *Component) Start() error { return nil }

func (c *Component) Stop() error {
	if c.provider == nil {
		return nil
	}
	return stopTracerProvider(c.provider, 5*time.Second)
}

func stopTracerProvider(provider tracerProviderLifecycle, timeout time.Duration) error {
	flushCtx, cancelFlush := context.WithTimeout(context.Background(), timeout)
	flushErr := provider.ForceFlush(flushCtx)
	cancelFlush()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), timeout)
	shutdownErr := provider.Shutdown(shutdownCtx)
	cancelShutdown()
	return errors.Join(flushErr, shutdownErr)
}

func (c *Component) GetManager() *Manager { return c.manager }

// GetActiveManager is used by the agent hot path. The component retains its
// manager for later registrations, but injects nil when no hooks are active.
func (c *Component) GetActiveManager() *Manager {
	if !c.manager.HasHooks() {
		return nil
	}
	return c.manager
}

func NewLoggingHook(logger *slog.Logger) Hook {
	if logger == nil {
		logger = slog.Default()
	}
	return func(ctx context.Context, state State) context.Context {
		attrs := []any{
			"event", state.Event,
			"interaction_id", state.InteractionID,
			"session_id", state.SessionID,
			"iteration", state.Iteration,
			"stage", state.Stage,
			"model", state.Model,
			"tool", state.ToolName,
			"tool_call_id", state.ToolCallID,
			"degraded", state.Degraded,
			"fallback_used", state.FallbackUsed,
		}
		if state.FallbackReason != "" {
			attrs = append(attrs, "fallback_reason", state.FallbackReason)
		}
		if state.Error != "" {
			attrs = append(attrs, "error", state.Error)
		}
		logger.InfoContext(ctx, "Agent hook event", attrs...)
		return ctx
	}
}
