# Nacos Helm Chart

* Apache [Nacos](https://nacos.io/) is committed to help you discover, configure, and manage your microservices. It provides a set of simple and useful features enabling you to realize dynamic service discovery, service configuration, service metadata and traffic management.


## Tips
If you use a custom database, please initialize the database [script](https://github.com/alibaba/nacos/blob/develop/distribution/conf/mysql-schema.sql)  yourself first.


## Installing the Chart

To install the my-release deployment:

```shell
helm install my-release nacos
```

## Uninstalling the Chart

To uninstall/delete the my-release deployment:

```shell
helm delete my-release
```

The command deploys Nacos on the Kubernetes cluster in the default configuration. It will run without a mysql chart and persistent volume. The [configuration](#configuration) section lists the parameters that can be configured during installation.

### Service & Configuration Management

#### Service registration

```shell
curl -X POST 'http://$NODE_IP:$NODE_PORT/nacos/v1/ns/instance?serviceName=nacos.naming.serviceName&ip=20.18.7.10&port=8080'
```

#### Service discovery

```shell
curl -X GET 'http://$NODE_IP:$NODE_PORT/nacos/v1/ns/instance/list?serviceName=nacos.naming.serviceName'
```

#### Publish config

```shell
curl -X POST "http://$NODE_IP:$NODE_PORT/nacos/v1/cs/configs?dataId=nacos.cfg.dataId&group=test&content=helloWorld"
```

#### Get config

```shell
curl -X GET "http://$NODE_IP:$NODE_PORT/nacos/v1/cs/configs?dataId=nacos.cfg.dataId&group=test"
```

## Configuration

The following table lists the configurable parameters of the Skywalking chart and their default values.

| Parameter                    | Description                                                                                            | Default                                                                                         |
|------------------------------|--------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------|
| `global.mode`                | Nacos Mode standalone or cluster.                                                                      | `standalone`                                                                                    |
| `nodeSelector`               | Nacos labels for pod assignment.                                                                       | `{}`                                                                                            |
| `affinity`                   | Nacos affinity policy.                                                                                 | `{}`                                                                                            |
| `tolerations`                | Nacos tolerations.                                                                                     | `{}`                                                                                            |
| `replicas`                   | Number of desired nacos pods, the number should be 1 as run standalone mode.                           | `1`                                                                                             |
| `maxUnavailable`             | The parameter for specifying the maximum allowed number of unavailable replicas.                       | `[]`                                                                                            |
| `tolerations`                | allow pods to be scheduled on nodes with specific taints by tolerating those taints.                   | `[]`                                                                                            |
| `domainName`                 | Enable Nacos cluster node domain name support.                                                         | `cluster.local`                                                                                 |
| `resources.limits`           | Resource Object Limits.                                                                                | `{}`                                                                                            |
| `resources.requests`         | Resource Object Requests.                                                                              | `{}`                                                                                            |
| `image.registry`             | Nacos container image registry.                                                                        | `nacos/nacos-server`                                                                            |
| `image.repository`           | Nacos container image repository.                                                                      | `nacos/nacos-server`                                                                            |
| `image.tag`                  | Nacos container image tag.                                                                             | `latest`                                                                                        |
| `image.pullPolicy`           | Nacos container image pull policy.                                                                     | `IfNotPresent`                                                                                  |
| `plugin.enabled`             | Nacos cluster plugin that is auto scale.                                                               | `true`                                                                                          |
| `plugin.image.repository`    | Nacos cluster plugin image name.                                                                       | `nacos/nacos-peer-finder-plugin`                                                                |
| `plugin.image.tag`           | Nacos cluster plugin image tag.                                                                        | `1.1`                                                                                           |
| `plugin.image.pullPolicy`    | Nacos cluster plugin image pull policy.                                                                | `IfNotPresent`                                                                                  |
| `health.enabled`             | Enable health check or not.                                                                            | `false`                                                                                         |
| `env.preferhostmode`         | Nacos env Preferred Host Mode.                                                                         | `hostname`                                                                                      |
| `env.serverPort`             | Nacos env Server Port.                                                                                 | `8848`                                                                                          |
| `storage.type`               | Nacos storage method `mysql` or `embedded`. The `embedded` supports either standalone or cluster mode. | `embedded`                                                                                      |
| `storage.db.host`            | Nacos Mysql storage host.                                                                              | `""`                                                                                            |
| `storage.db.name`            | Nacos Mysql storage database name.                                                                     | `""`                                                                                            |
| `storage.db.port`            | Nacos Mysql storage port.                                                                              | `3306`                                                                                          |
| `storage.db.username`        | Username of database.                                                                                  | `""`                                                                                            |
| `storage.db.password`        | Password of database.                                                                                  | `""`                                                                                            |
| `storage.db.param`           | Database url parameter.                                                                                | `characterEncoding=utf8&connectTimeout=1000&socketTimeout=3000&autoReconnect=true&useSSL=false` |
| `persistence.enabled`        | Enable or disable persistence functionality.                                                           | `false`                                                                                         |
| `persistence.accessModes`    | Defining Access Patterns for Persistent Storage Volume.                                                | `ReadWriteOnce`		                                                                               |
| `persistence.storageClassName` | Defines the type or class of persistent storage volume.                                                | `manual`                                                                                        |
| `persistence.size`           | Defines the size of the Kubernetes persistent storage volume.                                          | `5G`                                                                                            |
| `persistence.ClaimName`      | Define the name of the persistence store declaration to bind to.                                       | `{}`                                                                                            |
| `persistence.emptyDir`       | Configuration for creating a temporary blank directory.                                                | `{}`                                                                                            |
| `podDisruptionBudget.enabled` | Enable or disable podDisruptionBudget functionality.                                                   | `false`                                                                                         |
| `podDisruptionBudget.minAvailable` | Defines the minimum number of available Pods that must be maintained in a Kubernetes cluster.          | `1`                                                                                             |
| `podDisruptionBudget.maxUnavailable` | Defines the maximum number of available Pods that must be maintained in a Kubernetes cluster.          | `1`                                                                                             |
| `service.type`               | Nacos http service type.                                                                               | `NodePort`                                                                                      |
| `service.port`               | Nacos http service port.                                                                               | `8848`                                                                                          |
| `service.nodePort`           | Nacos http service nodeport.                                                                           | `30000`                                                                                         |
| `ingress.enabled`            | Enable or disable ingress functionality.                                                               | `false`                                                                                         |
| `ingress.annotations`        | Adding additional configuration and metadata information to ingress.                                   | `{}`                                                                                            |
| `ingress.labels`             | Add custom labels for Ingress objects.                                                                 | `{}`                                                                                            |
| `ingress.hosts`              | The host of nacos service in ingress rule.                                                             | `nacos.example.com`	                                                                            |
| `ingress.path`               | Ingress defines mapping rules between request paths and backend services.                              | `/`                                                                                             |
| `ingress.pathType`           | Ingress specifies the type of request path matching.                                                   | `Prefix`                                                                                        |
| `ingress.extraPaths`         | Ingress defines mapping rules between request extra paths and backend services.                        | `[]`                                                                                            |
| `ingress.tls`                | Configure TLS certificate information for encrypted connections received through Ingress objects.      | `[]`                                                                                            |
| `networkPolicy.enabled`      | Enable or disable networkpolicy functionality.                                                         | `false`                                                                                         |
| `networkPolicy.ingress`      | Enable or disable networkpolicy ingress.                                                               | `true`                                                                                          |
| `networkPolicy.egress.enabled` | Enable or disable networkpolicy egress.                                                                | `false`                                                                                         |
| `networkPolicy.egress.ports` | Enable or disable networkpolicy egress ports.                                                          | `80`                                                                                            |