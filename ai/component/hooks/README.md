# Agent hooks and tracing

The `hooks` component observes stable Agent lifecycle events without coupling
the ReAct strategy to a telemetry backend. Logging and OTLP tracing are the
initial observers.

```yaml
type: hooks
spec:
  hooks:
    - name: lifecycle-log
      type: logging
      enabled: true
      config:
        level: info # debug | info | warn | error
    - name: otel-tracing
      type: tracing
      enabled: true
      config:
        protocol: grpc # grpc | http/protobuf
        service_name: dubbo-admin-ai
        sample_ratio: 1.0
        capture_content: none # none | truncated | full; opt-in tracing payloads
```

## Configuration-based registration

`spec.hooks` registers the listed built-in observers at startup. `name` is a
unique instance name and `type` selects `logging` or `tracing`. Multiple logging
instances may subscribe to different events and tools. Hook implementations are
built into the binary; this configuration does not load plugins or scripts.
Custom Go hooks can still register through `GetManager().Register(...)`.

Each entry defaults to `enabled: true`. Omitting `events` selects all supported
events, and omitting `tools` selects all tools (`["*"]`). Explicit empty selector
arrays are rejected; use `enabled: false` to disable an entry. `spec: {hooks: []}`
disables all configured observers, while leaving programmatic registration
available. Unknown types, events, options, duplicate names/selectors and blank
names are errors, including in disabled entries.

For example, register a warning-level observer for failures of one tool:

```yaml
type: hooks
spec:
  hooks:
    - name: service-tool-failures
      type: logging
      events: [tool_call.error]
      tools: [get_service_detail]
      config:
        level: warn
```

Logging defaults to `info`, respects the application's log-level threshold,
and includes the configured instance name as `hook`. It records metadata only;
`logging.config.capture_content` is not supported. The existing Go
`NewLoggingHook` API continues to use the info level.

Only one tracing entry may be configured, even if disabled. It always observes
all lifecycle events and tools to preserve complete span trees. Omit its
`events` and `tools` selectors, or provide the full event set and `["*"]`.
Partial selections are rejected before creating an exporter. Tracing parameters
default to gRPC, service name `dubbo-admin-ai`, sampling ratio `1.0`, and content
capture `none`. An explicit sampling ratio of `0` is preserved.

### Compatibility with the previous configuration

Existing `spec.logging` and `spec.tracing` configurations remain supported:

```yaml
type: hooks
spec:
  logging:
    enabled: true
  tracing:
    enabled: false
```

The configuration loader preserves legacy defaults: `spec: {}` enables logging
and disables tracing. New-list entries do not inherit an extra legacy logger.
Do not combine `hooks` with `logging` or `tracing` in the same `spec`; mixed
formats are rejected even when the legacy block is empty or disabled. Both
formats use the same registration and shutdown paths. Direct Go construction
retains its existing zero-value behavior: `NewComponent(Spec{})` enables neither
built-in observer.

## Exporting traces

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

Hooks observe lifecycle events. `hooks.schema.json` enumerates these 16 values
for `events`; Go callers use the `Event` constants or `AllEvents()`:

- **Interaction**: `interaction.start`, `interaction.end`, `interaction.error`, `interaction.cancel`, `interaction.degrade`
- **Iteration**: `iteration.start`, `iteration.end`
- **Stage**: `stage.start`, `stage.end`, `stage.error`
- **Model call**: `model_call.start`, `model_call.end`, `model_call.error`
- **Tool call**: `tool_call.start`, `tool_call.end`, `tool_call.error`

`model_call.chunk` is reserved and is not emitted or accepted in subscriptions.

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
provide additional context. Logging subscriptions can be selected in YAML or
in Go. Tracing keeps the complete event set; Go observers use
`Registration.Events` and `Registration.Tools`.

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

The opt-in integration test loads hook registrations through the configuration
loader, checks logging filters, flushes traces on shutdown, and verifies the
span tree through Jaeger's Query API:

```shell
DUBBO_ADMIN_OTEL_E2E=1 \
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 \
go test -run TestJaegerOTLPEndToEnd -v ./component/hooks
```

To verify the HTTP exporter against the same Jaeger instance:

```shell
DUBBO_ADMIN_OTEL_E2E=1 \
DUBBO_ADMIN_OTEL_E2E_PROTOCOL=http/protobuf \
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 \
go test -run TestJaegerOTLPEndToEnd -v ./component/hooks
```

Set `JAEGER_QUERY_URL` when the query UI is served somewhere other than
`http://localhost:16686`.

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

Go tool-call registrations must explicitly select one or more exact tool names.
Use `hooks.AllTools` (`"*"`) when a hook intentionally observes every tool.

A registration that returns a context used by nested Agent work must set
`DerivesContext: true`. A Manager accepts one such registration because a Go
context can carry only one active span lineage; use an OpenTelemetry Collector
to fan that trace out to multiple backends. Contexts returned by ordinary
observational hooks are ignored. Context-deriving registrations must include
both the Start and End event for every selected lifecycle. Use
`hooks.NewTracingRegistration` for OpenTelemetry so the complete, paired event
set and content-capture behavior cannot be configured inconsistently.
