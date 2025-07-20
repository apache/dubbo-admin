# Addons

Example deployments of various plugins that integrate with Dubbo.

The deployments here are intended to get up and running quickly and may not be suitable for production environments.

## Getting started

To quickly deploy all addons:

```shell script
kubectl apply -f samples/addons
```

Alternatively, you can deploy individual addons:

```shell script
kubectl apply -f samples/addons/prometheus.yaml
```