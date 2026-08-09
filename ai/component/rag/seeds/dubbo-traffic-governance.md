# Dubbo Traffic Governance

Traffic governance lets you control how requests flow between consumers and
providers at runtime, without changing or redeploying application code. Rules are
distributed through the configuration center and take effect dynamically.

## Routing Rules

- **Conditional routing**: route requests based on conditions on the request
  (method, arguments) or the consumer (host, application). Example: send all
  requests from a gray consumer group to providers tagged `gray`.
- **Tag routing**: tag provider instances (e.g. `tag=gray`) and direct tagged
  traffic to them, used for canary / gray releases.
- **Mesh / virtual-service style rules**: Dubbo 3 supports rules modeled after
  service-mesh `VirtualService` and `DestinationRule` for weighted traffic
  splitting.

## Load Balancing

Client-side strategies selected per service or method:

- **Random** (default): weighted random selection.
- **RoundRobin**: weighted round robin.
- **LeastActive**: favors providers with fewer active calls (faster responders).
- **ConsistentHash**: same arguments always map to the same provider.
- **ShortestResponse**: favors providers with the shortest recent response time.

## Fault Tolerance (Cluster Strategies)

- **Failover** (default): retry other providers on failure (`retries=2` by
  default). Suitable for idempotent reads.
- **Failfast**: fail immediately, no retry. Use for non-idempotent writes.
- **Failsafe**: ignore failures, return empty. Use for non-critical operations
  like audit logs.
- **Failback**: record failed calls and retry them in the background.
- **Forking**: call multiple providers in parallel, return the first success.
- **Broadcast**: call all providers; fails if any fails.

## Rate Limiting and Circuit Breaking

Dubbo integrates with **Sentinel** for flow control, circuit breaking, and
system-load protection. You can also configure provider-side
`executes` (max concurrent invocations) and consumer-side `actives` limits.

## Example: Weighted Gray Release via Tag Routing

```yaml
configVersion: v3.0
force: false
enabled: true
key: my-service
tags:
  - name: gray
    match:
      - key: env
        value:
          exact: gray
```
