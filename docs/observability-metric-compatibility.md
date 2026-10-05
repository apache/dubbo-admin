# Dubbo Java and Go metric compatibility

Dubbo Admin uses the same Application, Service, and Instance metric dashboards for dubbo-java and dubbo-go. The shared query identities are `application_name` for an application and `interface` for a service. `method` is an optional service refinement: an interface-only query remains valid. Provider and consumer data come from their respective `dubbo_provider_*` and `dubbo_consumer_*` metric families; there is no shared `side` label.

## Instance resolution

The product identity of a Dubbo instance is its native, bare IP address. Prometheus's `instance` label, commonly an `IP:port` scrape target, is not the product identity and must not be shown or requested as one.

The shared Instance dashboard supports two paths:

1. If an RPC metric has `ip`, filter it directly by the selected bare IP. This is the path observed with dubbo-go 3.3.2.
2. If an RPC metric lacks `ip`, correlate `application_name` and the selected IP through `dubbo_application_info_total`, then match the resulting scrape target to the RPC metric on `(application_name, instance)`. The `instance` label is only an internal join key. This is the path observed with dubbo-java 3.2.16.

The dashboard combines the paths with PromQL `or`, which returns an identical series once if both paths find it. The bridge requires the application-info series to identify the correct scrape target. Do not guess a QoS, RPC, or exporter port from the bare IP.

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

This compatibility work does not align cross-language error or duration semantics, add metrics to dubbo-go, or change Trace-to-Metric links. Those are outside D009.

## Verification scope

The compatibility targets are dubbo-java 3.2.0 and later and dubbo-go 3.1.1 and later. Source inspection at those starting tags supports the identity and metric-family assumptions. Runtime checks used dubbo-java 3.2.16 and dubbo-go 3.3.2; they do not constitute E2E verification of every release in either range.

For the tested Java runtime, RPC request series lacked native `ip`, while application-info series included `application_name`, `ip`, and the scrape `instance`. Two `shop-detail` IPs resolved to distinct RPC series through the bridge. For the tested Go runtime, RPC series contained `application_name` and `ip`, and direct-IP queries returned the fixture's real provider and consumer request data. Shared dashboards were checked against Grafana 12.1.0 and Prometheus data. Browser acceptance and a complete Go discovery-to-Admin-to-Grafana journey should be recorded separately from these API checks.

A complete product acceptance should use a registered dubbo-go application with two distinguishable instances: confirm Prometheus ingestion, open its Application, Instance, and Service Metric views from Admin, trigger real RPCs, verify provider and/or consumer data, and verify that selecting one instance excludes the other's series. Repeat representative views with dubbo-java to check for regression. These browser and discovery checks have not been claimed by the runtime query checks above.
