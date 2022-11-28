# Dubbo Admin

![GitHub Workflow Status](https://img.shields.io/github/workflow/status/apache/dubbo-admin/CI)
[![codecov](https://codecov.io/gh/apache/dubbo-admin/branch/develop/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-admin/branches/develop)
![license](https://img.shields.io/github/license/apache/dubbo-admin.svg)
[![Average time to resolve an issue](http://isitmaintained.com/badge/resolution/apache/dubbo-admin.svg)](http://isitmaintained.com/project/apache/dubbo-admin "Average time to resolve an issue")
[![Percentage of issues still open](http://isitmaintained.com/badge/open/apache/dubbo-admin.svg)](http://isitmaintained.com/project/apache/dubbo-admin "Percentage of issues still open")

[中文说明](README_ZH.md)
# 1 Quick start

![index](https://raw.githubusercontent.com/apache/dubbo-admin/develop/doc/images/index.png)

## 1.1 Run With Helm

There are two ways to deploy Dubbo Admin distinguished by how you get the helm chart resources, any one of them is fine.

### 1.1.1 Run from helm chart sources
**1. Get helm chart sources**

Clone the Dubbo Admin repo:

```sh
$ git clone https://github.com/apache/dubbo-admin.git
```


From the repo's root directory, change your working directory to `deploy/helm/dubbo-admin`
```sh
$ cd /dubbo-admin/deploy/helm/dubbo-admin
```
**2. Install helm chart**

```sh
$ helm install dubbo-admin .
```

if you want to customize the configuration of Admin to let it connects to your registry or centers:
```sh
$ helm install dubbo-admin -f properties.yaml .
```
`properties.yaml` should contain the items in [application.properties]() that you want to override, refer to [example/xxxx]() for how to a demo.

**3. Visit Dubbo Admin**

Dubbo Admin should now has been successfully installed, run the following command or follow instruction of helm installation output to get the address of Admin:

```sh
$ kubectl --namespace default port-forward service/dubbo-admin 8080:8080
```

Open browser and visit http://127.0.0.1:8080

### 1.1.2 Run from helm chart repository
**1. Add helm chart repository**

```sh
$ helm repo add dubbo-charts https://dubbo.apache.org/dubbo-charts
$ helm repo update
```

**2. Install helm chart**
```sh
$ helm install dubbo-admin dubbo-charts/dubbo-admin
```

if you want to customize the configuration of Admin to let it connects to your registry or centers:
```sh
$ helm install dubbo-admin -f properties.yaml dubbo-charts/dubbo-admin
```
`properties.yaml` should contain the items in [application.properties]() that you want to override, refer to [example/xxxx]() for how to a demo.

**3. Visit Dubbo Admin**

Dubbo Admin should now has been successfully installed, run the following command or follow instruction of helm installation output to get the address of Admin:

```sh
$ kubectl --namespace default port-forward service/dubbo-admin 8080:8080
```

## 1.2 Run With Kubernetes

1. Get Kubernetes manifests
```sh
$ git clone https://github.com/apache/dubbo-admin.git
```

All we need from this step is the Admin kubernetes manifest files in `deploy/k8s`
```sh
$ cd /dubbo-admin/deploy/k8s
```

2. Deploy Dubbo Admin
```sh
# Change configuration in ./deploy/application.yml before apply
$ kubectl apply -f ./
```
3. Visit Admin
```sh
$ kubectl port-forward service dubbo-admin 8080:8080
```

Open web browser and visit `http://localhost:8080`, default username and password are `root`

## 1.3 Compile From Source
1. Clone source code on `develop` branch `git clone https://github.com/apache/dubbo-admin.git`
2. Specify registry address in `dubbo-admin-server/src/main/resources/application.properties`
3. Build
    - `mvn clean package -Dmaven.test.skip=true`
4. Start
    * `mvn --projects dubbo-admin-server spring-boot:run`
    OR
    * `cd dubbo-admin-distribution/target`;   `java -jar dubbo-admin-0.1.jar`
5. Visit `http://localhost:8080`, default username and password are `root`

# 2 Want To Contribute
service governance follows the version of Dubbo 2.7, and compatible for Dubbo 2.6, please refer to [here](https://github.com/apache/dubbo-admin/wiki/The-compatibility-of-service-governance)
## 2.1 admin UI

- [Vue.js](https://vuejs.org) and [Vue Cli](https://cli.vuejs.org/)
- [dubbo-admin-ui/README.md](dubbo-admin-ui/README.md) for more detail
- Set npm **proxy mirror**:

  if you have network issue, you can set npm proxy mirror to speedup npm install:

  add `registry=https://registry.npmmirror.com` to ~/.npmrc

## 2.2 admin Server

* Standard spring boot project
* [configurations in application.properties](https://github.com/apache/dubbo-admin/wiki/Dubbo-Admin-configuration)


## 2.3 Development Setup
* Run admin server project

  backend is a standard spring boot project, you can run it in any java IDE

* Run admin ui project

  run with `npm run dev`.

* visit web page

  visit `http://localhost:8080`, frontend supports hot reload.
  
## 2.4 Swagger support

Once deployed, you can check http://localhost:8080/swagger-ui.html to check all restful api and models


# 3 License

Apache Dubbo admin is under the Apache 2.0 license, Version 2.0.
See [LICENSE](https://github.com/apache/dubbo-admin/blob/develop/LICENSE) for full license text.
