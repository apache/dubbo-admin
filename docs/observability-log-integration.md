# Dubbo Admin Log integration

Dubbo Admin provides connect-only views for Application Log and Instance Log.
It does not deploy or operate a logging backend. Users must provide Grafana,
configure a Loki datasource in Grafana, and operate an external log collection
pipeline that makes the required fields queryable.

The minimum log contract is:

- `app` is the Loki index label used to select an application.
- `pod_ip` is query-visible structured metadata and is preferred for instance
  filtering.
- The Loki entry timestamp is retained for time-range queries.
- The complete raw log line is retained as the log message.
- `trace_id` is optional source data. When present it is preserved, but it is
  not a Loki index label.

The project validation environment uses OTel Collector 0.112.0, Loki 3.4.2,
and Grafana 12.1.0. These versions describe the validation path; they are not
bundled into the Dubbo Admin release.

## Elasticsearch provider

The release also provisions Application and Instance Elasticsearch Log
Dashboards. The deployment remains connect-only: an administrator supplies a
Grafana Elasticsearch datasource and the Elasticsearch index or data stream.
No datasource UID, endpoint, credentials, or index is embedded in the
Dashboards. The default Admin `console.logDashboards` URLs continue to select
Loki. To select Elasticsearch, replace all three URLs with the commented
Elasticsearch alternatives beside `logDashboards` in
`release/kubernetes/dubbo-system/dubbo-admin.yaml`. The `trace` URL reuses the
Application Elasticsearch Log Dashboard; there is no third dashboard or
provider switch in Admin.

The validated Elasticsearch path uses Elasticsearch 8.18.0 and OTel Collector
Contrib 0.112.0 with the Elasticsearch exporter `mapping.mode = raw`. The
OTel LogRecords carry `app`, `pod_ip`, `trace_id` (when present), `level`,
`pod_name`, and `logger` as attributes. Raw mapping writes
these as top-level fields, writes the timestamp to `@timestamp`, and preserves
the complete original message in `Body`. Application and Instance queries use
`app.keyword` and `pod_ip.keyword`; the unescaped `applicationRaw` URL variable
is used for exact application matching, including names such as `shop.user`.
Manual Trace ID searches resolve the Trace start time through Jaeger, then
query `trace_id.keyword` within T±5 minutes. They do not scan `Body`.

This first version does not use an ECS adapter. The Elasticsearch index or
data stream is an environment choice configured in Grafana; Admin does not
access Elasticsearch directly. The official release does not deploy
Elasticsearch, a production OTel logs pipeline, Filebeat, or Logstash.
Service Log, namespace Log, cluster Log, Trace-to-Log navigation,
Log-to-Trace navigation, and Trace to Metric are outside this integration.
