# Dubbo Configuration Reference

Dubbo applications are commonly configured with Spring Boot `application.yml`
under the `dubbo:` prefix, or programmatically via the configuration API. The
same options can be overridden dynamically through the configuration center.

## Core Configuration Blocks

```yaml
dubbo:
  application:
    name: order-service          # required, application identity
    qos-enable: true             # online ops (QoS) port
  registry:
    address: nacos://127.0.0.1:8848
  protocol:
    name: tri                    # tri (triple/HTTP2), dubbo, rest
    port: 50051
  provider:
    timeout: 3000                # default call timeout (ms)
    retries: 0                   # retries for non-idempotent writes
    loadbalance: leastactive
  consumer:
    check: false                 # do not fail startup if provider absent
    timeout: 2000
```

## Frequently Tuned Options

- **timeout**: maximum time (ms) to wait for an RPC response. Set on provider as
  a default; consumers can override per reference/method.
- **retries**: number of additional attempts under the Failover strategy. Set to
  `0` for non-idempotent operations to avoid duplicate side effects.
- **loadbalance**: `random` (default), `roundrobin`, `leastactive`,
  `consistenthash`, `shortestresponse`.
- **cluster**: fault-tolerance strategy: `failover` (default), `failfast`,
  `failsafe`, `failback`, `forking`, `broadcast`.
- **version** / **group**: isolate multiple implementations of the same
  interface; consumer and provider must match.
- **serialization**: `hessian2` (default), `fastjson2`, `protobuf`, `kryo`.

## Configuration Priority

From highest to lowest precedence:

1. Method-level configuration.
2. Reference (consumer) / Service (provider) level.
3. Consumer / Provider global level.
4. Application / framework defaults.

Dynamic overrides pushed from the configuration center take effect at runtime
and override the static `application.yml` values, which is how Dubbo Admin
changes behavior without a redeploy.

## Common Issues

- **Consumer startup fails with "No provider available"**: set
  `dubbo.consumer.check=false`, or start the provider first.
- **Calls time out under load**: increase `timeout`, switch `loadbalance` to
  `leastactive` or `shortestresponse`, and check provider thread-pool / `executes`
  limits.
- **Duplicate writes after a timeout**: set `retries=0` and use the `failfast`
  cluster strategy for non-idempotent methods.
