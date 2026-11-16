# Dubbo Admin Log integration

## Architecture and boundary

Dubbo Application (pod stdout/file log) → OTel Collector logs pipeline →
Loki **or** Elasticsearch → Grafana datasource and Dashboard → Dubbo Admin.

Dubbo Admin is **connect-only**. Its release provides Log pages, Dashboard
definitions, Grafana provisioning, and Dashboard URLs. It does **not** deploy
or manage production Loki, Elasticsearch, an OTel logs pipeline, Filebeat, or
Logstash. The release's existing OTel Collector configuration is for traces;
do not replace it with these examples. No ECS adapter, Service Log,
Trace-to-Log, or Log-to-Trace navigation is part of this integration.

These files are **reference configurations only**, derived from the project's
validation harness, not release manifests:

- [Loki Collector example](examples/observability/otel-collector-loki.yaml)
- [Elasticsearch Collector example](examples/observability/otel-collector-elasticsearch.yaml)

An optional [Elasticsearch index template](examples/observability/elasticsearch-index-template.json)
is also provided for the raw log field contract. It is not installed by the
release or Collector; the Elasticsearch administrator owns its installation.

Both Collector examples target OTel Collector Contrib 0.112.0, Kubernetes pod stdout files under
`/var/log/pods`, and namespace `dubbo-samples-shop`. Adapt the namespace,
file glob, pod labels, endpoint, index, and credentials. Deploy **one**
example for the selected backend. `start_at: end` ingests new lines only;
change to `beginning` temporarily if historical files are needed.

## Shared field contract

| Field | Requirement | Source and use |
| --- | --- | --- |
| `app` | Required | Pod label `app`; Application Log filter. |
| `pod_ip` | Required | Pod IP from `k8sattributes`; with `app` filters Instance Log. It must equal the IP known to Admin. |
| timestamp | Required | Container log timestamp retained by OTel; becomes ES `@timestamp`. |
| raw log | Required | Complete original application line retained in OTel `Body`; displayed as the message. |
| `trace_id` | Optional | Lowercase 16- or 32-character hex ID for manual Log-page search; never a Loki index label. |
| `level` | Optional | Parsed severity, if available. |
| `pod_name` | Optional | Kubernetes pod name for display. |
| `logger` | Optional | Parsed logger name, if available. |

The `filelog` receiver reads pod stdout container files. The `container`
operator obtains timestamp and pod identity. JSON and regex parsers add
attributes without replacing raw `Body`. `k8sattributes` needs pod `get`,
`list`, and `watch` permissions in the application namespace and pod UID
association. The pod must carry an `app` label matching its Admin application
name. Adapt parsing operators for other log formats without changing the
field contract.

## Deploy a reference Collector

Operate the logging backend separately. Install OTel Collector **Contrib**
0.112.0 as a DaemonSet, mounting host `/var/log/pods` read-only at the same
path. Give its service account a Role and RoleBinding allowing `get`, `list`,
and `watch` of `pods` in the application namespace. Mount the selected file
as `/etc/otelcol-contrib/config.yaml` (for example, create a ConfigMap with
`--from-file=config.yaml=<chosen-example>`), and start the Collector with
`--config=/etc/otelcol-contrib/config.yaml`. The example backend URLs are
placeholders; supply reachable endpoints and production TLS/authentication.
Do not apply these files to the release's trace Collector.

For example, from the repository root, create the configuration in your
existing logging namespace (choose the Elasticsearch file instead for ES):

```sh
kubectl -n logging create configmap dubbo-log-collector-reference \
  --from-file=config.yaml=docs/examples/observability/otel-collector-loki.yaml \
  --dry-run=client -o yaml | kubectl apply -f -
```

Reference this ConfigMap in your separately managed Collector DaemonSet.
This command creates configuration only; it does not install the DaemonSet,
RBAC or backend. With a Contrib 0.112.0 binary available, validate the chosen
file before deploying it:

```sh
otelcol-contrib validate --config=docs/examples/observability/otel-collector-loki.yaml
otelcol-contrib validate --config=docs/examples/observability/otel-collector-elasticsearch.yaml
```

Before connecting Grafana, confirm a newly emitted line reaches the backend
with `app`, `pod_ip`, timestamp, and the full raw `Body`.

## Loki setup

1. Start with `otel-collector-loki.yaml`. Set its file glob, namespace, app
   label and `otlphttp/loki.endpoint` (example:
   `http://loki.logging.svc:3100/otlp`). Loki 3.4.2 was used for validation.
2. Configure Loki's OTLP mapping: `app` is an **index label**, `pod_ip` is
   **query-visible structured metadata**, and optional `trace_id` is
   structured metadata, **not** an index label. The validated mapping is:

   Structured metadata requires a compatible Loki schema (TSDB, schema v13
   in the validation environment). Merge the following into your externally
   managed Loki configuration; it is not a complete Loki deployment file.

   ```yaml
   limits_config:
     allow_structured_metadata: true
     otlp_config:
       resource_attributes:
         ignore_defaults: true
         attributes_config:
           - action: index_label
             attributes: [app]
           - action: structured_metadata
             attributes: [pod_ip, k8s.pod.name, k8s.pod.uid, k8s.namespace.name]
       log_attributes:
         - action: structured_metadata
           attributes: [trace_id, level, pod_name, logger]
   ```

3. Add a Grafana Loki datasource pointing at your Loki query endpoint.
   Confirm `{app="<your-app>"}` returns lines and `pod_ip` is available
   for instance filtering. Set the Loki Admin URLs below. `trace` reuses the
   Application Dashboard; a Trace ID may also be found in the raw line.

## Elasticsearch setup

1. Run Elasticsearch 8.18.0. Use `otel-collector-elasticsearch.yaml` with
   OTel Collector Contrib 0.112.0. Set its file glob, namespace, app label,
   `elasticsearch.endpoints`, `logs_index`, and authentication/TLS.
   `elasticsearchexporter` must use `mapping.mode: raw`; do not add an ECS
   adapter or convert `Body` into `message`.
2. Raw mapping writes timestamp to `@timestamp`, unchanged body to `Body`,
   and `app`, `pod_ip`, `trace_id`, `level`, `pod_name`, `logger` as top-level
   LogRecord attributes when present. `app` and `pod_ip` are required.
   Ensure the ES mapping exposes `app.keyword`, `pod_ip.keyword`, and
   `trace_id.keyword` (if present): Dashboards use these exact filters and
   do not search `Body` for Trace IDs. A dotted app such as `shop.user` uses
   Grafana's unescaped `applicationRaw` variable for exact matching.
3. Add a Grafana Elasticsearch datasource with the appropriate endpoint,
   index/data-stream pattern, and `@timestamp` time field. Confirm new
   documents are queryable, then replace **all three** Admin URLs below.
   There is no third ES Dashboard or provider switch.

### Optional Elasticsearch index template

`elasticsearch-index-template.json` is a composable index template for
Elasticsearch 8.18.0 and the Collector's example `logs_index: dubbo-logs`.
It matches only `dubbo-logs` and `dubbo-logs-*`. It maps `@timestamp` as
`date`, preserves the original `Body` in `_source` as a `text` field, and
provides the `app.keyword`, `pod_ip.keyword`, and `trace_id.keyword` exact
match fields required by the Dashboards. `level`, `pod_name`, and `logger`
are `keyword` fields. No ECS conversion or Admin-side ES access is added.

Before installing, adapt the index patterns and inspect your existing
templates. Composable templates select the highest matching priority; this
example uses priority `100` and does not merge another matching index
template. A same-priority overlap can reject installation. Do not supersede
an existing production template: merge the required fields into your own
template instead. This example targets ordinary indices, not data streams;
data-stream deployments must use their separately managed templates.

From the repository root, use a reachable ES URL and add your environment's
authentication/TLS options to these commands. Installation is optional and
manual. `create=true` refuses to replace a template with the same name:

```sh
ES_URL=http://127.0.0.1:9200
curl --fail-with-body "$ES_URL/_index_template?pretty"
curl --fail-with-body -X PUT \
  "$ES_URL/_index_template/dubbo-admin-log-reference?create=true" \
  -H 'Content-Type: application/json' \
  --data-binary @docs/examples/observability/elasticsearch-index-template.json
curl --fail-with-body -X POST \
  "$ES_URL/_index_template/_simulate_index/dubbo-logs?pretty"
```

Check the simulation's effective mapping before starting ingestion into a
new index. Installing a template does not create an index, alter existing
indices, or reindex documents. If `dubbo-logs` already exists with incompatible
fields, use an administrator-approved new matching index and update the
Collector and Grafana index pattern, or plan a migration separately.

After new logs arrive, check the actual index fields (not just the template):

```sh
curl --fail-with-body \
  "$ES_URL/dubbo-logs*/_field_caps?fields=@timestamp,Body,app.keyword,pod_ip.keyword,trace_id.keyword,level,pod_name,logger&pretty"
```

Expect `@timestamp` to be `date`, `Body` to be `text`, and the other listed
fields to be `keyword` and searchable, without conflicting types across
the indices selected by Grafana. A missing `trace_id` value on an individual
log is allowed; a log must carry that value to support Trace ID search.

## Dubbo Admin configuration

Set `console.logDashboards.application`, `console.logDashboards.instance`,
and `console.logDashboards.trace` in Admin configuration. The release example
is `release/kubernetes/dubbo-system/dubbo-admin.yaml`. Grafana Dashboards
are provisioned by the release, but you must supply the selected datasource.
Replace the Grafana host if needed: embedded URLs must be reachable by the
user's browser, not only by the Admin server.

Default Loki URLs:

```yaml
console:
  logDashboards:
    application: http://grafana.monitoring.svc:3000/d/dubbo-application-loki-logs/dubbo-application-logs?theme=light
    instance: http://grafana.monitoring.svc:3000/d/dubbo-instance-loki-logs/dubbo-instance-logs?theme=light
    trace: http://grafana.monitoring.svc:3000/d/dubbo-application-loki-logs/dubbo-application-logs?theme=light
```

Elasticsearch alternatives (replace the whole group):

```yaml
console:
  logDashboards:
    application: http://grafana.monitoring.svc:3000/d/dubbo-application-elasticsearch-logs/dubbo-application-elasticsearch-logs?theme=light
    instance: http://grafana.monitoring.svc:3000/d/dubbo-instance-elasticsearch-logs/dubbo-instance-elasticsearch-logs?theme=light
    trace: http://grafana.monitoring.svc:3000/d/dubbo-application-elasticsearch-logs/dubbo-application-elasticsearch-logs?theme=light
```

Admin supplies application and instance variables. Manual Trace ID search
gets the Trace start time from Jaeger and opens the selected log Dashboard
over T±5 minutes. Jaeger query access is needed for this optional search,
not for ordinary Log views.

Manual search accepts ranges no longer than ten minutes, with an end time
at most five minutes ahead of the Admin server's current time. This allows
the full T±5-minute window for a just-created trace while rejecting clearly
future ranges. Accepted ranges are not clipped; keep server clocks synchronized.

Set `observability.jaeger` to the Jaeger **query HTTP** base URL reachable
by the Admin server (for example, `http://127.0.0.1:16686`), not its OTLP
ingestion endpoint. The query client has a five-second timeout and accepts
at most 16 MiB per trace response; oversized responses are reported as
query failures, not as missing traces. The local example in
`app/dubbo-admin/dubbo-admin.yaml` includes optional Jaeger and Log settings.

### Dashboard variable contract

The two Loki Dashboards intentionally consume different representations
of `var-application`. These are existing contracts, not provider detection:

| Entry | Value supplied by Admin | Loki interpolation |
| --- | --- | --- |
| Application Log | `regexp.QuoteMeta(app)`, then `strconv.Quote` with the outer quotes removed | `${application:raw}` inside a quoted LogQL regex matcher |
| Instance Log | Original application name | `${application:regex}` inside a quoted LogQL regex matcher |
| Both Log entries | `var-applicationRaw` is the original application name | Not used by the Loki Dashboards; available for exact-match Dashboards |

For example, for application `shop.user`, the decoded Application Log URL
parameter contains `shop\\.user` (two backslashes), whereas Instance Log
receives `shop.user`. The Application value is escaped for both the regex
and its surrounding LogQL string; the Instance Dashboard delegates regex
formatting to Grafana. `var-instance` is the bare Pod IP, without a QoS port.
Do not substitute one representation for the other when customizing a
Dashboard. URL percent-encoding is separate from these query escapes.

`console.logDashboards.trace` is used only for manual Trace ID search on a
Log page. Although `GrafanaDashboardConfig` is shared, its `trace` field is
not a Metric or Trace Dashboard setting. It normally reuses the Application
Log Dashboard in trace-focused mode; a separate URL setting allows operators
to override that target without changing the ordinary Application view.
There is no third Dashboard and no automatic fallback when `trace` is absent:
manual search reports that its Dashboard is not configured. The handler
checks for the Log configuration object and the service checks its target
URL, so both direct service calls and HTTP requests remain protected.

### Elasticsearch exact-match constraints

The ES Dashboards use Lucene quoted exact-match filters, for example
`app.keyword:"${applicationRaw:raw}"`. Dots and hyphens are supported, but
application names containing a double quote (`"`) or backslash (`\`) are
**not supported by these templates**. `applicationRaw` deliberately remains
provider-neutral and unescaped; Admin does not apply ES-specific escaping.
Use a supported application identity or an independently adapted Dashboard
for those names; do not change the shared identity or the Loki variables.

Manual Trace ID search sends `var-application=.*`, `var-instance=.*`, and
`var-traceId`, but does not send `var-applicationRaw`. The ES Application
Dashboard query is:

```text
app.keyword:"${applicationRaw:raw}" OR trace_id.keyword:"${traceId:raw}"
```

The Instance query has the same `OR trace_id.keyword` branch after its
application/IP filter. In trace-focused mode, `applicationRaw` defaults to
`.*`; this is a literal value inside a quoted keyword filter, **not** a
match-all expression. The `OR trace_id` branch enables the exact Trace ID
search without an application or instance restriction. Trace IDs are
validated as hexadecimal before URL generation. Outside trace-focused mode,
the default `traceId=.*` is also a literal keyword value, not a match-all
expression; it does not replace the ordinary application filter.

Admin lowercases accepted Trace IDs before Jaeger lookup and Log URL
generation. Configure ingestion to store ES `trace_id` values in lowercase
as well: keyword matching is case-sensitive, and historical uppercase values
are not automatically converted. The reference parsing examples do not
themselves normalize uppercase IDs. A zero-padded 32-character ID and its
16-character equivalent compare equally for Jaeger response validation,
but Log queries retain the input width; the stored ES value must use that
same width. No `Body` full-text fallback is performed.

### Shared Grafana embedding behavior

The shared `GrafanaPage.vue` is used by Metric, Trace, and Log views. It no
longer accesses the iframe's cross-origin DOM or clips a fixed-size frame
with negative offsets. It sizes the frame from its container, observes
container/window resizing, and adds `kiosk=1` only when the URL has no
`kiosk` parameter (explicit values are preserved). Regression checks should
include existing Metric and Trace pages as well as the new Log pages.

## Minimum verification

1. Emit new logs from two pods with the same `app`: Application Log shows both.
2. Open one Instance Log: only the target `pod_ip` appears.
3. Compare source stdout with the Dashboard message: the raw log is complete.
4. When a valid `trace_id` exists, manual Log-page search finds the line in
   the Jaeger-derived T±5-minute range.
5. A normal log without `trace_id` remains visible in ordinary Log views.

If a view is empty, inspect backend records and Grafana datasource health,
then compare `app` and `pod_ip` with Admin identity. For ES, check `.keyword`
fields; for Loki, check the `app` label and `pod_ip` structured metadata.
