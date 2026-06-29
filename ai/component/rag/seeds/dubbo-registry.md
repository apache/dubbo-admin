# Dubbo Service Discovery and Registry

Dubbo decouples service consumers from providers through a **registry**. Providers
register their runtime addresses; consumers subscribe and receive a live,
push-updated address list. When a provider scales up, scales down, or fails, the
registry notifies consumers so the address list stays current.

## Supported Registries

- **Nacos** (recommended): also serves as a configuration center and metadata
  center. Address form: `nacos://host:8848`.
- **ZooKeeper**: classic registry, strongly consistent. Address form:
  `zookeeper://host:2181`.
- **Kubernetes**: Dubbo can use the Kubernetes API server / native Service
  discovery for registration-free deployment.
- **Redis**, **Consul**, and others via SPI extensions.

## Application-Level vs Interface-Level Discovery

Dubbo 3 introduced **application-level service discovery**. Instead of
registering one entry per interface (interface-level, used in Dubbo 2.x), an
instance registers once per application. This drastically reduces the data
pushed by the registry and improves scalability for large clusters. Interface
metadata is stored separately in a **metadata center** and fetched on demand.

## Configuration Example

```yaml
dubbo:
  registry:
    address: nacos://127.0.0.1:8848
    # username/password optional
  application:
    name: my-service
    # register-mode: instance | interface | all  (default: instance in Dubbo 3)
```

## Troubleshooting Discovery Issues

- **Consumer cannot find provider**: verify both use the same registry address
  and namespace/group, and that the provider actually registered (check the
  registry's service list or the Dubbo Admin instance view).
- **Stale addresses**: confirm the registry push is working and that the
  provider's heartbeat/health is reported; restart the provider if its session
  expired.
- **Mixed Dubbo 2 / Dubbo 3**: set a compatible `register-mode` (such as `all`)
  so older consumers relying on interface-level discovery still work.
