# 运行 Admin
## 启动 Zookeeper
首先，你需要在本地启动一个 [zookeeper server](https://zookeeper.apache.org/doc/current/zookeeperStarted.html)，用作 Admin 连接的注册/配置中心。

## 启动 Admin

### Run with IDE
Once open this project in GoLand, a pre-configured Admin runnable task can be found from "Run Configuration" pop up menu as shown below.

![image.png](../../docs/images/ide_configuration.png)

Click the `Run` button and you can get the Admin process started locally. 

> But before doing that, you might need to change the configuration file located at `/conf/dubboadmin.yml` to make sure `registry.address` is pointed to the zookeeper server you started before.

```yaml
admin:
  registry:
    address: zookeeper://127.0.0.1:2181
  config-center: zookeeper://127.0.0.1:2181
  metadata-report:
    address: zookeeper://127.0.0.1:2181
```

### Run with command line 
```shell
$ export ADMIN_CONFIG_PATH=/path/to/your/admin/project/conf/admin.yml
$ cd cmd/admin
$ go run . 
```

> If you also have the Java version admin running, make sure to use different port to avoid conflict.

## 启动示例
为了能在 Admin 控制台看到一些示例数据，可以在本地启动一些示例项目。可参考以下两个链接，务必确保示例使用的注册中心指向你之前启动的 zookeeper server，如果示例中有使用 embeded zookeeper 则应该进行修改并指向你本地起的 zookeeper 集群。

1. https://github.com/apache/dubbo-samples/tree/master/1-basic/dubbo-samples-spring-boot
2. https://dubbo.apache.org/zh-cn/overview/quickstart/java/brief/