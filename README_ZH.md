# Dubbo Admin

[![Build Status](https://travis-ci.org/apache/dubbo-admin.svg?branch=develop)](https://travis-ci.org/apache/dubbo-admin)
[![codecov](https://codecov.io/gh/apache/dubbo-admin/branch/develop/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-admin/branches/develop)
![license](https://img.shields.io/github/license/apache/dubbo-admin.svg)

[English version](README.md).

Dubbo Admin 是一个控制台，为 Dubbo 集群提供更好可视化服务。Admin 支持 Dubbo3 并很好的兼容 2.7.x、2.6.x 和 2.5.x。

![index](https://raw.githubusercontent.com/apache/dubbo-admin/develop/doc/images/index.png)


有四种将 Dubbo Admin 部署到生产环境的方式

1. [Linux 运行 Admin](#11-linux-运行-admin)
2. [Docker 运行 Admin](#12-docker-运行-admin)
3. [Kubernetes 运行 Admin](#13-kubernetes-运行-admin)
4. [Helm 运行 Admin](#14-helm-运行-admin)

基于环境选择任一方式，其中，Helm 是推荐的安装方式，因为 Helm 可以做到一键安装，自动帮助管理所有 Admin 必须的生产环境依赖。

## 1.1 Linux 运行 Admin

1. 下载代码: `git clone https://github.com/apache/dubbo-admin.git`
2. 在 `dubbo-admin-server/src/main/resources/application.properties`中指定注册中心地址
3. 构建
    - `mvn clean package -Dmaven.test.skip=true`
4. 启动
    * `mvn --projects dubbo-admin-server spring-boot:run`
      或者
    * `cd dubbo-admin-distribution/target; java -jar dubbo-admin-${project.version}.jar`
5. 访问 `http://localhost:38080`

## 1.2 Docker 运行 Admin
 Admin 镜像托管在： https://hub.docker.com/repository/docker/apache/dubbo-admin

  1、下面 `172.17.0.2` 注册中心地址是 docker 运行 zookeeper 注册中心地址，修改 `application.properties` 文件默认参数，如注册中心地址等。
  2、通过 `docker inspect` 获取 zookeeper 注册中心地址。
  3、将 `172.17.0.2` 注册中心地址修改成你当前 docker 运行 zookeeper 注册中心地址。
```
  admin.registry.address: zookeeper://172.17.0.2:2181
  admin.config-center: zookeeper://172.17.0.2:2181
  admin.metadata-report.address: zookeeper://172.17.0.2:2181
```

```sh
$ docker run -p 38080:38080 --name dubbo-admin -d dubbo-admin
```

打开浏览器并访问 `http://localhost:38080`

## 1.3 Kubernetes 运行 Admin

**1. 下载 Kubernetes manifests**
```sh
$ git clone https://github.com/apache/dubbo-admin.git
```

切换到 `deploy/k8s` 目录可以看到 Admin kubernetes 资源文件
```sh
$ cd /dubbo-admin/deploy/kubernetes
```

**2. 部署 Dubbo Admin**

修改 [application.properties](./dubbo-admin-server/src/main/resources/application.properties)  `configmap.yaml` 中的参数配置，只定义要覆盖参数即可。

执行以下命令：

```sh
$ kubectl apply -f ./
```

**3. 访问 Admin**
```sh
$ kubectl port-forward service dubbo-admin 38080:38080
```

打开浏览器并访问 `http://localhost:38080`


## 1.4 Helm 运行 Admin
通过 Helm 运行 Admin 有两种方式，它们起到相同的效果，因此可以选择以下任意一种。

### 1.4.1 基于 Chart 源文件运行 Admin
**1. 下载 chart 源文件**

克隆 Dubbo Admin 项目仓库:

```sh
$ git clone https://github.com/apache/dubbo-admin.git
```

从仓库根目录切换到以下目录 `deploy/charts/dubbo-admin`

```sh
$ cd dubbo-admin/deploy/charts/dubbo-admin
```
**2. 安装 helm chart**

Admin 的启动参数，以便让 Admin 连接到真实的生产环境注册中心或配置中心，可以通过以下 `-f` helm 参数指定自定义配置文件：

```yaml
properties:
  admin.registry.address: zookeeper://zookeeper:2181
  admin.config-center: zookeeper://zookeeper:2181
  admin.metadata-report.address: zookeeper://zookeeper:2181
```

`zookeeper://zookeeper:2181`  Kubernetes 集群注册中心 zookeeper 的访问地址。
```sh
$ helm install dubbo-admin -f values.yaml .
```

`properties` 字段指定的内容将会覆盖 Admin [application.properties](./dubbo-admin-server/src/main/resources/application.properties) 指定的默认配置，除了 `properties` 之外，还可以定制 Admin helm chart 定义的其他属性，这里是可供使用的[完整参数](./deploy/helm/dubbo-admin/values.yaml)。

**3. 访问 Admin**

打开浏览器并访问 http://127.0.0.1:38080

### 1.4.2 基于 Chart 仓库运行 Admin

**1. 添加 helm chart 仓库 (暂时不可用)**

```sh
$ helm repo add dubbo-charts https://dubbo.apache.org/dubbo-charts
$ helm repo update
```

**2. 安装 helm chart**
```sh
$ helm install dubbo-admin dubbo-charts/dubbo-admin
```

参考 [1.1.1 基于 Chart 源文件运行 Admin](1.1.1-Run-from-helm-chart-sources) 中相应环节了解如何定制安装参数

```sh
$ helm install dubbo-admin -f properties.yaml dubbo-charts/dubbo-admin
```

**3. 访问 Dubbo Admin**

Dubbo Admin 现在应该已经成功安装，运行以下命令获得访问地址:

```sh
$ kubectl --namespace default port-forward service/dubbo-admin 38080:38080
```

打开浏览器并访问 http://127.0.0.1:38080

# 2 参与项目贡献

以下是项目架构介绍，适合想贡献源码的开发者阅读。

## 2.1 前端部分

- 使用[Vue.js](https://vuejs.org)作为javascript框架
- [dubbo-admin-ui/README.md](dubbo-admin-ui/README.md)中有更详细的介绍
- 设置 npm **代理镜像** :

    如果遇到了网络问题，可以设置npm代理镜像来加速npm install的过程：

    在~/.npmrc中增加 `registry=https://registry.npmmirror.com`

## 2.2 后端部分

* 标准spring boot工程
* [application.properties配置说明](https://github.com/apache/dubbo-admin/wiki/Dubbo-Admin%E9%85%8D%E7%BD%AE%E8%AF%B4%E6%98%8E)

### 2.2.1 开发环境配置
* 运行`dubbo-admin-server`

  `dubbo-admin-server`是一个标准的spring boot项目, 可以在任何java IDE中运行它

* 运行`dubbo-admin-ui`

  `dubbo-admin-ui`由npm管理和构建，在`dubbo-admin-ui`目录中，可以单独运行: `npm run dev`

* 页面访问

  访问 `http://localhost:38082`, 由于前后端分开部署，前端支持热加载，任何页面的修改都可以实时反馈，不需要重启应用。

### 2.2.2 Swagger 支持

部署完成后，可以访问 http://localhost:38080/swagger-ui.html 来查看所有的restful api
