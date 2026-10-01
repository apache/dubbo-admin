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

The official Dubbo Admin release does not deploy Loki, an OTel logs pipeline,
or Alloy. Service Log, Elasticsearch, namespace Log, cluster Log, and Trace to
Metric are outside this integration scope.
