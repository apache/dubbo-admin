# Dubbo Java and Go metric compatibility

Dubbo Admin uses the same Application, Service, and Instance metric dashboards for dubbo-java and dubbo-go. The shared query identities are `application_name` for an application and `interface` for a service. `method` is an optional service refinement: an interface-only query remains valid. Provider and consumer data come from their respective `dubbo_provider_*` and `dubbo_consumer_*` metric families; there is no shared `side` label.

## Instance resolution

The Instance Metric dashboard selects data by application name and native, bare IP address. Prometheus's `instance` label, commonly an `IP:port` scrape target, is an internal association key, not the dashboard's instance selection parameter.

The shared Instance dashboard supports two paths:

1. If an RPC metric has `ip`, filter it directly by the selected bare IP. This is the path observed with dubbo-go 3.3.2.
2. If an RPC metric lacks `ip`, correlate `application_name` and the selected IP through `dubbo_application_info_total`, then match the resulting scrape target to the RPC metric on `(application_name, instance)`. The `instance` label is only an internal join key. This is the path observed with dubbo-java 3.2.16.

The dashboard combines the paths with PromQL `or`, which returns an identical series once if both paths find it. The bridge requires the application-info series to identify the correct scrape target. Do not guess a QoS, RPC, or exporter port from the bare IP.

This selection is IP-scoped, not process- or port-scoped. If multiple instances of the same application share an IP but use different ports, selecting that IP can include all of them. The direct-IP path cannot distinguish those processes, and the metadata path can match multiple scrape targets for the same application/IP. Use distinguishable IPs when validating instance isolation; this dashboard does not promise per-process isolation for a shared IP.

## Upgrade and deployment notes

The Instance Metric URL parameter `var-instance` changes from `IP:QoSPort` to bare IP. Deploy the updated Admin backend and Instance Metric dashboard together, and reload or redeploy Grafana's provisioned dashboard definitions. Mixing an old backend with the new dashboard, or a new backend with a dashboard expecting `IP:port`, can produce empty or incorrect results.

Update custom dashboards and saved links that treat `var-instance` as a scrape address. This parameter change does not rename or rewrite Prometheus's `instance` label, change application/service identity parameters, or configure scrape discovery.

## Scraping a dubbo-go application

Dashboard query compatibility does not configure a Go application or make Prometheus scrape it. In dubbo-go, the metrics master switch is disabled by default. Enable `metrics.enable`, use the Prometheus exporter, and expose its configured HTTP port and path (defaults: `9090` and `/metrics`). The exporter being enabled by default does not override the disabled metrics master switch.

For example, the dubbo-go application configuration can include:

```yaml
dubbo:
  metrics:
    enable: true
    protocol: prometheus
    port: 9090
    path: /metrics
    prometheus:
      exporter:
        enabled: true
```

The release's [`dubbo-serviceMonitor.yaml`](../release/kubernetes/monitoring/dubbo-serviceMonitor.yaml) selects Services labeled `org.apache.dubbo/service: dubbo-qos` and scrapes a Service port named `dubbo-qos`. It does not automatically discover a Go application's default `9090/metrics` endpoint. Expose the Go exporter through a Kubernetes Service and configure a Prometheus scrape target or ServiceMonitor that selects that Service and its actual named metrics port. An existing ServiceMonitor can be reused only if its selector and port name really match the Go Service; do not infer a scrape port from the Admin instance IP or QoS port. Point the Grafana dashboards at the Prometheus datasource that receives those samples.

Before testing Admin, check Prometheus targets and confirm `dubbo_application_info_total{application_name="<go-app>"}` and the relevant `dubbo_provider_requests_total` or `dubbo_consumer_requests_total` series. After an RPC call, verify that the series carries the expected application, `interface`, and native `ip` labels. A consumer-only application need not have provider request series.

## Metric capabilities

Request counters are the common basis for the dashboards. Other families, including processing and histograms, depend on what the running application actually exports; they are not inferred from a Java/Go runtime name. A missing optional family must not make the whole application appear to lack metric support.

This compatibility work does not align cross-language error or duration semantics, add metrics to dubbo-go, or change Trace-to-Metric links. Those are outside the scope of this dashboard compatibility work.

## Verification scope

The compatibility targets are dubbo-java 3.2.0 and later and dubbo-go 3.1.1 and later. Source inspection at those starting tags supports the identity and metric-family assumptions. Runtime checks used dubbo-java 3.2.16 and dubbo-go 3.3.2; they do not constitute E2E verification of every release in either range.

For the tested Java runtime, RPC request series lacked native `ip`, while application-info series included `application_name`, `ip`, and the scrape `instance`. Two `shop-detail` IPs resolved to distinct RPC series through the bridge. For the tested Go runtime, RPC series contained `application_name` and `ip`, and direct-IP queries returned the fixture's real provider and consumer request data. Shared dashboards were checked against Grafana 12.1.0 and Prometheus data. Browser acceptance and a complete Go discovery-to-Admin-to-Grafana journey should be recorded separately from these API checks.

### Recorded runtime query results

The following summarizes the historical API/query observations above, not a new runtime run of the network/codec summary fix:

| Runtime | Observed labels and query path | Recorded result |
| --- | --- | --- |
| dubbo-java 3.2.16 | RPC request series lacked `ip`; `dubbo_application_info_total` supplied `application_name`, `ip`, and scrape `instance`; RPC series were matched on `(application_name, instance)` | The two tested `shop-detail` IPs resolved to distinct RPC series through the metadata bridge. |
| dubbo-go 3.3.2 | RPC series supplied `application_name` and native `ip`; queries selected them directly | The fixture's real provider and consumer request data were returned. |

These observations do not establish network/codec counter behavior, shared-IP process isolation, or all-version compatibility. For runtime acceptance of the error summaries, record the concrete Java/Go label sets, selected application/IP, rendered PromQL, time range, and returned values for two distinguishable instances. A zero fallback alone is not evidence of successful isolation: use nonzero samples in the selected and excluded instances, or explicitly record that an optional metric is not exported. Retain the corresponding Grafana screenshots or query results with the PR's acceptance evidence.

### Automated dashboard contract check

```sh
go test ./release/kubernetes/monitoring -run '^TestInstanceMetricQueriesScopeSelectedIP$' -count=1 -v
go test ./pkg/console/service ./pkg/console/handler -count=1
```

The dashboard check parses the provisioning YAML and embedded dashboard JSON, inspects every Instance Metric query expression (not only those already containing `$instance`), and checks that the provider network/codec summaries retain both selection paths and the zero fallback. It is a structural regression test, not a PromQL evaluator or browser acceptance test.

A complete product acceptance should use a registered dubbo-go application with two distinguishable instances: confirm Prometheus ingestion, open its Application, Instance, and Service Metric views from Admin, trigger real RPCs, verify provider and/or consumer data, and verify that selecting one instance excludes the other's series. Repeat representative views with dubbo-java to check for regression. These browser and discovery checks have not been claimed by the runtime query checks above.
