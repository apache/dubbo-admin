# Refactor With Go
此分支是 Dubbo Admin 正在基于 Go 语言重构的开发分支，目前仍在开发过程中。
如您正寻求将 Dubbo Admin 用作生产环境，想了解 Admin 的能力及安装方式，请参见 [develop 分支](https://github.com/apache/dubbo-admin/tree/develop#dubbo-admin) 及内部相关使用说明。

# 重构版本本地开发说明
## 启动 Zookeeper
首先，你需要在本地启动一个 [zookeeper server](https://zookeeper.apache.org/doc/current/zookeeperStarted.html)，用作 Admin 连接的注册/配置中心。

## 启动 Admin
1. Java 版本。Java 版本作为对原有功能的参考。
2. Go 版本。Go 版本是当前实际要开发的版本。

### Java

1. Clone source code on `develop` branch
```shell
git clone -b develop https://github.com/apache/dubbo-admin.git
```

2. [Optional] Specify registry address in `dubbo-admin-server/src/main/resources/application.properties`
3. Build `mvn clean package -Dmaven.test.skip=true`
4. Start
   - `mvn --projects dubbo-admin-server spring-boot:run`
     OR
   - `cd dubbo-admin-distribution/target & java -jar dubbo-admin-{the-package-version}.jar`
5. Visit `http://localhost:38080`, default username and password are `root`

可以在此查看 [Java 版本的更多启动方式](../README.md)。

### Go
Once open this project in GoLand, a pre-configured Admin runnable task can be found from "Run Configuration" pop up menu as shown below.

![image.png](../doc/images/ide_configuration.png)

Click the `Run`button and you can get the Admin process started locally. But before doing that, you might need to change the configuration file located at `dubbo-admin/dubbo-admin-server/pkg/conf/dubboadmin.yml`to make sure `registry.address` is pointed to the zookeeper server you started before.

```yaml
admin:
  registry:
    address: zookeeper://127.0.0.1:2181
  config-center: zookeeper://127.0.0.1:2181
  metadata-report:
    address: zookeeper://127.0.0.1:2181
```

> If you also have the Java version admin running, make sure to use different port to avoid confliction.

## 启动示例
为了能在 Admin 控制台看到一些示例数据，可以在本地启动一些示例项目。可参考以下两个链接，务必确保示例使用的注册中心指向你之前启动的 zookeeper server，如果示例中有使用 embeded zookeeper 则应该进行适当修改。

1. https://github.com/apache/dubbo-samples/tree/master/1-basic/dubbo-samples-spring-boot
2. https://dubbo.apache.org/zh-cn/overview/quickstart/java/brief/