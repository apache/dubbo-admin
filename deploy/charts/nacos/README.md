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

| Parameter                             | Description                                                        | Default                             |
|---------------------------------------|--------------------------------------------------------------------|-------------------------------------|
| `global.mode`                         | Run Mode (~~quickstart,~~ standalone, cluster; )   | `standalone`            |
| `resources`                          | The [resources] to allocate for nacos container                    | `{}`                                |
| `nodeSelector`                       | Nacos labels for pod assignment                   | `{}`                                |
| `affinity`                           | Nacos affinity policy                                              | `{}`                                |
| `tolerations`                         | Nacos tolerations                                                  | `{}`                                |
| `resources.requests.cpu`|nacos requests cpu resource|`500m`|
| `resources.requests.memory`|nacos requests memory resource|`2G`|
| `nacos.replicaCount`                        | Number of desired nacos pods, the number should be 1 as run standalone mode| `1`           |
| `nacos.image.repository`                    | Nacos container image name                                      | `nacos/nacos-server`                   |
| `nacos.image.tag`                           | Nacos container image tag                                       | `latest`                                |
| `nacos.image.pullPolicy`                    | Nacos container image pull policy                                | `IfNotPresent`                        |
| `nacos.plugin.enable`                    | Nacos cluster plugin that is auto scale                                       | `true`                   |
| `nacos.plugin.image.repository`                    | Nacos cluster plugin image name                                      | `nacos/nacos-peer-finder-plugin`                   |
| `nacos.plugin.image.tag`                           | Nacos cluster plugin image tag                                       | `1.1`                                |
| `nacos.health.enabled`                      | Enable health check or not                                         | `false`                              |
| `nacos.env.preferhostmode`                  | Enable Nacos cluster node domain name support                      | `hostname`                         |
| `nacos.env.serverPort`                      | Nacos port                                                         | `8848`                               |
| `nacos.storage.type`                      | Nacos data storage method `mysql` or `embedded`. The `embedded` supports either standalone or cluster mode                                                       | `embedded`                               |
| `nacos.storage.db.host`                      | mysql  host                                                       |                                |
| `nacos.storage.db.name`                      | mysql  database name                                                      |                                |
| `nacos.storage.db.port`                      | mysql port                                                       | 3306                               |
| `nacos.storage.db.username`                      | username of  database                                                       |                               |
| `nacos.storage.db.password`                      | password of  database                                                       |                               |
| `nacos.storage.db.param`                      | Database url parameter                                                       | `characterEncoding=utf8&connectTimeout=1000&socketTimeout=3000&autoReconnect=true&useSSL=false`                               |
| `persistence.enabled`                 | Enable the nacos data persistence or not                           | `false`                              |
| `persistence.data.accessModes`					| Nacos data pvc access mode										| `ReadWriteOnce`		|
| `persistence.data.storageClassName`				| Nacos data pvc storage class name									| `manual`			|
| `persistence.data.resources.requests.storage`		| Nacos data pvc requests storage									| `5G`					|
| `service.type`									| http service type													| `NodePort`			|
| `service.port`									| http service port													| `8848`				|
| `service.nodePort`								| http service nodeport												| `30000`				|
| `ingress.enabled`									| Enable ingress or not												| `false`				|
| `ingress.annotations`								| The annotations used in ingress									| `{}`					|
| `ingress.hosts`									| The host of nacos service in ingress rule							| `nacos.example.com`	|
