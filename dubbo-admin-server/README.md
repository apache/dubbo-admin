# Refactor With Go
此分支是 Dubbo Admin 正在基于 Go 语言重构的开发分支，目前仍在开发过程中。
如您正寻求将 Dubbo Admin 用作生产环境，想了解 Admin 的能力及安装方式，请参见 [develop 分支](https://github.com/apache/dubbo-admin/tree/develop#dubbo-admin) 及内部相关使用说明。

# 重构版本本地开发说明
## 启动 Zookeeper
首先，你需要在本地启动一个 [zookeeper server](https://zookeeper.apache.org/doc/current/zookeeperStarted.html)，用作 Admin 连接的注册/配置中心。

## 启动 Admin
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

### Go
Once open this project in GoLand, a pre-configured Admin runnable task can be found from "Run Configuration" pop up menu as shown below.
![image.png](https://intranetproxy.alipay.com/skylark/lark/0/2023/png/54037/1677484872987-5a568293-74f9-4612-86c9-5c7112f3ac70.png#clientId=u4a56b9a9-a507-4&from=paste&height=165&id=ucdc7d17b&name=image.png&originHeight=330&originWidth=672&originalType=binary&ratio=2&rotation=0&showTitle=false&size=115664&status=done&style=none&taskId=u8b7fff84-e1b5-443a-9068-f67902132e5&title=&width=336)

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