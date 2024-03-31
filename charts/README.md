# Apache Dubbo Admin

A Helm chart for Dubbo Admin The ops and reference implementation for Apache Dubbo.

## Prerequisites

* Kubernetes v1.14+
* Helm v3+

### Install

```bash
helm install [RELEASE_NAME] dubbo-admin --namespace dubbo-system --create-namespace
```

### Uninstall

```bash
helm delete [RELEASE_NAME] --namespace dubbo-system
```