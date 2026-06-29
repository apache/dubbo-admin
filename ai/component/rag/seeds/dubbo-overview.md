# Apache Dubbo Overview

Apache Dubbo is a high-performance, open-source RPC (Remote Procedure Call)
framework for building microservice applications. It originated at Alibaba and
is now a top-level Apache project. Dubbo handles service definition, service
discovery, remote communication, load balancing, and traffic governance so that
distributed services can call each other as if they were local methods.

## Core Concepts

- **Provider**: a service that exposes one or more remote interfaces.
- **Consumer**: a service that invokes a remote interface exposed by a provider.
- **Registry**: a coordination center (such as Nacos, ZooKeeper, or Kubernetes)
  where providers register their addresses and consumers subscribe to discover
  them.
- **Invoker**: the runtime abstraction of an invokable service used internally
  for RPC calls.
- **Protocol**: the wire protocol used between consumer and provider. Dubbo
  supports the `dubbo`, `triple` (gRPC-compatible HTTP/2), `rest`, and other
  protocols.

## Key Features

- Transparent RPC with multiple serialization options (Hessian2, Protobuf, JSON,
  Fastjson2, Kryo).
- Built-in service discovery and address-list push from the registry.
- Client-side load balancing with strategies such as Random, RoundRobin,
  LeastActive, ConsistentHash, and ShortestResponse.
- Traffic governance: routing rules, tag routing, traffic weighting, and
  application-level / interface-level configuration overrides.
- Fault tolerance strategies: Failover, Failfast, Failsafe, Failback, Forking,
  and Broadcast.
- Observability through metrics, tracing, and the Dubbo Admin console.

## Typical Architecture

A consumer subscribes to the registry to obtain the live address list of a
provider, applies load balancing to pick one address, and sends the request over
the configured protocol. Configuration and governance rules are distributed
through the configuration center (often the same component as the registry,
e.g. Nacos) so that behavior can be changed at runtime without redeployment.
