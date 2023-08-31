# Dubbo Admin

![GitHub Workflow Status](https://img.shields.io/github/workflow/status/apache/dubbo-admin/CI)
[![codecov](https://codecov.io/gh/apache/dubbo-admin/branch/develop/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-admin/branches/develop)
![license](https://img.shields.io/github/license/apache/dubbo-admin.svg)
[![Average time to resolve an issue](http://isitmaintained.com/badge/resolution/apache/dubbo-admin.svg)](http://isitmaintained.com/project/apache/dubbo-admin "Average time to resolve an issue")
[![Percentage of issues still open](http://isitmaintained.com/badge/open/apache/dubbo-admin.svg)](http://isitmaintained.com/project/apache/dubbo-admin "Percentage of issues still open")

[中文说明](README_ZH.md)

Dubbo Admin is the console designed for better visualization of Dubbo services, it provides support for Dubbo3 and is compatible with 2.7.x, 2.6.x and 2.5.x.

![index](https://raw.githubusercontent.com/apache/dubbo-admin/develop/doc/images/index.png)

There are four ways to deploy Dubbo Admin to a production environment.

1. [Linux with Admin](#11-linux-with-admin)
2. [Docker with Admin](#12-docker-with-admin)
3. [Kubernetes with Admin](#13-kubernetes-with-admin)
4. [Helm with Admin](#14-helm-with-admin)

Choose either method based on your environment, where Helm is the recommended installation method because Helm can be installed with a single click and automatically helps manage all of Admin's required production environment dependencies.

## 1.1 Linux with Admin

1. Download code: `git clone https://github.com/apache/dubbo-admin.git`
2. `dubbo-admin-server/src/main/resources/application.properties` Designated Registration Center Address
3. Build
    - `mvn clean package -Dmaven.test.skip=true`
4. Start
    * `mvn --projects dubbo-admin-server spring-boot:run`
      or
    * `cd dubbo-admin-distribution/target; java -jar dubbo-admin-${project.version}.jar`
5. Visit  `http://localhost:38080`

## 1.2 Docker with Admin
Admin image hosting at： https://hub.docker.com/repository/docker/apache/dubbo-admin

  1, the following `172.17.0.2` registry address is the docker run zookeeper registry address, modify the `application.properties` file default parameters, such as registry address, etc.
  2、Get the zookeeper registry address through `docker inspect`.
  3.Change `172.17.0.2` registry address to your current docker running zookeeper registry address.
```
  admin.registry.address: zookeeper://172.17.0.2:2181
  admin.config-center: zookeeper://172.17.0.2:2181
  admin.metadata-report.address: zookeeper://172.17.0.2:2181
```
docker start
```sh
$ docker run -p 38080:38080 --name dubbo-admin -d dubbo-admin
```

Visit `http://localhost:38080`

## 1.3 Kubernetes with Admin

**1. Download Kubernetes manifests**
```sh
$ git clone https://github.com/apache/dubbo-admin.git
```

Switch to the 'deploy/k8s' directory to see the Admin kubernetes resource file
```sh
$ cd /dubbo-admin/deploy/kubernetes
```

**2. Install Dubbo Admin**

modify [application.properties](./dubbo-admin-server/src/main/resources/application.properties)  Parameter configuration in `configmap.yaml` ，Just define the parameters to be overwritten。

Run the following command：

```sh
$ kubectl apply -f ./
```

**3. Visit Admin**
```sh
$ kubectl port-forward service dubbo-admin 38080:38080
```

Visit `http://localhost:38080`


## 1.4 Helm with Admin
There are two ways to run Admin through Help. They have the same effect, so you can choose any of the following.

### 1.4.1 Run Admin based on Chart source file
**1. Download chart source file**

clone Dubbo Admin project storehouse:

```sh
$ git clone https://github.com/apache/dubbo-admin.git
```

Switch from the warehouse root directory to the following directory `deploy/charts/dubbo-admin`

```sh
$ cd dubbo-admin/deploy/charts/dubbo-admin
```
**2. Install helm chart**

Start parameters of Admin so that Admin can connect to the real production environment registry or configuration center. You can specify a custom configuration file through the following `-f` help parameter:
```yaml
properties:
  admin.registry.address: zookeeper://zookeeper:2181
  admin.config-center: zookeeper://zookeeper:2181
  admin.metadata-report.address: zookeeper://zookeeper:2181
```

`zookeeper://zookeeper:2181`  Visit address of the Kubernetes Cluster registration center zookeeper。
```sh
$ helm install dubbo-admin -f values.yaml .
```

`properties` The content specified in the field will be overwritten Admin [application.properties](./dubbo-admin-server/src/main/resources/application.properties) Specified default configuration，In addition to 'properties', you can customize other properties defined by Admin help chart，Here is available[Complete parameters](./deploy/helm/dubbo-admin/values.yaml)。

**3. Visit Admin**

Visit http://127.0.0.1:38080

### 1.4.2 Run Admin based on Chart warehouse

**1. Add helm chart  (Temporarily unavailable)**

```sh
$ helm repo add dubbo-charts https://dubbo.apache.org/dubbo-charts
$ helm repo update
```

**2. Install helm chart**
```sh
$ helm install dubbo-admin dubbo-charts/dubbo-admin
```

reference resources [1.4.1 Run Admin based on Chart warehouse](1.4.1-Run-from-helm-chart-sources) Learn how to customize installation parameters.

```sh
$ helm install dubbo-admin -f properties.yaml dubbo-charts/dubbo-admin
```

**3. Visit Dubbo Admin**

Dubbo Admin Now that the installation should be successful, run the following command to obtain the access address:

```sh
$ kubectl --namespace default port-forward service/dubbo-admin 38080:38080
```

Visit http://127.0.0.1:38080

# 2. Want To Contribute

Below contains the description of the project structure for developers who want to contribute to make Dubbo Admin better.

## 2.1 Admin UI

- [Vue.js](https://vuejs.org) and [Vue Cli](https://cli.vuejs.org/)
- [dubbo-admin-ui/README.md](dubbo-admin-ui/README.md) for more detail
- Set npm **proxy mirror**:

  If you have network issue, you can set npm proxy mirror to speedup npm install:

  add `registry=https://registry.npmmirror.com` to ~/.npmrc

## 2.2 Admin Server

* Standard spring boot project
* [configurations in application.properties](https://github.com/apache/dubbo-admin/wiki/Dubbo-Admin-configuration)


## 2.3 Setting up a local developing environment
* Run admin server project

  backend is a standard spring boot project, you can run it in any java IDE

* Run admin ui project

  at directory `dubbo-admin-ui`, run with `npm run dev`.

* visit web page

  visit `http://localhost:38082`, frontend supports hot reload.

# 3 License

Apache Dubbo admin is under the Apache 2.0 license, Version 2.0.
See [LICENSE](https://github.com/apache/dubbo-admin/blob/develop/LICENSE) for full license text.
