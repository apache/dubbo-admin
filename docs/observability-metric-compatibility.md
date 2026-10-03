# Dubbo Java and Go metric compatibility

## Instance resolution

The product identity of a Dubbo instance is its native, bare IP address. Prometheus's `instance` label, commonly an `IP:port` scrape target, is not the product identity and must not be shown or requested as one.

The shared Instance dashboard supports two paths:

1. If an RPC metric has `ip`, filter it directly by the selected bare IP. This is the path observed with dubbo-go 3.3.2.
2. If an RPC metric lacks `ip`, correlate `application_name` and the selected IP through `dubbo_application_info_total`, then match the resulting scrape target to the RPC metric on `(application_name, instance)`. The `instance` label is only an internal join key. This is the path observed with dubbo-java 3.2.16.

The dashboard combines the paths with PromQL `or`, which returns an identical series once if both paths find it. The bridge requires the application-info series to identify the correct scrape target. Do not guess a QoS, RPC, or exporter port from the bare IP.
