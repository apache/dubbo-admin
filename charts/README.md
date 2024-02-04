## Apache Dubbo Admin for Kubernetes

### Install

```bash
helm install [RELEASE_NAME] dubbo-admin --namespace dubbo-system --create-namespace
```

### Uninstall

```bash
helm delete [RELEASE_NAME] --namespace dubbo-system
```