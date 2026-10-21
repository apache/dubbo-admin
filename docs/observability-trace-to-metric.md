# Trace to metric correlation

Dubbo Admin's Application Trace view opens a Jaeger trace in Grafana Explore. On a Dubbo RPC span, Grafana's native `tracesToMetrics` links offer **Provider Request Rate** and **Consumer Request Rate**. Selecting a link opens Split Explore: the trace remains on the left and the corresponding Prometheus query appears on the right.

The metric is the aggregate request rate for that RPC around the span's start time, **not a metric belonging only to the selected trace**. Grafana queries the interval from five minutes before to five minutes after the span. Both links are offered on each Dubbo RPC span; the link does not infer provider or consumer side from `span.kind`.

## Data flow and identity

The existing OTel Collector traces pipeline sends spans to Jaeger. Its transform processor adds `dubbo.application` to business Dubbo RPC spans from the resource's `service.name`. It also normalizes the Dubbo RPC service identity for metric correlation without changing the original `rpc.service`, resource identity, or unrelated HTTP, database, and internal spans. Grafana maps span attributes to labels on existing Dubbo Prometheus metrics:

| Span attribute | Prometheus label |
| --- | --- |
| `dubbo.application` | `application_name` |
| `dubbo.metric.interface` | `interface` |
| `rpc.method` | `method` |

Some dubbo-go versions, including v3.1.1 and v3.3.2, can emit `rpc.service` as the legacy ServiceKey `[group/]interface[:version]`, while the corresponding metric uses the plain `interface` label. For non-empty group/version, the Collector extracts `dubbo.metric.interface`, `dubbo.metric.group`, and `dubbo.metric.version`. When a span already has explicit `dubbo.group` or `dubbo.version`, those attributes take precedence over parsed ServiceKey values. Missing group/version attributes stay absent. Pure interface names, including Java RPC services, remain unchanged. The upstream dubbo-go #3724 change improves the source semantics; Collector normalization provides compatibility with versions that emit the legacy ServiceKey format, without requiring an SDK upgrade.

The Collector requires `rpc.system=apache_dubbo` and a non-empty `rpc.service` before adding normalized metric attributes. It also requires non-null `rpc.method` and resource `service.name` before adding `dubbo.application`. The datasource provisioning declares the stable Prometheus UID `dubbo-admin-prometheus`, which the Grafana Jaeger datasource references. Group/version are retained as span attributes but are not yet mapped into Grafana's `$__tags` selector: static provisioning evidence does not establish whether an absent optional span attribute is omitted rather than emitted as an empty matcher. Consequently the current links correlate by application, interface, and method, not by group/version. No spanmetrics pipeline or new RPC metric is required.

## V1 queries

```promql
sum by (application_name, interface, method) (
  rate(dubbo_provider_requests_total{$__tags}[5m])
)
```

```promql
sum by (application_name, interface, method) (
  rate(dubbo_consumer_requests_total{$__tags}[5m])
)
```

Grafana substitutes `$__tags` with the three label matchers from the selected span. In the datasource provisioning file, write `$$__tags` so Grafana's provisioning interpolation leaves `$__tags` in the runtime datasource. Use `spanStartTimeShift: "-5m"` and `spanEndTimeShift: "5m"`; Grafana's interval parser does not accept a leading `+` for the end shift.

## Verification scope and limits

### Apply to an existing Kubernetes deployment

Apply the Collector ConfigMap and Grafana datasource Secret in your chosen
cluster/context, then explicitly restart the two deployments. The Collector
configuration uses a `subPath` mount, which does not receive ConfigMap updates
in an existing container. Grafana reads datasource provisioning at startup;
updating the Secret alone does not reload it. These changes do not update
Deployment checksum annotations or trigger an automatic rollout.

```sh
kubectl --context YOUR_CONTEXT apply -f release/kubernetes/monitoring/otelCollector-configuration.yaml
kubectl --context YOUR_CONTEXT apply -f release/kubernetes/monitoring/grafana-dashboardDatasources.yaml
kubectl --context YOUR_CONTEXT -n monitoring rollout restart deployment/otel-collector deployment/grafana
kubectl --context YOUR_CONTEXT -n monitoring rollout status deployment/otel-collector --timeout=120s
kubectl --context YOUR_CONTEXT -n monitoring rollout status deployment/grafana --timeout=120s
```

Use your existing Grafana connection and authentication to read
`GET /api/datasources/uid/fe670vj54q48wc`. Confirm `jsonData.tracesToMetrics`
contains the three mappings above, the two queries, Prometheus UID
`dubbo-admin-prometheus`, and shifts `-5m` / `5m`. In the API response the
query contains `$__tags`, not the provisioning escape `$$__tags`. Also check
`GET /api/datasources/uid/fe670vj54q48wc/health` and
`GET /api/datasources/uid/dubbo-admin-prometheus/health`.

Emit a new RPC after rollout and read it from Jaeger. Confirm its span has
`dubbo.application` and the normalized `dubbo.metric.interface` alongside
unchanged `rpc.service` and `rpc.method`. Old stored traces are not enriched
retroactively. A Ready Pod or datasource health response alone does not prove
the new identity transform ran. Fresh installations load both configurations
on startup; the same API and new-span checks still apply.

### Repeatable local configuration and transform checks

The static check includes traces pipeline wiring, not only statement text:

```powershell
pwsh -File scripts/verify-trace-metric-servicekey.ps1
```

For actual OTTL execution, install Python 3 with PyYAML and supply a native
OTel Collector Contrib **0.112.0** binary:

```sh
python scripts/verify-trace-metric-collector.py --collector /path/to/otelcol-contrib
```

This validates the release config without starting its cluster-dependent
pipeline, then copies the actual release transform into an isolated Collector
with a loopback OTLP HTTP receiver and temporary file exporter. It verifies
Java/plain interfaces, legacy ServiceKeys, explicit group/version precedence,
missing identity, and non-Dubbo spans against the exported attributes. It does
not reimplement OTTL in Python, connect to Kubernetes/Jaeger/Prometheus, or
perform browser verification. Missing prerequisites are a failed verification,
not a PASS. The test stops only its own Collector process and removes its own
temporary files.

Mapping a legacy ServiceKey directly from `rpc.service` to the metric's plain `interface` label does not match when group/version are present; the normalized `dubbo.metric.interface` avoids that mismatch. The checks above cover static wiring, configuration validation and isolated execution of the normalization processor with Collector Contrib 0.112.0; they do not establish the full RPC → Jaeger → Grafana → Prometheus runtime or browser journey. Validate that end-to-end behavior with fresh RPCs in the target deployment. A selected RPC and time window may have no metric series when that side does not emit the corresponding counter, has insufficient Prometheus samples, or was not scraped.

V1 intentionally excludes success rate, error rate, latency/percentiles, instance metrics, exemplars, service graphs, and automatic side selection. Trace-to-metric does not change the separate Log integration or add a Dubbo Admin backend API or frontend RPC selector.
