# Agent hooks and tracing

The `hooks` component observes stable Agent lifecycle events without coupling
the ReAct strategy to a telemetry backend. Logging and OTLP tracing are the
initial observers.

```yaml
type: hooks
spec:
  logging:
    enabled: true
  tracing:
    enabled: true
    protocol: grpc # grpc | http/protobuf
    service_name: dubbo-admin-ai
    sample_ratio: 1.0
    capture_content: none # none | truncated | full; content export is opt-in
```

The exporter reads standard OpenTelemetry environment variables. Use `grpc`
for an OpenTelemetry Collector or Jaeger OTLP endpoint. Use `http/protobuf`
for an OTLP/HTTP endpoint, including direct Langfuse ingestion.
Keep credentials outside YAML and provide them through variables such as:

```shell
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
export OTEL_EXPORTER_OTLP_HEADERS=authorization=Bearer%20token
```

The default ports are conventionally 4317 for gRPC and 4318 for HTTP/protobuf.

`capture_content` defaults to `none` deliberately: model messages and tool
payloads may contain credentials or user data. Payloads are not serialized on
the agent hot path unless a matching hook explicitly opts in; the tracing hook
defers serialization until the created span reports `IsRecording()`. `truncated`
limits each exported JSON attribute to 4096 bytes; `full` should only be used
in a controlled environment.

## Available events

Hooks observe lifecycle events. The current event vocabulary:

- **Interaction**: `interaction.start`, `interaction.end`, `interaction.error`, `interaction.cancel`, `interaction.degrade`
- **Iteration**: `iteration.start`, `iteration.end`
- **Stage**: `stage.start`, `stage.end`, `stage.error`
- **Model call**: `model_call.start`, `model_call.end`, `model_call.error`
- **Tool call**: `tool_call.start`, `tool_call.end`, `tool_call.error`

Error events (`*.error`) are emitted when an operation fails, before its 
corresponding `.end` event. Cancel events are emitted when a context cancellation
causes an interaction to abort. Degrade events signal that an interaction 
completed with degraded quality (tool failure or fallback model response).

The single ReAct loop emits one iteration and one stage per model call.
Stages are `reasonAct` while tools are available and `answer` on the forced
final iteration. There is no separate observe stage. An empty forced answer
uses `FallbackReason: empty_response`; model errors and timeouts propagate
as errors, while tool errors allow the next iteration to answer with degraded context.

Each `State` carries metadata (session/interaction ID, iteration, stage, model,
tool name) and optional fields like `Degraded`, `FallbackUsed`, `Error` to
provide additional context. The built-in logging and tracing hooks subscribe to 
all events; custom hooks can filter by event and tool name through the 
`Registration` API.

## Local Jaeger verification

Start an ephemeral Jaeger all-in-one instance:

```shell
docker run --rm --name dubbo-admin-jaeger \
  -p 16686:16686 -p 4317:4317 -p 4318:4318 \
  jaegertracing/all-in-one:1.76.0
```

Use `protocol: grpc`, set `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317`,
start dubbo-admin, and complete one Agent interaction. Open
`http://localhost:16686`, select the configured `service_name`, and verify the
trace tree contains `invoke_agent`, `chat <model>`, and any
`execute_tool <tool>` spans.

The opt-in integration test performs the same OTLP export and verifies the
trace through Jaeger's Query API:

```shell
DUBBO_ADMIN_OTEL_E2E=1 \
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 \
go test -run TestJaegerOTLPEndToEnd -v ./component/hooks
```

## Langfuse verification

Langfuse accepts OTLP over HTTP, not gRPC. Create project API keys, then set:

```shell
export AUTH_STRING="$(printf '%s' 'pk-lf-...:sk-lf-...' | base64)"
export OTEL_EXPORTER_OTLP_ENDPOINT=https://cloud.langfuse.com/api/public/otel
export OTEL_EXPORTER_OTLP_HEADERS="Authorization=Basic ${AUTH_STRING},x-langfuse-ingestion-version=4"
```

Use `protocol: http/protobuf`, complete an Agent interaction, and verify that
Langfuse groups its model and tool spans beneath one trace. Other Langfuse
regions require their corresponding regional endpoint.

Use an OpenTelemetry Collector to fan traces out to multiple backends. The
streaming endpoint returns the active trace identifier in `X-Trace-ID`, and MCP
HTTP requests propagate the W3C `traceparent` and `tracestate` headers. To
verify downstream propagation, log the MCP server's inbound `traceparent` and
confirm its trace ID matches `X-Trace-ID` (the middle 32 hexadecimal digits).

Tool-call registrations must explicitly select one or more exact tool names.
Use `hooks.AllTools` (`"*"`) when a hook intentionally observes every tool.

A registration that returns a context used by nested Agent work must set
`DerivesContext: true`. A Manager accepts one such registration because a Go
context can carry only one active span lineage; use an OpenTelemetry Collector
to fan that trace out to multiple backends. Contexts returned by ordinary
observational hooks are ignored. Context-deriving registrations must include
both the Start and End event for every selected lifecycle. Use
`hooks.NewTracingRegistration` for OpenTelemetry so the complete, paired event
set and content-capture behavior cannot be configured inconsistently.
