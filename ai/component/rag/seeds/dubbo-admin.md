# Dubbo Admin Console

Dubbo Admin is the official web console for operating and governing Dubbo
service clusters. It connects to the registry, configuration center, and
metadata center to give operators a single place to observe and manage services.

## Main Capabilities

- **Service query**: list registered applications, services, providers, and
  consumers; inspect their metadata, methods, and parameters.
- **Instance / metadata view**: see live provider instances, their addresses,
  health, and the interfaces they expose (application-level discovery).
- **Traffic governance**: create and edit conditional routes, tag routes,
  weight adjustments, and dynamic configuration overrides through the UI.
- **Configuration management**: push and version governance rules to the
  configuration center (e.g. Nacos) so they take effect at runtime.
- **Service testing / mock**: invoke service methods directly from the console
  and configure mock responses.
- **Observability**: surface metrics and the relationships (dependencies)
  between services.

## Deployment

Dubbo Admin typically runs as a standalone service (a Go/Java backend plus a
Vue front end). It is configured with the addresses of the registry and
configuration center it should manage:

```yaml
admin:
  registry:
    address: nacos://127.0.0.1:8848
  config-center:
    address: nacos://127.0.0.1:8848
  metadata-report:
    address: nacos://127.0.0.1:8848
```

## Common Operational Tasks

- **Diagnose a service that has no available providers**: open the service view,
  confirm providers are registered and healthy, and check that consumer and
  provider share the same group/version and registry namespace.
- **Roll out a gray release**: tag the new instances, create a tag-routing rule,
  then gradually shift weight to the tagged group.
- **Change behavior without redeploying**: use dynamic configuration overrides
  (timeout, retries, load balance) pushed through the configuration center.
